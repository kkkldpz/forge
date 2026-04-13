package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type PlanEnterTool struct {
	tool.BaseTool
}

func NewPlanEnterTool() *PlanEnterTool {
	return &PlanEnterTool{
		BaseTool: tool.BaseTool{
			NameStr:        "plan_enter",
			DescriptionStr: "进入计划模式，用于详细规划和验证复杂任务的执行步骤。",
		},
	}
}

func (t *PlanEnterTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"goal": {Type: "string", Description: "计划目标"},
		},
		Required: []string{"goal"},
	}
}

func (t *PlanEnterTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args struct {
		Goal string `json:"goal"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	content := fmt.Sprintf("已进入计划模式\n目标: %s\n\n请详细描述完成此目标的步骤计划。", args.Goal)

	return types.ToolResult{Content: content}
}

func (t *PlanEnterTool) IsReadOnly(input json.RawMessage) bool  { return true }
func (t *PlanEnterTool) IsConcurrencySafe(input json.RawMessage) bool { return true }