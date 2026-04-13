package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/task"
	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

// TaskUpdateTool 更新异步任务状态。
type TaskUpdateTool struct {
	toolkit.BaseTool
}

// NewTaskUpdateTool 创建新的 TaskUpdate 工具。
func NewTaskUpdateTool() *TaskUpdateTool {
	return &TaskUpdateTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "task_update",
			DescriptionStr: "更新异步任务的状态、进度或结果。",
		},
	}
}

func (t *TaskUpdateTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"id":          {Type: "string", Description: "任务 ID"},
			"title":       {Type: "string", Description: "新标题（可选）"},
			"description": {Type: "string", Description: "新描述（可选）"},
			"status":      {Type: "string", Description: "新状态: pending|running|completed|failed|cancelled"},
			"result":      {Type: "string", Description: "任务结果（可选）"},
		},
		Required: []string{"id"},
	}
}

func (t *TaskUpdateTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Status      string `json:"status"`
		Result      string `json:"result"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if err := task.GlobalRegistry().Update(args.ID, func(tk *task.Task) {
		tk.Update(args.Title, args.Description, args.Status, args.Result)
	}); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("更新任务失败: %v", err), IsError: true}
	}

	return types.ToolResult{Content: fmt.Sprintf("任务 %s 更新成功", args.ID)}
}

func (t *TaskUpdateTool) IsReadOnly(input json.RawMessage) bool  { return false }
func (t *TaskUpdateTool) IsConcurrencySafe(input json.RawMessage) bool { return true }
