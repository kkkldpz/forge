// Package task 实现异步任务的状态跟踪和管理。
package task

import (
	"sync"
)

// Registry 是全局任务注册表，提供便捷访问。
type Registry struct {
	store Store
}

var (
	globalRegistry     *Registry
	globalRegistryOnce sync.Once
)

// GlobalRegistry 返回全局任务注册表实例。
func GlobalRegistry() *Registry {
	globalRegistryOnce.Do(func() {
		globalRegistry = &Registry{
			store: NewMemoryStore(),
		}
	})
	return globalRegistry
}

// NewRegistry 创建新的任务注册表。
func NewRegistry() *Registry {
	return &Registry{
		store: NewMemoryStore(),
	}
}

// Store 返回底层存储实例。
func (r *Registry) Store() Store {
	return r.store
}

// Create 创建任务到底层存储。
func (r *Registry) Create(task *Task) error {
	return r.store.Create(task)
}

// Update 更新任务。
func (r *Registry) Update(id string, updater func(*Task)) error {
	return r.store.Update(id, updater)
}

// Get 获取任务。
func (r *Registry) Get(id string) (*Task, bool) {
	return r.store.Get(id)
}

// List 列出所有任务。
func (r *Registry) List() []*Task {
	return r.store.List(nil)
}

// ListByStatus 按状态列出任务。
func (r *Registry) ListByStatus(status Status) []*Task {
	return r.store.List(func(t *Task) bool {
		return t.Status == status
	})
}

// Stop 取消任务。
func (r *Registry) Stop(id string) (*Task, error) {
	return r.store.Stop(id)
}
