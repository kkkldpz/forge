package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

// MCPTool 将 MCP 服务器上的工具适配为 Forge 工具接口。
// 工具名称格式: mcp__{server}__{tool}
type MCPTool struct {
	toolkit.BaseTool

	// server 关联的 MCP 服务器名称
	server string
	// mcpClient MCP 客户端引用
	mcpClient *Client
	// schema MCP 工具的原始 JSON Schema
	schema json.RawMessage
}

// NewMCPTool 创建 MCP 工具适配器。
func NewMCPTool(server string, toolDef ToolDef, client *Client) *MCPTool {
	name := fmt.Sprintf("mcp__%s__%s", server, toolDef.Name)

	return &MCPTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        name,
			DescriptionStr: toolDef.Description,
		},
		server:    server,
		mcpClient: client,
		schema:    toolDef.InputSchema,
	}
}

// InputSchema 返回工具的输入参数 JSON Schema。
// 将 MCP 工具的原始 schema 转换为 Forge 的 ToolInputJSONSchema 格式。
func (t *MCPTool) InputSchema() types.ToolInputJSONSchema {
	schema := types.ToolInputJSONSchema{
		Type: "object",
	}

	if len(t.schema) == 0 {
		return schema
	}

	// 尝试将原始 schema 解析为 Forge 格式
	var rawSchema struct {
		Type       string                       `json:"type"`
		Properties map[string]toolSchemaPropRaw `json:"properties"`
		Required   []string                     `json:"required"`
	}
	if err := json.Unmarshal(t.schema, &rawSchema); err != nil {
		return schema
	}

	if rawSchema.Type != "" {
		schema.Type = rawSchema.Type
	}

	if len(rawSchema.Properties) > 0 {
		schema.Properties = make(map[string]types.ToolSchemaProperty, len(rawSchema.Properties))
		for k, v := range rawSchema.Properties {
			schema.Properties[k] = types.ToolSchemaProperty{
				Type:        v.Type,
				Description: v.Description,
				Enum:        v.Enum,
				Default:     v.Default,
			}
		}
	}

	if len(rawSchema.Required) > 0 {
		schema.Required = rawSchema.Required
	}

	return schema
}

// Call 执行 MCP 工具调用。
func (t *MCPTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	result, err := t.mcpClient.CallTool(ctx, t.mcpToolName(), input)
	if err != nil {
		return types.ToolResult{
			Content: fmt.Sprintf("MCP 工具调用失败: %v", err),
			IsError: true,
		}
	}

	// 将内容项拼接为文本
	content := formatContentItems(result.Content)

	return types.ToolResult{
		Content: content,
		IsError: result.IsError,
	}
}

// ValidateInput 验证输入参数是否为合法 JSON。
func (t *MCPTool) ValidateInput(ctx context.Context, input json.RawMessage) types.ValidationResult {
	if len(input) == 0 {
		return types.ValidationResult{
			Valid: true,
		}
	}
	if !json.Valid(input) {
		return types.ValidationResult{
			Valid: false,
			Msg:   "输入不是有效的 JSON",
		}
	}
	return types.ValidationResult{
		Valid: true,
	}
}

// IsReadOnly MCP 工具默认视为非只读（安全性考虑）。
func (t *MCPTool) IsReadOnly(input json.RawMessage) bool {
	return false
}

// IsConcurrencySafe MCP 工具调用是否并发安全取决于服务器实现，默认安全。
func (t *MCPTool) IsConcurrencySafe(input json.RawMessage) bool {
	return true
}

// IsEnabled MCP 工具始终启用。
func (t *MCPTool) IsEnabled() bool {
	return true
}

// Server 返回关联的 MCP 服务器名称。
func (t *MCPTool) Server() string {
	return t.server
}

// mcpToolName 提取 MCP 原始工具名称（去掉 mcp__{server}__ 前缀）。
func (t *MCPTool) mcpToolName() string {
	prefix := "mcp__" + t.server + "__"
	if strings.HasPrefix(t.Name(), prefix) {
		return strings.TrimPrefix(t.Name(), prefix)
	}
	return t.Name()
}

// formatContentItems 将工具结果内容项格式化为文本。
func formatContentItems(items []ContentItem) string {
	if len(items) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, item := range items {
		switch item.Type {
		case "text":
			builder.WriteString(item.Text)
		case "image", "resource":
			if item.Text != "" {
				builder.WriteString(item.Text)
			}
			if item.Data != "" {
				if builder.Len() > 0 {
					builder.WriteByte('\n')
				}
				builder.WriteString(item.Data)
			}
		default:
			if item.Text != "" {
				builder.WriteString(item.Text)
			}
		}
	}
	return builder.String()
}

// toolSchemaPropRaw 是解析 MCP 工具 schema 的中间结构。
type toolSchemaPropRaw struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum"`
	Default     any      `json:"default"`
}
