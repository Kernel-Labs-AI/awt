package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/decibelvc/awt/internal/errors"
	"github.com/decibelvc/awt/internal/git"
	"github.com/decibelvc/awt/internal/repo"
	"github.com/decibelvc/awt/internal/task"
	"github.com/spf13/cobra"
)

// PruneOptions contains options for the prune command
type PruneOptions struct {
	RepoPath   string
	DryRun     bool
	OutputJSON bool
}

// PruneResult represents the output of the prune command
type PruneResult struct {
	PrunedWorktrees int      `json:"pruned_worktrees"`
	DeletedTasks    []string `json:"deleted_tasks,omitempty"`
	DeletedLocks    []string `json:"deleted_locks,omitempty"`
}

// NewPruneCmd creates the prune command
func NewPruneCmd() *cobra.Command {
	opts := &PruneOptions{}

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Clean up orphaned tasks and stale locks",
		Long: `Clean up orphaned task metadata and stale locks.

This command performs the following cleanup operations:
  1. Runs git worktree prune to remove deleted worktrees
  2. Removes task metadata for non-existent worktrees
  3. Cleans up stale lock files

Example:
  awt prune
  awt prune --dry-run  # preview what would be cleaned`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrune(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "preview what would be cleaned without making changes")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	return cmd
}

func runPrune(opts *PruneOptions) error {
	// Discover repository
	r, err := repo.DiscoverRepo(opts.RepoPath)
	if err != nil {
		return errors.RepoNotFound(opts.RepoPath)
	}

	store := task.NewTaskStore(r.GitCommonDir)

	// Create Git wrapper
	g := git.New(r.WorkTreeRoot, false)

	result := PruneResult{}

	// Step 1: Run git worktree prune
	if !opts.OutputJSON && !opts.DryRun {
		fmt.Println("Pruning Git worktrees...")
	}

	if !opts.DryRun {
		pruneResult, err := g.WorktreePrune()
		if err != nil || pruneResult.ExitCode != 0 {
			// Don't fail if prune fails - just warn
			if !opts.OutputJSON {
				fmt.Printf("Warning: git worktree prune failed: %s\n", pruneResult.Stderr)
			}
		} else {
			result.PrunedWorktrees = 1 // git worktree prune doesn't report count
		}
	}

	// Step 2: Find orphaned task metadata
	if !opts.OutputJSON && !opts.DryRun {
		fmt.Println("Checking for orphaned task metadata...")
	}

	tasks, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	for _, t := range tasks {
		if t.WorktreePath == "" {
			// Task has no worktree, skip
			continue
		}

		// Check if worktree exists
		if _, err := os.Stat(t.WorktreePath); os.IsNotExist(err) {
			// Worktree doesn't exist, delete task metadata
			if !opts.DryRun {
				if !opts.OutputJSON {
					fmt.Printf("Deleting orphaned task: %s\n", t.ID)
				}
				if err := store.Delete(t.ID); err != nil {
					if !opts.OutputJSON {
						fmt.Printf("Warning: failed to delete task %s: %v\n", t.ID, err)
					}
				} else {
					result.DeletedTasks = append(result.DeletedTasks, t.ID)
				}
			} else {
				if !opts.OutputJSON {
					fmt.Printf("Would delete orphaned task: %s\n", t.ID)
				}
				result.DeletedTasks = append(result.DeletedTasks, t.ID)
			}
		}
	}

	// Step 3: Clean up stale lock files
	if !opts.OutputJSON && !opts.DryRun {
		fmt.Println("Checking for stale locks...")
	}

	locksDir := filepath.Join(r.GitCommonDir, "awt", "locks")
	if entries, err := os.ReadDir(locksDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				lockPath := filepath.Join(locksDir, entry.Name())

				// Try to check if lock is stale
				// A lock file without an active process holding it is stale
				if info, err := os.Stat(lockPath); err == nil {
					// If file size is 0, it's likely stale
					if info.Size() == 0 {
						if !opts.DryRun {
							if !opts.OutputJSON {
								fmt.Printf("Deleting stale lock: %s\n", entry.Name())
							}
							if err := os.Remove(lockPath); err != nil {
								if !opts.OutputJSON {
									fmt.Printf("Warning: failed to remove lock %s: %v\n", entry.Name(), err)
								}
							} else {
								result.DeletedLocks = append(result.DeletedLocks, entry.Name())
							}
						} else {
							if !opts.OutputJSON {
								fmt.Printf("Would delete stale lock: %s\n", entry.Name())
							}
							result.DeletedLocks = append(result.DeletedLocks, entry.Name())
						}
					}
				}
			}
		}
	}

	// Output result
	if opts.OutputJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Println("\nPrune completed!")
		if opts.DryRun {
			fmt.Println("  Mode: dry-run (no changes made)")
		}
		fmt.Printf("  Orphaned tasks deleted: %d\n", len(result.DeletedTasks))
		fmt.Printf("  Stale locks deleted: %d\n", len(result.DeletedLocks))
	}

	return nil
}
