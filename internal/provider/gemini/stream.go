package gemini

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/kkkldpz/forge/internal/types"
)

// AdaptStream 读取 Gemini SSE 响应体，将其转换为 Anthropic 格式的 StreamEvent channel。
func AdaptStream(respBody io.Reader) <-chan types.StreamEvent {
	ch := make(chan types.StreamEvent, 64)
	go func() {
		defer close(ch)
		adaptGeminiSSE(respBody, ch)
	}()
	return ch
}

// adaptGeminiSSE 逐行解析 Gemini SSE 数据并发射 Anthropic 流事件。
func adaptGeminiSSE(body io.Reader, ch chan<- types.StreamEvent) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	messageStarted := false
	blockIndex := 0
	openBlocks := make(map[int]*geminiBlockState)
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

		// 解析 JSON 块
		var resp GenerateContentResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			ch <- &types.ErrorEvent{Error: types.ErrorDetail{
				Type:    "parse_error",
				Message: fmt.Sprintf("解析 Gemini SSE 块失败: %v", err),
			}}
			continue
		}

		if len(resp.Candidates) == 0 {
			continue
		}

		candidate := resp.Candidates[0]
		content := candidate.Content
		if len(content.Parts) == 0 {
			continue
		}

		// 发射 message_start（仅一次）
		if firstChunk {
			messageStarted = true
			ch <- &types.MessageStartEvent{
				ID:    "gemini-msg",
				Type:  "message",
				Role:  "assistant",
				Model: "gemini",
			}
			firstChunk = false
		}

		// 处理每个 part
		for _, part := range content.Parts {
			switch {
			case part.FunctionCall != nil:
				handleGeminiFunctionCall(part.FunctionCall, openBlocks, &blockIndex, ch)
			case part.Text != "" && part.Thought:
				// thinking 块
				ensureThinkingBlock(openBlocks, &blockIndex, ch)
				ch <- &types.ContentBlockDeltaEvent{
					Index:     blockIndex - 1,
					DeltaType: "thinking_delta",
					Text:      part.Text,
				}
			case part.Text != "":
				// 普通文本块
				ensureTextBlock(openBlocks, &blockIndex, ch)
				ch <- &types.ContentBlockDeltaEvent{
					Index:     blockIndex - 1,
					DeltaType: "text_delta",
					Text:      part.Text,
				}
			}
		}

		// 处理 finishReason
		if candidate.FinishReason != "" {
			closeGeminiOpenBlocks(openBlocks, blockIndex, ch)

			stopReason := mapGeminiFinishReason(candidate.FinishReason)

			usageJSON := geminiUsageToJSON(resp.UsageMetadata)
			ch <- &types.MessageDeltaEvent{
				StopReason: stopReason,
				Usage:      usageJSON,
			}
			ch <- &types.MessageStopEvent{}
			return
		}
	}

	// 扫描结束但未收到 finishReason——关闭所有打开的块
	closeGeminiOpenBlocks(openBlocks, blockIndex, ch)
	if messageStarted {
		ch <- &types.MessageStopEvent{}
	}
}

// geminiBlockState 跟踪当前打开的流式内容块。
type geminiBlockState struct {
	Type       string // "text", "tool_use", "thinking"
	ToolName   string
	ArgsBuf    strings.Builder
}

// ensureTextBlock 确保文本内容块已打开。
func ensureTextBlock(openBlocks map[int]*geminiBlockState, blockIndex *int, ch chan<- types.StreamEvent) {
	lastIdx := *blockIndex - 1
	if lastIdx >= 0 {
		if state, ok := openBlocks[lastIdx]; ok && state.Type == "text" {
			return
		}
	}

	// 关闭之前的非文本块
	for idx, state := range openBlocks {
		if state.Type != "text" {
			ch <- &types.ContentBlockStopEvent{Index: idx}
			delete(openBlocks, idx)
		}
	}

	idx := *blockIndex
	*blockIndex++
	openBlocks[idx] = &geminiBlockState{Type: "text"}
	ch <- &types.ContentBlockStartEvent{
		Index: idx,
		Type:  "text",
	}
}

// ensureThinkingBlock 确保 thinking 内容块已打开。
func ensureThinkingBlock(openBlocks map[int]*geminiBlockState, blockIndex *int, ch chan<- types.StreamEvent) {
	lastIdx := *blockIndex - 1
	if lastIdx >= 0 {
		if state, ok := openBlocks[lastIdx]; ok && state.Type == "thinking" {
			return
		}
	}

	// 关闭之前的非 thinking 块
	for idx, state := range openBlocks {
		if state.Type != "thinking" {
			ch <- &types.ContentBlockStopEvent{Index: idx}
			delete(openBlocks, idx)
		}
	}

	idx := *blockIndex
	*blockIndex++
	openBlocks[idx] = &geminiBlockState{Type: "thinking"}
	ch <- &types.ContentBlockStartEvent{
		Index: idx,
		Type:  "thinking",
	}
}

// handleGeminiFunctionCall 处理 Gemini 的函数调用。
func handleGeminiFunctionCall(
	fc *FunctionCall,
	openBlocks map[int]*geminiBlockState,
	blockIndex *int,
	ch chan<- types.StreamEvent,
) {
	// 关闭所有非 tool_use 块
	for idx, state := range openBlocks {
		if state.Type != "tool_use" {
			ch <- &types.ContentBlockStopEvent{Index: idx}
			delete(openBlocks, idx)
		}
	}

	idx := *blockIndex
	*blockIndex++
	openBlocks[idx] = &geminiBlockState{Type: "tool_use", ToolName: fc.Name}

	// Gemini 的 functionCall 在单个 chunk 中完整到达，直接发射 start + delta + stop
	ch <- &types.ContentBlockStartEvent{
		Index: idx,
		Type:  "tool_use",
		ID:    fc.Name, // Gemini 没有独立的调用 ID，使用函数名
		Name:  fc.Name,
	}

	if len(fc.Args) > 0 {
		ch <- &types.ContentBlockDeltaEvent{
			Index:       idx,
			DeltaType:   "input_json_delta",
			PartialJSON: string(fc.Args),
		}
	}

	ch <- &types.ContentBlockStopEvent{Index: idx}
	delete(openBlocks, idx)
}

// closeGeminiOpenBlocks 关闭所有仍处于打开状态的内容块。
func closeGeminiOpenBlocks(openBlocks map[int]*geminiBlockState, blockIndex int, ch chan<- types.StreamEvent) {
	for i := 0; i < blockIndex; i++ {
		if _, ok := openBlocks[i]; ok {
			ch <- &types.ContentBlockStopEvent{Index: i}
			delete(openBlocks, i)
		}
	}
}

// mapGeminiFinishReason 将 Gemini 的 finishReason 映射为 Anthropic 的 stop_reason。
func mapGeminiFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "end_turn"
	case "MAX_TOKENS":
		return "max_tokens"
	case "SAFETY":
		return "end_turn"
	case "RECITATION":
		return "end_turn"
	case "TOOL_CALLS":
		return "tool_use"
	default:
		return "end_turn"
	}
}

// geminiUsageToJSON 将 UsageMetadata 转换为 Anthropic 格式的 JSON。
func geminiUsageToJSON(usage *UsageMetadata) json.RawMessage {
	if usage == nil {
		return nil
	}
	data, _ := json.Marshal(struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	}{
		InputTokens:  usage.PromptTokenCount,
		OutputTokens: usage.CandidatesTokenCount,
	})
	return data
}
