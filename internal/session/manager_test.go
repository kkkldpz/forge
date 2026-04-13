package session

import (
	"testing"
	"time"
)

func TestGlobalManager(t *testing.T) {
	m1 := GlobalManager()
	m2 := GlobalManager()

	if m1 != m2 {
		t.Error("GlobalManager should return the same instance")
	}
}

func TestManager_Add(t *testing.T) {
	m := &Manager{sessions: make(map[string]*SessionInfo)}
	s := &SessionInfo{
		ID:     "test-session-1",
		PID:    12345,
		Status: "running",
		Cwd:    "/tmp",
	}

	err := m.Add(s)
	if err != nil {
		t.Errorf("Add failed: %v", err)
	}

	retrieved, ok := m.Get("test-session-1")
	if !ok {
		t.Error("Session not found after Add")
	}
	if retrieved.PID != 12345 {
		t.Errorf("Expected PID 12345, got %d", retrieved.PID)
	}
}

func TestManager_Add_Duplicate(t *testing.T) {
	m := &Manager{sessions: make(map[string]*SessionInfo)}
	s := &SessionInfo{ID: "dup-session", PID: 100}

	m.Add(s)
	err := m.Add(s)

	if err == nil {
		t.Error("Expected error for duplicate session")
	}
}

func TestManager_Get_NotFound(t *testing.T) {
	m := &Manager{sessions: make(map[string]*SessionInfo)}
	_, ok := m.Get("non-existent")

	if ok {
		t.Error("Expected not found for non-existent session")
	}
}

func TestManager_List(t *testing.T) {
	m := &Manager{sessions: make(map[string]*SessionInfo)}
	m.Add(&SessionInfo{ID: "s1", PID: 1})
	m.Add(&SessionInfo{ID: "s2", PID: 2})

	sessions := m.List()
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}
}

func TestManager_Remove(t *testing.T) {
	m := &Manager{sessions: make(map[string]*SessionInfo)}
	m.Add(&SessionInfo{ID: "remove-me", PID: 1})

	err := m.Remove("remove-me")
	if err != nil {
		t.Errorf("Remove failed: %v", err)
	}

	_, ok := m.Get("remove-me")
	if ok {
		t.Error("Session still exists after Remove")
	}
}

func TestManager_Remove_NotFound(t *testing.T) {
	m := &Manager{sessions: make(map[string]*SessionInfo)}
	err := m.Remove("non-existent")

	if err == nil {
		t.Error("Expected error for removing non-existent session")
	}
}

func TestManager_UpdateStatus(t *testing.T) {
	m := &Manager{sessions: make(map[string]*SessionInfo)}
	m.Add(&SessionInfo{ID: "update-me", PID: 1, Status: "running"})

	err := m.UpdateStatus("update-me", "stopped")
	if err != nil {
		t.Errorf("UpdateStatus failed: %v", err)
	}

	s, _ := m.Get("update-me")
	if s.Status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", s.Status)
	}
}

func TestSessionInfo_ToJSON(t *testing.T) {
	s := &SessionInfo{
		ID:        "json-test",
		PID:       12345,
		Status:    "running",
		StartedAt: time.Now(),
		Cwd:       "/tmp",
	}

	data, err := s.ToJSON()
	if err != nil {
		t.Errorf("ToJSON failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty JSON")
	}
}