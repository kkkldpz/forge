package daemon

import (
	"fmt"
	"sync"
	"time"
)

type Worker struct {
	ID       string
	Name     string
	Status   string
	Session  string
	Started  time.Time
	LastBeat time.Time
}

type Registry struct {
	mu      sync.RWMutex
	workers map[string]*Worker
}

var (
	globalRegistry     *Registry
	globalRegistryOnce sync.Once
)

func GlobalRegistry() *Registry {
	globalRegistryOnce.Do(func() {
		globalRegistry = &Registry{
			workers: make(map[string]*Worker),
		}
	})
	return globalRegistry
}

func (r *Registry) Register(worker *Worker) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.workers[worker.ID]; exists {
		return fmt.Errorf("worker %s 已存在", worker.ID)
	}

	worker.Started = time.Now()
	worker.LastBeat = time.Now()
	r.workers[worker.ID] = worker
	return nil
}

func (r *Registry) Unregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workers, id)
}

func (r *Registry) Get(id string) (*Worker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	w, ok := r.workers[id]
	return w, ok
}

func (r *Registry) List() []*Worker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Worker, 0, len(r.workers))
	for _, w := range r.workers {
		copyW := *w
		result = append(result, &copyW)
	}
	return result
}

func (r *Registry) Heartbeat(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, ok := r.workers[id]
	if !ok {
		return fmt.Errorf("worker %s 不存在", id)
	}

	w.LastBeat = time.Now()
	return nil
}

func (r *Registry) UpdateSession(id, session string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	w, ok := r.workers[id]
	if !ok {
		return fmt.Errorf("worker %s 不存在", id)
	}

	w.Session = session
	return nil
}