package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kernel-labs-ai/awt/internal/task"
)

func TestRunTaskEditorInvalidTaskID(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	opts := &EditorOptions{
		RepoPath: repoPath,
		TaskID:   "non-existent-task",
		Editor:   "echo", // Use echo as a safe "editor" for testing
	}

	err := runTaskEditor(opts)
	if err == nil {
		t.Error("expected error for invalid task ID, got nil")
	}
}

func TestRunTaskEditorWorktreeNotFound(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Start a task first
	startOpts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "test-editor-task",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	if err := runTaskStart(startOpts); err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	// Load task to get the actual worktree path
	store := task.NewTaskStore(filepath.Join(repoPath, ".git"))
	tsk, err := store.Load("test-editor-task")
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}

	// Remove the worktree directory to simulate missing worktree
	if err := os.RemoveAll(tsk.WorktreePath); err != nil {
		t.Fatalf("failed to remove worktree: %v", err)
	}

	opts := &EditorOptions{
		RepoPath: repoPath,
		TaskID:   "test-editor-task",
		Editor:   "echo",
	}

	err = runTaskEditor(opts)
	if err == nil {
		t.Error("expected error for missing worktree, got nil")
	}
}

func TestRunTaskEditorWithValidTask(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Start a task first
	startOpts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "test-valid-task",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	if err := runTaskStart(startOpts); err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	// Use 'true' command which always succeeds and doesn't do anything
	// This is available on Unix systems and Git Bash on Windows
	opts := &EditorOptions{
		RepoPath: repoPath,
		TaskID:   "test-valid-task",
		Editor:   "true",
	}

	err := runTaskEditor(opts)
	if err != nil {
		t.Errorf("runTaskEditor() failed: %v", err)
	}
}

func TestRunTaskEditorWithBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Start a task first
	startOpts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "test-branch-task",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	if err := runTaskStart(startOpts); err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	opts := &EditorOptions{
		RepoPath: repoPath,
		Branch:   "awt/test-agent/test-branch-task",
		Editor:   "true",
	}

	err := runTaskEditor(opts)
	if err != nil {
		t.Errorf("runTaskEditor() with branch failed: %v", err)
	}
}

func TestRunTaskEditorNoEditorFound(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Start a task first
	startOpts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "test-no-editor-task",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	if err := runTaskStart(startOpts); err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	// Save current EDITOR env var
	oldEditor := os.Getenv("EDITOR")
	defer func() {
		if oldEditor != "" {
			os.Setenv("EDITOR", oldEditor)
		} else {
			os.Unsetenv("EDITOR")
		}
	}()

	// Unset EDITOR and use a non-existent editor
	os.Unsetenv("EDITOR")

	opts := &EditorOptions{
		RepoPath: repoPath,
		TaskID:   "test-no-editor-task",
		Editor:   "completely-non-existent-editor-12345",
	}

	err := runTaskEditor(opts)
	if err == nil {
		t.Error("expected error for non-existent editor, got nil")
	}
}

func TestRunTaskEditorWithEnvVar(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Start a task first
	startOpts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "test-env-editor-task",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	if err := runTaskStart(startOpts); err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	// Save current EDITOR env var
	oldEditor := os.Getenv("EDITOR")
	defer func() {
		if oldEditor != "" {
			os.Setenv("EDITOR", oldEditor)
		} else {
			os.Unsetenv("EDITOR")
		}
	}()

	// Set EDITOR to a valid command
	os.Setenv("EDITOR", "true")

	opts := &EditorOptions{
		RepoPath: repoPath,
		TaskID:   "test-env-editor-task",
		// Don't set Editor, should use EDITOR env var
	}

	err := runTaskEditor(opts)
	if err != nil {
		t.Errorf("runTaskEditor() with EDITOR env var failed: %v", err)
	}
}

func TestRunTaskEditorRepoNotFound(t *testing.T) {
	opts := &EditorOptions{
		RepoPath: "/non/existent/path",
		TaskID:   "some-task",
		Editor:   "echo",
	}

	err := runTaskEditor(opts)
	if err == nil {
		t.Error("expected error for non-existent repository, got nil")
	}
}

func TestRunTaskEditorInferFromDirectory(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Start a task first
	startOpts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "test-infer-task",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	if err := runTaskStart(startOpts); err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	// Load task to get worktree path
	store := task.NewTaskStore(filepath.Join(repoPath, ".git"))
	tsk, err := store.Load("test-infer-task")
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}

	// Change to worktree directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	if err := os.Chdir(tsk.WorktreePath); err != nil {
		t.Fatalf("failed to change to worktree directory: %v", err)
	}

	opts := &EditorOptions{
		// Don't set TaskID or Branch - should infer from current directory
		Editor: "true",
	}

	err = runTaskEditor(opts)
	if err != nil {
		t.Errorf("runTaskEditor() with inferred task ID failed: %v", err)
	}
}

func TestRunTaskEditorInvalidBranch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	opts := &EditorOptions{
		RepoPath: repoPath,
		Branch:   "invalid-branch-format",
		Editor:   "true",
	}

	err := runTaskEditor(opts)
	if err == nil {
		t.Error("expected error for invalid branch format, got nil")
	}
}
