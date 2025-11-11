package errors

import (
	"encoding/json"
	"fmt"
	"os"
)

// ExitCode represents an AWT error exit code
type ExitCode int

const (
	// ExitSuccess is the success exit code
	ExitSuccess ExitCode = 0

	// Repository errors (10-19)
	ExitRepoNotFound ExitCode = 10
	ExitGitTooOld    ExitCode = 11

	// Branch/worktree errors (20-29)
	ExitBranchExists             ExitCode = 20
	ExitBranchCheckedOutElsewhere ExitCode = 21
	ExitWorktreeExists           ExitCode = 22
	ExitWorktreeNotFound         ExitCode = 23
	ExitDetachFailed             ExitCode = 24
	ExitRemoveFailed             ExitCode = 25

	// Sync/push errors (30-39)
	ExitSyncConflicts ExitCode = 30
	ExitPushRejected  ExitCode = 31

	// Lock errors (40-49)
	ExitLockTimeout ExitCode = 40
	ExitLockHeld    ExitCode = 41

	// Tool errors (50-59)
	ExitToolMissing ExitCode = 50

	// Task errors (60-69)
	ExitInvalidTaskID      ExitCode = 60
	ExitCaseOnlyCollision  ExitCode = 61
)

// AWTError represents an AWT-specific error with an exit code and hint
type AWTError struct {
	Code    ExitCode
	Message string
	Hint    string
	Cause   error
}

// Error implements the error interface
func (e *AWTError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s\nHint: %s", e.Message, e.Hint)
	}
	return e.Message
}

// Unwrap returns the underlying cause
func (e *AWTError) Unwrap() error {
	return e.Cause
}

// JSONError represents the JSON format for errors
type JSONError struct {
	Error string   `json:"error"`
	Code  ExitCode `json:"code"`
	Hint  string   `json:"hint,omitempty"`
}

// ToJSON returns the JSON representation of the error
func (e *AWTError) ToJSON() string {
	je := JSONError{
		Error: e.Message,
		Code:  e.Code,
		Hint:  e.Hint,
	}
	data, _ := json.MarshalIndent(je, "", "  ")
	return string(data)
}

// New creates a new AWT error
func New(code ExitCode, message string, hint string, cause error) *AWTError {
	return &AWTError{
		Code:    code,
		Message: message,
		Hint:    hint,
		Cause:   cause,
	}
}

