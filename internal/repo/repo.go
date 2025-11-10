package repo

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// MinGitVersion is the minimum required Git version
	MinGitVersion = "2.33"
)

// Repo represents a discovered Git repository
type Repo struct {
	// WorkTreeRoot is the root of the worktree (output of git rev-parse --show-toplevel)
	WorkTreeRoot string
	// GitCommonDir is the common Git directory (output of git rev-parse --git-common-dir)
	GitCommonDir string
}

// DiscoverRepo discovers the Git repository from the current directory
// or from the path specified by the --repo flag
func DiscoverRepo(repoPath string) (*Repo, error) {
	// Validate Git version first
	if err := validateGitVersion(); err != nil {
		return nil, err
	}

	// If repoPath is specified, use it as working directory for git commands
	// Otherwise, git will use the current directory

	// Get worktree root
	wtRoot, err := getWorkTreeRoot(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find worktree root: %w", err)
	}

	// Get git common dir
	gitCommon, err := getGitCommonDir(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find git common dir: %w", err)
	}

	return &Repo{
		WorkTreeRoot: wtRoot,
		GitCommonDir: gitCommon,
	}, nil
}

// getWorkTreeRoot executes git rev-parse --show-toplevel
func getWorkTreeRoot(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if repoPath != "" {
		cmd.Dir = repoPath
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository or unable to find repository root: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getGitCommonDir executes git rev-parse --git-common-dir
func getGitCommonDir(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	if repoPath != "" {
		cmd.Dir = repoPath
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("unable to find git common directory: %w", err)
	}

	path := strings.TrimSpace(string(output))

	// If the path is relative, we need to make it absolute
	// git rev-parse --git-common-dir can return relative paths like ".git"
	if !filepath.IsAbs(path) {
		// Get the absolute path by combining with worktree root
		wtRoot, err := getWorkTreeRoot(repoPath)
		if err != nil {
			return "", err
		}
		path = filepath.Join(wtRoot, path)
	}

	// Resolve any symlinks and clean the path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("unable to resolve git common directory path: %w", err)
	}

	return absPath, nil
}

// validateGitVersion checks if Git version is >= 2.33
func validateGitVersion() error {
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git not found: %w", err)
	}

	version := strings.TrimSpace(string(output))
	// Parse version string: "git version 2.33.0" -> "2.33.0"
	parts := strings.Fields(version)
	if len(parts) < 3 {
		return fmt.Errorf("unable to parse git version: %s", version)
	}

	versionNum := parts[2]

	// Simple version comparison - check major.minor
	if !isVersionAtLeast(versionNum, MinGitVersion) {
		return fmt.Errorf("git version %s is too old, minimum required: %s", versionNum, MinGitVersion)
	}

	return nil
}

// isVersionAtLeast checks if version is >= minVersion
// Simplified version comparison for major.minor
func isVersionAtLeast(version, minVersion string) bool {
	// Split on '.' and compare major.minor
	vParts := strings.Split(version, ".")
	minParts := strings.Split(minVersion, ".")

	if len(vParts) < 2 || len(minParts) < 2 {
		return false
	}

	// Compare major version
	if vParts[0] > minParts[0] {
		return true
	}
	if vParts[0] < minParts[0] {
		return false
	}

	// Major versions equal, compare minor
	return vParts[1] >= minParts[1]
}
