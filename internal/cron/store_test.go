package cron

import (
	"testing"
)

func TestMemoryStore_Create(t *testing.T) {
	store := NewMemoryStore()
	job := &Job{
		ID:       "test-1",
		Command:  "echo hello",
		Schedule: "* * * * *",
	}

	err := store.Create(job)
	if err != nil {
		t.Errorf("Create failed: %v", err)
	}

	retrieved, ok := store.Get("test-1")
	if !ok {
		t.Error("Job not found after create")
	}
	if retrieved.Command != "echo hello" {
		t.Errorf("Expected command 'echo hello', got '%s'", retrieved.Command)
	}
}

func TestMemoryStore_Create_Duplicate(t *testing.T) {
	store := NewMemoryStore()
	job := &Job{ID: "dup-1", Command: "test"}

	store.Create(job)
	err := store.Create(job)

	if err == nil {
		t.Error("Expected error for duplicate ID")
	}
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	store := NewMemoryStore()
	_, ok := store.Get("non-existent")

	if ok {
		t.Error("Expected not found for non-existent ID")
	}
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore()
	store.Create(&Job{ID: "job-1", Command: "cmd1"})
	store.Create(&Job{ID: "job-2", Command: "cmd2"})

	jobs := store.List()
	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	store.Create(&Job{ID: "del-1", Command: "test"})

	err := store.Delete("del-1")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	_, ok := store.Get("del-1")
	if ok {
		t.Error("Job still exists after delete")
	}
}

func TestMemoryStore_Delete_NotFound(t *testing.T) {
	store := NewMemoryStore()
	err := store.Delete("non-existent")

	if err == nil {
		t.Error("Expected error for deleting non-existent job")
	}
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()
	store.Create(&Job{ID: "upd-1", Command: "original"})

	err := store.Update("upd-1", func(j *Job) {
		j.Command = "updated"
	})
	if err != nil {
		t.Errorf("Update failed: %v", err)
	}

	job, _ := store.Get("upd-1")
	if job.Command != "updated" {
		t.Errorf("Expected command 'updated', got '%s'", job.Command)
	}
}