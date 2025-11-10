package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kernel-labs-ai/awt/internal/errors"
	"github.com/kernel-labs-ai/awt/internal/git"
	"github.com/kernel-labs-ai/awt/internal/repo"
	"github.com/kernel-labs-ai/awt/internal/task"
	"github.com/spf13/cobra"
)

// SyncOptions contains options for the sync command
type SyncOptions struct {
	RepoPath   string
	TaskID     string
	Branch     string
	Merge      bool
	Rebase     bool
	Submodules bool
	OutputJSON bool
}

// SyncResult represents the output of the sync command
type SyncResult struct {
	TaskID   string `json:"task_id"`
	Strategy string `json:"strategy"`
	Base     string `json:"base"`
	Success  bool   `json:"success"`
}

// NewTaskSyncCmd creates the task sync command
func NewTaskSyncCmd() *cobra.Command {
	opts := &SyncOptions{}

	cmd := &cobra.Command{
		Use:   "sync [task-id]",
		Short: "Sync task branch with base branch",
		Long: `Sync the task's branch with its base branch.

The task can be specified by:
  1. Providing the task ID as an argument
  2. Using --branch flag
  3. Inferring from current worktree (if in a worktree)

By default, the command uses rebase. Use --merge to merge instead.

Example:
  awt task sync 20250110-120000-abc123
  awt task sync --merge
  awt task sync --submodules  # also update submodules`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.TaskID = args[0]
			}
			return runTaskSync(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "branch name")
	cmd.Flags().BoolVar(&opts.Merge, "merge", false, "use merge instead of rebase")
	cmd.Flags().BoolVar(&opts.Rebase, "rebase", true, "use rebase (default)")
	cmd.Flags().BoolVar(&opts.Submodules, "submodules", false, "update submodules after sync")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	return cmd
}

func runTaskSync(opts *SyncOptions) error {
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
		// Try to infer from current worktree
		taskID, err = inferTaskIDFromCurrentDirectory(r)
		if err != nil {
			return fmt.Errorf("could not infer task ID: %w\nProvide task ID as argument or use --branch flag", err)
		}
	}

	// Load task
	t, err := store.Load(taskID)
	if err != nil {
		return errors.InvalidTaskID(taskID)
	}

	// Create Git wrapper for the worktree
	g := git.New(t.WorktreePath, false)

	// Fetch base ref
	result, err := g.Fetch("", "")
	if err != nil || result.ExitCode != 0 {
		// Check if it's a shallow clone
		if strings.Contains(result.Stderr, "shallow") {
			// Try to unshallow
			result, err = g.FetchUnshallow()
			if err != nil || result.ExitCode != 0 {
				return fmt.Errorf("failed to unshallow repository: %s", result.Stderr)
			}
		} else {
			// Fetch failed, but continue anyway (might be offline)
			// Log warning but don't fail
			if !opts.OutputJSON {
				fmt.Printf("Warning: fetch failed, continuing with local refs: %s\n", result.Stderr)
			}
		}
	}

	// Determine strategy (merge or rebase)
	strategy := "rebase"
	if opts.Merge {
		strategy = "merge"
	}

	// Execute sync
	var syncResult *git.Result
	if strategy == "merge" {
		syncResult, err = g.Merge(t.Base)
	} else {
		syncResult, err = g.Rebase(t.Base)
	}

	if err != nil || syncResult.ExitCode != 0 {
		// Check for conflicts
		if strings.Contains(syncResult.Stderr, "conflict") || strings.Contains(syncResult.Stdout, "conflict") {
			return errors.SyncConflicts(t.Branch)
		}
		return fmt.Errorf("failed to %s: %s", strategy, syncResult.Stderr)
	}

	// Update submodules if requested
	if opts.Submodules {
		subResult, err := g.SubmoduleUpdate()
		if err != nil || subResult.ExitCode != 0 {
			return fmt.Errorf("failed to update submodules: %s", subResult.Stderr)
		}
	}

	// Output result
	if opts.OutputJSON {
		output := SyncResult{
			TaskID:   taskID,
			Strategy: strategy,
			Base:     t.Base,
			Success:  true,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Synced successfully!\n")
		fmt.Printf("  Task: %s\n", taskID)
		fmt.Printf("  Strategy: %s\n", strategy)
		fmt.Printf("  Base: %s\n", t.Base)
		if opts.Submodules {
			fmt.Printf("  Submodules: updated\n")
		}
	}

	return nil
}
