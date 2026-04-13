package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/kkkldpz/forge/internal/task"
	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

// TaskCreateTool 创建异步任务。
type TaskCreateTool struct {
	toolkit.BaseTool
}

// NewTaskCreateTool 创建新的 TaskCreate 工具。
func NewTaskCreateTool() *TaskCreateTool {
	return &TaskCreateTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "task_create",
			DescriptionStr: "创建一个新的异步任务，用于跟踪工作进度。",
		},
	}
}

func (t *TaskCreateTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"title":       {Type: "string", Description: "任务标题"},
			"description": {Type: "string", Description: "任务描述"},
		},
		Required: []string{"title"},
	}
}

func (t *TaskCreateTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	tk := &task.Task{
		ID:          uuid.New().String(),
		Title:       args.Title,
		Description: args.Description,
		Status:      task.StatusPending,
	}

	if err := task.GlobalRegistry().Create(tk); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("创建任务失败: %v", err), IsError: true}
	}

	return types.ToolResult{
		Content: fmt.Sprintf("任务创建成功: %s (%s)", tk.ID, tk.Title),
	}
}

func (t *TaskCreateTool) IsReadOnly(input json.RawMessage) bool  { return false }
func (t *TaskCreateTool) IsConcurrencySafe(input json.RawMessage) bool { return true }
