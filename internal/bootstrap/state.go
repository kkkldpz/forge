// Package bootstrap 管理会话全局状态，包括 sessionID、工作目录和费用追踪。
package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kkkldpz/forge/internal/types"
)

// contextKey 用于在 context.Context 中存储 SessionState。
type contextKey struct{}

// ModelUsage 跟踪单个模型的 token 用量。
type ModelUsage struct {
	InputTokens              int64   `json:"inputTokens"`
	OutputTokens             int64   `json:"outputTokens"`
	CacheReadInputTokens     int64   `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int64   `json:"cacheCreationInputTokens"`
	CostUSD                  float64 `json:"costUSD"`
}

// SessionState 会话全局可变状态。
type SessionState struct {
	mu sync.RWMutex

	// 身份标识
	SessionID       types.SessionID `json:"sessionId"`
	ParentSessionID types.SessionID `json:"parentSessionId,omitempty"`

	// 目录信息
	OriginalCWD string `json:"originalCwd"`
	ProjectRoot string `json:"projectRoot"`
	CWD         string `json:"cwd"`

	// 时间记录
	StartTime           time.Time `json:"startTime"`
	LastInteractionTime time.Time `json:"lastInteractionTime"`

	// 费用追踪
	TotalCostUSD float64 `json:"totalCostUSD"`

	// Token 追踪
	TotalInputTokens  int64 `json:"totalInputTokens"`
	TotalOutputTokens int64 `json:"totalOutputTokens"`

	// 各模型的用量统计
	ModelUsage map[string]*ModelUsage `json:"modelUsage"`

	// 模型设置
	MainLoopModelOverride string `json:"mainLoopModelOverride,omitempty"`
	InitialMainLoopModel  string `json:"initialMainLoopModel"`

	// 会话标志
	IsInteractive            bool `json:"isInteractive"`
	SessionBypassPermissions bool `json:"sessionBypassPermissions"`
	HasExitedPlanMode        bool `json:"hasExitedPlanMode"`
	IsRemoteMode             bool `json:"isRemoteMode"`

	// 代码行数变更统计
	TotalLinesAdded   int `json:"totalLinesAdded"`
	TotalLinesRemoved int `json:"totalLinesRemoved"`
}

// NewSessionState 为指定工作目录创建新的会话状态。
func NewSessionState(cwd string) *SessionState {
	projectRoot := findProjectRoot(cwd)
	return &SessionState{
		SessionID:        types.NewSessionID(uuid.New().String()),
		OriginalCWD:      cwd,
		ProjectRoot:      projectRoot,
		CWD:              cwd,
		StartTime:        time.Now(),
		LastInteractionTime: time.Now(),
		ModelUsage:       make(map[string]*ModelUsage),
		IsInteractive:    true,
	}
}

// AddCost 累加费用。
func (s *SessionState) AddCost(cost float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalCostUSD += cost
}

// AddModelUsage 记录指定模型的 token 用量。
func (s *SessionState) AddModelUsage(model string, usage *ModelUsage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.ModelUsage[model]; ok {
		existing.InputTokens += usage.InputTokens
		existing.OutputTokens += usage.OutputTokens
		existing.CacheReadInputTokens += usage.CacheReadInputTokens
		existing.CacheCreationInputTokens += usage.CacheCreationInputTokens
		existing.CostUSD += usage.CostUSD
	} else {
		s.ModelUsage[model] = &ModelUsage{
			InputTokens:              usage.InputTokens,
			OutputTokens:             usage.OutputTokens,
			CacheReadInputTokens:     usage.CacheReadInputTokens,
			CacheCreationInputTokens: usage.CacheCreationInputTokens,
			CostUSD:                  usage.CostUSD,
		}
	}
	s.TotalInputTokens += usage.InputTokens
	s.TotalOutputTokens += usage.OutputTokens
}

// Touch 更新最后交互时间。
func (s *SessionState) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastInteractionTime = time.Now()
}

// SnapshotData 是 SessionState 的只读快照（不含互斥锁）。
type SnapshotData struct {
	SessionID         types.SessionID         `json:"sessionId"`
	OriginalCWD       string                  `json:"originalCwd"`
	ProjectRoot       string                  `json:"projectRoot"`
	CWD               string                  `json:"cwd"`
	StartTime         time.Time               `json:"startTime"`
	LastInteraction   time.Time               `json:"lastInteractionTime"`
	TotalCostUSD      float64                 `json:"totalCostUSD"`
	TotalInputTokens  int64                   `json:"totalInputTokens"`
	TotalOutputTokens int64                   `json:"totalOutputTokens"`
	ModelUsage        map[string]*ModelUsage  `json:"modelUsage"`
	IsInteractive     bool                    `json:"isInteractive"`
}

// Snapshot 返回状态的只读快照。
func (s *SessionState) Snapshot() SnapshotData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mu := make(map[string]*ModelUsage, len(s.ModelUsage))
	for k, v := range s.ModelUsage {
		vc := *v
		mu[k] = &vc
	}
	return SnapshotData{
		SessionID:         s.SessionID,
		OriginalCWD:       s.OriginalCWD,
		ProjectRoot:       s.ProjectRoot,
		CWD:               s.CWD,
		StartTime:         s.StartTime,
		LastInteraction:   s.LastInteractionTime,
		TotalCostUSD:      s.TotalCostUSD,
		TotalInputTokens:  s.TotalInputTokens,
		TotalOutputTokens: s.TotalOutputTokens,
		ModelUsage:        mu,
		IsInteractive:     s.IsInteractive,
	}
}

// WithContext 将 SessionState 存入 context.Context。
func WithContext(ctx context.Context, state *SessionState) context.Context {
	return context.WithValue(ctx, contextKey{}, state)
}

// FromContext 从 context.Context 中取出 SessionState。
func FromContext(ctx context.Context) *SessionState {
	s, _ := ctx.Value(contextKey{}).(*SessionState)
	return s
}

// findProjectRoot 向上遍历目录树，查找包含 .git 或 .forge 的项目根目录。
func findProjectRoot(cwd string) string {
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, ".forge")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}