// Handle handles an error by printing it and exiting with the appropriate code
func Handle(err error, useJSON bool) {
	if err == nil {
		return
	}

	// Check if it's an AWTError
	if awtErr, ok := err.(*AWTError); ok {
		if useJSON {
			fmt.Fprintln(os.Stderr, awtErr.ToJSON())
		} else {
			fmt.Fprintln(os.Stderr, awtErr.Error())
		}
		os.Exit(int(awtErr.Code))
	}

	// Generic error
	if useJSON {
		je := JSONError{
			Error: err.Error(),
			Code:  1,
		}
		data, _ := json.MarshalIndent(je, "", "  ")
		fmt.Fprintln(os.Stderr, string(data))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	os.Exit(1)
}

// Predefined error constructors for common cases

// RepoNotFound creates a REPO_NOT_FOUND error
func RepoNotFound(path string) *AWTError {
	return New(
		ExitRepoNotFound,
		fmt.Sprintf("Git repository not found at: %s", path),
		"Make sure you're running this command from within a Git repository, or use --repo to specify the path.",
		nil,
	)
}

// GitTooOld creates a GIT_TOO_OLD error
func GitTooOld(version, minVersion string) *AWTError {
	return New(
		ExitGitTooOld,
		fmt.Sprintf("Git version %s is too old (minimum required: %s)", version, minVersion),
		"Please upgrade Git to version 2.33 or later.",
		nil,
	)
}

// BranchExists creates a BRANCH_EXISTS error
func BranchExists(branch string) *AWTError {
	return New(
		ExitBranchExists,
		fmt.Sprintf("Branch already exists: %s", branch),
		"Use a different task ID or delete the existing branch first.",
		nil,
	)
}

// BranchCheckedOutElsewhere creates a BRANCH_CHECKED_OUT_ELSEWHERE error
func BranchCheckedOutElsewhere(branch, worktree string) *AWTError {
	return New(
		ExitBranchCheckedOutElsewhere,
		fmt.Sprintf("Branch %s is checked out at: %s", branch, worktree),
		"Use 'awt task unlock' to detach the branch, or check out a different branch in that worktree.",
		nil,
	)
}

// WorktreeExists creates a WORKTREE_EXISTS error
func WorktreeExists(path string) *AWTError {
	return New(
		ExitWorktreeExists,
		fmt.Sprintf("Worktree already exists at: %s", path),
		"Remove the existing worktree or choose a different path.",
		nil,
	)
}

// WorktreeNotFound creates a WORKTREE_NOT_FOUND error
func WorktreeNotFound(path string) *AWTError {
	return New(
		ExitWorktreeNotFound,
		fmt.Sprintf("Worktree not found at: %s", path),
		"The worktree may have been removed. Use 'awt list' to see available tasks.",
		nil,
	)
}

// DetachFailed creates a DETACH_FAILED error
func DetachFailed(worktree string, cause error) *AWTError {
	return New(
		ExitDetachFailed,
		fmt.Sprintf("Failed to detach HEAD in worktree: %s", worktree),
		"Check if the worktree still exists and is in a valid state.",
		cause,
	)
}

// RemoveFailed creates a REMOVE_FAILED error
func RemoveFailed(worktree string, cause error) *AWTError {
	return New(
		ExitRemoveFailed,
		fmt.Sprintf("Failed to remove worktree: %s", worktree),
		"Check if the worktree is locked by another process or has uncommitted changes.",
		cause,
	)
}

// SyncConflicts creates a SYNC_CONFLICTS error
func SyncConflicts(branch string) *AWTError {
	return New(
		ExitSyncConflicts,
		fmt.Sprintf("Conflicts detected while syncing branch: %s", branch),
		"Resolve conflicts manually in the worktree, then run 'git rebase --continue' or 'git merge --continue'.",
		nil,
	)
}

// PushRejected creates a PUSH_REJECTED error
func PushRejected(branch string, cause error) *AWTError {
	return New(
		ExitPushRejected,
		fmt.Sprintf("Push rejected for branch: %s", branch),
		"The remote may have been updated. Run 'awt task sync' to update your branch, then try again.",
		cause,
	)
}

// LockTimeout creates a LOCK_TIMEOUT error
func LockTimeout(lockName string) *AWTError {
	return New(
		ExitLockTimeout,
		fmt.Sprintf("Timeout waiting for lock: %s", lockName),
		"Another AWT operation may be in progress. Wait for it to complete or check for stale locks.",
		nil,
	)
}

// LockHeld creates a LOCK_HELD error
func LockHeld(lockName string) *AWTError {
	return New(
		ExitLockHeld,
		fmt.Sprintf("Lock is held: %s", lockName),
		"Another AWT operation is currently using this lock.",
		nil,
	)
}

// ToolMissing creates a TOOL_MISSING error
func ToolMissing(tool string) *AWTError {
	return New(
		ExitToolMissing,
		fmt.Sprintf("Required tool not found: %s", tool),
		fmt.Sprintf("Please install %s and make sure it's in your PATH.", tool),
		nil,
	)
}

// InvalidTaskID creates an INVALID_TASK_ID error
func InvalidTaskID(taskID string) *AWTError {
	return New(
		ExitInvalidTaskID,
		fmt.Sprintf("Invalid or unknown task ID: %s", taskID),
		"Use 'awt list' to see available tasks. Custom task IDs must not contain special characters (/, \\, :, *, ?, \", <, >, |, etc.) and must be 1-255 characters long.",
		nil,
	)
}

// CaseOnlyCollision creates a CASE_ONLY_COLLISION error
func CaseOnlyCollision(branch1, branch2 string) *AWTError {
	return New(
		ExitCaseOnlyCollision,
		fmt.Sprintf("Case-only collision detected: %s vs %s", branch1, branch2),
		"macOS filesystems are case-insensitive. Use branch names that differ by more than just case.",
		nil,
	)
}
