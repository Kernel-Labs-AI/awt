package safety

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAgentName(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		agent   string
		wantErr bool
	}{
		{"valid simple", "claude", false},
		{"valid with dash", "claude-3", false},
		{"valid with underscore", "claude_agent", false},
		{"valid mixed", "Claude_Agent-1", false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 51), true},
		{"with space", "claude agent", true},
		{"with special chars", "claude!", true},
		{"with dot", "claude.ai", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateAgentName(tt.agent)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAgentName(%q) error = %v, wantErr %v", tt.agent, err, tt.wantErr)
			}
		})
	}
}

func TestValidateTaskTitle(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"valid simple", "Add feature", false},
		{"valid with special chars", "Fix bug: issue #123", false},
		{"valid long", strings.Repeat("a", 200), false},
		{"empty", "", true},
		{"too long", strings.Repeat("a", 201), true},
		{"with newline", "Test\nwith\nnewline", true},
		{"with tab", "Test\twith\ttab", true},
		{"with carriage return", "Test\rwith\rCR", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateTaskTitle(tt.title)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskTitle(%q) error = %v, wantErr %v", tt.title, err, tt.wantErr)
			}
		})
	}
}

func TestValidateBranchName(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		branch  string
		wantErr bool
	}{
		{"valid simple", "feature/add-auth", false},
		{"valid with numbers", "feature/123-add-auth", false},
		{"empty", "", true},
		{"starts with dash", "-invalid", true},
		{"ends with dot", "invalid.", true},
		{"ends with .lock", "invalid.lock", true},
		{"contains ..", "feature..invalid", true},
		{"contains ~", "feature~invalid", true},
		{"contains space", "feature invalid", true},
		{"contains @{", "feature@{invalid", true},
		{"just @", "@", true},
		{"contains colon", "feature:invalid", true},
		{"contains asterisk", "feature*invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateBranchName(tt.branch)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBranchName(%q) error = %v, wantErr %v", tt.branch, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCommitMessage(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		message string
		wantErr bool
	}{
		{"valid short", "Fix bug", false},
		{"valid with body", "Fix bug\n\nDetailed explanation", false},
		{"valid max subject", strings.Repeat("a", 100), false},
		{"empty", "", true},
		{"too long overall", strings.Repeat("a", 10001), true},
		{"subject too long", strings.Repeat("a", 101), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateCommitMessage(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommitMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"already valid", "feature/add-auth", "feature/add-auth"},
		{"with spaces", "feature add auth", "feature-add-auth"},
		{"with dots", "feature..invalid", "feature-invalid"},
		{"leading dash", "-invalid", "invalid"},
		{"trailing dot", "invalid.", "invalid"},
		{"multiple dashes", "feature---auth", "feature-auth"},
		{"empty", "", "branch"},
		{"complex", "Feature: Add Auth (v2)", "Feature-Add-Auth-(v2)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeBranchName(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitizeTaskTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"already valid", "Add feature", "Add feature"},
		{"with newlines", "Add\nfeature\ntest", "Add feature test"},
		{"with tabs", "Add\tfeature\ttest", "Add feature test"},
		{"multiple spaces", "Add  feature  test", "Add feature test"},
		{"too long", strings.Repeat("a", 250), strings.Repeat("a", 197) + "..."},
		{"leading/trailing spaces", "  Add feature  ", "Add feature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeTaskTitle(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeTaskTitle(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateWorktreePath(t *testing.T) {
	v := NewValidator()

	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "awt-safety-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Create a test file (not a directory)
	testFile := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a non-empty directory
	nonEmptyDir := filepath.Join(tempDir, "nonempty")
	if err := os.MkdirAll(nonEmptyDir, 0755); err != nil {
		t.Fatalf("failed to create non-empty dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create file in non-empty dir: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		repoRoot string
		wantErr  bool
	}{
		{
			name:     "valid new path",
			path:     filepath.Join(tempDir, "valid-worktree"),
			repoRoot: tempDir,
			wantErr:  false,
		},
		{
			name:     "empty path",
			path:     "",
			repoRoot: tempDir,
			wantErr:  true,
		},
		{
			name:     "path is file not directory",
			path:     testFile,
			repoRoot: tempDir,
			wantErr:  true,
		},
		{
			name:     "non-empty directory",
			path:     nonEmptyDir,
			repoRoot: tempDir,
			wantErr:  true,
		},
		{
			name:     "inside .git directory",
			path:     filepath.Join(tempDir, ".git", "worktrees", "test"),
			repoRoot: tempDir,
			wantErr:  true,
		},
		{
			name:     "same as repo root",
			path:     tempDir,
			repoRoot: tempDir,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateWorktreePath(tt.path, tt.repoRoot)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWorktreePath(%q, %q) error = %v, wantErr %v", tt.path, tt.repoRoot, err, tt.wantErr)
			}
		})
	}
}

func TestIsSafeToRemoveWorktree(t *testing.T) {
	v := NewValidator()

	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "awt-safety-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Create a test worktree
	worktree := filepath.Join(tempDir, "test-worktree")
	if err := os.MkdirAll(worktree, 0755); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create a file (not a directory)
	testFile := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		force   bool
		wantErr bool
	}{
		{
			name:    "non-existent path",
			path:    filepath.Join(tempDir, "does-not-exist"),
			force:   false,
			wantErr: false, // Already removed, safe
		},
		{
			name:    "valid worktree",
			path:    worktree,
			force:   false,
			wantErr: false,
		},
		{
			name:    "path is file not directory",
			path:    testFile,
			force:   false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.IsSafeToRemoveWorktree(tt.path, tt.force)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsSafeToRemoveWorktree(%q, %v) error = %v, wantErr %v", tt.path, tt.force, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRemoteName(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		remote  string
		wantErr bool
	}{
		{"valid simple", "origin", false},
		{"valid with dash", "my-remote", false},
		{"valid with slash", "upstream/main", false},
		{"empty", "", true},
		{"starts with dash", "-invalid", true},
		{"contains ..", "remote..invalid", true},
		{"contains space", "remote invalid", true},
		{"contains tab", "remote\tinvalid", true},
		{"contains tilde", "remote~invalid", true},
		{"contains caret", "remote^invalid", true},
		{"contains colon", "remote:invalid", true},
		{"contains question", "remote?invalid", true},
		{"contains asterisk", "remote*invalid", true},
		{"contains bracket", "remote[invalid", true},
		{"contains backslash", "remote\\invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateRemoteName(tt.remote)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRemoteName(%q) error = %v, wantErr %v", tt.remote, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRefspec(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		refspec string
		wantErr bool
	}{
		{"valid simple", "refs/heads/main:refs/remotes/origin/main", false},
		{"valid with plus", "+refs/heads/*:refs/remotes/origin/*", false},
		{"valid force fetch", "refs/heads/feature", false},
		{"empty", "", true},
		{"starts with dash", "-invalid", true},
		{"contains control char", "refs/heads/\x00invalid", true},
		{"contains delete char", "refs/heads/\x7finvalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateRefspec(tt.refspec)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRefspec(%q) error = %v, wantErr %v", tt.refspec, err, tt.wantErr)
			}
		})
	}
}
