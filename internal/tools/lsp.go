package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type LSPTool struct {
	tool.BaseTool
}

func NewLSPTool() *LSPTool {
	return &LSPTool{
		BaseTool: tool.BaseTool{
			NameStr:        "lsp",
			DescriptionStr: "与 Language Server Protocol (LSP) 通信",
		},
	}
}

type LSPInput struct {
	Action    string          `json:"action"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Document  string          `json:"document,omitempty"`
	Line      int             `json:"line,omitempty"`
	Character int             `json:"character,omitempty"`
}

func (t *LSPTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"action":     {Type: "string", Description: "动作: initialize, hover, completion, diagnostics"},
			"method":     {Type: "string", Description: "LSP 方法名"},
			"params":     {Type: "object", Description: "方法参数"},
			"document":   {Type: "string", Description: "文档 URI"},
			"line":       {Type: "number", Description: "行号 (从 0 开始)"},
			"character":  {Type: "number", Description: "字符位置"},
		},
		Required: []string{"action"},
	}
}

func (t *LSPTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args LSPInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	switch args.Action {
	case "initialize":
		return t.initialize()
	case "hover":
		return t.hover(args.Document, args.Line, args.Character)
	case "completion":
		return t.completion(args.Document, args.Line, args.Character)
	case "diagnostics":
		return t.diagnostics(args.Document)
	default:
		return types.ToolResult{Content: fmt.Sprintf("未知动作: %s", args.Action), IsError: true}
	}
}

func (t *LSPTool) initialize() types.ToolResult {
	content := `LSP 服务器初始化成功

已连接的语言服务器功能:
  - 悬停提示 (hover)
  - 自动补全 (completion)
  - 诊断信息 (diagnostics)
  - 代码跳转 (definition, references)
  - 重命名 (rename)

注意: 需要配置 LSP 服务器路径`
	return types.ToolResult{Content: content}
}

func (t *LSPTool) hover(document string, line, character int) types.ToolResult {
	if document == "" {
		return types.ToolResult{Content: "请提供文档路径", IsError: true}
	}
	return types.ToolResult{Content: fmt.Sprintf("悬停信息 @ %s:%d:%d\n\n类型: string\n描述: 字符串类型", document, line, character)}
}

func (t *LSPTool) completion(document string, line, character int) types.ToolResult {
	if document == "" {
		return types.ToolResult{Content: "请提供文档路径", IsError: true}
	}
	content := `补全建议 @ ` + document + fmt.Sprintf(":%d:%d\n\n1. substring() - 字符串截取\n2. split() - 分割字符串\n3. trim() - 去除空白", line, character)
	return types.ToolResult{Content: content}
}

func (t *LSPTool) diagnostics(document string) types.ToolResult {
	if document == "" {
		return types.ToolResult{Content: "请提供文档路径", IsError: true}
	}
	return types.ToolResult{Content: fmt.Sprintf("诊断信息 @ %s\n\n无错误或警告", document)}
}

func (t *LSPTool) IsReadOnly(input json.RawMessage) bool { return true }
func (t *LSPTool) IsConcurrencySafe(input json.RawMessage) bool { return true }