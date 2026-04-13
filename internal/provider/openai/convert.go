// Package openai 实现 OpenAI 兼容 API 的流式适配器。
package openai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kkkldpz/forge/internal/api"
)

// ConvertMessages 将 Anthropic 格式的消息和 system 块转换为 OpenAI 消息列表。
func ConvertMessages(system []api.SystemBlock, anthropic []api.MessageParam) []OpenAIMessage {
	var result []OpenAIMessage

	// 将 system 块拼接为一条 role:"system" 消息
	if len(system) > 0 {
		var sb strings.Builder
		for _, block := range system {
			if block.Text != "" {
				if sb.Len() > 0 {
					sb.WriteByte('\n')
				}
				sb.WriteString(block.Text)
			}
		}
		if sb.Len() > 0 {
			result = append(result, OpenAIMessage{
				Role:    "system",
				Content: sb.String(),
			})
		}
	}

	// 逐条转换消息
	for _, msg := range anthropic {
		switch msg.Role {
		case "user":
			result = append(result, convertUserMessages(msg)...)
		case "assistant":
			result = append(result, convertAssistantMessage(msg))
		default:
			// 忽略未知角色
		}
	}

	return result
}

// convertUserMessages 将 user 消息从 Anthropic 格式转换为 OpenAI 格式。
// 可能产生多条消息（文本 + tool_result）。
func convertUserMessages(msg api.MessageParam) []OpenAIMessage {
	blocks := parseContentBlocks(msg.Content)
	var texts []string
	var toolResults []OpenAIMessage

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		case "tool_result":
			toolResults = append(toolResults, OpenAIMessage{
				Role:       "tool",
				ToolCallID: block.ToolUseID,
				Content:    formatToolResultContent(block.Content, block.IsError),
			})
		default:
			// 忽略未知块类型（如 image）
		}
	}

	var result []OpenAIMessage
	// 先输出文本部分
	if len(texts) > 0 {
		result = append(result, OpenAIMessage{
			Role:    "user",
			Content: strings.Join(texts, "\n"),
		})
	}
	// 再输出 tool_result 部分
	result = append(result, toolResults...)

	// 如果什么都没有，返回空 user 消息
	if len(result) == 0 {
		return []OpenAIMessage{{Role: "user", Content: ""}}
	}
	return result
}

// convertAssistantMessage 将 assistant 消息从 Anthropic 格式转换为 OpenAI 格式。
func convertAssistantMessage(msg api.MessageParam) OpenAIMessage {
	blocks := parseContentBlocks(msg.Content)
	var texts []string
	var toolCalls []ToolCall

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: FunctionCall{
					Name:      block.Name,
					Arguments: string(args),
				},
			})
		default:
			// 忽略 thinking 等块
		}
	}

	oai := OpenAIMessage{
		Role:      "assistant",
		ToolCalls: toolCalls,
	}
	if len(texts) > 0 {
		oai.Content = strings.Join(texts, "\n")
	}
	return oai
}

// parsedBlock 是解析后的内容块。
type parsedBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   any             `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// parseContentBlocks 将 Content (json.RawMessage) 解析为内容块列表。
// Content 可以是字符串或内容块数组。
func parseContentBlocks(raw json.RawMessage) []parsedBlock {
	if len(raw) == 0 {
		return nil
	}

	// 尝试解析为字符串
	var textStr string
	if err := json.Unmarshal(raw, &textStr); err == nil {
		if textStr != "" {
			return []parsedBlock{{Type: "text", Text: textStr}}
		}
		return nil
	}

	// 尝试解析为块数组
	var blocks []parsedBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		// 解析失败，当作纯文本处理
		return []parsedBlock{{Type: "text", Text: string(raw)}}
	}
	return blocks
}

// formatToolResultContent 将工具结果格式化为可读内容。
func formatToolResultContent(content any, isError bool) string {
	switch v := content.(type) {
	case string:
		return v
	case nil:
		if isError {
			return "错误：工具执行失败"
		}
		return ""
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

// ConvertTools 将 Anthropic 的 ToolParam 列表转换为 OpenAI 的工具定义。
func ConvertTools(tools []api.ToolParam) []OpenAITool {
	result := make([]OpenAITool, 0, len(tools))
	for _, t := range tools {
		result = append(result, OpenAITool{
			Type: "function",
			Function: FunctionObject{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}
	return result
}

// BuildRequest 构建完整的 OpenAI 请求体。
func BuildRequest(
	model string,
	messages []OpenAIMessage,
	tools []OpenAITool,
	opts *api.RequestOptions,
) *ChatCompletionRequest {
	req := &ChatCompletionRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	if len(tools) > 0 {
		req.Tools = tools
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			req.MaxTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			req.Temperature = opts.Temperature
		}
	}

	return req
}

// ValidateRequest 验证请求参数是否合法。
func ValidateRequest(req *ChatCompletionRequest) error {
	if req.Model == "" {
		return fmt.Errorf("model 不能为空")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("messages 不能为空")
	}
	return nil
}
