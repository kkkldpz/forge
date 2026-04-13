// Package task 实现异步任务的状态跟踪和管理。
package task

import (
	"fmt"
	"sync"
	"time"
)

// Store 是任务的内存存储接口。
type Store interface {
	Create(task *Task) error
	Update(id string, updater func(*Task)) error
	Get(id string) (*Task, bool)
	List(filter func(*Task) bool) []*Task
	Stop(id string) (*Task, error)
}

// MemoryStore 是内存中的任务存储实现。
type MemoryStore struct {
	mu     sync.RWMutex
	tasks  map[string]*Task
}

// NewMemoryStore 创建新的内存任务存储。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks: make(map[string]*Task),
	}
}

// Create 创建新任务。
func (s *MemoryStore) Create(task *Task) error {
	if task.ID == "" {
		return fmt.Errorf("任务 ID 不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	task.CreatedAt = time.Now()
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = task.CreatedAt
	}
	s.tasks[task.ID] = task
	return nil
}

// Update 更新任务。
func (s *MemoryStore) Update(id string, updater func(*Task)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("任务 %s 不存在", id)
	}

	updater(task)
	task.UpdatedAt = time.Now()
	return nil
}

// Get 获取任务。
func (s *MemoryStore) Get(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]
	return task, ok
}

// List 列出任务，支持过滤。
func (s *MemoryStore) List(filter func(*Task) bool) []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Task
	for _, task := range s.tasks {
		if filter == nil || filter(task) {
			// 返回副本避免外部修改
			copyTask := *task
			result = append(result, &copyTask)
		}
	}

	// 按创建时间降序
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// Stop 取消任务。
func (s *MemoryStore) Stop(id string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("任务 %s 不存在", id)
	}

	if task.IsTerminal() {
		return nil, fmt.Errorf("任务 %s 已结束，无法取消", id)
	}

	task.MarkCancelled()
	copyTask := *task
	return &copyTask, nil
}
