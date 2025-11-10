package safety

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validator provides safety checks for AWT operations
type Validator struct{}

// NewValidator creates a new safety validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateTaskTitle validates a task title
func (v *Validator) ValidateTaskTitle(title string) error {
	if title == "" {
		return fmt.Errorf("task title cannot be empty")
	}

	if len(title) > 200 {
		return fmt.Errorf("task title too long (max 200 characters)")
	}

	// Check for problematic characters
	if strings.ContainsAny(title, "\n\r\t") {
		return fmt.Errorf("task title cannot contain newlines or tabs")
	}

	return nil
}

// ValidateBranchName validates a branch name
func (v *Validator) ValidateBranchName(branch string) error {
	if branch == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Git branch name restrictions
	if strings.HasPrefix(branch, "-") {
		return fmt.Errorf("branch name cannot start with a dash")
	}

	if strings.HasSuffix(branch, ".") {
		return fmt.Errorf("branch name cannot end with a dot")
	}

	if strings.HasSuffix(branch, ".lock") {
		return fmt.Errorf("branch name cannot end with .lock")
	}

	// Check for problematic characters
	forbidden := []string{"..", "~", "^", ":", "?", "*", "[", " ", "\t", "\n", "\\"}
	for _, char := range forbidden {
		if strings.Contains(branch, char) {
			return fmt.Errorf("branch name contains forbidden character: %s", char)
		}
	}

	// Check for @ without braces (git reflog syntax)
	if strings.Contains(branch, "@{") {
		return fmt.Errorf("branch name cannot contain @{")
	}

	// Cannot be just @ alone
	if branch == "@" {
		return fmt.Errorf("branch name cannot be @")
	}

	// Cannot contain ASCII control characters
	for _, c := range branch {
		if c < 32 || c == 127 {
			return fmt.Errorf("branch name contains control character")
		}
	}

	return nil
}

// ValidateAgentName validates an agent name
func (v *Validator) ValidateAgentName(agent string) error {
	if agent == "" {
		return fmt.Errorf("agent name cannot be empty")
	}

	if len(agent) > 50 {
		return fmt.Errorf("agent name too long (max 50 characters)")
	}

	// Allow alphanumeric, dash, underscore
	for _, c := range agent {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return fmt.Errorf("agent name can only contain alphanumeric, dash, and underscore")
		}
	}

	return nil
}

// ValidateWorktreePath validates a worktree path
func (v *Validator) ValidateWorktreePath(path, repoRoot string) error {
	if path == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if path already exists
	if info, err := os.Stat(absPath); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path exists and is not a directory: %s", absPath)
		}
		// Check if directory is empty
		entries, err := os.ReadDir(absPath)
		if err != nil {
			return fmt.Errorf("cannot read directory: %w", err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("directory is not empty: %s", absPath)
		}
	}

	// Ensure path is not inside .git directory
	absRepoRoot, _ := filepath.Abs(repoRoot)
	gitDir := filepath.Join(absRepoRoot, ".git")
	if strings.HasPrefix(absPath, gitDir+string(filepath.Separator)) {
		return fmt.Errorf("worktree path cannot be inside .git directory")
	}

	// Ensure path is not the repository root itself
	if absPath == absRepoRoot {
		return fmt.Errorf("worktree path cannot be the repository root")
	}

	return nil
}

// ValidateCommitMessage validates a commit message
func (v *Validator) ValidateCommitMessage(message string) error {
	if message == "" {
		return fmt.Errorf("commit message cannot be empty")
	}

	if len(message) > 10000 {
		return fmt.Errorf("commit message too long (max 10000 characters)")
	}

	// Warn if first line is too long (common convention is 50-72 chars)
	lines := strings.Split(message, "\n")
	if len(lines[0]) > 100 {
		return fmt.Errorf("commit message subject line too long (max 100 characters)")
	}

	return nil
}

// IsSafeToRemoveWorktree checks if it's safe to remove a worktree
func (v *Validator) IsSafeToRemoveWorktree(worktreePath string, force bool) error {
	absPath, err := filepath.Abs(worktreePath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Check if worktree exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already removed, safe
		}
		return fmt.Errorf("cannot access worktree: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("worktree path is not a directory: %s", absPath)
	}

	// Check if current working directory is inside worktree
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current directory: %w", err)
	}

	cwdAbs, _ := filepath.Abs(cwd)
	rel, err := filepath.Rel(absPath, cwdAbs)
	if err == nil && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..") {
		if !force {
			return fmt.Errorf("cannot remove worktree: current directory is inside it\nUse --force-remove to override, or change directory first")
		}
	}

	return nil
}

// ValidateRemoteName validates a git remote name
func (v *Validator) ValidateRemoteName(remote string) error {
	if remote == "" {
		return fmt.Errorf("remote name cannot be empty")
	}

	// Git remote name restrictions (similar to branch names)
	if strings.HasPrefix(remote, "-") {
		return fmt.Errorf("remote name cannot start with a dash")
	}

	if strings.Contains(remote, "..") {
		return fmt.Errorf("remote name cannot contain ..")
	}

	// Check for problematic characters
	forbidden := []string{" ", "\t", "\n", "~", "^", ":", "?", "*", "[", "\\"}
	for _, char := range forbidden {
		if strings.Contains(remote, char) {
			return fmt.Errorf("remote name contains forbidden character: %s", char)
		}
	}

	return nil
}

// ValidateRefspec validates a git refspec
func (v *Validator) ValidateRefspec(refspec string) error {
	if refspec == "" {
		return fmt.Errorf("refspec cannot be empty")
	}

	// Basic refspec validation
	if strings.HasPrefix(refspec, "-") {
		return fmt.Errorf("refspec cannot start with a dash")
	}

	// Check for control characters
	for _, c := range refspec {
		if c < 32 || c == 127 {
			return fmt.Errorf("refspec contains control character")
		}
	}

	return nil
}

// SanitizeBranchName sanitizes a string to be a valid branch name
func SanitizeBranchName(name string) string {
	// Replace forbidden characters with dash
	sanitized := name
	forbidden := []string{"..", "~", "^", ":", "?", "*", "[", " ", "\t", "\n", "\\", "@{"}
	for _, char := range forbidden {
		sanitized = strings.ReplaceAll(sanitized, char, "-")
	}

	// Remove leading dashes
	sanitized = strings.TrimPrefix(sanitized, "-")

	// Remove trailing dots and .lock
	sanitized = strings.TrimSuffix(sanitized, ".")
	sanitized = strings.TrimSuffix(sanitized, ".lock")

	// Remove control characters
	var result strings.Builder
	for _, c := range sanitized {
		if c >= 32 && c != 127 {
			result.WriteRune(c)
		}
	}

	sanitized = result.String()

	// Collapse multiple dashes
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	// Ensure not empty
	if sanitized == "" {
		sanitized = "branch"
	}

	return sanitized
}

// SanitizeTaskTitle sanitizes a task title
func SanitizeTaskTitle(title string) string {
	// Remove newlines and tabs
	sanitized := strings.ReplaceAll(title, "\n", " ")
	sanitized = strings.ReplaceAll(sanitized, "\r", " ")
	sanitized = strings.ReplaceAll(sanitized, "\t", " ")

	// Collapse multiple spaces
	for strings.Contains(sanitized, "  ") {
		sanitized = strings.ReplaceAll(sanitized, "  ", " ")
	}

	// Trim whitespace
	sanitized = strings.TrimSpace(sanitized)

	// Truncate if too long
	if len(sanitized) > 200 {
		sanitized = sanitized[:197] + "..."
	}

	return sanitized
}
