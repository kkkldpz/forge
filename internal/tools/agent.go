package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kkkldpz/forge/internal/api"
	"github.com/kkkldpz/forge/internal/engine"
	"github.com/kkkldpz/forge/internal/provider"
	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

// AgentTool 实现子代理执行工具。
// 通过创建独立的 QueryEngine 实例运行子任务。
type AgentTool struct {
	tool.BaseTool
	mu      sync.RWMutex
	agents  map[string]*AgentRun
	results map[string]*AgentResult
}

// AgentRun 跟踪正在运行的代理。
type AgentRun struct {
	ID        string
	Name      string
	Prompt    string
	Status    string // "running", "completed", "failed", "cancelled"
	StartedAt time.Time
	cancel    context.CancelFunc
}

// AgentResult 存储代理执行的最终结果。
type AgentResult struct {
	ID        string
	Name      string
	Status    string
	Output    string
	Usage     api.Usage
	CostUSD   float64
	Error     error
	StartedAt time.Time
	EndedAt   time.Time
}

// AgentInput 是 agent 工具的输入参数。
type AgentInput struct {
	Prompt          string `json:"prompt"`
	Description     string `json:"description"`
	Name            string `json:"name,omitempty"`
	Model           string `json:"model,omitempty"`
	RunInBackground bool   `json:"run_in_background"`
	MaxTurns        int    `json:"max_turns,omitempty"`
	Isolation       string `json:"isolation,omitempty"` // "worktree" 或空
}

func NewAgentTool() *AgentTool {
	return &AgentTool{
		BaseTool: tool.BaseTool{
			NameStr:        "agent",
			DescriptionStr: "启动子代理独立执行复杂任务，子代理运行在独立的对话循环中并拥有完整的工具访问权限",
		},
		agents:  make(map[string]*AgentRun),
		results: make(map[string]*AgentResult),
	}
}

func (t *AgentTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"prompt":            {Type: "string", Description: "子代理需要执行的任务描述"},
			"description":       {Type: "string", Description: "任务简短摘要（3-5 个词）"},
			"name":              {Type: "string", Description: "代理名称（为空时自动生成）"},
			"model":             {Type: "string", Description: "可选的模型覆盖"},
			"run_in_background": {Type: "boolean", Description: "异步执行并立即返回（默认 false）"},
			"max_turns":         {Type: "number", Description: "子代理最大对话轮数"},
			"isolation":         {Type: "string", Description: "隔离模式：'worktree' 表示 git worktree 隔离"},
		},
		Required: []string{"prompt"},
	}
}

func (t *AgentTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args AgentInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Prompt == "" {
		return types.ToolResult{Content: "任务描述不能为空", IsError: true}
	}

	// 从上下文中获取 Provider
	prov, ok := tuc.Provider.(provider.Provider)
	if !ok || prov == nil {
		return types.ToolResult{Content: "agent 工具需要配置 API Provider", IsError: true}
	}

	// 确定使用的模型
	model := args.Model
	if model == "" {
		model = tuc.Model
	}

	agentName := args.Name
	if agentName == "" {
		agentName = fmt.Sprintf("agent-%d", len(t.agents)+1)
	}

	agentID := uuid.New().String()[:8]

	// 构建子代理工具列表（排除 agent 工具以防无限嵌套）
	subTools := filterToolsForSubAgent(tuc.Tools)

	// 创建代理运行记录
	run := &AgentRun{
		ID:        agentID,
		Name:      agentName,
		Prompt:    args.Prompt,
		Status:    "running",
		StartedAt: time.Now(),
	}
	t.mu.Lock()
	t.agents[agentID] = run
	t.mu.Unlock()

	if args.RunInBackground {
		return t.runAsync(ctx, run, args, prov, model, subTools, tuc)
	}
	return t.runSync(ctx, run, args, prov, model, subTools, tuc)
}

