// Package task 实现异步任务的状态跟踪和管理。
package task

import (
	"encoding/json"
	"time"
)

// Status 表示任务的执行状态。
type Status string

const (
	StatusPending    Status = "pending"
	StatusRunning    Status = "running"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

// Task 表示一个可跟踪的异步任务。
type Task struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Status      Status          `json:"status"`
	Result      string          `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	CompletedAt *time.Time      `json:"completedAt,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// IsTerminal 判断任务是否已处于终止状态。
func (t *Task) IsTerminal() bool {
	return t.Status == StatusCompleted || t.Status == StatusFailed || t.Status == StatusCancelled
}

// MarkRunning 将任务标记为运行中。
func (t *Task) MarkRunning() {
	t.Status = StatusRunning
	t.UpdatedAt = time.Now()
}

// MarkCompleted 将任务标记为已完成。
func (t *Task) MarkCompleted(result string) {
	t.Status = StatusCompleted
	t.Result = result
	t.UpdatedAt = time.Now()
	now := time.Now()
	t.CompletedAt = &now
}

// MarkFailed 将任务标记为失败。
func (t *Task) MarkFailed(err string) {
	t.Status = StatusFailed
	t.Error = err
	t.UpdatedAt = time.Now()
	now := time.Now()
	t.CompletedAt = &now
}

// MarkCancelled 将任务标记为已取消。
func (t *Task) MarkCancelled() {
	t.Status = StatusCancelled
	t.UpdatedAt = time.Now()
	now := time.Now()
	t.CompletedAt = &now
}

// Update 更新任务字段并刷新更新时间。
func (t *Task) Update(title, description, status, result string) {
	if title != "" {
		t.Title = title
	}
	if description != "" {
		t.Description = description
	}
	if status != "" {
		t.Status = Status(status)
	}
	if result != "" {
		t.Result = result
	}
	t.UpdatedAt = time.Now()
}
