package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/kernel-labs-ai/awt/internal/errors"
	"github.com/kernel-labs-ai/awt/internal/git"
	"github.com/kernel-labs-ai/awt/internal/lock"
	"github.com/kernel-labs-ai/awt/internal/repo"
	"github.com/kernel-labs-ai/awt/internal/task"
	"github.com/spf13/cobra"
)

// CheckoutOptions contains options for the checkout command
type CheckoutOptions struct {
	RepoPath   string
	TaskID     string
	Branch     string
	Path       string
	Submodules bool
	OutputJSON bool
}

// CheckoutResult represents the output of the checkout command
type CheckoutResult struct {
	TaskID       string `json:"task_id"`
	Branch       string `json:"branch"`
	WorktreePath string `json:"worktree_path"`
}

// NewTaskCheckoutCmd creates the task checkout command
func NewTaskCheckoutCmd() *cobra.Command {
	opts := &CheckoutOptions{}

	cmd := &cobra.Command{
		Use:   "checkout [task-id]",
		Short: "Checkout a task for review or modification",
		Long: `Checkout a task by creating a new worktree.

The task can be specified by:
  1. Providing the task ID as an argument
  2. Using --branch flag

This creates a new worktree for human validation or modification.

Example:
  awt task checkout 20250110-120000-abc123
  awt task checkout --branch=awt/claude/20250110-120000-abc123
  awt task checkout 20250110-120000-abc123 --path=./review/task
  awt task checkout 20250110-120000-abc123 --submodules`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.TaskID = args[0]
			}
			return runTaskCheckout(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "branch name")
	cmd.Flags().StringVar(&opts.Path, "path", "", "worktree path (default: ./wt/<id>)")
	cmd.Flags().BoolVar(&opts.Submodules, "submodules", false, "initialize and update submodules")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	return cmd
}

func runTaskCheckout(opts *CheckoutOptions) error {
	// Discover repository
	r, err := repo.DiscoverRepo(opts.RepoPath)
	if err != nil {
		return errors.RepoNotFound(opts.RepoPath)
	}

	store := task.NewTaskStore(r.GitCommonDir)

	// Determine task ID
	taskID := opts.TaskID

	if taskID == "" && opts.Branch != "" {
		// Extract task ID from branch name
		taskID = extractTaskIDFromBranch(opts.Branch)
		if taskID == "" {
			return fmt.Errorf("could not extract task ID from branch: %s", opts.Branch)
		}
	}

	if taskID == "" {
		return fmt.Errorf("task ID is required\nProvide task ID as argument or use --branch flag")
	}

	// Load task
	t, err := store.Load(taskID)
	if err != nil {
		return errors.InvalidTaskID(taskID)
	}

	// Determine worktree path
	worktreePath := opts.Path
	if worktreePath == "" {
		worktreePath = filepath.Join(r.WorkTreeRoot, "wt", taskID)
	} else {
		// Make path absolute
		worktreePath = filepath.Join(r.WorkTreeRoot, worktreePath)
	}

	// Acquire global lock for worktree creation
	lm := lock.NewLockManager(r.GitCommonDir)
	ctx := context.Background()
	globalLock, err := lm.AcquireGlobal(ctx)
	if err != nil {
		return errors.LockTimeout("global")
	}
	defer globalLock.Release()

	// Create Git wrapper
	g := git.New(r.WorkTreeRoot, false)

	// Check if worktree already exists at path
	worktrees, err := g.WorktreeList()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	for _, wt := range worktrees {
		wtAbs, _ := filepath.Abs(wt.Path)
		pathAbs, _ := filepath.Abs(worktreePath)
		if wtAbs == pathAbs {
			return errors.WorktreeExists(worktreePath)
		}
	}

	// Create worktree
	branchName := t.Branch
	if len(branchName) > 11 && branchName[:11] == "refs/heads/" {
		branchName = branchName[11:]
	}

	result, err := g.WorktreeAddExisting(worktreePath, branchName)
	if err != nil || result.ExitCode != 0 {
		return fmt.Errorf("failed to create worktree: %s", result.Stderr)
	}

	// Initialize/update submodules if requested
	if opts.Submodules {
		wtGit := git.New(worktreePath, false)
		subResult, err := wtGit.SubmoduleUpdate()
		if err != nil || subResult.ExitCode != 0 {
			return fmt.Errorf("failed to update submodules: %s", subResult.Stderr)
		}
	}

	// Output result
	if opts.OutputJSON {
		output := CheckoutResult{
			TaskID:       taskID,
			Branch:       t.Branch,
			WorktreePath: worktreePath,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Checked out task successfully!\n")
		fmt.Printf("  Task: %s\n", taskID)
		fmt.Printf("  Branch: %s\n", t.Branch)
		fmt.Printf("  Worktree: %s\n", worktreePath)
		if opts.Submodules {
			fmt.Printf("  Submodules: initialized\n")
		}
	}

	return nil
}
