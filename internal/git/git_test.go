package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "awt-git-test-*")
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

func TestGitNew(t *testing.T) {
	g := New("/tmp/test", false)
	if g == nil {
		t.Fatal("New returned nil")
	}
	if g.workTreeRoot != "/tmp/test" {
		t.Errorf("workTreeRoot = %s, expected /tmp/test", g.workTreeRoot)
	}
}

func TestGitRun(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoPath, false)

	// Test successful command
	result, err := g.run("status", "--short")
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	// Test command with error
	result, err = g.run("log", "nonexistent-ref")
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}
	if result.ExitCode == 0 {
		t.Error("expected non-zero exit code for invalid ref")
	}
}

func TestGitBranchExists(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoPath, false)

	// Check for main/master branch (depends on git version)
	exists, err := g.BranchExists("master")
	if err != nil {
		t.Fatalf("BranchExists failed: %v", err)
	}
	if !exists {
		// Try main instead
		exists, err = g.BranchExists("main")
		if err != nil {
			t.Fatalf("BranchExists failed: %v", err)
		}
		if !exists {
			t.Fatalf("Neither master nor main branch exists")
		}
	}

	// Check for non-existent branch
	exists, err = g.BranchExists("nonexistent")
	if err != nil {
		t.Fatalf("BranchExists failed: %v", err)
	}
	if exists {
		t.Error("expected branch 'nonexistent' to not exist")
	}
}

func TestGitWorktreeOperations(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoPath, false)

	// Get current branch
	currentBranch, err := g.CurrentBranch()
	if err != nil {
		t.Fatalf("CurrentBranch failed: %v", err)
	}
	if currentBranch == "" {
		t.Fatal("current branch is empty")
	}

	// List worktrees (should have just the main one)
	worktrees, err := g.WorktreeList()
	if err != nil {
		t.Fatalf("WorktreeList failed: %v", err)
	}
	if len(worktrees) != 1 {
		t.Errorf("expected 1 worktree, got %d", len(worktrees))
	}

	// Add a worktree
	wtPath := filepath.Join(repoPath, "wt-test")
	result, err := g.WorktreeAdd(wtPath, "test-branch", currentBranch)
	if err != nil {
		t.Fatalf("WorktreeAdd failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("WorktreeAdd failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// Verify worktree was created
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Fatal("worktree directory was not created")
	}

	// List worktrees again
	worktrees, err = g.WorktreeList()
	if err != nil {
		t.Fatalf("WorktreeList failed: %v", err)
	}
	if len(worktrees) != 2 {
		t.Errorf("expected 2 worktrees, got %d", len(worktrees))
	}

	// Check if branch is checked out
	isCheckedOut, path, err := g.IsBranchCheckedOut("test-branch")
	if err != nil {
		t.Fatalf("IsBranchCheckedOut failed: %v", err)
	}
	if !isCheckedOut {
		t.Error("expected test-branch to be checked out")
	}
	// Resolve symlinks for comparison (macOS /var -> /private/var)
	expectedPath, _ := filepath.EvalSymlinks(wtPath)
	actualPath, _ := filepath.EvalSymlinks(path)
	if actualPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, actualPath)
	}

	// Remove worktree
	result, err = g.WorktreeRemove(wtPath, false)
	if err != nil {
		t.Fatalf("WorktreeRemove failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("WorktreeRemove failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// Verify worktree was removed
	worktrees, err = g.WorktreeList()
	if err != nil {
		t.Fatalf("WorktreeList failed: %v", err)
	}
	if len(worktrees) != 1 {
		t.Errorf("expected 1 worktree after removal, got %d", len(worktrees))
	}
}

func TestParseWorktreeList(t *testing.T) {
	output := `worktree /path/to/repo
HEAD 1234567890abcdef
branch refs/heads/main

worktree /path/to/worktree
HEAD abcdef1234567890
branch refs/heads/feature

worktree /path/to/detached
HEAD 9876543210fedcba
detached
`

	worktrees := parseWorktreeList(output)

	if len(worktrees) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(worktrees))
	}

	// Check first worktree
	if worktrees[0].Path != "/path/to/repo" {
		t.Errorf("worktree 0 path = %s, expected /path/to/repo", worktrees[0].Path)
	}
	if worktrees[0].Branch != "refs/heads/main" {
		t.Errorf("worktree 0 branch = %s, expected refs/heads/main", worktrees[0].Branch)
	}
	if worktrees[0].Commit != "1234567890abcdef" {
		t.Errorf("worktree 0 commit = %s, expected 1234567890abcdef", worktrees[0].Commit)
	}

	// Check second worktree
	if worktrees[1].Path != "/path/to/worktree" {
		t.Errorf("worktree 1 path = %s, expected /path/to/worktree", worktrees[1].Path)
	}
	if worktrees[1].Branch != "refs/heads/feature" {
		t.Errorf("worktree 1 branch = %s, expected refs/heads/feature", worktrees[1].Branch)
	}

	// Check third worktree (detached HEAD)
	if worktrees[2].Path != "/path/to/detached" {
		t.Errorf("worktree 2 path = %s, expected /path/to/detached", worktrees[2].Path)
	}
	if worktrees[2].Branch != "" {
		t.Errorf("worktree 2 branch = %s, expected empty (detached)", worktrees[2].Branch)
	}
}

func TestGitFetch(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoPath, false)

	// Test fetch (will fail but tests the method)
	result, err := g.Fetch("", "")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	// Exit code doesn't matter for this test, we just want to cover the code path
	_ = result.ExitCode
}

func TestGitAdd(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoPath, false)

	// Create a new file
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test add
	result, err := g.Add(testFile)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("Add returned non-zero exit code: %d", result.ExitCode)
	}
}

func TestGitCommit(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoPath, false)

	// Create and add a file
	testFile := filepath.Join(repoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Add the file
	_, _ = g.Add(testFile)

	// Test commit
	result, err := g.Commit("Test commit message", false, false, false)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("Commit returned non-zero exit code: %d, stderr: %s", result.ExitCode, result.Stderr)
	}
}

func TestGitStatus(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoPath, false)

	// Test status
	result, err := g.Status()
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("Status returned non-zero exit code: %d", result.ExitCode)
	}
}

func TestGitRevParse(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoPath, false)

	// Test RevParse with HEAD
	sha, err := g.RevParse("HEAD")
	if err != nil {
		t.Fatalf("RevParse failed: %v", err)
	}
	if sha == "" {
		t.Error("RevParse returned empty SHA")
	}
}

func TestGitWorktreePrune(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	g := New(repoPath, false)

	// Test worktree prune (won't do much but covers the code)
	result, err := g.WorktreePrune()
	if err != nil {
		t.Fatalf("WorktreePrune failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("WorktreePrune returned non-zero exit code: %d", result.ExitCode)
	}
}