// runSync 同步执行代理并返回结果。
func (t *AgentTool) runSync(
	ctx context.Context,
	run *AgentRun,
	args AgentInput,
	prov provider.Provider,
	model string,
	subTools []tool.Tool,
	tuc tool.ToolUseContext,
) types.ToolResult {
	logger := slog.Default().With("component", "agent", "agent_id", run.ID, "agent_name", run.Name)

	// 创建可取消的上下文
	agentCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	run.cancel = cancel

	// 构建子引擎配置
	engineCfg := engine.QueryEngineConfig{
		Cwd:      tuc.WorkingDir,
		HomeDir:  tuc.HomeDir,
		Tools:    subTools,
		Provider: prov,
		Model:    model,
		MaxTurns: args.MaxTurns,
	}

	logger.Info("启动子代理", "prompt_length", len(args.Prompt))

	// 创建子引擎并执行
	eng := engine.NewQueryEngine(engineCfg)
	eventCh := eng.SubmitMessage(agentCtx, args.Prompt)
	var output string
	var totalUsage api.Usage

	for evt := range eventCh {
		switch evt.Type {
		case "assistant":
			// 累积助手响应
		case "tool_result":
			// 跟踪工具执行
		case "complete":
			if evt.Usage != nil {
				totalUsage = *evt.Usage
			}
			// 获取引擎最终消息
			messages := eng.GetMessages()
			output = extractLastAssistantText(messages)
		case "error":
			run.Status = "failed"
			return t.buildResult(run, args, "", totalUsage, evt.Error)
		case "cancelled":
			run.Status = "cancelled"
			return types.ToolResult{
				Content: fmt.Sprintf("代理 %s 已被取消", run.Name),
				Extra:   map[string]any{"agent_id": run.ID, "status": "cancelled"},
			}
		}
	}

	run.Status = "completed"
	cost := api.CalculateCost(model, &totalUsage)

	result := &AgentResult{
		ID:        run.ID,
		Name:      run.Name,
		Status:    "completed",
		Output:    output,
		Usage:     totalUsage,
		CostUSD:   cost,
		StartedAt: run.StartedAt,
		EndedAt:   time.Now(),
	}

	t.mu.Lock()
	t.results[run.ID] = result
	t.mu.Unlock()

	logger.Info("子代理执行完成",
		"input_tokens", totalUsage.InputTokens,
		"output_tokens", totalUsage.OutputTokens,
		"cost", api.FormatCost(cost),
	)

	return types.ToolResult{
		Content: formatAgentOutput(result),
		Extra: map[string]any{
			"agent_id":      run.ID,
			"status":        "completed",
			"input_tokens":  totalUsage.InputTokens,
			"output_tokens": totalUsage.OutputTokens,
			"cost_usd":      cost,
		},
	}
}

// runAsync 在后台 goroutine 中启动代理并立即返回。
func (t *AgentTool) runAsync(
	ctx context.Context,
	run *AgentRun,
	args AgentInput,
	prov provider.Provider,
	model string,
	subTools []tool.Tool,
	tuc tool.ToolUseContext,
) types.ToolResult {
	agentCtx, cancel := context.WithCancel(ctx)
	run.cancel = cancel

	go func() {
		defer cancel()
		logger := slog.Default().With("component", "agent", "agent_id", run.ID)

		engineCfg := engine.QueryEngineConfig{
			Cwd:      tuc.WorkingDir,
			HomeDir:  tuc.HomeDir,
			Tools:    subTools,
			Provider: prov,
			Model:    model,
			MaxTurns: args.MaxTurns,
		}

		eng := engine.NewQueryEngine(engineCfg)
		eventCh := eng.SubmitMessage(agentCtx, args.Prompt)

		var totalUsage api.Usage
		var output string
		var agentErr error

		for evt := range eventCh {
			if evt.Usage != nil {
				totalUsage.InputTokens += evt.Usage.InputTokens
				totalUsage.OutputTokens += evt.Usage.OutputTokens
			}
			if evt.Type == "complete" {
				messages := eng.GetMessages()
				output = extractLastAssistantText(messages)
			}
			if evt.Type == "error" {
				agentErr = evt.Error
			}
		}

		status := "completed"
		if agentErr != nil {
			status = "failed"
		}

		t.mu.Lock()
		run.Status = status
		t.results[run.ID] = &AgentResult{
			ID:        run.ID,
			Name:      run.Name,
			Status:    status,
			Output:    output,
			Usage:     totalUsage,
			CostUSD:   api.CalculateCost(model, &totalUsage),
			Error:     agentErr,
			StartedAt: run.StartedAt,
			EndedAt:   time.Now(),
		}
		t.mu.Unlock()

		logger.Info("后台代理执行完成", "status", status)
	}()

	return types.ToolResult{
		Content: fmt.Sprintf("代理 %s 已在后台启动\nID: %s\n任务: %s",
			run.Name, run.ID, args.Prompt),
		Extra: map[string]any{
			"agent_id": run.ID,
			"status":   "running",
			"async":    true,
		},
	}
}

