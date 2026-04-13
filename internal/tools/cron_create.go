package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kkkldpz/forge/internal/cron"
	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type CronCreateTool struct {
	tool.BaseTool
}

func NewCronCreateTool() *CronCreateTool {
	return &CronCreateTool{
		BaseTool: tool.BaseTool{
			NameStr:        "cron_create",
			DescriptionStr: "创建一个新的定时任务",
		},
	}
}

type CronCreateInput struct {
	Command     string `json:"command"`
	Schedule    string `json:"schedule"`
	Description string `json:"description,omitempty"`
}

func (t *CronCreateTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"command":     {Type: "string", Description: "要执行的命令"},
			"schedule":    {Type: "string", Description: "Cron 表达式 (例如: * * * * *)"},
			"description": {Type: "string", Description: "任务描述（可选）"},
		},
		Required: []string{"command", "schedule"},
	}
}

func (t *CronCreateTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args CronCreateInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Command == "" {
		return types.ToolResult{Content: "命令不能为空", IsError: true}
	}
	if args.Schedule == "" {
		return types.ToolResult{Content: "定时计划不能为空", IsError: true}
	}

	job := &cron.Job{
		ID:          uuid.New().String(),
		Command:     args.Command,
		Schedule:    args.Schedule,
		Description: args.Description,
		Enabled:     true,
		CreatedAt:   time.Now(),
	}

	if err := cron.GlobalStore().Create(job); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("创建定时任务失败: %v", err), IsError: true}
	}

	return types.ToolResult{
		Content: fmt.Sprintf("✓ 定时任务创建成功\nID: %s\n命令: %s\n计划: %s", job.ID, job.Command, job.Schedule),
	}
}

func (t *CronCreateTool) IsReadOnly(input json.RawMessage) bool  { return false }
func (t *CronCreateTool) IsConcurrencySafe(input json.RawMessage) bool { return true }