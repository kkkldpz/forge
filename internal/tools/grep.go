// Package tools 实现文件操作工具。
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

// GrepTool 使用 ripgrep 搜索文件内容。
type GrepTool struct {
	tool.BaseTool
}

// GrepInput Grep 工具的输入参数。
type GrepInput struct {
	Pattern      string `json:"pattern"`              // 正则表达式模式
	Path         string `json:"path,omitempty"`       // 搜索路径
	Glob         string `json:"glob,omitempty"`       // 文件 glob 过滤
	OutputMode   string `json:"output_mode,omitempty"` // 输出模式
	Before       int    `json:"-B,omitempty"`         // 匹配前的行数
	After        int    `json:"-A,omitempty"`         // 匹配后的行数
	Context      int    `json:"context,omitempty"`    // 上下文的行数
	LineNumber   bool   `json:"-n,omitempty"`         // 显示行号
	IgnoreCase   bool   `json:"-i,omitempty"`         // 忽略大小写
	HeadLimit    int    `json:"head_limit,omitempty"` // 结果数量限制
}

// NewGrepTool 创建新的 Grep 工具实例。
func NewGrepTool() *GrepTool {
	return &GrepTool{
		BaseTool: tool.BaseTool{
			NameStr:        "grep",
			DescriptionStr: "使用 ripgrep 搜索文件内容",
		},
	}
}

// InputSchema 返回工具的输入参数 JSON Schema。
func (t *GrepTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"pattern": {
				Type:        "string",
				Description: "正则表达式模式",
			},
			"path": {
				Type:        "string",
				Description: "搜索的文件或目录（可选）",
			},
			"glob": {
				Type:        "string",
				Description: "文件 glob 过滤（如 *.go）（可选）",
			},
			"output_mode": {
				Type:        "string",
				Description: "输出模式: content(显示内容), files_with_matches(只显示文件), count(计数)",
				Default:     "files_with_matches",
				Enum:        []string{"content", "files_with_matches", "count"},
			},
			"-B": {
				Type:        "number",
				Description: "显示匹配前的 N 行（可选）",
			},
			"-A": {
				Type:        "number",
				Description: "显示匹配后的 N 行（可选）",
			},
			"context": {
				Type:        "number",
				Description: "显示上下文的 N 行（可选）",
			},
			"-n": {
				Type:        "boolean",
				Description: "显示行号（可选，默认 true）",
				Default:     true,
			},
			"-i": {
				Type:        "boolean",
				Description: "忽略大小写（可选）",
			},
			"head_limit": {
				Type:        "number",
				Description: "限制结果数量（可选）",
			},
		},
		Required: []string{"pattern"},
	}
}

// Call 执行 grep 搜索。
func (t *GrepTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args GrepInput
	if err := tool.ParseToolInput(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析错误: %v", err), IsError: true}
	}

	if args.Pattern == "" {
		return types.ToolResult{Content: "错误: pattern 不能为空", IsError: true}
	}

	// 确定搜索路径
	searchPath := args.Path
	if searchPath == "" {
		searchPath = tuc.WorkingDir
	}
	if !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(tuc.WorkingDir, searchPath)
	}

	// 构建 ripgrep 参数
	rgArgs := []string{"--color=never"}

	// 输出模式
	switch args.OutputMode {
	case "files_with_matches":
		rgArgs = append(rgArgs, "-l")
	case "count":
		rgArgs = append(rgArgs, "-c")
	case "content":
		// 默认就是 content 模式
		if args.LineNumber {
			rgArgs = append(rgArgs, "-n")
		}
	default:
		rgArgs = append(rgArgs, "-l")
	}

	// 上下文选项
	if args.Before > 0 {
		rgArgs = append(rgArgs, "-B", strconv.Itoa(args.Before))
	}
	if args.After > 0 {
		rgArgs = append(rgArgs, "-A", strconv.Itoa(args.After))
	}
	if args.Context > 0 {
		rgArgs = append(rgArgs, "-C", strconv.Itoa(args.Context))
	}

	// 忽略大小写
	if args.IgnoreCase {
		rgArgs = append(rgArgs, "-i")
	}

	// glob 过滤
	if args.Glob != "" {
		rgArgs = append(rgArgs, "-g", args.Glob)
	}

	// 结果限制
	if args.HeadLimit > 0 {
		rgArgs = append(rgArgs, "-m", strconv.Itoa(args.HeadLimit))
	}

	// 添加模式和路径
	rgArgs = append(rgArgs, args.Pattern, searchPath)

	// 执行 ripgrep
	cmd := exec.CommandContext(ctx, "rg", rgArgs...)
	output, err := cmd.CombinedOutput()

	// ripgrep 找不到匹配时返回退出码 1，这不是错误
	if err != nil && cmd.ProcessState.ExitCode() != 1 {
		return types.ToolResult{Content: fmt.Sprintf("搜索失败: %v\n%s", err, string(output)), IsError: true}
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		result = fmt.Sprintf("未找到匹配: %s", args.Pattern)
	}

	return types.ToolResult{Content: result}
}

// IsReadOnly Grep 搜索是只读操作。
func (t *GrepTool) IsReadOnly(input json.RawMessage) bool {
	return true
}

// IsConcurrencySafe Grep 搜索是并发安全的。
func (t *GrepTool) IsConcurrencySafe(input json.RawMessage) bool {
	return true
}
