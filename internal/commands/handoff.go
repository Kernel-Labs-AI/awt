package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kernel-labs-ai/awt/internal/errors"
	"github.com/kernel-labs-ai/awt/internal/git"
	"github.com/kernel-labs-ai/awt/internal/lock"
	"github.com/kernel-labs-ai/awt/internal/repo"
	"github.com/kernel-labs-ai/awt/internal/task"
	"github.com/spf13/cobra"
)

// HandoffOptions contains options for the handoff command
type HandoffOptions struct {
	RepoPath     string
	TaskID       string
	Branch       string
	Push         bool
	CreatePR     bool
	KeepWorktree bool
	ForceRemove  bool
	OutputJSON   bool
}

// HandoffResult represents the output of the handoff command
type HandoffResult struct {
	TaskID       string `json:"task_id"`
	Branch       string `json:"branch"`
	Pushed       bool   `json:"pushed"`
	PRURL        string `json:"pr_url,omitempty"`
	WorktreeKept bool   `json:"worktree_kept"`
}

// NewTaskHandoffCmd creates the task handoff command
func NewTaskHandoffCmd() *cobra.Command {
	opts := &HandoffOptions{}

	cmd := &cobra.Command{
		Use:   "handoff [task-id]",
		Short: "Hand off a task for review",
		Long: `Hand off a task for review by pushing and optionally creating a PR.

The task can be specified by:
  1. Providing the task ID as an argument
  2. Using --branch flag
  3. Inferring from current worktree (if in a worktree)

This command performs the following steps:
  1. Commits any staged changes (optional)
  2. Syncs with base branch
  3. Pushes to remote (if --push)
  4. Creates PR/MR (if --create-pr, requires gh/glab)
  5. Detaches HEAD in worktree
  6. Removes worktree (unless --keep-worktree)
  7. Updates task state to HANDOFF_READY

Example:
  awt task handoff 20250110-120000-abc123 --push --create-pr
  awt task handoff --push
  awt task handoff --keep-worktree`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.TaskID = args[0]
			}
			return runTaskHandoff(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "branch name")
	cmd.Flags().BoolVar(&opts.Push, "push", false, "push to remote")
	cmd.Flags().BoolVar(&opts.CreatePR, "create-pr", false, "create pull/merge request (requires --push)")
	cmd.Flags().BoolVar(&opts.KeepWorktree, "keep-worktree", false, "keep worktree after handoff")
	cmd.Flags().BoolVar(&opts.ForceRemove, "force-remove", false, "force remove worktree even if CWD is inside")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	return cmd
}

