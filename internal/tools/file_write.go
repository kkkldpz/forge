// Package tools 实现文件操作工具。
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

// FileWriteTool 创建或覆写文件。
type FileWriteTool struct {
	toolkit.BaseTool
}

// FileWriteInput FileWrite 工具的输入参数。
type FileWriteInput struct {
	FilePath string `json:"file_path"` // 文件路径
	Content  string `json:"content"`   // 文件内容
}

// NewFileWriteTool 创建新的文件写入工具实例。
func NewFileWriteTool() *FileWriteTool {
	return &FileWriteTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "file_write",
			DescriptionStr: "创建新文件或覆写已有文件",
		},
	}
}

// InputSchema 返回工具的输入参数 JSON Schema。
func (t *FileWriteTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"file_path": {
				Type:        "string",
				Description: "要写入的文件的绝对路径",
			},
			"content": {
				Type:        "string",
				Description: "要写入文件的内容",
			},
		},
		Required: []string{"file_path", "content"},
	}
}

// Call 执行文件写入。
func (t *FileWriteTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args FileWriteInput
	if err := toolkit.ParseToolInput(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析错误: %v", err), IsError: true}
	}

	if args.FilePath == "" {
		return types.ToolResult{Content: "错误: file_path 不能为空", IsError: true}
	}

	// 解析路径
	path := args.FilePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(tuc.WorkingDir, path)
	}
	path = filepath.Clean(path)

	// 确保父目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("创建目录失败: %v", err), IsError: true}
	}

	// 检查文件是否已存在
	_, exists := os.Stat(path)
	action := "创建"
	if exists == nil {
		action = "覆写"
	}

	// 写入文件
	if err := os.WriteFile(path, []byte(args.Content), 0644); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("写入文件失败: %v", err), IsError: true}
	}

	output := fmt.Sprintf("文件已%s: %s\n", action, path)
	output += fmt.Sprintf("写入 %d 字节", len(args.Content))

	return types.ToolResult{Content: output}
}

// IsReadOnly 文件写入不是只读操作。
func (t *FileWriteTool) IsReadOnly(input json.RawMessage) bool {
	return false
}

// IsConcurrencySafe 文件写入不是并发安全的。
func (t *FileWriteTool) IsConcurrencySafe(input json.RawMessage) bool {
	return false
}
