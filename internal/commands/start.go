package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/decibelvc/awt/internal/errors"
	"github.com/decibelvc/awt/internal/git"
	"github.com/decibelvc/awt/internal/idgen"
	"github.com/decibelvc/awt/internal/lock"
	"github.com/decibelvc/awt/internal/repo"
	"github.com/decibelvc/awt/internal/task"
	"github.com/spf13/cobra"
)

// StartOptions contains options for the start command
type StartOptions struct {
	RepoPath      string
	Agent         string
	Title         string
	Base          string
	ID            string
	NoFetch       bool
	BranchPrefix  string
	WorktreeDir   string
	OutputJSON    bool
}

// StartResult represents the output of the start command
type StartResult struct {
	ID           string `json:"id"`
	Branch       string `json:"branch"`
	WorktreePath string `json:"worktree_path"`
}

// NewTaskCmd creates the task command group
func NewTaskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage AWT tasks",
		Long:  "Commands for managing AWT tasks (start, status, commit, sync, handoff, etc.)",
	}

	// Add subcommands
	cmd.AddCommand(NewTaskStartCmd())
	cmd.AddCommand(NewTaskStatusCmd())
	cmd.AddCommand(NewTaskExecCmd())
	cmd.AddCommand(NewTaskCommitCmd())
	cmd.AddCommand(NewTaskSyncCmd())
	cmd.AddCommand(NewTaskHandoffCmd())
	cmd.AddCommand(NewTaskCheckoutCmd())
	cmd.AddCommand(NewTaskAdoptCmd())
	cmd.AddCommand(NewTaskUnlockCmd())

	return cmd
}

// NewTaskStartCmd creates the task start command
func NewTaskStartCmd() *cobra.Command {
	opts := &StartOptions{
		BranchPrefix: "awt",
		WorktreeDir:  ".awt/wt",
	}

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a new task",
		Long: `Start a new task by creating a branch and worktree.

This command:
  1. Generates a unique task ID (or uses --id if provided)
  2. Creates a branch: <prefix>/<agent>/<id>
  3. Creates a worktree at: <worktree-dir>/<id>
  4. Saves task metadata
  5. Outputs the task details

Example:
  awt task start --agent=claude --title="Add user authentication"
  awt task start --agent=claude --title="Fix bug" --base=develop --no-fetch`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskStart(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Agent, "agent", "", "agent name (required)")
	cmd.Flags().StringVar(&opts.Title, "title", "", "task title (required)")
	cmd.Flags().StringVar(&opts.Base, "base", "origin/main", "base branch")
	cmd.Flags().StringVar(&opts.ID, "id", "", "task ID (auto-generated if not provided)")
	cmd.Flags().BoolVar(&opts.NoFetch, "no-fetch", false, "skip git fetch")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	cmd.MarkFlagRequired("agent")
	cmd.MarkFlagRequired("title")

	return cmd
}

func runTaskStart(opts *StartOptions) error {
	// Discover repository
	r, err := repo.DiscoverRepo(opts.RepoPath)
	if err != nil {
		return errors.RepoNotFound(opts.RepoPath)
	}

	// Create Git wrapper
	g := git.New(r.WorkTreeRoot, false)

	// Acquire global lock for worktree creation
	lm := lock.NewLockManager(r.GitCommonDir)
	ctx := context.Background()
	globalLock, err := lm.AcquireGlobal(ctx)
	if err != nil {
		return errors.LockTimeout("global")
	}
	defer globalLock.Release()

	// Generate or validate task ID
	taskID := opts.ID
	if taskID == "" {
		taskID, err = idgen.GenerateTaskID()
		if err != nil {
			return fmt.Errorf("failed to generate task ID: %w", err)
		}
	} else if !idgen.ValidateTaskID(taskID) {
		return errors.InvalidTaskID(taskID)
	}

	// Generate branch name
	branchName := idgen.GenerateBranchName(opts.BranchPrefix, opts.Agent, taskID)

	// Generate worktree path
	worktreePath := filepath.Join(r.WorkTreeRoot, opts.WorktreeDir, taskID)

	// Fetch unless --no-fetch
	if !opts.NoFetch {
		result, err := g.Fetch("", "")
		if err != nil || result.ExitCode != 0 {
			// Fetch failed, but continue anyway (might be offline)
			// TODO: log warning
		}
	}

	// Check if branch already exists
	exists, err := g.BranchExists(branchName)
	if err != nil {
		return fmt.Errorf("failed to check branch existence: %w", err)
	}
	if exists {
		return errors.BranchExists(branchName)
	}

	// Check if branch is checked out elsewhere
	checkedOut, path, err := g.IsBranchCheckedOut(branchName)
	if err != nil {
		return fmt.Errorf("failed to check branch checkout status: %w", err)
	}
	if checkedOut {
		return errors.BranchCheckedOutElsewhere(branchName, path)
	}

	// Create worktree
	result, err := g.WorktreeAdd(worktreePath, branchName, opts.Base)
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to create worktree: %s", result.Stderr)
	}

	// Create task metadata
	t := &task.Task{
		ID:           taskID,
		Agent:        opts.Agent,
		Title:        opts.Title,
		Branch:       branchName,
		Base:         opts.Base,
		CreatedAt:    time.Now(),
		State:        task.StateActive,
		WorktreePath: worktreePath,
	}

	// Save task
	store := task.NewTaskStore(r.GitCommonDir)
	if err := store.Save(t); err != nil {
		// Try to clean up worktree
		g.WorktreeRemove(worktreePath, true)
		return fmt.Errorf("failed to save task: %w", err)
	}

	// Output result
	if opts.OutputJSON {
		output := StartResult{
			ID:           taskID,
			Branch:       branchName,
			WorktreePath: worktreePath,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Task started successfully!\n")
		fmt.Printf("  ID: %s\n", taskID)
		fmt.Printf("  Branch: %s\n", branchName)
		fmt.Printf("  Worktree: %s\n", worktreePath)
		fmt.Printf("  Agent: %s\n", opts.Agent)
		fmt.Printf("  Title: %s\n", opts.Title)
	}

	return nil
}