// GetResult 返回已完成代理的执行结果。
func (t *AgentTool) GetResult(id string) (*AgentResult, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	r, ok := t.results[id]
	return r, ok
}

// GetAgent 返回指定代理的运行信息。
func (t *AgentTool) GetAgent(id string) (*AgentRun, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	a, ok := t.agents[id]
	return a, ok
}

// ListAgents 返回所有已知的代理。
func (t *AgentTool) ListAgents() []*AgentRun {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]*AgentRun, 0, len(t.agents))
	for _, a := range t.agents {
		cp := *a
		result = append(result, &cp)
	}
	return result
}

// CancelAgent 取消正在运行的代理。
func (t *AgentTool) CancelAgent(id string) bool {
	t.mu.RLock()
	a, ok := t.agents[id]
	t.mu.RUnlock()
	if !ok || a.cancel == nil {
		return false
	}
	a.cancel()
	a.Status = "cancelled"
	return true
}

func (t *AgentTool) IsReadOnly(input json.RawMessage) bool      { return false }
func (t *AgentTool) IsConcurrencySafe(input json.RawMessage) bool { return false }

// filterToolsForSubAgent 从子代理的工具列表中排除 agent 工具，防止无限嵌套。
func filterToolsForSubAgent(tools []tool.Tool) []tool.Tool {
	filtered := make([]tool.Tool, 0, len(tools))
	for _, t := range tools {
		if t.Name() != "agent" {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// extractLastAssistantText 从最后一条助手消息中提取文本内容。
func extractLastAssistantText(messages []types.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Type == types.MessageTypeAssistant && msg.Message != nil {
			var blocks []map[string]any
			if err := json.Unmarshal(msg.Message.Content, &blocks); err == nil {
				for _, b := range blocks {
					if b["type"] == "text" {
						if text, ok := b["text"].(string); ok {
							return text
						}
					}
				}
			}
			// 降级：返回原始内容
			return string(msg.Message.Content)
		}
	}
	return ""
}

// buildResult 为失败的代理运行创建错误结果。
func (t *AgentTool) buildResult(run *AgentRun, _ AgentInput, output string, usage api.Usage, err error) types.ToolResult {
	result := &AgentResult{
		ID:        run.ID,
		Name:      run.Name,
		Status:    "failed",
		Output:    output,
		Usage:     usage,
		Error:     err,
		StartedAt: run.StartedAt,
		EndedAt:   time.Now(),
	}

	t.mu.Lock()
	t.results[run.ID] = result
	t.mu.Unlock()

	return types.ToolResult{
		Content: fmt.Sprintf("代理 %s 执行失败: %v", run.Name, err),
		IsError: true,
		Extra: map[string]any{
			"agent_id": run.ID,
			"status":   "failed",
			"error":    err.Error(),
		},
	}
}

// formatAgentOutput 格式化代理执行结果用于展示。
func formatAgentOutput(r *AgentResult) string {
	duration := r.EndedAt.Sub(r.StartedAt).Round(time.Second)
	out := fmt.Sprintf("代理: %s (id: %s)\n", r.Name, r.ID)
	out += fmt.Sprintf("状态: %s\n", r.Status)
	out += fmt.Sprintf("耗时: %s\n", duration)
	out += fmt.Sprintf("Token: %d 输入 / %d 输出\n", r.Usage.InputTokens, r.Usage.OutputTokens)
	out += fmt.Sprintf("费用: %s\n\n", api.FormatCost(r.CostUSD))
	if r.Output != "" {
		out += r.Output
	}
	return out
}
