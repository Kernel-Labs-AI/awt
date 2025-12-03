package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kernel-labs-ai/awt/internal/task"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "awt-cmd-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(tempDir)
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git
	_ = exec.Command("git", "-C", tempDir, "config", "user.name", "Test User").Run()
	_ = exec.Command("git", "-C", tempDir, "config", "user.email", "test@example.com").Run()

	// Create initial commit
	readmePath := filepath.Join(tempDir, "README.md")
	_ = os.WriteFile(readmePath, []byte("# Test Repo\n"), 0644)
	_ = exec.Command("git", "-C", tempDir, "add", "README.md").Run()
	_ = exec.Command("git", "-C", tempDir, "commit", "-m", "Initial commit").Run()

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestRunTaskStart(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	tests := []struct {
		name    string
		opts    *StartOptions
		wantErr bool
	}{
		{
			name: "valid task",
			opts: &StartOptions{
				RepoPath:     repoPath,
				Agent:        "test-agent",
				Title:        "Test task",
				Base:         "HEAD",
				NoFetch:      true,
				BranchPrefix: "awt",
				WorktreeDir:  ".awt/wt",
				OutputJSON:   true,
			},
			wantErr: false,
		},
		{
			name: "invalid agent name",
			opts: &StartOptions{
				RepoPath:     repoPath,
				Agent:        "invalid agent!",
				Title:        "Test task",
				Base:         "HEAD",
				NoFetch:      true,
				BranchPrefix: "awt",
				WorktreeDir:  ".awt/wt",
			},
			wantErr: true,
		},
		{
			name: "empty title",
			opts: &StartOptions{
				RepoPath:     repoPath,
				Agent:        "test-agent",
				Title:        "",
				Base:         "HEAD",
				NoFetch:      true,
				BranchPrefix: "awt",
				WorktreeDir:  ".awt/wt",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := runTaskStart(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("runTaskStart() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If successful, verify the worktree was created
			if err == nil {
				// Load all tasks and find the one we just created
				gitCommonDir := filepath.Join(repoPath, ".git")
				store := task.NewTaskStore(gitCommonDir)
				tasks, err := store.List()
				if err != nil {
					t.Fatalf("failed to list tasks: %v", err)
				}
				if len(tasks) == 0 {
					t.Error("no tasks found after creation")
					return
				}
				// Get the latest task (should be the one we just created)
				latestTask := tasks[len(tasks)-1]
				if _, err := os.Stat(latestTask.WorktreePath); os.IsNotExist(err) {
					t.Errorf("worktree was not created at: %s", latestTask.WorktreePath)
				}
				// Clean up the global worktree directory
				defer os.RemoveAll(latestTask.WorktreePath)
			}
		})
	}
}

func TestRunTaskStartWithCustomID(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	opts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task with custom ID",
		Base:         "HEAD",
		ID:           "20251110-120000-abc123",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
		OutputJSON:   true,
	}

	err := runTaskStart(opts)
	if err != nil {
		t.Fatalf("runTaskStart() failed: %v", err)
	}

	// Load task metadata to get the actual worktree path
	gitCommonDir := filepath.Join(repoPath, ".git")
	store := task.NewTaskStore(gitCommonDir)
	savedTask, err := store.Load(opts.ID)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}

	// Verify worktree exists at the path stored in task metadata
	if _, err := os.Stat(savedTask.WorktreePath); os.IsNotExist(err) {
		t.Errorf("worktree was not created at expected path: %s", savedTask.WorktreePath)
	}

	// Clean up the global worktree directory
	defer os.RemoveAll(savedTask.WorktreePath)
}

func TestRunTaskStartInvalidTaskID(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	opts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "invalid/id", // Use a task ID with invalid character (slash)
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	err := runTaskStart(opts)
	if err == nil {
		t.Error("expected error for invalid task ID, got nil")
	}
}

func TestStartResultJSON(t *testing.T) {
	result := StartResult{
		ID:           "20251110-120000-abc123",
		Branch:       "awt/test-agent/20251110-120000-abc123",
		WorktreePath: "/path/to/worktree",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal StartResult: %v", err)
	}

	var decoded StartResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal StartResult: %v", err)
	}

	if decoded.ID != result.ID {
		t.Errorf("ID mismatch: got %q, want %q", decoded.ID, result.ID)
	}
	if decoded.Branch != result.Branch {
		t.Errorf("Branch mismatch: got %q, want %q", decoded.Branch, result.Branch)
	}
	if decoded.WorktreePath != result.WorktreePath {
		t.Errorf("WorktreePath mismatch: got %q, want %q", decoded.WorktreePath, result.WorktreePath)
	}
}

func TestRunTaskStartSetsUpstreamTracking(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Add a remote for testing
	_ = exec.Command("git", "-C", repoPath, "remote", "add", "origin", "https://example.com/repo.git").Run()

	// Initialize AWT config directory
	awtConfigDir := filepath.Join(repoPath, ".git", "awt")
	_ = os.MkdirAll(awtConfigDir, 0755)

	opts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task with upstream tracking",
		Base:         "HEAD",
		ID:           "20251110-120000-test123",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	err := runTaskStart(opts)
	if err != nil {
		t.Fatalf("runTaskStart() failed: %v", err)
	}

	// Load task metadata to get the actual worktree path
	gitCommonDir := filepath.Join(repoPath, ".git")
	store := task.NewTaskStore(gitCommonDir)
	savedTask, err := store.Load(opts.ID)
	if err != nil {
		t.Fatalf("failed to load task: %v", err)
	}

	worktreePath := savedTask.WorktreePath
	defer os.RemoveAll(worktreePath)

	// Verify worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		t.Fatal("worktree was not created")
	}

	// Check that upstream tracking is set correctly by checking git config
	expectedBranch := fmt.Sprintf("awt/%s/%s", opts.Agent, opts.ID)

	remoteCmd := exec.Command("git", "-C", worktreePath, "config", fmt.Sprintf("branch.%s.remote", expectedBranch))
	remoteOutput, err := remoteCmd.Output()
	if err != nil {
		t.Fatalf("failed to check remote config: %v", err)
	}
	remote := strings.TrimSpace(string(remoteOutput))
	if remote != "origin" {
		t.Errorf("branch remote = %q, want %q", remote, "origin")
	}

	mergeCmd := exec.Command("git", "-C", worktreePath, "config", fmt.Sprintf("branch.%s.merge", expectedBranch))
	mergeOutput, err := mergeCmd.Output()
	if err != nil {
		t.Fatalf("failed to check merge config: %v", err)
	}
	mergeRef := strings.TrimSpace(string(mergeOutput))
	expectedMergeRef := fmt.Sprintf("refs/heads/%s", expectedBranch)
	if mergeRef != expectedMergeRef {
		t.Errorf("branch merge ref = %q, want %q", mergeRef, expectedMergeRef)
	}
}
