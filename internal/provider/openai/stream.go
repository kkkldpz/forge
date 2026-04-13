package openai

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/kkkldpz/forge/internal/types"
)

// AdaptStream 读取 OpenAI SSE 响应体，将其转换为 Anthropic 格式的 StreamEvent channel。
func AdaptStream(respBody io.Reader) <-chan types.StreamEvent {
	ch := make(chan types.StreamEvent, 64)
	go func() {
		defer close(ch)
		adaptSSELoop(respBody, ch)
	}()
	return ch
}

// adaptSSELoop 逐行解析 OpenAI SSE 数据并发射 Anthropic 流事件。
func adaptSSELoop(body io.Reader, ch chan<- types.StreamEvent) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// 跟踪流式状态
	messageStarted := false
	blockIndex := 0
	openBlocks := make(map[int]*StreamBlockState) // index -> state
	firstChunk := true

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 解析 "data: ..." 格式
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// 流结束标记
		if data == "[DONE]" {
			// 关闭所有打开的内容块
			closeOpenBlocks(openBlocks, blockIndex, ch)
			ch <- &types.MessageStopEvent{}
			return
		}

		// 解析 JSON 块
		var chunk ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			ch <- &types.ErrorEvent{Error: types.ErrorDetail{
				Type:    "parse_error",
				Message: fmt.Sprintf("解析 OpenAI SSE 块失败: %v", err),
			}}
			continue
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		choice := chunk.Choices[0]
		delta := choice.Delta

		// 发射 message_start（仅一次）
		if firstChunk && (delta.Role != "" || delta.Content != nil) {
			messageStarted = true
			ch <- &types.MessageStartEvent{
				ID:    chunk.ID,
				Type:  "message",
				Role:  "assistant",
				Model: chunk.Model,
			}
			firstChunk = false
		}

		// 处理文本增量
		if delta.Content != nil && *delta.Content != "" {
			ensureTextBlock(openBlocks, &blockIndex, ch)
			ch <- &types.ContentBlockDeltaEvent{
				Index:     blockIndex - 1, // 当前文本块的 index
				DeltaType: "text_delta",
				Text:      *delta.Content,
			}
		}

		// 处理工具调用增量
		for _, tc := range delta.ToolCalls {
			handleToolCallDelta(tc, openBlocks, &blockIndex, ch)
		}

		// 处理 finish_reason
		if choice.FinishReason != nil {
			closeOpenBlocks(openBlocks, blockIndex, ch)
			ch <- &types.MessageDeltaEvent{
				StopReason: mapFinishReason(choice.FinishReason),
				Usage:      UsageToJSON(chunk.Usage),
			}
			ch <- &types.MessageStopEvent{}
			return
		}
	}

	// 扫描结束但未收到 [DONE]——关闭所有打开的块
	closeOpenBlocks(openBlocks, blockIndex, ch)
	if messageStarted {
		ch <- &types.MessageStopEvent{}
	}
}

// ensureTextBlock 确保文本内容块已打开。如果没有打开的文本块，则打开一个。
func ensureTextBlock(openBlocks map[int]*StreamBlockState, blockIndex *int, ch chan<- types.StreamEvent) {
	// 查找最后一个块是否是 text
	lastIdx := *blockIndex - 1
	if lastIdx >= 0 {
		if state, ok := openBlocks[lastIdx]; ok && state.Type == "text" {
			return // 已有打开的文本块
		}
	}

	// 关闭之前的非文本块
	for idx, state := range openBlocks {
		if state.Type != "text" {
			ch <- &types.ContentBlockStopEvent{Index: idx}
			delete(openBlocks, idx)
		}
	}

	// 打开新的文本块
	idx := *blockIndex
	*blockIndex++
	openBlocks[idx] = &StreamBlockState{Type: "text"}
	ch <- &types.ContentBlockStartEvent{
		Index: idx,
		Type:  "text",
	}
}

// handleToolCallDelta 处理工具调用的增量数据。
// OpenAI 的 tool_calls 通过 index 标识，且可能是分片到达的。
func handleToolCallDelta(tc ToolCall, openBlocks map[int]*StreamBlockState, blockIndex *int, ch chan<- types.StreamEvent) {
	idx := tc.Index
	state, exists := openBlocks[idx]

	if !exists {
		// 新的工具调用，打开新块
		*blockIndex = max(*blockIndex, idx+1)
		state = &StreamBlockState{
			Type:       "tool_use",
			ToolCallID: tc.ID,
			ToolName:   tc.Function.Name,
		}
		openBlocks[idx] = state

		ch <- &types.ContentBlockStartEvent{
			Index: idx,
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Function.Name,
		}
	}

	// 追加函数名（某些模型会分片发送函数名）
	if tc.Function.Name != "" && state.ToolName != "" && tc.Function.Name != state.ToolName {
		state.ToolName = tc.Function.Name
	}

	// 追加参数
	if tc.Function.Arguments != "" {
		ch <- &types.ContentBlockDeltaEvent{
			Index:       idx,
			DeltaType:   "input_json_delta",
			PartialJSON: tc.Function.Arguments,
		}
		state.ArgsBuf.WriteString(tc.Function.Arguments)
	}

	// 补充 ID（某些模型在后续块中才发送 ID）
	if tc.ID != "" && state.ToolCallID == "" {
		state.ToolCallID = tc.ID
	}
}

// closeOpenBlocks 关闭所有仍处于打开状态的内容块。
func closeOpenBlocks(openBlocks map[int]*StreamBlockState, blockIndex int, ch chan<- types.StreamEvent) {
	// 按 index 顺序关闭
	for i := 0; i < blockIndex; i++ {
		if _, ok := openBlocks[i]; ok {
			ch <- &types.ContentBlockStopEvent{Index: i}
			delete(openBlocks, i)
		}
	}
}
