// Package compact 实现上下文压缩，包括完整压缩、自动压缩和微压缩。
package compact

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/types"
)

const (
	// AutocompactBufferTokens 自动压缩触发后的缓冲 token 数。
	AutocompactBufferTokens = 13_000
	// WarningThresholdTokens 警告阈值缓冲。
	WarningThresholdTokens = 20_000
	// DefaultContextWindow 默认上下文窗口大小。
	DefaultContextWindow = 200_000
	// MaxConsecutiveFailures 最大连续压缩失败次数。
	MaxConsecutiveFailures = 3
	// MicroCompactLimit 微压缩单条工具结果的最大字符数。
	MicroCompactLimit = 8000
)

// CompactionResult 压缩结果。
type CompactionResult struct {
	// Summary 压缩后的摘要文本。
	Summary string
	// MessagesRemoved 被移除的消息数。
	MessagesRemoved int
	// MessagesKept 保留的消息数。
	MessagesKept int
	// PreTokenCount 压缩前估算 token 数。
	PreTokenCount int
	// PostTokenCount 压缩后估算 token 数。
	PostTokenCount int
}

// AutoCompactTracker 自动压缩状态追踪。
type AutoCompactTracker struct {
	consecutiveFailures int
	lastCompactTokens   int
}

// NewAutoCompactTracker 创建自动压缩追踪器。
func NewAutoCompactTracker() *AutoCompactTracker {
	return &AutoCompactTracker{}
}

// ShouldCompact 判断是否需要自动压缩。
// tokenUsage: 当前 token 使用量（估算）
// contextWindow: 模型上下文窗口大小
func (t *AutoCompactTracker) ShouldCompact(tokenUsage, contextWindow int) bool {
	if contextWindow <= 0 {
		contextWindow = DefaultContextWindow
	}
	threshold := contextWindow - AutocompactBufferTokens
	return tokenUsage >= threshold
}

// ShouldWarn 判断是否需要发出 token 警告。
func ShouldWarn(tokenUsage, contextWindow int) bool {
	if contextWindow <= 0 {
		contextWindow = DefaultContextWindow
	}
	threshold := contextWindow - WarningThresholdTokens
	return tokenUsage >= threshold
}

// TokenWarningState 返回 token 使用状态的详细信息。
type TokenWarningState struct {
	PercentUsed               float64
	IsAboveWarning            bool
	IsAboveAutoCompact        bool
	IsAtBlockingLimit         bool
}

// CalculateTokenState 计算 token 使用状态。
func CalculateTokenState(used, window int) TokenWarningState {
	if window <= 0 {
		window = DefaultContextWindow
	}
	pct := float64(used) / float64(window) * 100
	return TokenWarningState{
		PercentUsed:        pct,
		IsAboveWarning:     used >= window-WarningThresholdTokens,
		IsAboveAutoCompact: used >= window-AutocompactBufferTokens,
		IsAtBlockingLimit:  used >= window-5000,
	}
}

// RecordFailure 记录一次压缩失败。
func (t *AutoCompactTracker) RecordFailure() {
	t.consecutiveFailures++
}

// RecordSuccess 记录一次压缩成功。
func (t *AutoCompactTracker) RecordSuccess(tokens int) {
	t.consecutiveFailures = 0
	t.lastCompactTokens = tokens
}

// IsCircuitOpen 熔断器是否打开（连续失败过多）。
func (t *AutoCompactTracker) IsCircuitOpen() bool {
	return t.consecutiveFailures >= MaxConsecutiveFailures
}

// CompactConversation 完整压缩：调用 API 生成对话摘要，替换历史消息。
func CompactConversation(
	ctx context.Context,
	client *api.Client,
	messages []types.Message,
	model string,
) (*CompactionResult, error) {
	if len(messages) <= 2 {
		return nil, fmt.Errorf("消息太少，无需压缩")
	}

	slog.Info("开始压缩对话", "消息数", len(messages), "模型", model)

	// 1. 将消息格式化为摘要请求
	conversationText := formatConversationForSummary(messages)
	summaryPrompt := buildCompactPrompt(conversationText)

	// 2. 调用 API 生成摘要
	summary, err := requestSummary(ctx, client, model, summaryPrompt)
	if err != nil {
		return nil, fmt.Errorf("压缩请求失败: %w", err)
	}

	if strings.TrimSpace(summary) == "" {
		return nil, fmt.Errorf("压缩摘要为空")
	}

	preTokens := estimateTokenCount(formatMessagesAsString(messages))

	// 3. 构建压缩后的消息列表
	result := &CompactionResult{
		Summary:          summary,
		MessagesRemoved:  len(messages),
		PreTokenCount:    preTokens,
	}

	slog.Info("压缩完成",
		"移除消息", result.MessagesRemoved,
		"压缩前token", preTokens,
		"摘要长度", len(summary),
	)

	return result, nil
}

