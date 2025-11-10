package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State represents the task state in the state machine
type State string

const (
	// StateNew is the initial state when metadata is reserved
	StateNew State = "NEW"
	// StateActive means the worktree exists and owns the branch
	StateActive State = "ACTIVE"
	// StateHandoffReady means changes are committed/pushed and branch is detached
	StateHandoffReady State = "HANDOFF_READY"
	// StateMerged means the task was integrated into the base branch
	StateMerged State = "MERGED"
	// StateAbandoned means the task was closed without merge
	StateAbandoned State = "ABANDONED"
)

// Task represents a single agent task
type Task struct {
	// ID is the unique task identifier (YYYYmmdd-HHMMSS-<6random>)
	ID string `json:"id"`

	// Agent is the name of the agent working on this task
	Agent string `json:"agent"`

	// Title is a human-readable description of the task
	Title string `json:"title"`

	// Branch is the git branch name for this task (awt/<agent>/<id>)
	Branch string `json:"branch"`

	// Base is the base branch this task branches from
	Base string `json:"base"`

	// CreatedAt is when the task was created
	CreatedAt time.Time `json:"created_at"`

	// State is the current state of the task
	State State `json:"state"`

	// WorktreePath is the path to the worktree (empty if not checked out)
	WorktreePath string `json:"worktree_path"`

	// LastCommit is the SHA of the last commit (optional)
	LastCommit string `json:"last_commit,omitempty"`

	// PRURL is the URL of the pull/merge request (optional)
	PRURL string `json:"pr_url,omitempty"`
}

// TaskStore handles persistence of task metadata
type TaskStore struct {
	// tasksDir is the directory where task JSON files are stored
	tasksDir string
}

// NewTaskStore creates a new task store
func NewTaskStore(gitCommonDir string) *TaskStore {
	return &TaskStore{
		tasksDir: filepath.Join(gitCommonDir, "awt", "tasks"),
	}
}

// Save saves the task to disk atomically
func (ts *TaskStore) Save(task *Task) error {
	// Ensure tasks directory exists
	if err := os.MkdirAll(ts.tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Marshal task to JSON
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// Write atomically: write to temp file, then rename
	taskPath := ts.taskPath(task.ID)
	tempPath := taskPath + ".tmp"

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Rename to final location (atomic on POSIX systems)
	if err := os.Rename(tempPath, taskPath); err != nil {
		// Clean up temp file on error
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Load loads a task from disk
func (ts *TaskStore) Load(taskID string) (*Task, error) {
	taskPath := ts.taskPath(taskID)

	data, err := os.ReadFile(taskPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		return nil, fmt.Errorf("failed to read task file: %w", err)
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task (corrupted JSON?): %w", err)
	}

	// Validate task
	if err := task.Validate(); err != nil {
		return nil, fmt.Errorf("task validation failed: %w", err)
	}

	return &task, nil
}

// List returns all tasks
func (ts *TaskStore) List() ([]*Task, error) {
	entries, err := os.ReadDir(ts.tasksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Task{}, nil
		}
		return nil, fmt.Errorf("failed to read tasks directory: %w", err)
	}

	var tasks []*Task
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		taskID := entry.Name()[:len(entry.Name())-5] // Remove .json extension
		task, err := ts.Load(taskID)
		if err != nil {
			// Log error but continue with other tasks
			continue
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// Delete removes a task from disk
func (ts *TaskStore) Delete(taskID string) error {
	taskPath := ts.taskPath(taskID)
	if err := os.Remove(taskPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

// taskPath returns the file path for a task
func (ts *TaskStore) taskPath(taskID string) string {
	return filepath.Join(ts.tasksDir, taskID+".json")
}

// Validate validates the task fields
func (t *Task) Validate() error {
	if t.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if t.Agent == "" {
		return fmt.Errorf("agent is required")
	}
	if t.Title == "" {
		return fmt.Errorf("title is required")
	}
	if t.Branch == "" {
		return fmt.Errorf("branch is required")
	}
	if t.Base == "" {
		return fmt.Errorf("base is required")
	}
	if t.State == "" {
		return fmt.Errorf("state is required")
	}

	// Validate state is one of the valid states
	switch t.State {
	case StateNew, StateActive, StateHandoffReady, StateMerged, StateAbandoned:
		// Valid state
	default:
		return fmt.Errorf("invalid state: %s", t.State)
	}

	return nil
}
