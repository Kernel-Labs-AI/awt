package safety

import (
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
