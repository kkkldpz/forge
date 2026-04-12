// Package query 实现流事件处理。
package query

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/types"
)

// StreamProcessor 处理流式 API 响应，累积 content blocks。
type StreamProcessor struct {
	partialMessage    *types.MessageContent
	contentBlocks     []map[string]any
	currentBlockIndex int
	currentBlockType  string
	currentBlockData  map[string]any
	usage             api.Usage
	stopReason        string
}

// NewStreamProcessor 创建新的流处理器。
func NewStreamProcessor() *StreamProcessor {
	return &StreamProcessor{
		contentBlocks: make([]map[string]any, 0),
		currentBlockData: make(map[string]any),
	}
}

// HandleMessageStart 处理 message_start 事件。
func (p *StreamProcessor) HandleMessageStart(evt *types.MessageStartEvent) {
	p.partialMessage = &types.MessageContent{
		Role:    evt.Role,
		ID:      evt.ID,
		Content: json.RawMessage("[]"),
	}
}

// HandleContentBlockStart 处理 content_block_start 事件。
func (p *StreamProcessor) HandleContentBlockStart(evt *types.ContentBlockStartEvent) {
	p.currentBlockIndex = evt.Index
	p.currentBlockType = evt.Type
	p.currentBlockData = map[string]any{
		"type": evt.Type,
	}

	// 根据类型初始化
	switch evt.Type {
	case "text":
		p.currentBlockData["text"] = ""
	case "tool_use":
		p.currentBlockData["id"] = evt.ID
		p.currentBlockData["name"] = evt.Name
		p.currentBlockData["input"] = ""
	case "thinking":
		p.currentBlockData["thinking"] = ""
		p.currentBlockData["signature"] = ""
	}
}

// HandleContentBlockDelta 处理 content_block_delta 事件。
func (p *StreamProcessor) HandleContentBlockDelta(evt *types.ContentBlockDeltaEvent) {
	switch evt.DeltaType {
	case "text_delta":
		if text, ok := p.currentBlockData["text"].(string); ok {
			p.currentBlockData["text"] = text + evt.Text
		} else {
			p.currentBlockData["text"] = evt.Text
		}
	case "input_json_delta":
		if input, ok := p.currentBlockData["input"].(string); ok {
			p.currentBlockData["input"] = input + evt.PartialJSON
		} else {
			p.currentBlockData["input"] = evt.PartialJSON
		}
	case "thinking_delta":
		if thinking, ok := p.currentBlockData["thinking"].(string); ok {
			p.currentBlockData["thinking"] = thinking + evt.Text
		} else {
			p.currentBlockData["thinking"] = evt.Text
		}
	}
}

// HandleContentBlockStop 处理 content_block_stop 事件。
func (p *StreamProcessor) HandleContentBlockStop(evt *types.ContentBlockStopEvent) {
	// 解析 tool_use 的 input JSON
	if p.currentBlockType == "tool_use" {
		if inputStr, ok := p.currentBlockData["input"].(string); ok && inputStr != "" {
			var inputMap map[string]any
			if err := json.Unmarshal([]byte(inputStr), &inputMap); err == nil {
				p.currentBlockData["input"] = inputMap
			}
		}
	}

	// 添加到 blocks 列表
	p.contentBlocks = append(p.contentBlocks, p.currentBlockData)
	p.currentBlockData = make(map[string]any)
}

// HandleMessageDelta 处理 message_delta 事件。
func (p *StreamProcessor) HandleMessageDelta(evt *types.MessageDeltaEvent) {
	p.stopReason = evt.StopReason
	// 解析 usage
	if evt.Usage != nil {
		json.Unmarshal(evt.Usage, &p.usage)
	}
}

// BuildAssistantMessage 从累积的数据构建完整的 assistant message。
func (p *StreamProcessor) BuildAssistantMessage() *types.Message {
	if p.partialMessage == nil {
		return nil
	}

	// 构建 content JSON
	contentJSON, _ := json.Marshal(p.contentBlocks)

	return &types.Message{
		Type: types.MessageTypeAssistant,
		UUID: uuid.New().String(),
		Message: &types.MessageContent{
			Role:       "assistant",
			ID:         p.partialMessage.ID,
			Content:    contentJSON,
			Usage:      mustMarshal(p.usage),
		},
	}
}

// GetUsage 返回累积的用量信息。
func (p *StreamProcessor) GetUsage() api.Usage {
	return p.usage
}

// GetStopReason 返回停止原因。
func (p *StreamProcessor) GetStopReason() string {
	return p.stopReason
}

// mustMarshal 将值序列化为 JSON，失败时返回 nil。
func mustMarshal(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
