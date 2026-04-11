package types

import "encoding/json"

// MessageType 区分对话消息的类型。
type MessageType string

const (
	MessageTypeUser                MessageType = "user"
	MessageTypeAssistant           MessageType = "assistant"
	MessageTypeSystem              MessageType = "system"
	MessageTypeAttachment          MessageType = "attachment"
	MessageTypeProgress            MessageType = "progress"
	MessageTypeGroupedToolUse      MessageType = "grouped_tool_use"
	MessageTypeCollapsedReadSearch MessageType = "collapsed_read_search"
)

// Message 是基础消息类型，所有对话消息都基于此结构。
type Message struct {
	Type                      MessageType     `json:"type"`
	UUID                      string          `json:"uuid"`
	IsMeta                    bool            `json:"isMeta,omitempty"`
	IsCompactSummary          bool            `json:"isCompactSummary,omitempty"`
	ToolUseResult             json.RawMessage `json:"toolUseResult,omitempty"`
	IsVisibleInTranscriptOnly bool            `json:"isVisibleInTranscriptOnly,omitempty"`
	Message                   *MessageContent `json:"message,omitempty"`
}

// MessageContent 包含 API 消息的角色、内容和用量信息。
type MessageContent struct {
	Role    string          `json:"role,omitempty"`
	ID      string          `json:"id,omitempty"`
	Content json.RawMessage `json:"content,omitempty"`
	Usage   json.RawMessage `json:"usage,omitempty"`
}

// UserMessage 是用户发送的消息。
type UserMessage struct {
	Message
	ImagePasteIDs []int `json:"imagePasteIds,omitempty"`
}

// AssistantMessage 是助手返回的消息。
type AssistantMessage struct {
	Message
}

// SystemMessage 是系统级消息。
type SystemMessage struct {
	Message
}

// ProgressMessage 携带工具执行进度更新。
type ProgressMessage struct {
	Message
	Data json.RawMessage `json:"data"`
}

// AttachmentMessage 携带文件/数据附件。
type AttachmentMessage struct {
	Message
	Attachment json.RawMessage `json:"attachment"`
}

// ContentBlock 表示消息中的单个内容块。
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ToolUseBlock 是请求工具执行的内容块。
type ToolUseBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResultBlock 是携带工具执行结果的内容块。
type ToolResultBlock struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}
