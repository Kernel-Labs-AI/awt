package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kernel-labs-ai/awt/internal/errors"
	"github.com/kernel-labs-ai/awt/internal/git"
	"github.com/kernel-labs-ai/awt/internal/lock"
	"github.com/kernel-labs-ai/awt/internal/repo"
	"github.com/kernel-labs-ai/awt/internal/task"
	"github.com/spf13/cobra"
)

// UnlockOptions contains options for the unlock command
type UnlockOptions struct {
	RepoPath   string
	TaskID     string
	Branch     string
	Remove     bool
	OutputJSON bool
}

// UnlockResult represents the output of the unlock command
type UnlockResult struct {
	TaskID          string   `json:"task_id"`
	Branch          string   `json:"branch"`
	WorktreesFreed  []string `json:"worktrees_freed"`
	WorktreesRemoved []string `json:"worktrees_removed,omitempty"`
}

// NewTaskUnlockCmd creates the task unlock command
func NewTaskUnlockCmd() *cobra.Command {
	opts := &UnlockOptions{}

	cmd := &cobra.Command{
		Use:   "unlock [task-id]",
		Short: "Unlock a task branch by detaching worktrees",
		Long: `Unlock a task branch by detaching HEAD in worktrees where it's checked out.

The task can be specified by:
  1. Providing the task ID as an argument
  2. Using --branch flag

This command is useful when a branch is locked by another worktree and you
want to free it up for other operations.

Example:
  awt task unlock 20250110-120000-abc123
  awt task unlock --branch=awt/claude/20250110-120000-abc123
  awt task unlock 20250110-120000-abc123 --remove  # also remove worktrees`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.TaskID = args[0]
			}
			return runTaskUnlock(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "branch name")
	cmd.Flags().BoolVar(&opts.Remove, "remove", false, "remove worktrees after detaching")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	return cmd
}

func runTaskUnlock(opts *UnlockOptions) error {
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

	// Create Git wrapper
	g := git.New(r.WorkTreeRoot, false)

	// Find worktrees where this branch is checked out
	worktrees, err := g.WorktreeList()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktreesWithBranch []*git.Worktree
	branchRef := t.Branch
	if !strings.HasPrefix(branchRef, "refs/heads/") {
		branchRef = "refs/heads/" + branchRef
	}

	for _, wt := range worktrees {
		if wt.Branch == branchRef {
			worktreesWithBranch = append(worktreesWithBranch, wt)
		}
	}

	if len(worktreesWithBranch) == 0 {
		if !opts.OutputJSON {
			fmt.Printf("Branch %s is not checked out in any worktree\n", t.Branch)
		}
		return nil
	}

	// Acquire global lock for safety
	lm := lock.NewLockManager(r.GitCommonDir)
	ctx := context.Background()
	globalLock, err := lm.AcquireGlobal(ctx)
	if err != nil {
		return errors.LockTimeout("global")
	}
	defer globalLock.Release()

	var worktreesFreed []string
	var worktreesRemoved []string

	// Detach HEAD in each worktree
	for _, wt := range worktreesWithBranch {
		if !opts.OutputJSON {
			fmt.Printf("Detaching HEAD in worktree: %s\n", wt.Path)
		}

		// Create git wrapper for the worktree
		wtGit := git.New(wt.Path, false)
		result, err := wtGit.Switch("HEAD", true)
		if err != nil || result.ExitCode != 0 {
			return errors.DetachFailed(wt.Path, err)
		}

		worktreesFreed = append(worktreesFreed, wt.Path)

		// Remove worktree if requested
		if opts.Remove {
			if !opts.OutputJSON {
				fmt.Printf("Removing worktree: %s\n", wt.Path)
			}

			// Resolve absolute path
			wtPathAbs, _ := filepath.Abs(wt.Path)

			removeResult, err := g.WorktreeRemove(wtPathAbs, true)
			if err != nil || removeResult.ExitCode != 0 {
				// Don't fail if removal fails - just warn
				if !opts.OutputJSON {
					fmt.Printf("Warning: failed to remove worktree %s: %s\n", wt.Path, removeResult.Stderr)
				}
			} else {
				worktreesRemoved = append(worktreesRemoved, wt.Path)
			}
		}
	}

	// Output result
	if opts.OutputJSON {
		output := UnlockResult{
			TaskID:           taskID,
			Branch:           t.Branch,
			WorktreesFreed:   worktreesFreed,
			WorktreesRemoved: worktreesRemoved,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("\nUnlock completed successfully!\n")
		fmt.Printf("  Task: %s\n", taskID)
		fmt.Printf("  Branch: %s\n", t.Branch)
		fmt.Printf("  Worktrees freed: %d\n", len(worktreesFreed))
		if opts.Remove {
			fmt.Printf("  Worktrees removed: %d\n", len(worktreesRemoved))
		}
	}

	return nil
}
