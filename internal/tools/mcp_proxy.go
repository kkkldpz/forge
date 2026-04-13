package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type MCPProxyTool struct {
	tool.BaseTool
}

func NewMCPProxyTool() *MCPProxyTool {
	return &MCPProxyTool{
		BaseTool: tool.BaseTool{
			NameStr:        "mcp_proxy",
			DescriptionStr: "通过 MCP 协议代理工具调用",
		},
	}
}

type MCPProxyInput struct {
	Server   string          `json:"server"`
	Method   string          `json:"method"`
	Params   json.RawMessage `json:"params,omitempty"`
}

func (t *MCPProxyTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"server": {Type: "string", Description: "MCP 服务器名称"},
			"method": {Type: "string", Description: "要调用的方法"},
			"params": {Type: "object", Description: "方法参数（可选）"},
		},
		Required: []string{"server", "method"},
	}
}

func (t *MCPProxyTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args MCPProxyInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Server == "" || args.Method == "" {
		return types.ToolResult{Content: "服务器名称和方法不能为空", IsError: true}
	}

	content := fmt.Sprintf("MCP Proxy 调用\n服务器: %s\n方法: %s\n\n注意: MCP Proxy 功能需要 MCP 服务器配置", args.Server, args.Method)

	return types.ToolResult{Content: content}
}

func (t *MCPProxyTool) IsReadOnly(input json.RawMessage) bool { return false }
func (t *MCPProxyTool) IsConcurrencySafe(input json.RawMessage) bool { return true }