package bridge

import (
	"sync"
	"time"
)

// SessionManager 跟踪活跃的桥接会话。
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*BridgeSession
}

// BridgeSession 表示一个活跃的桥接会话。
type BridgeSession struct {
	ID           string
	PeerID       string
	StartedAt    time.Time
	LastActivity time.Time
	Metadata     map[string]string
}

// NewSessionManager 创建新的会话管理器。
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*BridgeSession),
	}
}

// Create 创建新的会话。
func (sm *SessionManager) Create(id, peerID string) *BridgeSession {
	now := time.Now()
	s := &BridgeSession{
		ID:           id,
		PeerID:       peerID,
		StartedAt:    now,
		LastActivity: now,
		Metadata:     make(map[string]string),
	}
	sm.mu.Lock()
	sm.sessions[id] = s
	sm.mu.Unlock()
	return s
}

// Get 按 ID 查找会话。
func (sm *SessionManager) Get(id string) (*BridgeSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[id]
	return s, ok
}

// Touch 更新会话的最后活跃时间。
func (sm *SessionManager) Touch(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if s, ok := sm.sessions[id]; ok {
		s.LastActivity = time.Now()
	}
}

// Remove 移除会话。
func (sm *SessionManager) Remove(id string) {
	sm.mu.Lock()
	delete(sm.sessions, id)
	sm.mu.Unlock()
}

// List 返回所有会话。
func (sm *SessionManager) List() []*BridgeSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	list := make([]*BridgeSession, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		cp := *s
		list = append(list, &cp)
	}
	return list
}

// ExpireStale 清除超过指定时间未活跃的会话。
func (sm *SessionManager) ExpireStale(maxAge time.Duration) int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	expired := 0
	for id, s := range sm.sessions {
		if now.Sub(s.LastActivity) > maxAge {
			delete(sm.sessions, id)
			expired++
		}
	}
	return expired
}
