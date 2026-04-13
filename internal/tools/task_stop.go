package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/task"
	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type TaskStopTool struct {
	tool.BaseTool
}

func NewTaskStopTool() *TaskStopTool {
	return &TaskStopTool{
		BaseTool: tool.BaseTool{
			NameStr:        "task_stop",
			DescriptionStr: "取消正在运行的任务。",
		},
	}
}

func (t *TaskStopTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"id": {Type: "string", Description: "任务 ID"},
		},
		Required: []string{"id"},
	}
}

func (t *TaskStopTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	tk, err := task.GlobalRegistry().Stop(args.ID)
	if err != nil {
		return types.ToolResult{Content: fmt.Sprintf("停止任务失败: %v", err), IsError: true}
	}

	return types.ToolResult{Content: fmt.Sprintf("任务 %s 已取消\n标题: %s\n原状态: %s", tk.ID[:8], tk.Title, tk.Status)}
}

func (t *TaskStopTool) IsReadOnly(input json.RawMessage) bool  { return false }
func (t *TaskStopTool) IsConcurrencySafe(input json.RawMessage) bool { return true }