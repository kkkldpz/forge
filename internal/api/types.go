// Package api 实现 Anthropic Messages API 客户端。
package api

import "encoding/json"

// Beta header 常量，用于启用 API 的实验性功能。
const (
	BetaContext1M         = "context-1m-2025-01-24"
	BetaFastMode          = "speed-2025-03-23"
	BetaStructuredOutputs = "output-2024-08-06"
	BetaTaskBudgets       = "task-budgets-2026-03-13"
	BetaRedactThinking    = "redo-thinking-2024-09-24"
	BetaTokenCounting     = "token-counting-2024-11-01"
	BetaInterleavedThinking = "interleaved-thinking-2025-05-14"
)

// 重试相关常量。
const (
	DefaultMaxRetries   = 10
	Max529Retries       = 3
	BaseDelayMs         = 500
	MaxDelayMs          = 5 * 60 * 1000 // 5 分钟
	PersistentMaxBackoffMs = 5 * 60 * 1000
	FloorOutputTokens   = 3000
	ClientRequestIDHeader = "x-client-request-id"
)

// RequestOptions 是 API 请求的配置选项。
type RequestOptions struct {
	Model         string         `json:"model"`
	MaxTokens     int            `json:"max_tokens"`
	Temperature   float64        `json:"temperature,omitempty"`
	TopP          float64        `json:"top_p,omitempty"`
	TopK          int            `json:"top_k,omitempty"`
	Stream        bool           `json:"stream"`
	Thinking      *ThinkingConfig `json:"thinking,omitempty"`
	ToolChoice    any            `json:"tool_choice,omitempty"`
	Betas         []string       `json:"-"` // 作为 header 发送，不在 body 中
	Metadata      any            `json:"metadata,omitempty"`
	Speed         string         `json:"speed,omitempty"` // "fast" 用于快速模式
	OutputConfig  any            `json:"output_config,omitempty"`
}

// ThinkingConfig 控制 extended thinking 行为。
type ThinkingConfig struct {
	Type         string `json:"type"`                    // "enabled", "disabled", "adaptive"
	BudgetTokens int    `json:"budget_tokens,omitempty"`  // 仅 type="enabled" 时使用
}

// ThinkingDisabled 返回禁用 thinking 的配置。
func ThinkingDisabled() *ThinkingConfig {
	return &ThinkingConfig{Type: "disabled"}
}

// ThinkingEnabled 返回启用 thinking 的配置，带 token 预算。
func ThinkingEnabled(budget int) *ThinkingConfig {
	return &ThinkingConfig{Type: "enabled", BudgetTokens: budget}
}

// ThinkingAdaptive 返回自适应 thinking 配置。
func ThinkingAdaptive() *ThinkingConfig {
	return &ThinkingConfig{Type: "adaptive"}
}

// SystemBlock 是 system prompt 的一个内容块。
type SystemBlock struct {
	Type         string       `json:"type"` // "text"
	Text         string       `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// CacheControl 控制 prompt 缓存行为。
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// NewSystemTextBlock 创建普通文本 system prompt 块。
func NewSystemTextBlock(text string) SystemBlock {
	return SystemBlock{Type: "text", Text: text}
}

// NewCachedSystemTextBlock 创建带缓存标记的 system prompt 块。
func NewCachedSystemTextBlock(text string) SystemBlock {
	return SystemBlock{
		Type:         "text",
		Text:         text,
		CacheControl: &CacheControl{Type: "ephemeral"},
	}
}

// MessageParam 是发送给 API 的消息参数。
type MessageParam struct {
	Role    string          `json:"role"`    // "user" 或 "assistant"
	Content json.RawMessage `json:"content"` // 可以是字符串或 ContentBlockParam 数组
}

// NewTextMessageParam 创建纯文本消息。
func NewTextMessageParam(role, text string) MessageParam {
	content, _ := json.Marshal(text)
	return MessageParam{Role: role, Content: content}
}

// NewContentMessageParam 创建包含内容块数组的消息。
func NewContentMessageParam(role string, blocks []any) MessageParam {
	content, _ := json.Marshal(blocks)
	return MessageParam{Role: role, Content: content}
}

// ToolParam 是发送给 API 的工具定义。
type ToolParam struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema"`
}

// ToolResultBlockParam 是工具执行结果的内容块。
type ToolResultBlockParam struct {
	Type      string `json:"type"`       // "tool_result"
	ToolUseID string `json:"tool_use_id"`
	Content   any    `json:"content"`    // 字符串或内容块数组
	IsError   bool   `json:"is_error,omitempty"`
}

// TextBlockParam 是文本内容块。
type TextBlockParam struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// ToolUseBlockParam 是工具调用请求的内容块。
type ToolUseBlockParam struct {
	Type  string          `json:"type"`  // "tool_use"
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ThinkingBlockParam 是 thinking 内容块。
type ThinkingBlockParam struct {
	Type      string `json:"type"`      // "thinking"
	Thinking  string `json:"thinking"`
	Signature string `json:"signature,omitempty"`
}

// APIResponse 是非流式 API 响应的完整结构。
type APIResponse struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
	Model   string          `json:"model"`
	StopReason string       `json:"stop_reason,omitempty"`
	Usage   *Usage          `json:"usage,omitempty"`
}

// Usage 记录 API 调用的 token 用量。
type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	ServerToolUseTokens      int `json:"server_tool_use,omitempty"`
}

// IsEmpty 检查用量是否为零值。
func (u *Usage) IsEmpty() bool {
	return u.InputTokens == 0 && u.OutputTokens == 0
}

// Add 累加另一个 Usage。
func (u *Usage) Add(other *Usage) {
	if other == nil {
		return
	}
	u.InputTokens += other.InputTokens
	u.OutputTokens += other.OutputTokens
	u.CacheReadInputTokens += other.CacheReadInputTokens
	u.CacheCreationInputTokens += other.CacheCreationInputTokens
	u.ServerToolUseTokens += other.ServerToolUseTokens
}
