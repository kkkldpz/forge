package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type WorktreeEnterTool struct {
	tool.BaseTool
}

func NewWorktreeEnterTool() *WorktreeEnterTool {
	return &WorktreeEnterTool{
		BaseTool: tool.BaseTool{
			NameStr:        "worktree_enter",
			DescriptionStr: "进入 Git worktree 工作目录",
		},
	}
}

type WorktreeEnterInput struct {
	Path string `json:"path"`
	Branch string `json:"branch"`
}

func (t *WorktreeEnterTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"path":   {Type: "string", Description: "Worktree 路径"},
			"branch": {Type: "string", Description: "分支名称"},
		},
		Required: []string{"path"},
	}
}

func (t *WorktreeEnterTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args WorktreeEnterInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Path == "" {
		return types.ToolResult{Content: "Worktree 路径不能为空", IsError: true}
	}

	cmd := exec.CommandContext(ctx, "git", "worktree", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return types.ToolResult{Content: fmt.Sprintf("执行 git 命令失败: %v\n%s", err, output), IsError: true}
	}

	content := fmt.Sprintf("已进入 worktree: %s\n分支: %s\n工作目录: %s", args.Path, args.Branch, tuc.WorkingDir)

	return types.ToolResult{Content: content}
}

func (t *WorktreeEnterTool) IsReadOnly(input json.RawMessage) bool { return false }
func (t *WorktreeEnterTool) IsConcurrencySafe(input json.RawMessage) bool { return false }

type WorktreeExitTool struct {
	tool.BaseTool
}

func NewWorktreeExitTool() *WorktreeExitTool {
	return &WorktreeExitTool{
		BaseTool: tool.BaseTool{
			NameStr:        "worktree_exit",
			DescriptionStr: "退出 Git worktree 工作目录",
		},
	}
}

type WorktreeExitInput struct {
	Path string `json:"path"`
}

func (t *WorktreeExitTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"path": {Type: "string", Description: "要退出的 Worktree 路径"},
		},
		Required: []string{"path"},
	}
}

func (t *WorktreeExitTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args WorktreeExitInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Path == "" {
		return types.ToolResult{Content: "Worktree 路径不能为空", IsError: true}
	}

	content := fmt.Sprintf("已退出 worktree: %s", args.Path)

	return types.ToolResult{Content: content}
}

func (t *WorktreeExitTool) IsReadOnly(input json.RawMessage) bool { return false }
func (t *WorktreeExitTool) IsConcurrencySafe(input json.RawMessage) bool { return false }