package plugin

import (
	"testing"
)

func TestGlobalManager(t *testing.T) {
	m1 := GlobalManager()
	m2 := GlobalManager()

	if m1 != m2 {
		t.Error("GlobalManager should return the same instance")
	}
}

func TestManager_AddSearchPath(t *testing.T) {
	m := &Manager{
		plugins: make(map[string]Plugin),
	}

	m.AddSearchPath("/path1")
	m.AddSearchPath("/path2")

	if len(m.paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(m.paths))
	}
}

func TestManager_List(t *testing.T) {
	m := &Manager{
		plugins: map[string]Plugin{
			"plugin1": nil,
			"plugin2": nil,
		},
	}

	names := m.List()
	if len(names) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(names))
	}
}

func TestManager_List_Empty(t *testing.T) {
	m := &Manager{
		plugins: make(map[string]Plugin),
	}

	names := m.List()
	if len(names) != 0 {
		t.Errorf("Expected 0 plugins, got %d", len(names))
	}
}

func TestManager_Get(t *testing.T) {
	m := &Manager{
		plugins: make(map[string]Plugin),
	}

	m.plugins["test-plugin"] = nil

	p, ok := m.Get("test-plugin")
	if !ok {
		t.Error("Expected to find plugin")
	}

	_, ok = m.Get("non-existent")
	if ok {
		t.Error("Expected not to find non-existent plugin")
	}

	_ = p
}

func TestManager_Load_NotFound(t *testing.T) {
	m := &Manager{
		plugins: make(map[string]Plugin),
		paths:   []string{"/nonexistent"},
	}

	err := m.Load("nonexistent-plugin")
	if err == nil {
		t.Error("Expected error for non-existent plugin")
	}
}

func TestManager_Unload_NotLoaded(t *testing.T) {
	m := &Manager{
		plugins: make(map[string]Plugin),
	}

	err := m.Unload("non-existent")
	if err == nil {
		t.Error("Expected error for unloading non-loaded plugin")
	}
}