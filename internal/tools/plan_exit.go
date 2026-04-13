package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

type PlanExitTool struct {
	toolkit.BaseTool
}

func NewPlanExitTool() *PlanExitTool {
	return &PlanExitTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "plan_exit",
			DescriptionStr: "退出计划模式，恢复正常执行。",
		},
	}
}

func (t *PlanExitTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"summary": {Type: "string", Description: "计划执行总结（可选）"},
		},
	}
}

func (t *PlanExitTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	content := "已退出计划模式"
	if args.Summary != "" {
		content += fmt.Sprintf("\n总结: %s", args.Summary)
	}
	content += "\n\n现在可以执行实际的修改操作。"

	return types.ToolResult{Content: content}
}

func (t *PlanExitTool) IsReadOnly(input json.RawMessage) bool  { return true }
func (t *PlanExitTool) IsConcurrencySafe(input json.RawMessage) bool { return true }