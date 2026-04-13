package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

type AskUserTool struct {
	toolkit.BaseTool
}

func NewAskUserTool() *AskUserTool {
	return &AskUserTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "ask_user",
			DescriptionStr: "向用户提问并等待回答。",
		},
	}
}

type AskUserInput struct {
	Question string   `json:"question"`
	Options  []string `json:"options,omitempty"`
}

type AskUserOutput struct {
	Answer   string `json:"answer"`
	Cancelled bool  `json:"cancelled"`
}

func (t *AskUserTool) InputSchema() types.ToolInputJSONSchema {
	schema := types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"question": {Type: "string", Description: "要向用户提问的问题"},
		},
		Required: []string{"question"},
	}
	return schema
}

func (t *AskUserTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args AskUserInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Question == "" {
		return types.ToolResult{Content: "问题不能为空", IsError: true}
	}

	content := "问题: " + args.Question + "\n\n请在 TUI 中回答此问题。"

	return types.ToolResult{
		Content: content,
		Extra: map[string]any{
			"question": args.Question,
			"options":  args.Options,
		},
	}
}

func (t *AskUserTool) IsReadOnly(input json.RawMessage) bool  { return true }
func (t *AskUserTool) IsConcurrencySafe(input json.RawMessage) bool { return true }

type SleepTool struct {
	toolkit.BaseTool
}

func NewSleepTool() *SleepTool {
	return &SleepTool{
		BaseTool: toolkit.BaseTool{
			NameStr:        "sleep",
			DescriptionStr: "暂停执行指定时间。",
		},
	}
}

type SleepInput struct {
	DurationMs int `json:"duration_ms"`
}

func (t *SleepTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"duration_ms": {Type: "number", Description: "暂停时长（毫秒）"},
		},
		Required: []string{"duration_ms"},
	}
}

func (t *SleepTool) Call(ctx context.Context, input json.RawMessage, tuc toolkit.ToolUseContext) types.ToolResult {
	var args SleepInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.DurationMs <= 0 {
		return types.ToolResult{Content: "暂停时长必须大于 0", IsError: true}
	}

	select {
	case <-ctx.Done():
		return types.ToolResult{Content: "睡眠被取消", IsError: true}
	case <-time.After(time.Duration(args.DurationMs) * time.Millisecond):
		return types.ToolResult{Content: fmt.Sprintf("已暂停 %d 毫秒", args.DurationMs)}
	}
}

func (t *SleepTool) IsReadOnly(input json.RawMessage) bool  { return true }
func (t *SleepTool) IsConcurrencySafe(input json.RawMessage) bool { return true }