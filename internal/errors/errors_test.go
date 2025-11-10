package errors

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestAWTErrorBasic(t *testing.T) {
	err := New(ExitRepoNotFound, "Repository not found", "Check your path", nil)

	if err.Code != ExitRepoNotFound {
		t.Errorf("expected code %d, got %d", ExitRepoNotFound, err.Code)
	}
	if err.Message != "Repository not found" {
		t.Errorf("unexpected message: %s", err.Message)
	}
	if err.Hint != "Check your path" {
		t.Errorf("unexpected hint: %s", err.Hint)
	}

	// Test Error() method
	errStr := err.Error()
	if !strings.Contains(errStr, "Repository not found") {
		t.Error("Error() should contain the message")
	}
	if !strings.Contains(errStr, "Check your path") {
		t.Error("Error() should contain the hint")
	}
}

func TestAWTErrorWithCause(t *testing.T) {
	cause := errors.New("underlying error")
	err := New(ExitRemoveFailed, "Failed to remove", "Try again", cause)

	// Test Unwrap
	if err.Unwrap() != cause {
		t.Error("Unwrap() should return the cause")
	}
}

func TestAWTErrorJSON(t *testing.T) {
	err := New(ExitBranchExists, "Branch exists", "Use a different name", nil)

	jsonStr := err.ToJSON()

	// Parse JSON
	var je JSONError
	if err := json.Unmarshal([]byte(jsonStr), &je); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if je.Error != "Branch exists" {
		t.Errorf("unexpected error in JSON: %s", je.Error)
	}
	if je.Code != ExitBranchExists {
		t.Errorf("unexpected code in JSON: %d", je.Code)
	}
	if je.Hint != "Use a different name" {
		t.Errorf("unexpected hint in JSON: %s", je.Hint)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      *AWTError
		wantCode ExitCode
	}{
		{"RepoNotFound", RepoNotFound("/tmp/repo"), ExitRepoNotFound},
		{"GitTooOld", GitTooOld("2.30", "2.33"), ExitGitTooOld},
		{"BranchExists", BranchExists("feature"), ExitBranchExists},
		{"BranchCheckedOutElsewhere", BranchCheckedOutElsewhere("feature", "/tmp/wt"), ExitBranchCheckedOutElsewhere},
		{"WorktreeExists", WorktreeExists("/tmp/wt"), ExitWorktreeExists},
		{"DetachFailed", DetachFailed("/tmp/wt", nil), ExitDetachFailed},
		{"RemoveFailed", RemoveFailed("/tmp/wt", nil), ExitRemoveFailed},
		{"SyncConflicts", SyncConflicts("feature"), ExitSyncConflicts},
		{"PushRejected", PushRejected("feature", nil), ExitPushRejected},
		{"LockTimeout", LockTimeout("global"), ExitLockTimeout},
		{"LockHeld", LockHeld("global"), ExitLockHeld},
		{"ToolMissing", ToolMissing("gh"), ExitToolMissing},
		{"InvalidTaskID", InvalidTaskID("bad-id"), ExitInvalidTaskID},
		{"CaseOnlyCollision", CaseOnlyCollision("Feature", "feature"), ExitCaseOnlyCollision},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.wantCode {
				t.Errorf("expected code %d, got %d", tt.wantCode, tt.err.Code)
			}
			if tt.err.Message == "" {
				t.Error("message should not be empty")
			}
			if tt.err.Hint == "" {
				t.Error("hint should not be empty")
			}
		})
	}
}

func TestExitCodes(t *testing.T) {
	// Verify exit codes are unique
	codes := map[ExitCode]string{
		ExitSuccess:                   "ExitSuccess",
		ExitRepoNotFound:              "ExitRepoNotFound",
		ExitGitTooOld:                 "ExitGitTooOld",
		ExitBranchExists:              "ExitBranchExists",
		ExitBranchCheckedOutElsewhere: "ExitBranchCheckedOutElsewhere",
		ExitWorktreeExists:            "ExitWorktreeExists",
		ExitDetachFailed:              "ExitDetachFailed",
		ExitRemoveFailed:              "ExitRemoveFailed",
		ExitSyncConflicts:             "ExitSyncConflicts",
		ExitPushRejected:              "ExitPushRejected",
		ExitLockTimeout:               "ExitLockTimeout",
		ExitLockHeld:                  "ExitLockHeld",
		ExitToolMissing:               "ExitToolMissing",
		ExitInvalidTaskID:             "ExitInvalidTaskID",
		ExitCaseOnlyCollision:         "ExitCaseOnlyCollision",
	}

	seen := make(map[ExitCode]bool)
	for code, name := range codes {
		if seen[code] {
			t.Errorf("duplicate exit code: %d (%s)", code, name)
		}
		seen[code] = true
	}
}

func TestErrorWithoutHint(t *testing.T) {
	err := New(ExitRepoNotFound, "Repository not found", "", nil)

	errStr := err.Error()
	if strings.Contains(errStr, "Hint:") {
		t.Error("Error() should not contain 'Hint:' when hint is empty")
	}
	if !strings.Contains(errStr, "Repository not found") {
		t.Error("Error() should contain the message")
	}
}
