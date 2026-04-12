// Package tools 实现各种实用工具。
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

// BashTool 执行 shell 命令。
type BashTool struct {
	tool.BaseTool
}

// BashInput Bash 工具的输入参数。
type BashInput struct {
	Command    string `json:"command"`              // 要执行的命令
	Cwd        string `json:"cwd,omitempty"`        // 工作目录
	TimeoutMs  int    `json:"timeout_ms,omitempty"` // 超时时间（毫秒）
	Background bool   `json:"background,omitempty"` // 是否在后台运行
	Stdin      string `json:"stdin,omitempty"`      // 标准输入
}

// BashOutput Bash 工具的输出结果。
type BashOutput struct {
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	WorkingDir string `json:"working_directory,omitempty"`
	DurationMs int64  `json:"duration_ms,omitempty"`
}

// NewBashTool 创建新的 Bash 工具实例。
func NewBashTool() *BashTool {
	return &BashTool{
		BaseTool: tool.BaseTool{
			NameStr:        "bash",
			DescriptionStr: "执行 shell 命令",
		},
	}
}

// InputSchema 返回工具的输入参数 JSON Schema。
func (t *BashTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"command": {
				Type:        "string",
				Description: "要执行的 shell 命令",
			},
			"cwd": {
				Type:        "string",
				Description: "工作目录（可选，默认为当前目录）",
			},
			"timeout_ms": {
				Type:        "number",
				Description: "超时时间（毫秒，可选）",
			},
			"background": {
				Type:        "boolean",
				Description: "是否在后台运行（可选）",
			},
			"stdin": {
				Type:        "string",
				Description: "发送到命令标准输入的数据（可选）",
			},
		},
		Required: []string{"command"},
	}
}

// Call 执行 shell 命令。
func (t *BashTool) Call(ctx context.Context, input []byte, tuc tool.ToolUseContext) types.ToolResult {
	var args BashInput
	if err := tool.ParseToolInput(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析错误: %v", err), IsError: true}
	}

	if args.Command == "" {
		return types.ToolResult{Content: "错误: command 不能为空", IsError: true}
	}

	start := time.Now()

	// 确定工作目录
	cwd := args.Cwd
	if cwd == "" {
		cwd = tuc.WorkingDir
	}

	// 创建命令
	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)
	cmd.Dir = cwd

	// 设置超时
	if args.TimeoutMs > 0 {
		timeout := time.Duration(args.TimeoutMs) * time.Millisecond
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, "bash", "-c", args.Command)
		cmd.Dir = cwd
	}

	// 后台模式：启动后不等待完成
	if args.Background {
		if err := cmd.Start(); err != nil {
			return types.ToolResult{Content: fmt.Sprintf("启动后台进程失败: %v", err), IsError: true}
		}
		output := fmt.Sprintf("后台进程已启动，PID: %d", cmd.Process.Pid)
		return types.ToolResult{Content: output}
	}

	// 标准输入
	if args.Stdin != "" {
		cmd.Stdin = strings.NewReader(args.Stdin)
	}

	// 执行命令
	output, err := cmd.CombinedOutput()
	duration := time.Since(start).Milliseconds()

	// 格式化输出
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return types.ToolResult{Content: fmt.Sprintf("执行命令失败: %v", err), IsError: true}
		}
	}

	result := BashOutput{
		ExitCode:   exitCode,
		Stdout:     string(output),
		WorkingDir: cwd,
		DurationMs: duration,
	}

	// 格式化结果
	var content string
	if result.Stdout != "" {
		content = result.Stdout
	}
	if result.Stderr != "" {
		if content != "" {
			content += "\n"
		}
		content += "stderr: " + result.Stderr
	}
	if content == "" {
		content = fmt.Sprintf("命令执行完成，退出码: %d", result.ExitCode)
	}

	return types.ToolResult{
		Content: content,
		IsError: exitCode != 0,
	}
}

// IsConcurrencySafe 判断是否可以并发执行。
func (t *BashTool) IsConcurrencySafe(input []byte) bool {
	var args BashInput
	if err := json.Unmarshal(input, &args); err != nil {
		return false
	}

	// 搜索命令是安全的
	searchCommands := []string{"grep", "rg", "find", "locate", "ack", "ag", "which"}
	for _, cmd := range searchCommands {
		if strings.HasPrefix(args.Command, cmd+" ") || args.Command == cmd {
			return true
		}
	}

	// 读取命令也是安全的
	readCommands := []string{"cat", "head", "tail", "less", "more", "echo", "pwd"}
	for _, cmd := range readCommands {
		if strings.HasPrefix(args.Command, cmd+" ") || args.Command == cmd {
			return true
		}
	}

	// 列表类命令安全
	listCommands := []string{"ls", "ll", "la", "dir", "ps"}
	for _, cmd := range listCommands {
		if args.Command == cmd || strings.HasPrefix(args.Command, cmd+" ") {
			return true
		}
	}

	// 后台命令、带 stdin 的命令不安全
	if args.Background || args.Stdin != "" {
		return false
	}

	// 默认不安全
	return false
}

// IsReadOnly 判断是否为只读操作。
func (t *BashTool) IsReadOnly(input []byte) bool {
	var args BashInput
	if err := json.Unmarshal(input, &args); err != nil {
		return false
	}

	// 只读命令前缀列表
	readonlyCmds := []string{
		"cat", "head", "tail", "less", "more", "echo", "pwd", "ls",
		"grep", "rg", "find", "locate", "ack", "ag", "which", "whereis",
		"ps", "top", "htop", "whoami", "id", "uname", "date",
	}

	parts := strings.Fields(args.Command)
	if len(parts) == 0 {
		return false
	}

	for _, cmd := range readonlyCmds {
		if parts[0] == cmd {
			return true
		}
	}

	return false
}
