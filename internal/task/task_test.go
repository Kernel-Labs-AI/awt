package task

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTaskValidation(t *testing.T) {
	validTask := &Task{
		ID:        "20250110-120000-abc123",
		Agent:     "claude",
		Title:     "Test task",
		Branch:    "awt/claude/20250110-120000-abc123",
		Base:      "main",
		CreatedAt: time.Now(),
		State:     StateActive,
	}

	if err := validTask.Validate(); err != nil {
		t.Errorf("valid task failed validation: %v", err)
	}

	// Test missing fields
	invalidTasks := []*Task{
		{Agent: "claude", Title: "Test", Branch: "test", Base: "main", State: StateActive}, // missing ID
		{ID: "123", Title: "Test", Branch: "test", Base: "main", State: StateActive},       // missing Agent
		{ID: "123", Agent: "claude", Branch: "test", Base: "main", State: StateActive},     // missing Title
		{ID: "123", Agent: "claude", Title: "Test", Base: "main", State: StateActive},      // missing Branch
		{ID: "123", Agent: "claude", Title: "Test", Branch: "test", State: StateActive},    // missing Base
		{ID: "123", Agent: "claude", Title: "Test", Branch: "test", Base: "main"},          // missing State
		{ID: "123", Agent: "claude", Title: "Test", Branch: "test", Base: "main", State: "INVALID"}, // invalid State
	}

	for i, task := range invalidTasks {
		if err := task.Validate(); err == nil {
			t.Errorf("invalid task %d passed validation", i)
		}
	}
}

func TestTaskStore(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "awt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewTaskStore(tempDir)

	// Create a test task
	task := &Task{
		ID:        "20250110-120000-abc123",
		Agent:     "claude",
		Title:     "Test task",
		Branch:    "awt/claude/20250110-120000-abc123",
		Base:      "main",
		CreatedAt: time.Now(),
		State:     StateActive,
	}

	// Test Save
	if err := store.Save(task); err != nil {
		t.Fatalf("failed to save task: %v", err)
	}

	// Verify file exists
	taskPath := filepath.Join(tempDir, "awt", "tasks", task.ID+".json")
	if _, err := os.Stat(taskPath); os.IsNotExist(err) {
		t.Fatalf("task file was not created")
	}

	// Test Load
	loadedTask, err := store.Load(task.ID)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}

	// Verify loaded task matches
	if loadedTask.ID != task.ID {
		t.Errorf("loaded task ID mismatch: got %s, want %s", loadedTask.ID, task.ID)
	}
	if loadedTask.Agent != task.Agent {
		t.Errorf("loaded task Agent mismatch: got %s, want %s", loadedTask.Agent, task.Agent)
	}
	if loadedTask.Title != task.Title {
		t.Errorf("loaded task Title mismatch: got %s, want %s", loadedTask.Title, task.Title)
	}

	// Test List
	tasks, err := store.List()
	if err != nil {
		t.Fatalf("failed to list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// Test Delete
	if err := store.Delete(task.ID); err != nil {
		t.Fatalf("failed to delete task: %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(taskPath); !os.IsNotExist(err) {
		t.Fatalf("task file was not deleted")
	}

	// Test Load non-existent task
	_, err = store.Load("nonexistent")
	if err == nil {
		t.Error("expected error loading non-existent task")
	}
}

func TestAtomicWrite(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "awt-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store := NewTaskStore(tempDir)

	task := &Task{
		ID:        "20250110-120000-abc123",
		Agent:     "claude",
		Title:     "Test task",
		Branch:    "awt/claude/20250110-120000-abc123",
		Base:      "main",
		CreatedAt: time.Now(),
		State:     StateActive,
	}

	// Save multiple times to test atomicity
	for i := 0; i < 10; i++ {
		task.Title = fmt.Sprintf("Test task %d", i)
		if err := store.Save(task); err != nil {
			t.Fatalf("failed to save task on iteration %d: %v", i, err)
		}
	}

	// Verify we can still load the task
	loadedTask, err := store.Load(task.ID)
	if err != nil {
		t.Fatalf("failed to load task after multiple saves: %v", err)
	}
	if loadedTask.Title != "Test task 9" {
		t.Errorf("expected title 'Test task 9', got '%s'", loadedTask.Title)
	}

	// Verify no temp files left behind
	tasksDir := filepath.Join(tempDir, "awt", "tasks")
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		t.Fatalf("failed to read tasks dir: %v", err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Errorf("temp file left behind: %s", entry.Name())
		}
	}
}
