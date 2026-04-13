package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type SendTool struct {
	tool.BaseTool
}

func NewSendTool() *SendTool {
	return &SendTool{
		BaseTool: tool.BaseTool{
			NameStr:        "send",
			DescriptionStr: "发送消息到外部服务或通道",
		},
	}
}

type SendInput struct {
	Channel string `json:"channel"`
	Message string `json:"message"`
}

func (t *SendTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"channel": {Type: "string", Description: "目标通道或服务名称"},
			"message": {Type: "string", Description: "要发送的消息内容"},
		},
		Required: []string{"channel", "message"},
	}
}

func (t *SendTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args SendInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Channel == "" || args.Message == "" {
		return types.ToolResult{Content: "通道和消息不能为空", IsError: true}
	}

	content := fmt.Sprintf("消息已发送到 '%s':\n%s", args.Channel, args.Message)

	return types.ToolResult{Content: content}
}

func (t *SendTool) IsReadOnly(input json.RawMessage) bool { return false }
func (t *SendTool) IsConcurrencySafe(input json.RawMessage) bool { return true }