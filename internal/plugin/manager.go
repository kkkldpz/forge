package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"
)

type Plugin interface {
	Name() string
	Version() string
	Initialize() error
}

type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	paths   []string
}

var (
	globalManager     *Manager
	globalManagerOnce sync.Once
)

func GlobalManager() *Manager {
	globalManagerOnce.Do(func() {
		globalManager = &Manager{
			plugins: make(map[string]Plugin),
		}
	})
	return globalManager
}

func (m *Manager) AddSearchPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paths = append(m.paths, path)
}

func (m *Manager) Load(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("插件 %s 已加载", name)
	}

	for _, path := range m.paths {
		pluginPath := filepath.Join(path, name+".so")
		if _, err := os.Stat(pluginPath); err == nil {
			return m.loadFromFile(name, pluginPath)
		}
	}

	return fmt.Errorf("未找到插件: %s", name)
}

func (m *Manager) loadFromFile(name, path string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("打开插件失败: %w", err)
	}

	symbol, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("查找插件符号失败: %w", err)
	}

	pl, ok := symbol.(Plugin)
	if !ok {
		return fmt.Errorf("插件类型不正确")
	}

	if err := pl.Initialize(); err != nil {
		return fmt.Errorf("初始化插件失败: %w", err)
	}

	m.plugins[name] = pl
	return nil
}

func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.plugins[name]
	return p, ok
}

func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}

func (m *Manager) Unload(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[name]; !exists {
		return fmt.Errorf("插件 %s 未加载", name)
	}

	delete(m.plugins, name)
	return nil
}