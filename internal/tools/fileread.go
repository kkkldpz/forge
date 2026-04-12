// Package tools 实现文件操作工具。
package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

// FileReadTool 读取文件内容。
type FileReadTool struct {
	tool.BaseTool
}

// FileReadInput FileRead 工具的输入参数。
type FileReadInput struct {
	FilePath string `json:"file_path"`          // 文件路径
	Offset   int    `json:"offset,omitempty"`   // 起始行号（从1开始）
	Limit    int    `json:"limit,omitempty"`    // 读取行数限制
	Encoding string `json:"encoding,omitempty"` // 文件编码
}

// NewFileReadTool 创建新的文件读取工具实例。
func NewFileReadTool() *FileReadTool {
	return &FileReadTool{
		BaseTool: tool.BaseTool{
			NameStr:        "file_read",
			DescriptionStr: "读取文件内容，支持行号偏移和行数限制",
		},
	}
}

// InputSchema 返回工具的输入参数 JSON Schema。
func (t *FileReadTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"file_path": {
				Type:        "string",
				Description: "要读取的文件的绝对路径",
			},
			"offset": {
				Type:        "number",
				Description: "起始行号（从1开始，可选）",
				Default:     1,
			},
			"limit": {
				Type:        "number",
				Description: "读取的最大行数（可选）",
			},
			"encoding": {
				Type:        "string",
				Description: "文件编码（可选，默认UTF-8）",
				Default:     "utf-8",
			},
		},
		Required: []string{"file_path"},
	}
}

// Call 执行文件读取。
func (t *FileReadTool) Call(ctx context.Context, input []byte, tuc tool.ToolUseContext) types.ToolResult {
	var args FileReadInput
	if err := tool.ParseToolInput(input, &args); err != nil {
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

	// 检查文件是否存在
	info, err := os.Stat(path)
	if err != nil {
		return types.ToolResult{Content: fmt.Sprintf("无法访问文件: %v", err), IsError: true}
	}

	if info.IsDir() {
		return types.ToolResult{Content: "错误: 路径是目录而非文件", IsError: true}
	}

	// 读取文件内容
	content, err := os.ReadFile(path)
	if err != nil {
		return types.ToolResult{Content: fmt.Sprintf("读取文件失败: %v", err), IsError: true}
	}

	// 按行处理
	lines := strings.Split(string(content), "\n")
	totalLines := len(lines)

	// 处理偏移和限制
	offset := args.Offset
	if offset < 1 {
		offset = 1
	}
	if offset > totalLines {
		offset = totalLines
	}

	limit := args.Limit
	if limit <= 0 || limit > 500 {
		limit = 500 // 默认限制500行
	}

	// 提取指定范围的行
	startIdx := offset - 1
	endIdx := startIdx + limit
	if endIdx > totalLines {
		endIdx = totalLines
	}

	selectedLines := lines[startIdx:endIdx]

	// 添加行号
	var numberedLines []string
	for i, line := range selectedLines {
		lineNum := offset + i
		numberedLines = append(numberedLines, fmt.Sprintf("%6d\t%s", lineNum, line))
	}
	numberedContent := strings.Join(numberedLines, "\n")

	// 判断是否被截断
	truncated := totalLines > endIdx

	// 格式化输出
	output := fmt.Sprintf("文件: %s\n", path)
	output += fmt.Sprintf("总行数: %d, 显示行: %d-%d\n", totalLines, offset, endIdx)
	if truncated {
		output += "(内容已截断，使用 offset/limit 参数查看更多)\n"
	}
	output += "```\n"
	output += numberedContent
	output += "\n```"

	return types.ToolResult{Content: output}
}

// IsReadOnly 文件读取始终是只读操作。
func (t *FileReadTool) IsReadOnly(input []byte) bool {
	return true
}

// IsConcurrencySafe 文件读取是并发安全的。
func (t *FileReadTool) IsConcurrencySafe(input []byte) bool {
	return true
}
