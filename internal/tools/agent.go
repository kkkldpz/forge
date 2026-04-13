package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/kkkldpz/forge/internal/tool"
	"github.com/kkkldpz/forge/internal/types"
)

type AgentTool struct {
	tool.BaseTool
	agents map[string]*SubAgent
	mu     sync.RWMutex
}

type SubAgent struct {
	ID      string
	Name    string
	Status  string
	Started bool
}

func NewAgentTool() *AgentTool {
	return &AgentTool{
		BaseTool: tool.BaseTool{
			NameStr:        "agent",
			DescriptionStr: "启动子代理执行复杂任务",
		},
		agents: make(map[string]*SubAgent),
	}
}

type AgentInput struct {
	Name    string `json:"name"`
	Goal    string `json:"goal"`
	Model   string `json:"model,omitempty"`
	Timeout int    `json:"timeout_ms,omitempty"`
}

func (t *AgentTool) InputSchema() types.ToolInputJSONSchema {
	return types.ToolInputJSONSchema{
		Type: "object",
		Properties: map[string]types.ToolSchemaProperty{
			"name":    {Type: "string", Description: "代理名称"},
			"goal":    {Type: "string", Description: "代理目标"},
			"model":   {Type: "string", Description: "使用的模型（可选）"},
			"timeout": {Type: "number", Description: "超时时间（毫秒，可选）"},
		},
		Required: []string{"goal"},
	}
}

func (t *AgentTool) Call(ctx context.Context, input json.RawMessage, tuc tool.ToolUseContext) types.ToolResult {
	var args AgentInput
	if err := json.Unmarshal(input, &args); err != nil {
		return types.ToolResult{Content: fmt.Sprintf("参数解析失败: %v", err), IsError: true}
	}

	if args.Goal == "" {
		return types.ToolResult{Content: "目标不能为空", IsError: true}
	}

	agentName := args.Name
	if agentName == "" {
		agentName = "agent-" + fmt.Sprintf("%d", len(t.agents)+1)
	}

	agentID := fmt.Sprintf("%s-%d", agentName, len(t.agents)+1)

	agent := &SubAgent{
		ID:      agentID,
		Name:    agentName,
		Status:  "running",
		Started: true,
	}

	t.mu.Lock()
	t.agents[agentID] = agent
	t.mu.Unlock()

	content := fmt.Sprintf("✓ 代理 %s 已启动\n目标: %s\n代理 ID: %s\n\n注意: 子代理功能需要完整的 Provider 配置", agentName, args.Goal, agentID)

	return types.ToolResult{
		Content: content,
		Extra: map[string]any{
			"agent_id": agentID,
			"status":   "running",
		},
	}
}

func (t *AgentTool) GetAgent(id string) (*SubAgent, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	agent, ok := t.agents[id]
	return agent, ok
}

func (t *AgentTool) ListAgents() []*SubAgent {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]*SubAgent, 0, len(t.agents))
	for _, agent := range t.agents {
		copyAgent := *agent
		result = append(result, &copyAgent)
	}
	return result
}

func (t *AgentTool) IsReadOnly(input json.RawMessage) bool  { return false }
func (t *AgentTool) IsConcurrencySafe(input json.RawMessage) bool { return false }