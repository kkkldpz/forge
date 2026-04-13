package openai

import (
	"encoding/json"
	"strings"
)

// ChatCompletionRequest 是 OpenAI /chat/completions 请求体。
type ChatCompletionRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	Tools       []OpenAITool    `json:"tools,omitempty"`
	ToolChoice  any             `json:"tool_choice,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

// OpenAIMessage 是 OpenAI 格式的消息。
type OpenAIMessage struct {
	Role       string     `json:"role"`
	Content    any        `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall 表示 OpenAI 的工具调用。
type ToolCall struct {
	Index    int          `json:"index"`    // 流式传输中的工具调用索引
	ID       string       `json:"id"`
	Type     string       `json:"type"`     // "function"
	Function FunctionCall `json:"function"`
}

// FunctionCall 表示函数调用参数。
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAITool 是 OpenAI 格式的工具定义。
type OpenAITool struct {
	Type     string         `json:"type"` // "function"
	Function FunctionObject `json:"function"`
}

// FunctionObject 是函数的详细定义。
type FunctionObject struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"`
}

// ChatCompletionChunk 是 OpenAI SSE 流的单个数据块。
type ChatCompletionChunk struct {
	ID      string        `json:"id"`
	Object  string        `json:"object"`
	Created int64         `json:"created"`
	Model   string        `json:"model"`
	Choices []ChunkChoice `json:"choices"`
	Usage   *ChunkUsage   `json:"usage,omitempty"`
}

// ChunkChoice 是流式响应中的单个选项。
type ChunkChoice struct {
	Index        int         `json:"index"`
	Delta        ChunkDelta  `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// ChunkDelta 是流式响应的增量内容。
type ChunkDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   *string    `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ChunkUsage 是流式响应最后一个块中的 token 用量。
type ChunkUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamBlockState 跟踪当前打开的流式内容块。
type StreamBlockState struct {
	Type       string // "text" 或 "tool_use"
	ToolCallID string
	ToolName   string
	ArgsBuf    strings.Builder
}

// mapFinishReason 将 OpenAI 的 finish_reason 映射为 Anthropic 的 stop_reason。
func mapFinishReason(reason *string) string {
	if reason == nil {
		return ""
	}
	switch *reason {
	case "stop":
		return "end_turn"
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	default:
		return *reason
	}
}

// UsageToJSON 将 ChunkUsage 转换为 Anthropic 格式的 JSON。
func UsageToJSON(u *ChunkUsage) json.RawMessage {
	if u == nil {
		return nil
	}
	data, _ := json.Marshal(struct {
		OutputTokens int `json:"output_tokens"`
	}{
		OutputTokens: u.CompletionTokens,
	})
	return data
}
