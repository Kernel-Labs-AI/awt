package config

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config represents the AWT configuration
type Config struct {
	// DefaultAgent is the default agent name to use
	DefaultAgent string `json:"default_agent,omitempty"`

	// BranchPrefix is the prefix for AWT branches (default: awt)
	BranchPrefix string `json:"branch_prefix,omitempty"`

	// WorktreeDir is the default directory for worktrees (default: ./wt)
	// This is relative to the repository root and used when GlobalWorktreeDir is empty
	WorktreeDir string `json:"worktree_dir,omitempty"`

	// GlobalWorktreeDir is the global directory for worktrees (e.g., ~/.awt)
	// When set, worktrees are stored at <GlobalWorktreeDir>/<project-hash>/<task-id>
	// This prevents agents from seeing each other's worktrees in the same project
	GlobalWorktreeDir string `json:"global_worktree_dir,omitempty"`

	// RebaseDefault determines whether to use rebase or merge for sync (default: true)
	RebaseDefault bool `json:"rebase_default,omitempty"`

	// AutoPush determines whether to auto-push on handoff (default: true)
	AutoPush bool `json:"auto_push,omitempty"`

	// AutoPR determines whether to auto-create PR on handoff (default: true)
	AutoPR bool `json:"auto_pr,omitempty"`

	// RemoveName is the default remote name (default: origin)
	RemoteName string `json:"remote_name,omitempty"`

	// LockTimeout is the lock acquisition timeout in seconds (default: 30)
	LockTimeout int `json:"lock_timeout,omitempty"`

	// VerboseGit enables verbose git command output (default: false)
	VerboseGit bool `json:"verbose_git,omitempty"`
}

// Default returns a config with default values
func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		DefaultAgent:      "unknown",
		BranchPrefix:      "awt",
		WorktreeDir:       "./wt",
		GlobalWorktreeDir: filepath.Join(homeDir, ".awt"),
		RebaseDefault:     true,
		AutoPush:          true,
		AutoPR:            true,
		RemoteName:        "origin",
		LockTimeout:       30,
		VerboseGit:        false,
	}
}

// ConfigLoader loads configuration from multiple sources
type ConfigLoader struct {
	systemPath string
	userPath   string
	repoPath   string
}

// NewConfigLoader creates a new config loader
func NewConfigLoader(gitCommonDir string) *ConfigLoader {
	homeDir, _ := os.UserHomeDir()

	return &ConfigLoader{
		systemPath: "/etc/awt/config.json",
		userPath:   filepath.Join(homeDir, ".config", "awt", "config.json"),
		repoPath:   filepath.Join(gitCommonDir, "awt", "config.json"),
	}
}

// Load loads and merges configuration from all sources
// Precedence: env > repo > user > system > defaults
func (cl *ConfigLoader) Load() (*Config, error) {
	config := Default()

	// Layer 1: System config
	if err := cl.loadFromFile(cl.systemPath, config); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load system config: %w", err)
	}

	// Layer 2: User config
	if err := cl.loadFromFile(cl.userPath, config); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}

	// Layer 3: Repo config
	if err := cl.loadFromFile(cl.repoPath, config); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load repo config: %w", err)
	}

	// Layer 4: Environment variables (highest precedence)
	cl.loadFromEnv(config)

	return config, nil
}

// loadFromFile loads config from a JSON file, merging non-zero values
func (cl *ConfigLoader) loadFromFile(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var partial Config
	if err := json.Unmarshal(data, &partial); err != nil {
		return fmt.Errorf("invalid JSON in %s: %w", path, err)
	}

	// Merge non-zero values
	if partial.DefaultAgent != "" {
		config.DefaultAgent = partial.DefaultAgent
	}
	if partial.BranchPrefix != "" {
		config.BranchPrefix = partial.BranchPrefix
	}
	if partial.WorktreeDir != "" {
		config.WorktreeDir = partial.WorktreeDir
	}
	if partial.GlobalWorktreeDir != "" {
		config.GlobalWorktreeDir = partial.GlobalWorktreeDir
	}
	if partial.RemoteName != "" {
		config.RemoteName = partial.RemoteName
	}
	if partial.LockTimeout > 0 {
		config.LockTimeout = partial.LockTimeout
	}

	// For booleans, we need to check if they were explicitly set
	// This is tricky with JSON unmarshalling, so we use a workaround
	// by checking the raw JSON for the presence of these fields
	if strings.Contains(string(data), "\"rebase_default\"") {
		config.RebaseDefault = partial.RebaseDefault
	}
	if strings.Contains(string(data), "\"auto_push\"") {
		config.AutoPush = partial.AutoPush
	}
	if strings.Contains(string(data), "\"auto_pr\"") {
		config.AutoPR = partial.AutoPR
	}
	if strings.Contains(string(data), "\"verbose_git\"") {
		config.VerboseGit = partial.VerboseGit
	}

	return nil
}

