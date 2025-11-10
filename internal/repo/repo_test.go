package repo

import (
	"testing"
)

func TestDiscoverRepo(t *testing.T) {
	repo, err := DiscoverRepo("")
	if err != nil {
		t.Fatalf("DiscoverRepo failed: %v", err)
	}

	if repo.WorkTreeRoot == "" {
		t.Error("WorkTreeRoot is empty")
	}

	if repo.GitCommonDir == "" {
		t.Error("GitCommonDir is empty")
	}

	t.Logf("WorkTreeRoot: %s", repo.WorkTreeRoot)
	t.Logf("GitCommonDir: %s", repo.GitCommonDir)
}

func TestValidateGitVersion(t *testing.T) {
	err := validateGitVersion()
	if err != nil {
		t.Fatalf("validateGitVersion failed: %v", err)
	}
}

func TestIsVersionAtLeast(t *testing.T) {
	tests := []struct {
		version    string
		minVersion string
		expected   bool
	}{
		{"2.33.0", "2.33", true},
		{"2.34.0", "2.33", true},
		{"2.32.0", "2.33", false},
		{"3.0.0", "2.33", true},
		{"1.9.0", "2.33", false},
		{"2.33.1", "2.33", true},
	}

	for _, tt := range tests {
		result := isVersionAtLeast(tt.version, tt.minVersion)
		if result != tt.expected {
			t.Errorf("isVersionAtLeast(%s, %s) = %v, expected %v",
				tt.version, tt.minVersion, result, tt.expected)
		}
	}
}
