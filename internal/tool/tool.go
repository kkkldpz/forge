// Package tool 定义工具系统的核心接口和执行框架。
package tool

import (
	"context"
	"encoding/json"

	"github.com/kkkldpz/forge/internal/types"
)

// Tool 是所有工具必须实现的核心接口。
// CCB 的 Tool 类型有约 40 个方法，这里简化为 Go 习惯的 8 个核心方法。
type Tool interface {
	// Name 返回工具的唯一名称（如 "bash", "file_read"）。
	Name() string

	// Description 返回工具的人类可读描述。
	Description() string

	// InputSchema 返回工具输入参数的 JSON Schema。
	InputSchema() types.ToolInputJSONSchema

	// Call 执行工具并返回结果。
	Call(ctx context.Context, input json.RawMessage, tuc ToolUseContext) types.ToolResult

	// ValidateInput 验证输入参数是否合法。
	ValidateInput(ctx context.Context, input json.RawMessage) types.ValidationResult

	// IsReadOnly 判断该工具是否为只读操作（不修改文件系统）。
	IsReadOnly(input json.RawMessage) bool

	// IsConcurrencySafe 判断该工具是否可以与其他工具并发执行。
	IsConcurrencySafe(input json.RawMessage) bool

	// IsEnabled 判断该工具是否在当前环境中启用。
	IsEnabled() bool
}

// ToolUseContext 提供工具执行所需的上下文信息。
// 简化自 CCB 的 ToolUseContext（原约 50 个字段）。
type ToolUseContext struct {
	// AbortSignal 用于取消操作，通过 context.Context 传递
	Ctx context.Context

	// SessionID 当前会话唯一标识
	SessionID types.SessionID

	// WorkingDir 工作目录
	WorkingDir string

	// Debug 是否启用调试输出
	Debug bool

	// Verbose 是否启用详细输出
	Verbose bool

	// NonInteractive 是否非交互式会话
	NonInteractive bool

	// Tools 当前可用的所有工具列表
	Tools []Tool
}

// CallResult 表示工具调用的结果。
type CallResult struct {
	// Content 工具输出内容
	Content string

	// IsError 是否执行出错
	IsError bool

	// ToolUseID 关联的 tool_use ID
	ToolUseID string
}

// BaseTool 是所有工具的嵌入基结构体。
// 提供默认实现，工具只需重写需要自定义的方法。
type BaseTool struct {
	NameStr        string
	DescriptionStr string
}

// Name 返回工具名称。
func (b *BaseTool) Name() string { return b.NameStr }

// Description 返回工具描述。
func (b *BaseTool) Description() string { return b.DescriptionStr }

// ValidateInput 默认验证通过。
func (b *BaseTool) ValidateInput(ctx context.Context, input json.RawMessage) types.ValidationResult {
	return types.ValidationResult{Valid: true}
}

// IsReadOnly 默认非只读。
func (b *BaseTool) IsReadOnly(input json.RawMessage) bool { return false }

// IsConcurrencySafe 默认不安全。
func (b *BaseTool) IsConcurrencySafe(input json.RawMessage) bool { return false }

// IsEnabled 默认启用。
func (b *BaseTool) IsEnabled() bool { return true }
