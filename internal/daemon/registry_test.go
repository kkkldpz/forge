package daemon

import (
	"testing"
)

func TestGlobalRegistry(t *testing.T) {
	r1 := GlobalRegistry()
	r2 := GlobalRegistry()

	if r1 != r2 {
		t.Error("GlobalRegistry should return the same instance")
	}
}

func TestRegistry_Register(t *testing.T) {
	r := &Registry{workers: make(map[string]*Worker)}
	w := &Worker{ID: "worker-1", Name: "test-worker"}

	err := r.Register(w)
	if err != nil {
		t.Errorf("Register failed: %v", err)
	}

	retrieved, ok := r.Get("worker-1")
	if !ok {
		t.Error("Worker not found after register")
	}
	if retrieved.Name != "test-worker" {
		t.Errorf("Expected name 'test-worker', got '%s'", retrieved.Name)
	}
}

func TestRegistry_Register_Duplicate(t *testing.T) {
	r := &Registry{workers: make(map[string]*Worker)}
	r.Register(&Worker{ID: "dup", Name: "first"})

	err := r.Register(&Worker{ID: "dup", Name: "second"})
	if err == nil {
		t.Error("Expected error for duplicate worker ID")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := &Registry{workers: make(map[string]*Worker)}
	r.Register(&Worker{ID: "remove-me"})

	r.Unregister("remove-me")

	_, ok := r.Get("remove-me")
	if ok {
		t.Error("Worker still exists after unregister")
	}
}

func TestRegistry_List(t *testing.T) {
	r := &Registry{workers: make(map[string]*Worker)}
	r.Register(&Worker{ID: "w1", Name: "worker1"})
	r.Register(&Worker{ID: "w2", Name: "worker2"})

	workers := r.List()
	if len(workers) != 2 {
		t.Errorf("Expected 2 workers, got %d", len(workers))
	}
}

func TestRegistry_Heartbeat(t *testing.T) {
	r := &Registry{workers: make(map[string]*Worker)}
	r.Register(&Worker{ID: "hb-worker"})

	err := r.Heartbeat("hb-worker")
	if err != nil {
		t.Errorf("Heartbeat failed: %v", err)
	}

	err = r.Heartbeat("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent worker")
	}
}

func TestRegistry_UpdateSession(t *testing.T) {
	r := &Registry{workers: make(map[string]*Worker)}
	r.Register(&Worker{ID: "session-worker"})

	err := r.UpdateSession("session-worker", "new-session-id")
	if err != nil {
		t.Errorf("UpdateSession failed: %v", err)
	}

	w, _ := r.Get("session-worker")
	if w.Session != "new-session-id" {
		t.Errorf("Expected session 'new-session-id', got '%s'", w.Session)
	}
}