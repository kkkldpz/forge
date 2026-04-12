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

// FileEditTool 通过精确字符串替换编辑文件。
type FileEditTool struct {
	tool.BaseTool
}

// FileEditInput FileEdit 工具的输入参数。
type FileEditInput struct {
	FilePath    string `json:"file_path"`              // 文件路径
	OldString   string `json:"old_string"`             // 要替换的旧字符串
	NewString   string `json:"new_string"`             // 新字符串
	ReplaceAll  bool   `json:"replace_all,omitempty"`  // 是否替换所有匹配
}

// NewFileEditTool 创建新的文件编辑工具实例。
func NewFileEditTool() *FileEditTool {
	return &FileEditTool{
		BaseTool: tool.BaseTool{
			NameStr:        "file_edit",
			DescriptionStr: "通过精确字符串替换编辑文件",
		},
	}
}

// InputSchema 返回工具的输入参数 JSON Schema。
func (t *FileEditTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"file_path": {
				Type:        "string",
				Description: "要编辑的文件的绝对路径",
			},
			"old_string": {
				Type:        "string",
				Description: "要替换的确切字符串（必须完整匹配）",
			},
			"new_string": {
				Type:        "string",
				Description: "用于替换的新字符串",
			},
			"replace_all": {
				Type:        "boolean",
				Description: "是否替换所有匹配项（可选，默认只替换第一个）",
				Default:     false,
			},
		},
		Required: []string{"file_path", "old_string", "new_string"},
	}
}

// Call 执行文件编辑。
func (t *FileEditTool) Call(ctx context.Context, input []byte, tuc tool.ToolUseContext) types.ToolResult {
	var args FileEditInput
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

	// 读取原始文件内容
	content, err := os.ReadFile(path)
	if err != nil {
		return types.ToolResult{Content: fmt.Sprintf("读取文件失败: %v", err), IsError: true}
	}
	originalContent := string(content)

	// 检查 old_string 是否存在
	if !strings.Contains(originalContent, args.OldString) {
		return types.ToolResult{
			Content: fmt.Sprintf("错误: 找不到要替换的字符串: %q", args.OldString),
			IsError: true,
		}
	}

	// 执行替换
	var newContent string
	if args.ReplaceAll {
		newContent = strings.ReplaceAll(originalContent, args.OldString, args.NewString)
	} else {
		newContent = strings.Replace(originalContent, args.OldString, args.NewString, 1)
	}

	// 如果没有变化
	if newContent == originalContent {
		return types.ToolResult{Content: "文件内容未变化"}
	}

	// 写回文件
	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("写入文件失败: %v", err), IsError: true}
	}

	// 计算变更行数
	oldLines := strings.Count(args.OldString, "\n")
	newLines := strings.Count(args.NewString, "\n")
	lineChange := newLines - oldLines

	output := fmt.Sprintf("文件已编辑: %s\n", path)
	if args.ReplaceAll {
		output += fmt.Sprintf("替换了所有 %q 的匹配项\n", args.OldString)
	} else {
		output += fmt.Sprintf("替换了第一个 %q 的匹配项\n", args.OldString)
	}
	output += fmt.Sprintf("行数变化: %+d\n", lineChange)

	return types.ToolResult{Content: output}
}

// IsReadOnly 文件编辑不是只读操作。
func (t *FileEditTool) IsReadOnly(input []byte) bool {
	return false
}

// IsConcurrencySafe 文件编辑不是并发安全的。
func (t *FileEditTool) IsConcurrencySafe(input []byte) bool {
	return false
}