func runTaskHandoff(opts *HandoffOptions) error {
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

	// Step 1: Check for uncommitted changes (optional - just warn)
	statusResult, err := g.Status()
	if err == nil && statusResult.ExitCode == 0 {
		if !strings.Contains(statusResult.Stdout, "nothing to commit") {
			if !opts.OutputJSON {
				fmt.Println("Warning: uncommitted changes detected. Consider running 'awt task commit' first.")
			}
		}
	}

	// Step 2: Sync with base (rebase by default)
	if !opts.OutputJSON {
		fmt.Printf("Syncing with base branch %s...\n", t.Base)
	}

	syncResult, err := g.Rebase(t.Base)
	if err != nil || syncResult.ExitCode != 0 {
		// Check for conflicts
		if strings.Contains(syncResult.Stderr, "conflict") || strings.Contains(syncResult.Stdout, "conflict") {
			return errors.SyncConflicts(t.Branch)
		}
		// Rebase failed but not conflicts - continue anyway
		if !opts.OutputJSON {
			fmt.Printf("Warning: sync failed: %s\n", syncResult.Stderr)
		}
	}

	// Step 3: Push if requested
	pushed := false
	if opts.Push {
		if !opts.OutputJSON {
			fmt.Printf("Pushing to remote...\n")
		}

		// Extract branch name without refs/heads/
		branchName := strings.TrimPrefix(t.Branch, "refs/heads/")

		pushResult, err := g.Push("origin", branchName, true, false)
		if err != nil || pushResult.ExitCode != 0 {
			return errors.PushRejected(t.Branch, err)
		}
		pushed = true
	}

	// Step 4: Create PR if requested
	prURL := ""
	if opts.CreatePR {
		if !opts.Push {
			return fmt.Errorf("--create-pr requires --push")
		}

		if !opts.OutputJSON {
			fmt.Printf("Creating pull request...\n")
		}

		// Check if gh or glab is available
		ghAvailable := checkCommandExists("gh")
		glabAvailable := checkCommandExists("glab")

		if !ghAvailable && !glabAvailable {
			return errors.ToolMissing("gh or glab")
		}

		// Try to create PR
		var prResult *git.Result
		if ghAvailable {
			prResult, err = g.CreatePRWithGH(t.Title, fmt.Sprintf("Task: %s\nAgent: %s\nBranch: %s", t.ID, t.Agent, t.Branch), t.Base)
		} else {
			prResult, err = g.CreateMRWithGLab(t.Title, fmt.Sprintf("Task: %s\nAgent: %s\nBranch: %s", t.ID, t.Agent, t.Branch), t.Base)
		}

		if err != nil || prResult.ExitCode != 0 {
			// PR creation failed - don't fail the handoff, just warn
			if !opts.OutputJSON {
				fmt.Printf("Warning: failed to create PR: %s\n", prResult.Stderr)
			}
		} else {
			// Extract PR URL from output
			prURL = extractPRURL(prResult.Stdout)
			if prURL != "" {
				t.PRURL = prURL
			}
		}
	}

	// Step 5: Detach HEAD in worktree
	if !opts.OutputJSON {
		fmt.Printf("Detaching HEAD in worktree...\n")
	}

	detachResult, err := g.Switch("HEAD", true)
	if err != nil || detachResult.ExitCode != 0 {
		return errors.DetachFailed(t.WorktreePath, err)
	}

	// Step 6: Remove worktree (unless --keep-worktree)
	worktreeKept := opts.KeepWorktree
	if !opts.KeepWorktree {
		// Check if CWD is inside the worktree
		cwd, err := os.Getwd()
		if err == nil {
			wtPathAbs, _ := filepath.Abs(t.WorktreePath)
			cwdAbs, _ := filepath.Abs(cwd)

			// Check if cwd is inside worktree
			rel, err := filepath.Rel(wtPathAbs, cwdAbs)
			isInside := err == nil && !filepath.IsAbs(rel) && rel != ".." && !hasParentDir(rel)

			if isInside && !opts.ForceRemove {
				if !opts.OutputJSON {
					fmt.Printf("Warning: current directory is inside worktree. Keeping worktree.\n")
					fmt.Printf("Use --force-remove to remove anyway, or cd out of the worktree.\n")
				}
				worktreeKept = true
			} else if isInside && opts.ForceRemove {
				// Change to repository root before removing
				if err := os.Chdir(r.WorkTreeRoot); err != nil {
					return fmt.Errorf("failed to change directory: %w", err)
				}
			}
		}

		if !worktreeKept {
			if !opts.OutputJSON {
				fmt.Printf("Removing worktree...\n")
			}

			// Acquire global lock before removing worktree
			lm := lock.NewLockManager(r.GitCommonDir)
			ctx := context.Background()
			globalLock, err := lm.AcquireGlobal(ctx)
			if err != nil {
				return errors.LockTimeout("global")
			}
			defer func() {
			_ = globalLock.Release()
		}()

			// Create git wrapper from repo root
			repoGit := git.New(r.WorkTreeRoot, false)
			removeResult, err := repoGit.WorktreeRemove(t.WorktreePath, true)
			if err != nil || removeResult.ExitCode != 0 {
				return errors.RemoveFailed(t.WorktreePath, err)
			}
		}
	}

	// Update task state
	t.State = task.StateHandoffReady
	if err := store.Save(t); err != nil {
		return fmt.Errorf("failed to update task metadata: %w", err)
	}

	// Output result
	if opts.OutputJSON {
		output := HandoffResult{
			TaskID:       taskID,
			Branch:       t.Branch,
			Pushed:       pushed,
			PRURL:        prURL,
			WorktreeKept: worktreeKept,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("\nHandoff completed successfully!\n")
		fmt.Printf("  Task: %s\n", taskID)
		fmt.Printf("  Branch: %s\n", t.Branch)
		fmt.Printf("  State: %s\n", t.State)
		if pushed {
			fmt.Printf("  Pushed: yes\n")
		}
		if prURL != "" {
			fmt.Printf("  PR: %s\n", prURL)
		}
		if worktreeKept {
			fmt.Printf("  Worktree: kept at %s\n", t.WorktreePath)
		} else {
			fmt.Printf("  Worktree: removed\n")
		}
	}

	return nil
}

// checkCommandExists checks if a command exists in PATH
func checkCommandExists(cmd string) bool {
	_, err := os.Stat("/usr/bin/" + cmd)
	if err == nil {
		return true
	}
	_, err = os.Stat("/usr/local/bin/" + cmd)
	return err == nil
}

// extractPRURL extracts the PR URL from gh/glab output
func extractPRURL(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "http") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}