// loadFromEnv loads config from environment variables
func (cl *ConfigLoader) loadFromEnv(config *Config) {
	if val := os.Getenv("AWT_DEFAULT_AGENT"); val != "" {
		config.DefaultAgent = val
	}
	if val := os.Getenv("AWT_BRANCH_PREFIX"); val != "" {
		config.BranchPrefix = val
	}
	if val := os.Getenv("AWT_WORKTREE_DIR"); val != "" {
		config.WorktreeDir = val
	}
	if val := os.Getenv("AWT_GLOBAL_WORKTREE_DIR"); val != "" {
		config.GlobalWorktreeDir = val
	}
	if val := os.Getenv("AWT_REMOTE_NAME"); val != "" {
		config.RemoteName = val
	}
	if val := os.Getenv("AWT_LOCK_TIMEOUT"); val != "" {
		if timeout, err := strconv.Atoi(val); err == nil && timeout > 0 {
			config.LockTimeout = timeout
		}
	}
	if val := os.Getenv("AWT_REBASE_DEFAULT"); val != "" {
		config.RebaseDefault = parseBool(val)
	}
	if val := os.Getenv("AWT_AUTO_PUSH"); val != "" {
		config.AutoPush = parseBool(val)
	}
	if val := os.Getenv("AWT_AUTO_PR"); val != "" {
		config.AutoPR = parseBool(val)
	}
	if val := os.Getenv("AWT_VERBOSE_GIT"); val != "" {
		config.VerboseGit = parseBool(val)
	}
}

// parseBool parses a boolean from a string (supports 1/0, true/false, yes/no)
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}

// Save saves configuration to a file
func (cl *ConfigLoader) Save(config *Config, scope string) error {
	var path string
	switch scope {
	case "system":
		path = cl.systemPath
	case "user":
		path = cl.userPath
	case "repo":
		path = cl.repoPath
	default:
		return fmt.Errorf("invalid scope: %s (must be system, user, or repo)", scope)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write atomically
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// GetConfigPath returns the path for a given scope
func (cl *ConfigLoader) GetConfigPath(scope string) (string, error) {
	switch scope {
	case "system":
		return cl.systemPath, nil
	case "user":
		return cl.userPath, nil
	case "repo":
		return cl.repoPath, nil
	default:
		return "", fmt.Errorf("invalid scope: %s (must be system, user, or repo)", scope)
	}
}

// GetWorktreePath returns the worktree path for a given task.
// If GlobalWorktreeDir is set, returns: <GlobalWorktreeDir>/<project-id>/<taskID>
// Otherwise returns: <repoRoot>/<WorktreeDir>/<taskID>
func (c *Config) GetWorktreePath(repoRoot, taskID string) string {
	if c.GlobalWorktreeDir != "" {
		projectID := GenerateProjectID(repoRoot)
		return filepath.Join(c.GlobalWorktreeDir, projectID, taskID)
	}
	return filepath.Join(repoRoot, c.WorktreeDir, taskID)
}

// GenerateProjectID creates a deterministic identifier for a repository.
// Uses a hash of the absolute path to ensure uniqueness while keeping names short.
func GenerateProjectID(repoRoot string) string {
	// Get absolute path and clean it
	absPath, err := filepath.Abs(repoRoot)
	if err != nil {
		absPath = repoRoot
	}
	absPath = filepath.Clean(absPath)

	// Create a short hash of the path for uniqueness
	// Use first 8 characters of hex-encoded hash
	hash := hashPath(absPath)

	// Also include the directory name for readability
	dirName := filepath.Base(absPath)
	// Sanitize directory name - remove special characters
	dirName = sanitizeForPath(dirName)

	// Limit directory name length
	if len(dirName) > 30 {
		dirName = dirName[:30]
	}

	return fmt.Sprintf("%s-%s", dirName, hash)
}

// hashPath creates a short hash of a path string
func hashPath(path string) string {
	// Simple hash using FNV-1a for speed and good distribution
	h := fnv.New32a()
	h.Write([]byte(path))
	return fmt.Sprintf("%08x", h.Sum32())
}

// sanitizeForPath removes or replaces characters not suitable for directory names
func sanitizeForPath(name string) string {
	var result strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result.WriteRune(c)
		} else if c == ' ' || c == '.' {
			result.WriteRune('-')
		}
	}
	return result.String()
}
