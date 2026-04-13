// Package engine 实现查询引擎编排器。
package engine

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kkkldpz/forge/internal/api"
	promptctx "github.com/kkkldpz/forge/internal/context"
	"github.com/kkkldpz/forge/internal/provider"
	"github.com/kkkldpz/forge/internal/query"
	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

// QueryEngineConfig 是查询引擎的配置。
type QueryEngineConfig struct {
	Cwd            string
	HomeDir        string
	Tools          []tool.Tool
	Provider       provider.Provider
	Model          string
	ThinkingConfig *api.ThinkingConfig
	MaxTurns       int
	MaxBudgetUSD   float64
}

// QueryEngine 是对话引擎。
type QueryEngine struct {
	config        QueryEngineConfig
	messages      []types.Message
	sessionID     string
	totalUsage    api.Usage
	totalCost     float64
	cancelFunc    context.CancelFunc
	systemPrompt  []api.SystemBlock
}

// NewQueryEngine 创建新的查询引擎，自动构建系统提示词和上下文。
func NewQueryEngine(config QueryEngineConfig) *QueryEngine {
	// 收集已启用的工具名称
	enabledTools := make([]string, 0, len(config.Tools))
	for _, t := range config.Tools {
		enabledTools = append(enabledTools, t.Name())
	}

	// 获取完整上下文
	promptParts, userCtx, sysCtx := promptctx.FetchContext(
		config.Model,
		config.Cwd,
		config.HomeDir,
		enabledTools,
	)

	// 组装最终 SystemBlock 列表
	systemBlocks := promptctx.BuildSystemBlocks(promptParts, userCtx, sysCtx)

	slog.Info("查询引擎初始化完成",
		"model", config.Model,
		"systemBlocks", len(systemBlocks),
		"tools", len(config.Tools),
	)

	return &QueryEngine{
		config:       config,
		messages:     make([]types.Message, 0),
		sessionID:    uuid.New().String(),
		systemPrompt: systemBlocks,
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

		params := query.QueryParams{
			Messages:       e.messages,
			SystemPrompt:   e.systemPrompt,
			Tools:          e.config.Tools,
			ToolUseContext: tool.ToolUseContext{WorkingDir: e.config.Cwd},
			Provider:       e.config.Provider,
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
