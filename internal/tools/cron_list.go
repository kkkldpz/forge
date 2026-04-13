package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kkkldpz/forge/internal/cron"
	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

type CronListTool struct {
	toolkit.BaseTool
}

func NewCronListTool() *CronListTool {
	return &CronListTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "cron_list",
			DescriptionStr: "列出所有定时任务",
		},
	}
}

func (t *CronListTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{},
	}
}

func (t *CronListTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args struct{}
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	jobs := cron.GlobalStore().List()

	if len(jobs) == 0 {
		return types.ToolResult{Content: "没有定时任务"}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("共 %d 个定时任务:\n", len(jobs)))
	lines = append(lines, "ID                  状态    计划           命令")
	lines = append(lines, strings.Repeat("-", 70))

	for _, job := range jobs {
		status := "启用"
		if !job.Enabled {
			status = "停用"
		}
		lines = append(lines, fmt.Sprintf("%s  %s  %-14s %s", job.ID[:8], status, job.Schedule, job.Command))
	}

	return types.ToolResult{Content: strings.Join(lines, "\n")}
}

func (t *CronListTool) IsReadOnly(input json.RawMessage) bool  { return true }
func (t *CronListTool) IsConcurrencySafe(input json.RawMessage) bool { return true }