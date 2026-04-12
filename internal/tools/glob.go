// Package tools 实现文件操作工具。
package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

// GlobTool 使用 glob 模式搜索文件。
type GlobTool struct {
	tool.BaseTool
}

// GlobInput Glob 工具的输入参数。
type GlobInput struct {
	Pattern string `json:"pattern"`          // glob 模式
	Path    string `json:"path,omitempty"`   // 搜索目录（可选）
}

// GlobOutput Glob 工具的输出结果。
type GlobOutput struct {
	Filenames   []string `json:"filenames"`
	NumFiles    int      `json:"num_files"`
	DurationMs  int64    `json:"duration_ms"`
	Truncated   bool     `json:"truncated"`
}

// NewGlobTool 创建新的 Glob 工具实例。
func NewGlobTool() *GlobTool {
	return &GlobTool{
		BaseTool: tool.BaseTool{
			NameStr:        "glob",
			DescriptionStr: "使用 glob 模式搜索文件",
		},
	}
}

// InputSchema 返回工具的输入参数 JSON Schema。
func (t *GlobTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"pattern": {
				Type:        "string",
				Description: "glob 模式（如 *.go, **/*.txt）",
			},
			"path": {
				Type:        "string",
				Description: "搜索的目录（可选，默认为当前目录）",
			},
		},
		Required: []string{"pattern"},
	}
}

// Call 执行 glob 搜索。
func (t *GlobTool) Call(ctx context.Context, input []byte, tuc tool.ToolUseContext) types.ToolResult {
	var args GlobInput
	if err := tool.ParseToolInput(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析错误: %v", err), IsError: true}
	}

	if args.Pattern == "" {
		return types.ToolResult{Content: "错误: pattern 不能为空", IsError: true}
	}

	// 确定搜索目录
	searchDir := args.Path
	if searchDir == "" {
		searchDir = tuc.WorkingDir
	}
	if !filepath.IsAbs(searchDir) {
		searchDir = filepath.Join(tuc.WorkingDir, searchDir)
	}

	start := time.Now()

	// 执行 glob 搜索
	matches, err := doublestar.Glob(os.DirFS(searchDir), args.Pattern)
	if err != nil {
		return types.ToolResult{Content: fmt.Sprintf("搜索失败: %v", err), IsError: true}
	}

	duration := time.Since(start).Milliseconds()

	// 限制结果数量
	const maxResults = 100
	truncated := len(matches) > maxResults
	resultCount := len(matches)
	if truncated {
		matches = matches[:maxResults]
	}

	// 转换为绝对路径
	var filenames []string
	for _, match := range matches {
		absPath := filepath.Join(searchDir, match)
		filenames = append(filenames, absPath)
	}

	// 格式化输出
	output := fmt.Sprintf("找到 %d 个文件", resultCount)
	if truncated {
		output += fmt.Sprintf(" (显示前 %d 个)", maxResults)
	}
	output += fmt.Sprintf("，耗时 %d ms\n", duration)
	output += "```\n"
	for _, f := range filenames {
		output += f + "\n"
	}
	output += "```"

	return types.ToolResult{Content: output}
}

// IsReadOnly Glob 搜索是只读操作。
func (t *GlobTool) IsReadOnly(input []byte) bool {
	return true
}

// IsConcurrencySafe Glob 搜索是并发安全的。
func (t *GlobTool) IsConcurrencySafe(input []byte) bool {
	return true
}
