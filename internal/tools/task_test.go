package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kkkldpz/forge/internal/task"
	"github.com/kkkldpz/forge/internal/tool"
)

func TestTaskGetTool(t *testing.T) {
	tk := &task.Task{
		ID:     "test-task-123",
		Title:  "测试任务",
		Status: task.StatusPending,
	}
	task.GlobalRegistry().Create(tk)

	tkTool := NewTaskGetTool()
	if tkTool.Name() != "task_get" {
		t.Errorf("Expected name 'task_get', got '%s'", tkTool.Name())
	}

	input := `{"id":"test-task-123"}`
	result := tkTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{WorkingDir: "/tmp"})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}

	if result.Content == "" {
		t.Error("Expected content, got empty string")
	}
}

func TestTaskListTool(t *testing.T) {
	tkTool := NewTaskListTool()
	if tkTool.Name() != "task_list" {
		t.Errorf("Expected name 'task_list', got '%s'", tkTool.Name())
	}

	input := `{}`
	result := tkTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{WorkingDir: "/tmp"})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestTaskStopTool(t *testing.T) {
	tk := &task.Task{
		ID:     "stop-task-456",
		Title:  "停止测试",
		Status: task.StatusRunning,
	}
	task.GlobalRegistry().Create(tk)

	tkTool := NewTaskStopTool()
	if tkTool.Name() != "task_stop" {
		t.Errorf("Expected name 'task_stop', got '%s'", tkTool.Name())
	}

	input := `{"id":"stop-task-456"}`
	result := tkTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{WorkingDir: "/tmp"})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestTaskGetTool_NotFound(t *testing.T) {
	tkTool := NewTaskGetTool()
	input := `{"id":"non-existent-id"}`
	result := tkTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{WorkingDir: "/tmp"})

	if !result.IsError {
		t.Error("Expected error for non-existent task")
	}
}

func TestTaskListTool_FilterByStatus(t *testing.T) {
	tk := &task.Task{
		ID:     "pending-task-789",
		Title:  "待处理任务",
		Status: task.StatusPending,
	}
	task.GlobalRegistry().Create(tk)

	tkTool := NewTaskListTool()
	input := `{"status":"pending"}`
	result := tkTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{WorkingDir: "/tmp"})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestTaskCreateTool(t *testing.T) {
	tkTool := NewTaskCreateTool()
	if tkTool.Name() != "task_create" {
		t.Errorf("Expected name 'task_create', got '%s'", tkTool.Name())
	}

	input := `{"title":"新任务"}`
	result := tkTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{WorkingDir: "/tmp"})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}

func TestTaskUpdateTool(t *testing.T) {
	tk := &task.Task{
		ID:     "update-task-001",
		Title:  "原标题",
		Status: task.StatusPending,
	}
	task.GlobalRegistry().Create(tk)

	tkTool := NewTaskUpdateTool()
	input := `{"id":"update-task-001","title":"新标题","status":"completed"}`
	result := tkTool.Call(context.Background(), json.RawMessage(input), tool.ToolUseContext{WorkingDir: "/tmp"})

	if result.IsError {
		t.Errorf("Expected no error, got: %s", result.Content)
	}
}
