package session

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type SessionInfo struct {
	ID        string
	PID       int
	Status    string
	StartedAt time.Time
	Cwd       string
}

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*SessionInfo
}

var (
	globalManager     *Manager
	globalManagerOnce sync.Once
)

func GlobalManager() *Manager {
	globalManagerOnce.Do(func() {
		globalManager = &Manager{
			sessions: make(map[string]*SessionInfo),
		}
	})
	return globalManager
}

func (m *Manager) Add(s *SessionInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[s.ID]; exists {
		return fmt.Errorf("会话 %s 已存在", s.ID)
	}

	s.StartedAt = time.Now()
	m.sessions[s.ID] = s
	return nil
}

func (m *Manager) Get(id string) (*SessionInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

func (m *Manager) List() []*SessionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*SessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
		copyS := *s
		result = append(result, &copyS)
	}
	return result
}

func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[id]; !exists {
		return fmt.Errorf("会话 %s 不存在", id)
	}

	delete(m.sessions, id)
	return nil
}

func (m *Manager) UpdateStatus(id, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("会话 %s 不存在", id)
	}

	s.Status = status
	return nil
}

func (s *SessionInfo) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}