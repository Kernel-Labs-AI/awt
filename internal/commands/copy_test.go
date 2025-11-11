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

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths
		{
			name:    "simple file",
			path:    ".env",
			wantErr: false,
		},
		{
			name:    "nested file",
			path:    "config/local.json",
			wantErr: false,
		},
		{
			name:    "deeply nested",
			path:    "foo/bar/baz/file.txt",
			wantErr: false,
		},
		// Invalid paths - path traversal attempts
		{
			name:    "parent directory traversal",
			path:    "../etc/passwd",
			wantErr: true,
		},
		{
			name:    "multiple parent traversal",
			path:    "../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "parent in middle",
			path:    "foo/../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "absolute path unix",
			path:    "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "just parent directory",
			path:    "..",
			wantErr: true,
		},
		{
			name:    "parent with trailing",
			path:    "../",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestRunTaskCopyPathTraversal(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Start a task first
	startOpts := &StartOptions{
		RepoPath:     repoPath,
		Agent:        "test-agent",
		Title:        "Test task",
		Base:         "HEAD",
		ID:           "test-security-task",
		NoFetch:      true,
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	if err := runTaskStart(startOpts); err != nil {
		t.Fatalf("failed to start task: %v", err)
	}

	// Try various path traversal attacks
	pathTraversalTests := []struct {
		name string
		file string
	}{
		{"parent directory", "../etc/passwd"},
		{"multiple parent", "../../etc/passwd"},
		{"absolute path", "/etc/passwd"},
		{"nested parent", "foo/../../etc/passwd"},
	}

	for _, tt := range pathTraversalTests {
		t.Run(tt.name, func(t *testing.T) {
			copyOpts := &CopyOptions{
				RepoPath: repoPath,
				TaskID:   "test-security-task",
				Files:    []string{tt.file},
			}

			err := runTaskCopy(copyOpts)
			if err == nil {
				t.Errorf("expected error for path traversal attempt with %q, got nil", tt.file)
			}
		})
	}
}

func TestIsSubPath(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{
			name:   "direct child",
			parent: "/home/user/worktree",
			child:  "/home/user/worktree/file.txt",
			want:   true,
		},
		{
			name:   "nested child",
			parent: "/home/user/worktree",
			child:  "/home/user/worktree/foo/bar/file.txt",
			want:   true,
		},
		{
			name:   "same path",
			parent: "/home/user/worktree",
			child:  "/home/user/worktree",
			want:   true,
		},
		{
			name:   "outside parent",
			parent: "/home/user/worktree",
			child:  "/home/user/other/file.txt",
			want:   false,
		},
		{
			name:   "parent of parent",
			parent: "/home/user/worktree",
			child:  "/home/user",
			want:   false,
		},
		{
			name:   "sibling",
			parent: "/home/user/worktree",
			child:  "/home/user/worktree2/file.txt",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubPath(tt.parent, tt.child)
			if got != tt.want {
				t.Errorf("isSubPath(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
			}
		})
	}
}
