package idgen

import (
	"strings"
	"testing"
)

func TestGenerateTaskID(t *testing.T) {
	// Generate multiple IDs to test uniqueness
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id, err := GenerateTaskID()
		if err != nil {
			t.Fatalf("GenerateTaskID() failed: %v", err)
		}

		// Check format
		if !ValidateTaskID(id) {
			t.Errorf("Generated ID %q does not pass validation", id)
		}

		// Check uniqueness (random part should differ)
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true

		// Check length (YYYYMMDD-HHMMSS-XXXXXX = 8+1+6+1+6 = 22)
		if len(id) != 22 {
			t.Errorf("ID length = %d, want 22 (got %q)", len(id), id)
		}

		// Check format parts
		parts := strings.Split(id, "-")
		if len(parts) != 3 {
			t.Errorf("ID has %d parts, want 3", len(parts))
		}

		// Date part should be 8 digits
		if len(parts[0]) != 8 {
			t.Errorf("Date part has %d chars, want 8", len(parts[0]))
		}

		// Time part should be 6 digits
		if len(parts[1]) != 6 {
			t.Errorf("Time part has %d chars, want 6", len(parts[1]))
		}

		// Random part should be 6 hex chars
		if len(parts[2]) != 6 {
			t.Errorf("Random part has %d chars, want 6", len(parts[2]))
		}
	}
}

func TestGenerateBranchName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		agent    string
		taskID   string
		expected string
	}{
		{
			name:     "simple",
			prefix:   "awt",
			agent:    "claude",
			taskID:   "20251110-120000-abc123",
			expected: "awt/claude/20251110-120000-abc123",
		},
		{
			name:     "agent with spaces",
			prefix:   "task",
			agent:    "GPT 4",
			taskID:   "20251110-120000-def456",
			expected: "task/gpt-4/20251110-120000-def456",
		},
		{
			name:     "agent with special chars",
			prefix:   "feature",
			agent:    "agent@123",
			taskID:   "20251110-120000-xyz789",
			expected: "feature/agent@123/20251110-120000-xyz789", // @ is allowed, only @{ is forbidden
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateBranchName(tt.prefix, tt.agent, tt.taskID)
			if result != tt.expected {
				t.Errorf("GenerateBranchName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already clean",
			input:    "claude",
			expected: "claude",
		},
		{
			name:     "with spaces",
			input:    "GPT 4 Turbo",
			expected: "gpt-4-turbo",
		},
		{
			name:     "with special chars",
			input:    "agent@v1.2",
			expected: "agent@v1.2", // @ is allowed, only @{ is forbidden
		},
		{
			name:     "with shell metacharacters",
			input:    "agent$test&pipe|",
			expected: "agenttestpipe",
		},
		{
			name:     "with git invalid chars",
			input:    "feature~test:branch*",
			expected: "featuretestbranch",
		},
		{
			name:     "with dots",
			input:    "agent..test",
			expected: "agenttest",
		},
		{
			name:     "leading/trailing hyphens",
			input:    "-agent-",
			expected: "agent",
		},
		{
			name:     "mixed case",
			input:    "ClaudeAI",
			expected: "claudeai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateTaskID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		// Auto-generated format should still work
		{
			name:  "auto-generated format",
			id:    "20251110-120000-abc123",
			valid: true,
		},
		{
			name:  "auto-generated with different hex",
			id:    "20231225-235959-fedcba",
			valid: true,
		},
		// Custom IDs
		{
			name:  "simple custom id",
			id:    "my-task",
			valid: true,
		},
		{
			name:  "custom id with numbers",
			id:    "task-123",
			valid: true,
		},
		{
			name:  "custom id with underscores",
			id:    "my_custom_task_id",
			valid: true,
		},
		{
			name:  "custom id all caps",
			id:    "URGENT-FIX",
			valid: true,
		},
		{
			name:  "custom id mixed case",
			id:    "FeatureBranch",
			valid: true,
		},
		{
			name:  "custom id with @ symbol",
			id:    "task@v1.2",
			valid: true,
		},
		{
			name:  "single character",
			id:    "a",
			valid: true,
		},
		// Invalid IDs
		{
			name:  "empty",
			id:    "",
			valid: false,
		},
		{
			name:  "with forward slash",
			id:    "my/task",
			valid: false,
		},
		{
			name:  "with backslash",
			id:    "my\\task",
			valid: false,
		},
		{
			name:  "with colon",
			id:    "task:123",
			valid: false,
		},
		{
			name:  "with asterisk",
			id:    "task*",
			valid: false,
		},
		{
			name:  "with question mark",
			id:    "task?",
			valid: false,
		},
		{
			name:  "with quotes",
			id:    "task\"name",
			valid: false,
		},
		{
			name:  "with less than",
			id:    "task<123",
			valid: false,
		},
		{
			name:  "with greater than",
			id:    "task>123",
			valid: false,
		},
		{
			name:  "with pipe",
			id:    "task|name",
			valid: false,
		},
		{
			name:  "with tilde",
			id:    "task~123",
			valid: false,
		},
		{
			name:  "with caret",
			id:    "task^123",
			valid: false,
		},
		{
			name:  "with brackets",
			id:    "task[123",
			valid: false,
		},
		{
			name:  "with double dot",
			id:    "task..name",
			valid: false,
		},
		{
			name:  "with dollar sign",
			id:    "task$name",
			valid: false,
		},
		{
			name:  "with backtick",
			id:    "task`name",
			valid: false,
		},
		{
			name:  "with ampersand",
			id:    "task&name",
			valid: false,
		},
		{
			name:  "with semicolon",
			id:    "task;name",
			valid: false,
		},
		{
			name:  "with parentheses",
			id:    "task(123)",
			valid: false,
		},
		{
			name:  "with braces",
			id:    "task{123}",
			valid: false,
		},
		{
			name:  "with @{ sequence",
			id:    "task@{123",
			valid: false,
		},
		{
			name:  "with double slash",
			id:    "task//name",
			valid: false,
		},
		{
			name:  "with newline",
			id:    "task\nname",
			valid: false,
		},
		{
			name:  "with tab",
			id:    "task\tname",
			valid: false,
		},
		{
			name:  "starting with dot",
			id:    ".hidden",
			valid: false,
		},
		{
			name:  "ending with dot",
			id:    "task.",
			valid: false,
		},
		{
			name:  "starting with space",
			id:    " task",
			valid: false,
		},
		{
			name:  "ending with space",
			id:    "task ",
			valid: false,
		},
		{
			name:  "too long (over 255 chars)",
			id:    strings.Repeat("a", 256),
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateTaskID(tt.id)
			if result != tt.valid {
				t.Errorf("ValidateTaskID(%q) = %v, want %v", tt.id, result, tt.valid)
			}
		})
	}
}
