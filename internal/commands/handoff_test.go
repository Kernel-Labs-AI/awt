package commands

import (
	"testing"
)

func TestStripRemotePrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"origin/main", "main"},
		{"origin/develop", "develop"},
		{"upstream/feature-branch", "feature-branch"},
		{"main", "main"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripRemotePrefix(tt.input)
			if got != tt.want {
				t.Errorf("stripRemotePrefix(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCheckCommandExists(t *testing.T) {
	// "git" should exist on any system running these tests
	if !checkCommandExists("git") {
		t.Error("expected git to be found")
	}

	// A nonsense command should not exist
	if checkCommandExists("awt-nonexistent-command-12345") {
		t.Error("expected nonexistent command to not be found")
	}
}
