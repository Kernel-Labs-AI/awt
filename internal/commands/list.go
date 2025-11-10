package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/decibelvc/awt/internal/errors"
	"github.com/decibelvc/awt/internal/git"
	"github.com/decibelvc/awt/internal/repo"
	"github.com/decibelvc/awt/internal/task"
	"github.com/spf13/cobra"
)

// ListOptions contains options for the list command
type ListOptions struct {
	RepoPath   string
	OutputJSON bool
}

// TaskListItem represents a task in the list output
type TaskListItem struct {
	ID           string `json:"id"`
	Agent        string `json:"agent"`
	Title        string `json:"title"`
	State        string `json:"state"`
	Branch       string `json:"branch"`
	WorktreePath string `json:"worktree_path,omitempty"`
	CheckedOut   bool   `json:"checked_out"`
}

// NewListCmd creates the list command
func NewListCmd() *cobra.Command {
	opts := &ListOptions{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all AWT tasks",
		Long: `List all AWT tasks with their current status.

Shows task ID, agent, title, state, and checkout status.

Example:
  awt list
  awt list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	return cmd
}

func runList(opts *ListOptions) error {
	// Discover repository
	r, err := repo.DiscoverRepo(opts.RepoPath)
	if err != nil {
		return errors.RepoNotFound(opts.RepoPath)
	}

	store := task.NewTaskStore(r.GitCommonDir)

	// List all tasks
	tasks, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	if len(tasks) == 0 {
		if !opts.OutputJSON {
			fmt.Println("No tasks found")
		} else {
			fmt.Println("[]")
		}
		return nil
	}

	// Create Git wrapper to check worktree status
	g := git.New(r.WorkTreeRoot, false)
	worktrees, err := g.WorktreeList()
	if err != nil {
		// Don't fail if we can't list worktrees
		worktrees = nil
	}

	// Build worktree map for quick lookup
	worktreeMap := make(map[string]string) // branch -> path
	for _, wt := range worktrees {
		worktreeMap[wt.Branch] = wt.Path
	}

	// Build task list
	var items []TaskListItem
	for _, t := range tasks {
		branchRef := t.Branch
		if !strings.HasPrefix(branchRef, "refs/heads/") {
			branchRef = "refs/heads/" + branchRef
		}

		wtPath, checkedOut := worktreeMap[branchRef]
		if !checkedOut && t.WorktreePath != "" {
			// Check if the worktree path in metadata exists
			if _, err := os.Stat(t.WorktreePath); err == nil {
				wtPath = t.WorktreePath
				checkedOut = true
			}
		}

		item := TaskListItem{
			ID:           t.ID,
			Agent:        t.Agent,
			Title:        t.Title,
			State:        string(t.State),
			Branch:       t.Branch,
			WorktreePath: wtPath,
			CheckedOut:   checkedOut,
		}
		items = append(items, item)
	}

	// Output result
	if opts.OutputJSON {
		data, _ := json.MarshalIndent(items, "", "  ")
		fmt.Println(string(data))
	} else {
		// Print table header
		fmt.Printf("%-20s %-12s %-30s %-15s %-10s\n", "ID", "AGENT", "TITLE", "STATE", "CHECKED OUT")
		fmt.Println(strings.Repeat("-", 90))

		// Print tasks
		for _, item := range items {
			title := item.Title
			if len(title) > 30 {
				title = title[:27] + "..."
			}

			checkedOut := "no"
			if item.CheckedOut {
				checkedOut = "yes"
			}

			fmt.Printf("%-20s %-12s %-30s %-15s %-10s\n",
				item.ID,
				item.Agent,
				title,
				item.State,
				checkedOut,
			)
		}

		fmt.Printf("\nTotal: %d tasks\n", len(items))
	}

	return nil
}
