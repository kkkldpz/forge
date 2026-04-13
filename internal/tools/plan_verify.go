package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

type PlanVerifyTool struct {
	toolkit.BaseTool
}

func NewPlanVerifyTool() *PlanVerifyTool {
	return &PlanVerifyTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "plan_verify",
			DescriptionStr: "验证计划执行结果是否符合预期。",
		},
	}
}

type PlanVerifyInput struct {
	PlanID   string `json:"plan_id"`
	Step     string `json:"step"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
}

func (t *PlanVerifyTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"plan_id":  {Type: "string", Description: "计划 ID"},
			"step":    {Type: "string", Description: "要验证的步骤"},
			"expected": {Type: "string", Description: "预期结果"},
			"actual":   {Type: "string", Description: "实际结果"},
		},
		Required: []string{"step", "expected", "actual"},
	}
}

func (t *PlanVerifyTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args PlanVerifyInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Expected == args.Actual {
		return types.ToolResult{
			Content: fmt.Sprintf("✓ 验证通过: %s", args.Step),
			Extra: map[string]any{
				"verified": true,
				"step":    args.Step,
			},
		}
	}

	return types.ToolResult{
		Content: fmt.Sprintf("✗ 验证失败: %s\n预期: %s\n实际: %s", args.Step, args.Expected, args.Actual),
		IsError: false,
		Extra: map[string]any{
			"verified": false,
			"step":    args.Step,
		},
	}
}

func (t *PlanVerifyTool) IsReadOnly(input json.RawMessage) bool  { return true }
func (t *PlanVerifyTool) IsConcurrencySafe(input json.RawMessage) bool { return true }