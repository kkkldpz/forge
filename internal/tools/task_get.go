package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/task"
	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

type TaskGetTool struct {
	toolkit.BaseTool
}

func NewTaskGetTool() *TaskGetTool {
	return &TaskGetTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "task_get",
			DescriptionStr: "获取指定任务的详细信息。",
		},
	}
}

func (t *TaskGetTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"id": {Type: "string", Description: "任务 ID"},
		},
		Required: []string{"id"},
	}
}

func (t *TaskGetTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	tk, ok := task.GlobalRegistry().Get(args.ID)
	if !ok {
		return types.ToolResult{Content: fmt.Sprintf("任务 %s 不存在", args.ID), IsError: true}
	}

	content := fmt.Sprintf("任务 ID: %s\n标题: %s\n状态: %s\n描述: %s\n创建时间: %s\n更新时间: %s",
		tk.ID, tk.Title, tk.Status, tk.Description, tk.CreatedAt.Format("2006-01-02 15:04:05"), tk.UpdatedAt.Format("2006-01-02 15:04:05"))

	if tk.Result != "" {
		content += fmt.Sprintf("\n结果: %s", tk.Result)
	}
	if tk.Error != "" {
		content += fmt.Sprintf("\n错误: %s", tk.Error)
	}
	if tk.CompletedAt != nil {
		content += fmt.Sprintf("\n完成时间: %s", tk.CompletedAt.Format("2006-01-02 15:04:05"))
	}

	return types.ToolResult{Content: content}
}

func (t *TaskGetTool) IsReadOnly(input json.RawMessage) bool  { return true }
func (t *TaskGetTool) IsConcurrencySafe(input json.RawMessage) bool { return true }