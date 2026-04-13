// Package cron 实现定时任务的存储和调度。
package cron

import (
	"fmt"
	"sync"
	"time"
)

type Job struct {
	ID          string
	Command     string
	Schedule    string
	Description string
	Enabled     bool
	CreatedAt   time.Time
	LastRun     *time.Time
	NextRun     *time.Time
}

func (j *Job) Run() {
	now := time.Now()
	j.LastRun = &now
}

type Store interface {
	Create(job *Job) error
	Get(id string) (*Job, bool)
	List() []*Job
	Delete(id string) error
	Update(id string, updater func(*Job)) error
}

type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{jobs: make(map[string]*Job)}
}

func (s *MemoryStore) Create(job *Job) error {
	if job.ID == "" {
		return fmt.Errorf("任务 ID 不能为空")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.jobs[job.ID]; exists {
		return fmt.Errorf("任务 %s 已存在", job.ID)
	}
	job.CreatedAt = time.Now()
	s.jobs[job.ID] = job
	return nil
}

func (s *MemoryStore) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}

func (s *MemoryStore) List() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		copyJob := *job
		result = append(result, &copyJob)
	}
	return result
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[id]; !ok {
		return fmt.Errorf("任务 %s 不存在", id)
	}
	delete(s.jobs, id)
	return nil
}

func (s *MemoryStore) Update(id string, updater func(*Job)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return fmt.Errorf("任务 %s 不存在", id)
	}
	updater(job)
	return nil
}

var (
	globalStore     *MemoryStore
	globalStoreOnce sync.Once
)

func GlobalStore() *MemoryStore {
	globalStoreOnce.Do(func() {
		globalStore = NewMemoryStore()
	})
	return globalStore
}