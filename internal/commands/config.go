package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kernel-labs-ai/awt/internal/config"
	"github.com/kernel-labs-ai/awt/internal/errors"
	"github.com/kernel-labs-ai/awt/internal/repo"
	"github.com/spf13/cobra"
)

// ConfigOptions contains options for the config command
type ConfigOptions struct {
	RepoPath   string
	Scope      string
	OutputJSON bool
}

// NewConfigCmd creates the config command
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage AWT configuration",
		Long: `Manage AWT configuration settings.

Configuration can be set at three levels:
  - system: /etc/awt/config.json (affects all users)
  - user: ~/.config/awt/config.json (affects current user)
  - repo: <repo>/.git/awt/config.json (affects current repository)

Environment variables have the highest precedence and override all file-based config.

Example:
  awt config list
  awt config get default_agent
  awt config set default_agent claude --scope=user
  awt config unset auto_push --scope=repo`,
	}

	cmd.AddCommand(NewConfigListCmd())
	cmd.AddCommand(NewConfigGetCmd())
	cmd.AddCommand(NewConfigSetCmd())
	cmd.AddCommand(NewConfigUnsetCmd())
	cmd.AddCommand(NewConfigPathCmd())

	return cmd
}

// NewConfigListCmd creates the config list command
func NewConfigListCmd() *cobra.Command {
	opts := &ConfigOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration settings",
		Long: `List all configuration settings with their current values.

Shows the effective configuration after merging all sources.

Example:
  awt config list
  awt config list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	return cmd
}

// NewConfigGetCmd creates the config get command
func NewConfigGetCmd() *cobra.Command {
	opts := &ConfigOptions{}

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get the value of a configuration setting.

Available keys:
  - default_agent: Default agent name
  - branch_prefix: Prefix for AWT branches (default: awt)
  - worktree_dir: Default directory for worktrees (default: ./wt)
  - rebase_default: Use rebase instead of merge for sync (default: true)
  - auto_push: Automatically push on handoff (default: true)
  - auto_pr: Automatically create PR on handoff (default: true)
  - remote_name: Default remote name (default: origin)
  - lock_timeout: Lock acquisition timeout in seconds (default: 30)
  - verbose_git: Enable verbose git output (default: false)

Example:
  awt config get default_agent
  awt config get auto_push`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			return runConfigGet(opts, key)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")

	return cmd
}

// NewConfigSetCmd creates the config set command
func NewConfigSetCmd() *cobra.Command {
	opts := &ConfigOptions{
		Scope: "user", // default scope
	}

	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a configuration value at the specified scope.

The scope determines where the setting is stored:
  - system: /etc/awt/config.json
  - user: ~/.config/awt/config.json (default)
  - repo: <repo>/.git/awt/config.json

Example:
  awt config set default_agent claude
  awt config set auto_push false --scope=repo
  awt config set lock_timeout 60 --scope=user`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			return runConfigSet(opts, key, value)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Scope, "scope", "user", "configuration scope (system, user, or repo)")

	return cmd
}

// NewConfigUnsetCmd creates the config unset command
func NewConfigUnsetCmd() *cobra.Command {
	opts := &ConfigOptions{
		Scope: "user", // default scope
	}

	cmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Unset a configuration value",
		Long: `Unset a configuration value at the specified scope.

This removes the setting from the configuration file at the specified scope.
The effective value will fall back to lower-precedence sources.

Example:
  awt config unset default_agent
  awt config unset auto_push --scope=repo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			return runConfigUnset(opts, key)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Scope, "scope", "user", "configuration scope (system, user, or repo)")

	return cmd
}

