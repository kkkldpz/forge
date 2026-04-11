package types

import "encoding/json"

// StreamEvent 是 API SSE 流事件接口，所有流事件都实现此接口。
type StreamEvent interface {
	EventType() string
}

// MessageStartEvent 在新消息开始时触发。
type MessageStartEvent struct {
	ID    string          `json:"id"`
	Type  string          `json:"type"`
	Role  string          `json:"role"`
	Model string          `json:"model"`
	Usage json.RawMessage `json:"usage,omitempty"`
}

func (e *MessageStartEvent) EventType() string { return "message_start" }

// ContentBlockStartEvent 在新内容块开始时触发。
type ContentBlockStartEvent struct {
	Index int    `json:"index"`
	Type  string `json:"type"` // "text", "tool_use", "thinking"
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Text  string `json:"text,omitempty"`
}

func (e *ContentBlockStartEvent) EventType() string { return "content_block_start" }

// ContentBlockDeltaEvent 携带增量内容更新。
type ContentBlockDeltaEvent struct {
	Index       int    `json:"index"`
	DeltaType   string `json:"type"` // "text_delta", "input_json_delta", "thinking_delta"
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
}

func (e *ContentBlockDeltaEvent) EventType() string { return "content_block_delta" }

// ContentBlockStopEvent 在内容块结束时触发。
type ContentBlockStopEvent struct {
	Index int `json:"index"`
}

func (e *ContentBlockStopEvent) EventType() string { return "content_block_stop" }

// MessageDeltaEvent 携带消息级别的元数据更新。
type MessageDeltaEvent struct {
	StopReason string          `json:"stop_reason,omitempty"`
	Usage      json.RawMessage `json:"usage,omitempty"`
}

func (e *MessageDeltaEvent) EventType() string { return "message_delta" }

// MessageStopEvent 在消息完成时触发。
type MessageStopEvent struct{}

func (e *MessageStopEvent) EventType() string { return "message_stop" }

// PingEvent 是保活事件。
type PingEvent struct{}

func (e *PingEvent) EventType() string { return "ping" }

// ErrorEvent 携带流式传输过程中的 API 错误。
type ErrorEvent struct {
	Error ErrorDetail `json:"error"`
}

func (e *ErrorEvent) EventType() string { return "error" }

// ErrorDetail 描述 API 错误详情。
type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// QueryEvent 是查询循环产出的内部事件类型。
type QueryEvent struct {
	Type    QueryEventType `json:"type"`
	Message any            `json:"message,omitempty"`
}

// QueryEventType 枚举查询循环产出的所有事件类型。
type QueryEventType string

const (
	QueryEventMessageStart QueryEventType = "message_start"
	QueryEventContentDelta QueryEventType = "content_delta"
	QueryEventToolUse      QueryEventType = "tool_use"
	QueryEventToolResult   QueryEventType = "tool_result"
	QueryEventMessageStop  QueryEventType = "message_stop"
	QueryEventError        QueryEventType = "error"
	QueryEventTurnStart    QueryEventType = "turn_start"
	QueryEventTurnEnd      QueryEventType = "turn_end"
	QueryEventCompactStart QueryEventType = "compact_start"
	QueryEventCompactEnd   QueryEventType = "compact_end"
)
