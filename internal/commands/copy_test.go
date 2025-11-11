package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kernel-labs-ai/awt/internal/task"
)

func TestRunTaskCopy(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Start a task first
	startOpts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "test-copy-task",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	if err := runTaskStart(startOpts); err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	// Create a test file to copy (simulating a .env file)
	testFile := filepath.Join(repoPath, ".env")
	testContent := "TEST_VAR=test_value\nANOTHER_VAR=another_value\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a nested file to test directory creation
	nestedDir := filepath.Join(repoPath, "config")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}
	nestedFile := filepath.Join(nestedDir, "local.json")
	nestedContent := `{"test": "value"}`
	if err := os.WriteFile(nestedFile, []byte(nestedContent), 0644); err != nil {
		t.Fatalf("failed to create nested test file: %v", err)
	}

	// Copy files to the task worktree
	copyOpts := &CopyOptions{
		RepoPath: repoPath,
		TaskID:   "test-copy-task",
		Files:    []string{".env", "config/local.json"},
	}

	if err := runTaskCopy(copyOpts); err != nil {
		t.Fatalf("runTaskCopy() failed: %v", err)
	}

	// Verify files were copied
	store := task.NewTaskStore(filepath.Join(repoPath, ".git"))
	tsk, err := store.Load("test-copy-task")
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}

	// Check .env file
	copiedEnv := filepath.Join(tsk.WorktreePath, ".env")
	copiedContent, err := os.ReadFile(copiedEnv)
	if err != nil {
		t.Fatalf("copied .env file not found: %v", err)
	}
	if string(copiedContent) != testContent {
		t.Errorf("copied .env content = %q, want %q", string(copiedContent), testContent)
	}

	// Check nested file
	copiedNested := filepath.Join(tsk.WorktreePath, "config", "local.json")
	copiedNestedContent, err := os.ReadFile(copiedNested)
	if err != nil {
		t.Fatalf("copied nested file not found: %v", err)
	}
	if string(copiedNestedContent) != nestedContent {
		t.Errorf("copied nested content = %q, want %q", string(copiedNestedContent), nestedContent)
	}
}

func TestRunTaskCopyJSON(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Start a task first
	startOpts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "test-json-task",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	if err := runTaskStart(startOpts); err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(repoPath, ".env")
	if err := os.WriteFile(testFile, []byte("TEST=value\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Copy with JSON output
	copyOpts := &CopyOptions{
		RepoPath:   repoPath,
		TaskID:     "test-json-task",
		Files:      []string{".env"},
		OutputJSON: true,
	}

	// Capture output by running the function
	// (In real test we'd capture stdout, but for simplicity we just verify no error)
	if err := runTaskCopy(copyOpts); err != nil {
		t.Fatalf("runTaskCopy() with JSON failed: %v", err)
	}
}

func TestRunTaskCopyNonExistentFile(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Start a task first
	startOpts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "test-error-task",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	if err := runTaskStart(startOpts); err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	// Try to copy a non-existent file
	copyOpts := &CopyOptions{
		RepoPath: repoPath,
		TaskID:   "test-error-task",
		Files:    []string{"non-existent-file.txt"},
	}

	err := runTaskCopy(copyOpts)
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestRunTaskCopyInvalidTaskID(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	copyOpts := &CopyOptions{
		RepoPath: repoPath,
		TaskID:   "non-existent-task",
		Files:    []string{".env"},
	}

	err := runTaskCopy(copyOpts)
	if err == nil {
		t.Error("expected error for invalid task ID, got nil")
	}
}

func TestCopyResultJSON(t *testing.T) {
	result := CopyResult{
		TaskID:       "test-task",
		FilesCopied:  []string{".env", "config/local.json"},
		WorktreePath: "/path/to/worktree",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal CopyResult: %v", err)
	}

	var decoded CopyResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal CopyResult: %v", err)
	}

	if decoded.TaskID != result.TaskID {
		t.Errorf("TaskID mismatch: got %q, want %q", decoded.TaskID, result.TaskID)
	}

	if len(decoded.FilesCopied) != len(result.FilesCopied) {
		t.Errorf("FilesCopied length mismatch: got %d, want %d", len(decoded.FilesCopied), len(result.FilesCopied))
	}
}