// NewConfigPathCmd creates the config path command
func NewConfigPathCmd() *cobra.Command {
	opts := &ConfigOptions{
		Scope: "user", // default scope
	}

	cmd := &cobra.Command{
		Use:   "path",
		Short: "Show configuration file path",
		Long: `Show the path to the configuration file for the specified scope.

Example:
  awt config path
  awt config path --scope=system
  awt config path --scope=repo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigPath(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Scope, "scope", "user", "configuration scope (system, user, or repo)")

	return cmd
}

func runConfigList(opts *ConfigOptions) error {
	// Discover repository if available
	var gitCommonDir string
	if r, err := repo.DiscoverRepo(opts.RepoPath); err == nil {
		gitCommonDir = r.GitCommonDir
	} else {
		// Not in a repo - use empty string for loader
		gitCommonDir = ""
	}

	loader := config.NewConfigLoader(gitCommonDir)
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if opts.OutputJSON {
		data, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println("Configuration settings:")
		fmt.Printf("  default_agent:   %s\n", cfg.DefaultAgent)
		fmt.Printf("  branch_prefix:   %s\n", cfg.BranchPrefix)
		fmt.Printf("  worktree_dir:    %s\n", cfg.WorktreeDir)
		fmt.Printf("  rebase_default:  %t\n", cfg.RebaseDefault)
		fmt.Printf("  auto_push:       %t\n", cfg.AutoPush)
		fmt.Printf("  auto_pr:         %t\n", cfg.AutoPR)
		fmt.Printf("  remote_name:     %s\n", cfg.RemoteName)
		fmt.Printf("  lock_timeout:    %d\n", cfg.LockTimeout)
		fmt.Printf("  verbose_git:     %t\n", cfg.VerboseGit)
	}

	return nil
}

func runConfigGet(opts *ConfigOptions, key string) error {
	// Discover repository if available
	var gitCommonDir string
	if r, err := repo.DiscoverRepo(opts.RepoPath); err == nil {
		gitCommonDir = r.GitCommonDir
	} else {
		gitCommonDir = ""
	}

	loader := config.NewConfigLoader(gitCommonDir)
	cfg, err := loader.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	value, err := getConfigValue(cfg, key)
	if err != nil {
		return err
	}

	fmt.Println(value)
	return nil
}

func runConfigSet(opts *ConfigOptions, key, value string) error {
	// For repo scope, we need a repository
	if opts.Scope == "repo" {
		r, err := repo.DiscoverRepo(opts.RepoPath)
		if err != nil {
			return errors.RepoNotFound(opts.RepoPath)
		}
		opts.RepoPath = r.GitCommonDir
	}

	loader := config.NewConfigLoader(opts.RepoPath)

	// Load existing config from the specific scope
	var cfg *config.Config
	scopePath, _ := loader.GetConfigPath(opts.Scope)
	if data, err := os.ReadFile(scopePath); err == nil {
		cfg = config.Default()
		json.Unmarshal(data, cfg)
	} else {
		cfg = config.Default()
	}

	// Set the value
	if err := setConfigValue(cfg, key, value); err != nil {
		return err
	}

	// Save config
	if err := loader.Save(cfg, opts.Scope); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Set %s = %s (scope: %s)\n", key, value, opts.Scope)
	return nil
}

func runConfigUnset(opts *ConfigOptions, key string) error {
	// For repo scope, we need a repository
	if opts.Scope == "repo" {
		r, err := repo.DiscoverRepo(opts.RepoPath)
		if err != nil {
			return errors.RepoNotFound(opts.RepoPath)
		}
		opts.RepoPath = r.GitCommonDir
	}

	loader := config.NewConfigLoader(opts.RepoPath)

	// Load existing config from the specific scope
	scopePath, _ := loader.GetConfigPath(opts.Scope)
	data, err := os.ReadFile(scopePath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No configuration to unset at scope: %s\n", opts.Scope)
			return nil
		}
		return fmt.Errorf("failed to read config: %w", err)
	}

	cfg := config.Default()
	json.Unmarshal(data, cfg)

	// Unset the value (set to zero value)
	if err := unsetConfigValue(cfg, key); err != nil {
		return err
	}

	// Save config
	if err := loader.Save(cfg, opts.Scope); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Unset %s (scope: %s)\n", key, opts.Scope)
	return nil
}

func runConfigPath(opts *ConfigOptions) error {
	// For repo scope, we need a repository
	var gitCommonDir string
	if opts.Scope == "repo" {
		r, err := repo.DiscoverRepo(opts.RepoPath)
		if err != nil {
			return errors.RepoNotFound(opts.RepoPath)
		}
		gitCommonDir = r.GitCommonDir
	}

	loader := config.NewConfigLoader(gitCommonDir)
	path, err := loader.GetConfigPath(opts.Scope)
	if err != nil {
		return err
	}

	fmt.Println(path)
	return nil
}

func getConfigValue(cfg *config.Config, key string) (string, error) {
	key = strings.ReplaceAll(key, "-", "_")

	switch key {
	case "default_agent":
		return cfg.DefaultAgent, nil
	case "branch_prefix":
		return cfg.BranchPrefix, nil
	case "worktree_dir":
		return cfg.WorktreeDir, nil
	case "rebase_default":
		return strconv.FormatBool(cfg.RebaseDefault), nil
	case "auto_push":
		return strconv.FormatBool(cfg.AutoPush), nil
	case "auto_pr":
		return strconv.FormatBool(cfg.AutoPR), nil
	case "remote_name":
		return cfg.RemoteName, nil
	case "lock_timeout":
		return strconv.Itoa(cfg.LockTimeout), nil
	case "verbose_git":
		return strconv.FormatBool(cfg.VerboseGit), nil
	default:
		return "", fmt.Errorf("unknown configuration key: %s", key)
	}
}

func setConfigValue(cfg *config.Config, key, value string) error {
	key = strings.ReplaceAll(key, "-", "_")

	switch key {
	case "default_agent":
		cfg.DefaultAgent = value
	case "branch_prefix":
		cfg.BranchPrefix = value
	case "worktree_dir":
		cfg.WorktreeDir = value
	case "rebase_default":
		cfg.RebaseDefault = parseBool(value)
	case "auto_push":
		cfg.AutoPush = parseBool(value)
	case "auto_pr":
		cfg.AutoPR = parseBool(value)
	case "remote_name":
		cfg.RemoteName = value
	case "lock_timeout":
		timeout, err := strconv.Atoi(value)
		if err != nil || timeout <= 0 {
			return fmt.Errorf("lock_timeout must be a positive integer")
		}
		cfg.LockTimeout = timeout
	case "verbose_git":
		cfg.VerboseGit = parseBool(value)
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}

func unsetConfigValue(cfg *config.Config, key string) error {
	key = strings.ReplaceAll(key, "-", "_")

	// Set to default values
	defaults := config.Default()

	switch key {
	case "default_agent":
		cfg.DefaultAgent = defaults.DefaultAgent
	case "branch_prefix":
		cfg.BranchPrefix = defaults.BranchPrefix
	case "worktree_dir":
		cfg.WorktreeDir = defaults.WorktreeDir
	case "rebase_default":
		cfg.RebaseDefault = defaults.RebaseDefault
	case "auto_push":
		cfg.AutoPush = defaults.AutoPush
	case "auto_pr":
		cfg.AutoPR = defaults.AutoPR
	case "remote_name":
		cfg.RemoteName = defaults.RemoteName
	case "lock_timeout":
		cfg.LockTimeout = defaults.LockTimeout
	case "verbose_git":
		cfg.VerboseGit = defaults.VerboseGit
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return nil
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "1", "true", "yes", "on", "enabled":
		return true
	default:
		return false
	}
}
