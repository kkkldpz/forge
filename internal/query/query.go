// Package query 实现核心对话循环。
package query

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/provider"
	"github.com/kkkldpz/forge/internal/toolkit"
	"github.com/kkkldpz/forge/internal/types"
)

// QueryParams 是 Query 函数的参数。
type QueryParams struct {
	Messages       []types.Message
	SystemPrompt   []api.SystemBlock
	Tools          []toolkit.Tool
	ToolUseContext toolkit.ToolUseContext
	Provider       provider.Provider
	MaxTurns       int
	MaxBudgetUSD   float64
	Model          string
	ThinkingConfig *api.ThinkingConfig
}

// QueryEvent 是 Query 函数产出的事件。
type QueryEvent struct {
	Type  string
	Usage *api.Usage
	Error error
}

// Query 是核心对话循环函数。
func Query(ctx context.Context, params QueryParams) <-chan QueryEvent {
	ch := make(chan QueryEvent, 64)

	go func() {
		defer close(ch)

		state := &queryState{
			messages:       params.Messages,
			toolUseContext: params.ToolUseContext,
			turnCount:      0,
		}

		for {
			select {
			case <-ctx.Done():
				ch <- QueryEvent{Type: "cancelled"}
				return
			default:
			}

			if params.MaxTurns > 0 && state.turnCount >= params.MaxTurns {
				ch <- QueryEvent{Type: "max_turns"}
				return
			}

			shouldContinue, err := executeTurn(ctx, state, params, ch)
			if err != nil {
				ch <- QueryEvent{Type: "error", Error: err}
				return
			}

			if !shouldContinue {
				ch <- QueryEvent{Type: "complete", Usage: &state.totalUsage}
				return
			}

			state.turnCount++
		}
	}()

	return ch
}

type queryState struct {
	messages       []types.Message
	toolUseContext toolkit.ToolUseContext
	turnCount      int
	totalUsage     api.Usage
	totalCostUSD   float64
}

func executeTurn(ctx context.Context, state *queryState, params QueryParams, ch chan<- QueryEvent) (bool, error) {
	logger := slog.Default().With("component", "query", "turn", state.turnCount)

	// 准备消息
	messagesForAPI := normalizeMessages(state.messages)

	// 构建请求选项
	opts := &api.RequestOptions{
		Model:     params.Model,
		MaxTokens: 16384,
		Thinking:  params.ThinkingConfig,
	}

	// 转换消息为 API 格式
	apiMessages := convertToAPIMessages(messagesForAPI)
	toolsParams := convertToolsToParams(params.Tools)

	logger.Debug("发送请求到 API", "messageCount", len(apiMessages))

	// 调用 API
	streamCh := params.Provider.QueryModel(ctx, apiMessages, params.SystemPrompt, toolsParams, opts)

	// 处理流响应
	processor := NewStreamProcessor()
	var hasToolUse bool

	for event := range streamCh {
		if errEvt, ok := event.(*types.ErrorEvent); ok {
			return false, fmt.Errorf("API 错误: %s - %s", errEvt.Error.Type, errEvt.Error.Message)
		}

		switch evt := event.(type) {
		case *types.MessageStartEvent:
			processor.HandleMessageStart(evt)
		case *types.ContentBlockStartEvent:
			processor.HandleContentBlockStart(evt)
		case *types.ContentBlockDeltaEvent:
			processor.HandleContentBlockDelta(evt)
		case *types.ContentBlockStopEvent:
			processor.HandleContentBlockStop(evt)
		case *types.MessageDeltaEvent:
			processor.HandleMessageDelta(evt)
		case *types.MessageStopEvent:
			assistantMsg := processor.BuildAssistantMessage()
			if assistantMsg != nil {
				state.messages = append(state.messages, *assistantMsg)
				ch <- QueryEvent{Type: "assistant"}

				// 检测 tool_use
				toolUses := detectToolUses(assistantMsg)
				hasToolUse = len(toolUses) > 0

				if hasToolUse {
					ch <- QueryEvent{Type: "tool_start"}

					// 执行工具
					results := toolkit.ExecuteTools(ctx, toolUses, params.Tools, params.ToolUseContext)

					for _, result := range results {
						toolResultMsg := createToolResultMessage(result)
						state.messages = append(state.messages, toolResultMsg)
						ch <- QueryEvent{Type: "tool_result"}
					}

					ch <- QueryEvent{Type: "tool_stop"}
				}
			}
		}
	}

	return hasToolUse, nil
}

func normalizeMessages(messages []types.Message) []types.Message {
	result := make([]types.Message, 0, len(messages))
	for _, msg := range messages {
		if msg.Type == types.MessageTypeUser || msg.Type == types.MessageTypeAssistant {
			result = append(result, msg)
		}
	}
	return result
}

func convertToAPIMessages(messages []types.Message) []api.MessageParam {
	apiMessages := make([]api.MessageParam, 0, len(messages))
	for _, msg := range messages {
		if msg.Message != nil {
			apiMessages = append(apiMessages, api.MessageParam{
				Role:    string(msg.Type),
				Content: msg.Message.Content,
			})
		}
	}
	return apiMessages
}

func convertToolsToParams(tools []toolkit.Tool) []api.ToolParam {
	params := make([]api.ToolParam, 0, len(tools))
	for _, t := range tools {
		schema := t.InputSchema()
		params = append(params, api.ToolParam{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: schema,
		})
	}
	return params
}

func detectToolUses(msg *types.Message) []toolkit.ToolCall {
	if msg.Message == nil || len(msg.Message.Content) == 0 {
		return nil
	}

	var toolCalls []toolkit.ToolCall
	var blocks []map[string]any
	if err := json.Unmarshal(msg.Message.Content, &blocks); err != nil {
		return nil
	}

	for _, block := range blocks {
		if blockType, ok := block["type"].(string); ok && blockType == "tool_use" {
			call := toolkit.ToolCall{
				ID:   getStr(block, "id"),
				Name: getStr(block, "name"),
			}
			if input, ok := block["input"]; ok {
				if inputMap, ok := input.(map[string]any); ok {
					inputJSON, _ := json.Marshal(inputMap)
					call.Input = inputJSON
				}
			}
			toolCalls = append(toolCalls, call)
		}
	}

	return toolCalls
}

func createToolResultMessage(result toolkit.ToolCallResult) types.Message {
	content := result.Result.Content
	if result.Error != nil {
		content = fmt.Sprintf("错误: %v", result.Error)
	}

	blocks := []map[string]any{
		{
			"type":        "tool_result",
			"tool_use_id": result.CallID,
			"content":     content,
			"is_error":    result.Result.IsError || result.Error != nil,
		},
	}

	contentJSON, _ := json.Marshal(blocks)

	return types.Message{
		Type: types.MessageTypeUser,
		Message: &types.MessageContent{
			Role:    "user",
			Content: contentJSON,
		},
	}
}

func getStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
