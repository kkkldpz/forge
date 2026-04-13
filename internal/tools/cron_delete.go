package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/cron"
	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type CronDeleteTool struct {
	tool.BaseTool
}

func NewCronDeleteTool() *CronDeleteTool {
	return &CronDeleteTool{
		BaseTool: tool.BaseTool{
			NameStr:        "cron_delete",
			DescriptionStr: "删除定时任务",
		},
	}
}

type CronDeleteInput struct {
	ID string `json:"id"`
}

func (t *CronDeleteTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"id": {Type: "string", Description: "要删除的任务 ID"},
		},
		Required: []string{"id"},
	}
}

func (t *CronDeleteTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args CronDeleteInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if err := cron.GlobalStore().Delete(args.ID); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("删除失败: %v", err), IsError: true}
	}

	return types.ToolResult{Content: fmt.Sprintf("✓ 定时任务 %s 已删除", args.ID)}
}

func (t *CronDeleteTool) IsReadOnly(input json.RawMessage) bool  { return false }
func (t *CronDeleteTool) IsConcurrencySafe(input json.RawMessage) bool { return true }