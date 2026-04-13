package gemini

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kkkldpz/forge/internal/api"
)

// ConvertMessages 将 Anthropic 格式的消息和 system 块转换为 Gemini 的 Content 列表和 SystemInstruction。
// 返回值：(contents, systemInstruction)
func ConvertMessages(system []api.SystemBlock, msgs []api.MessageParam) ([]Content, *Content) {
	var contents []Content
	var sysInstr *Content

	// 将 system 块拼接为 SystemInstruction
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
			sysInstr = &Content{
				Role:  "user",
				Parts: []Part{{Text: sb.String()}},
			}
		}
	}

	// 逐条转换消息
	for _, msg := range msgs {
		blocks := parseAnthropicContent(msg.Content)
		switch msg.Role {
		case "user":
			contents = append(contents, convertGeminiUserMessage(blocks)...)
		case "assistant":
			contents = append(contents, convertGeminiAssistantMessage(blocks))
		}
	}

	return contents, sysInstr
}

// convertGeminiUserMessage 将 Anthropic user 消息转换为 Gemini 格式。
// tool_result 在 Gemini 中使用 role:"user" + functionResponse parts。
func convertGeminiUserMessage(blocks []anthropicBlock) []Content {
	var texts []string
	var funcResps []Part

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				texts = append(texts, block.Text)
			}
		case "tool_result":
			resp := formatFunctionResponse(block.ToolUseID, block.Content, block.IsError)
			funcResps = append(funcResps, resp)
		}
	}

	var result []Content

	// 文本部分
	if len(texts) > 0 {
		result = append(result, Content{
			Role:  "user",
			Parts: []Part{{Text: strings.Join(texts, "\n")}},
		})
	}

	// 函数响应部分
	if len(funcResps) > 0 {
		result = append(result, Content{
			Role:  "user",
			Parts: funcResps,
		})
	}

	// 空消息保护
	if len(result) == 0 {
		return []Content{{Role: "user", Parts: []Part{{Text: ""}}}}
	}
	return result
}

// convertGeminiAssistantMessage 将 Anthropic assistant 消息转换为 Gemini 格式。
// tool_use 在 Gemini 中使用 role:"model" + functionCall parts。
func convertGeminiAssistantMessage(blocks []anthropicBlock) Content {
	var parts []Part

	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				parts = append(parts, Part{Text: block.Text})
			}
		case "tool_use":
			parts = append(parts, Part{
				FunctionCall: &FunctionCall{
					Name: block.Name,
					Args: block.Input,
				},
			})
		case "thinking":
			if block.Thinking != "" {
				parts = append(parts, Part{
					Text:    block.Thinking,
					Thought: true,
				})
			}
		}
	}

	// 空内容保护
	if len(parts) == 0 {
		parts = []Part{{Text: ""}}
	}

	return Content{Role: "model", Parts: parts}
}

// formatFunctionResponse 将工具结果格式化为 Gemini 的 FunctionResponse。
func formatFunctionResponse(toolUseID string, content any, isError bool) Part {
	// Gemini 的 FunctionResponse 使用 name 和 response 字段
	// toolUseID 在 Gemini 协议中不直接映射，放入 response 中
	var respContent any
	switch v := content.(type) {
	case string:
		if isError {
			respContent = map[string]any{"error": v}
		} else {
			respContent = map[string]any{"result": v}
		}
	case nil:
		if isError {
			respContent = map[string]any{"error": "工具执行失败"}
		} else {
			respContent = map[string]any{"result": ""}
		}
	default:
		respContent = v
	}

	respBytes, err := json.Marshal(respContent)
	if err != nil {
		respBytes = []byte(`{"error":"序列化工具结果失败"}`)
	}

	// 使用 toolUseID 作为名称（Gemini 要求 name 字段）
	return Part{
		FunctionResp: &FunctionResp{
			Name:     toolUseID,
			Response: respBytes,
		},
	}
}

// ConvertTools 将 Anthropic 的 ToolParam 列表转换为 Gemini 的 ToolDecl 列表。
func ConvertTools(tools []api.ToolParam) []ToolDecl {
	if len(tools) == 0 {
		return nil
	}

	result := make([]ToolDecl, 1, 1) // 所有函数声明放在同一个 ToolDecl 中
	funcs := make([]FuncDecl, 0, len(tools))

	for _, t := range tools {
		params, _ := json.Marshal(t.InputSchema)
		funcs = append(funcs, FuncDecl{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  params,
		})
	}

	result[0] = ToolDecl{FunctionDecls: funcs}
	return result
}

// anthropicBlock 是解析后的 Anthropic 内容块。
type anthropicBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   any             `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
}

// parseAnthropicContent 将 json.RawMessage 解析为 Anthropic 内容块列表。
func parseAnthropicContent(raw json.RawMessage) []anthropicBlock {
	if len(raw) == 0 {
		return nil
	}

	// 尝试解析为字符串
	var textStr string
	if err := json.Unmarshal(raw, &textStr); err == nil {
		if textStr != "" {
			return []anthropicBlock{{Type: "text", Text: textStr}}
		}
		return nil
	}

	// 尝试解析为块数组
	var blocks []anthropicBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return []anthropicBlock{{Type: "text", Text: string(raw)}}
	}
	return blocks
}

// BuildRequest 构建完整的 Gemini 请求体。
func BuildRequest(
	model string,
	contents []Content,
	sysInstr *Content,
	tools []ToolDecl,
	opts *api.RequestOptions,
) *GenerateContentRequest {
	req := &GenerateContentRequest{
		Contents: contents,
	}

	if sysInstr != nil {
		req.SystemInstruction = sysInstr
	}

	if len(tools) > 0 {
		req.Tools = tools
	}

	if opts != nil {
		genConfig := &GenConfig{}
		if opts.MaxTokens > 0 {
			genConfig.MaxOutputTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			genConfig.Temperature = opts.Temperature
		}
		if opts.Thinking != nil && opts.Thinking.Type == "enabled" && opts.Thinking.BudgetTokens > 0 {
			genConfig.ThinkingConfig = &ThinkingConfig{
				ThinkingBudget: opts.Thinking.BudgetTokens,
			}
		}
		req.GenerationConfig = genConfig
	}

	// model 不在请求体中，而是在 URL 路径中，这里保留字段便于日志记录
	_ = model

	return req
}

// ValidateRequest 验证请求参数是否合法。
func ValidateRequest(req *GenerateContentRequest) error {
	if len(req.Contents) == 0 {
		return fmt.Errorf("contents 不能为空")
	}
	return nil
}
