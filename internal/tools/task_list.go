package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kkkldpz/forge/internal/task"
	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

type TaskListTool struct {
	toolkit.BaseTool
}

func NewTaskListTool() *TaskListTool {
	return &TaskListTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "task_list",
			DescriptionStr: "列出所有任务或按状态筛选。",
		},
	}
}

func (t *TaskListTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"status": {
				Type:        "string",
				Description: "按状态筛选: pending, running, completed, failed, cancelled",
				Enum:        []string{"pending", "running", "completed", "failed", "cancelled"},
			},
		},
	}
}

func (t *TaskListTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	var tasks []*task.Task
	if args.Status != "" {
		status := task.Status(strings.ToLower(args.Status))
		tasks = task.GlobalRegistry().ListByStatus(status)
	} else {
		tasks = task.GlobalRegistry().List()
	}

	if len(tasks) == 0 {
		return types.ToolResult{Content: "没有找到任务"}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("共 %d 个任务:\n", len(tasks)))
	lines = append(lines, "ID                  状态       标题")
	lines = append(lines, strings.Repeat("-", 60))

	for _, tk := range tasks {
		lines = append(lines, fmt.Sprintf("%s  %-10s  %s", tk.ID[:8], tk.Status, tk.Title))
	}

	return types.ToolResult{Content: strings.Join(lines, "\n")}
}

func (t *TaskListTool) IsReadOnly(input json.RawMessage) bool  { return true }
func (t *TaskListTool) IsConcurrencySafe(input json.RawMessage) bool { return true }