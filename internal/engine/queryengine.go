// Package engine 实现查询引擎编排器。
package engine

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/query"
	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

// QueryEngineConfig 是查询引擎的配置。
type QueryEngineConfig struct {
	Cwd              string
	Tools            []tool.Tool
	APIClient        *api.Client
	SystemPrompt     []api.SystemBlock
	Model            string
	ThinkingConfig   *api.ThinkingConfig
	MaxTurns         int
	MaxBudgetUSD     float64
}

// QueryEngine 是对话引擎。
type QueryEngine struct {
	config      QueryEngineConfig
	messages    []types.Message
	sessionID   string
	totalUsage  api.Usage
	totalCost   float64
	cancelFunc  context.CancelFunc
}

// NewQueryEngine 创建新的查询引擎。
func NewQueryEngine(config QueryEngineConfig) *QueryEngine {
	return &QueryEngine{
		config:    config,
		messages:  make([]types.Message, 0),
		sessionID: uuid.New().String(),
	}
}

// SubmitMessage 提交用户消息并获取回复。
func (e *QueryEngine) SubmitMessage(ctx context.Context, prompt string) <-chan query.QueryEvent {
	ch := make(chan query.QueryEvent, 64)

	// 添加用户消息
	userMsg := types.Message{
		Type: types.MessageTypeUser,
		Message: &types.MessageContent{
			Role:    "user",
			Content: []byte(fmt.Sprintf(`"%s"`, prompt)),
		},
	}
	e.messages = append(e.messages, userMsg)

	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	e.cancelFunc = cancel

	go func() {
		defer close(ch)

		// 调用 query
		params := query.QueryParams{
			Messages:       e.messages,
			SystemPrompt:   e.config.SystemPrompt,
			Tools:          e.config.Tools,
			ToolUseContext: tool.ToolUseContext{WorkingDir: e.config.Cwd},
			APIClient:      e.config.APIClient,
			MaxTurns:       e.config.MaxTurns,
			MaxBudgetUSD:   e.config.MaxBudgetUSD,
			Model:          e.config.Model,
			ThinkingConfig: e.config.ThinkingConfig,
		}

		for event := range query.Query(ctx, params) {
			ch <- event

			// 累积用量
			if event.Usage != nil {
				e.totalUsage.InputTokens += event.Usage.InputTokens
				e.totalUsage.OutputTokens += event.Usage.OutputTokens
			}
		}
	}()

	return ch
}

// Interrupt 中断当前对话。
func (e *QueryEngine) Interrupt() {
	if e.cancelFunc != nil {
		e.cancelFunc()
	}
}

// GetMessages 获取完整消息历史。
func (e *QueryEngine) GetMessages() []types.Message {
	return e.messages
}

// GetSessionID 获取会话 ID。
func (e *QueryEngine) GetSessionID() string {
	return e.sessionID
}

// GetTotalCost 获取累积费用。
func (e *QueryEngine) GetTotalCost() float64 {
	return e.totalCost
}

// GetTotalUsage 获取累积用量。
func (e *QueryEngine) GetTotalUsage() api.Usage {
	return e.totalUsage
}