// MicroCompact 微压缩：裁剪过大的工具结果。
func MicroCompact(messages []types.Message) []types.Message {
	result := make([]types.Message, 0, len(messages))
	trimmed := 0

	for _, msg := range messages {
		if msg.Message != nil && len(msg.Message.Content) > 0 {
			trimmedContent := trimToolResultContent(msg.Message.Content, MicroCompactLimit)
			if len(trimmedContent) < len(msg.Message.Content) {
				trimmed++
				newMsg := msg
				newMsg.Message = &types.MessageContent{
					Role:    msg.Message.Role,
					Content: trimmedContent,
				}
				result = append(result, newMsg)
				continue
			}
		}
		result = append(result, msg)
	}

	if trimmed > 0 {
		slog.Debug("微压缩完成", "裁剪消息数", trimmed)
	}

	return result
}

// trimToolResultContent 裁剪工具结果内容到最大字符数。
func trimToolResultContent(content []byte, maxChars int) []byte {
	if len(content) <= maxChars {
		return content
	}

	// 检查是否为工具结果 JSON 数组
	str := string(content)
	if strings.HasPrefix(str, "[") {
		// 简化处理：直接截断并添加提示
		truncated := str[:maxChars]
		truncated += "\n\n... (内容过长，已截断。如需完整内容请重新执行工具调用)"
		return []byte(truncated)
	}

	return content
}

// formatConversationForSummary 将消息列表格式化为用于摘要的文本。
func formatConversationForSummary(messages []types.Message) string {
	var sb strings.Builder

	for _, msg := range messages {
		role := string(msg.Type)
		if msg.Message != nil {
			role = msg.Message.Role
		}

		content := formatMessageContent(msg)
		if content == "" {
			continue
		}

		sb.WriteString(fmt.Sprintf("[%s]: %s\n\n", role, content))
	}

	return sb.String()
}

// formatMessageContent 格式化单条消息的内容。
func formatMessageContent(msg types.Message) string {
	if msg.Message == nil || len(msg.Message.Content) == 0 {
		return ""
	}

	content := string(msg.Message.Content)

	// 截断过长的消息
	if len(content) > 4000 {
		return content[:4000] + "\n... (已截断)"
	}

	return content
}

// formatMessagesAsString 将所有消息格式化为单行字符串。
func formatMessagesAsString(messages []types.Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		if msg.Message != nil {
			sb.Write(msg.Message.Content)
			sb.WriteString(" ")
		}
	}
	return sb.String()
}

// buildCompactPrompt 构建压缩用的提示词。
func buildCompactPrompt(conversation string) string {
	return fmt.Sprintf(`请总结以下对话的关键信息。保持简洁但完整，包含以下要素：

1. **主要请求和意图**：用户的核心目标
2. **关键技术概念**：涉及的技术栈、框架、工具
3. **文件和代码**：涉及的关键文件路径和代码变更
4. **错误和修复**：遇到的问题及解决方案
5. **当前工作状态**：进行到哪一步，还有什么待完成

以下是完整对话记录：

---

%s

---

请生成简洁的摘要：`, conversation)
}

// requestSummary 调用 API 生成摘要。
func requestSummary(ctx context.Context, client *api.Client, model string, prompt string) (string, error) {
	// 构建简单的摘要请求
	messages := []api.MessageParam{
		api.NewTextMessageParam("user", prompt),
	}

	systemBlocks := []api.SystemBlock{
		api.NewSystemTextBlock("你是一个对话总结助手。请将对话历史压缩为简洁但完整的摘要，保留所有关键信息。"),
	}

	opts := &api.RequestOptions{
		Model:     model,
		MaxTokens: 4096,
	}

	// 使用非流式请求
	streamCh := client.QueryModel(ctx, messages, systemBlocks, nil, opts)

	var fullText string
	for event := range streamCh {
		switch evt := event.(type) {
		case *types.ContentBlockDeltaEvent:
			if evt.Text != "" {
				fullText += evt.Text
			}
		case *types.ErrorEvent:
			return "", fmt.Errorf("API 错误: %s", evt.Error.Message)
		}
	}

	return fullText, nil
}

// estimateTokenCount 粗略估算 token 数（4 字符约 1 token）。
func estimateTokenCount(text string) int {
	return len(text) / 4
}
