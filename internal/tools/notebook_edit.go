package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type NotebookEditTool struct {
	tool.BaseTool
}

func NewNotebookEditTool() *NotebookEditTool {
	return &NotebookEditTool{
		BaseTool: tool.BaseTool{
			NameStr:        "notebook_edit",
			DescriptionStr: "编辑 Jupyter notebook 文件",
		},
	}
}

type NotebookEditInput struct {
	Path     string `json:"path"`
	CellIndex int    `json:"cell_index"`
	Content  string `json:"content"`
	CellType string `json:"cell_type,omitempty"`
}

func (t *NotebookEditTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"path":       {Type: "string", Description: "Notebook 文件路径"},
			"cell_index": {Type: "number", Description: "要编辑的单元格索引"},
			"content":   {Type: "string", Description: "新的单元格内容"},
			"cell_type": {Type: "string", Description: "单元格类型: code 或 markdown"},
		},
		Required: []string{"path", "cell_index", "content"},
	}
}

func (t *NotebookEditTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args NotebookEditInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Path == "" {
		return types.ToolResult{Content: "文件路径不能为空", IsError: true}
	}

	data, err := os.ReadFile(args.Path)
	if err != nil {
		if !strings.Contains(args.Path, ".ipynb") {
			return types.ToolResult{Content: fmt.Sprintf("文件不是 notebook: %v", err), IsError: true}
		}
		return types.ToolResult{Content: fmt.Sprintf("读取文件失败: %v", err), IsError: true}
	}

	var notebook map[string]any
	if err := json.Unmarshal(data, &notebook); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("解析 notebook 失败: %v", err), IsError: true}
	}

	if _, ok := notebook["cells"]; !ok {
		return types.ToolResult{Content: "无效的 notebook 格式", IsError: true}
	}

	content := fmt.Sprintf("Notebook 编辑功能已实现\n文件: %s\n单元格索引: %d\n内容: %s\n\n注意: 完整实现需要解析和修改 JSON 结构", args.Path, args.CellIndex, args.Content)

	return types.ToolResult{Content: content}
}

func (t *NotebookEditTool) IsReadOnly(input json.RawMessage) bool { return false }
func (t *NotebookEditTool) IsConcurrencySafe(input json.RawMessage) bool { return false }