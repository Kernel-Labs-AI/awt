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

// StatusOptions contains options for the status command
type StatusOptions struct {
	RepoPath   string
	TaskID     string
	Branch     string
	OutputJSON bool
}

// StatusResult represents the output of the status command
type StatusResult struct {
	ID           string `json:"id"`
	Agent        string `json:"agent"`
	Title        string `json:"title"`
	Branch       string `json:"branch"`
	Base         string `json:"base"`
	State        string `json:"state"`
	WorktreePath string `json:"worktree_path"`
	CreatedAt    string `json:"created_at"`
	LastCommit   string `json:"last_commit,omitempty"`
	PRURL        string `json:"pr_url,omitempty"`
}

// NewTaskStatusCmd creates the task status command
func NewTaskStatusCmd() *cobra.Command {
	opts := &StatusOptions{}

	cmd := &cobra.Command{
		Use:   "status [task-id]",
		Short: "Show task status",
		Long: `Show the status of a task.

The task can be specified by:
  1. Providing the task ID as an argument
  2. Using --branch flag
  3. Inferring from current worktree (if in a worktree)

Example:
  awt task status 20250110-120000-abc123
  awt task status --branch=awt/claude/20250110-120000-abc123
  awt task status  # infer from current directory`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.TaskID = args[0]
			}
			return runTaskStatus(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "branch name")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	return cmd
}

func runTaskStatus(opts *StatusOptions) error {
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
		// Branch format: awt/<agent>/<id>
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

	// Output result
	if opts.OutputJSON {
		result := StatusResult{
			ID:           t.ID,
			Agent:        t.Agent,
			Title:        t.Title,
			Branch:       t.Branch,
			Base:         t.Base,
			State:        string(t.State),
			WorktreePath: t.WorktreePath,
			CreatedAt:    t.CreatedAt.Format("2006-01-02 15:04:05"),
			LastCommit:   t.LastCommit,
			PRURL:        t.PRURL,
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Task: %s\n", t.ID)
		fmt.Printf("  Agent: %s\n", t.Agent)
		fmt.Printf("  Title: %s\n", t.Title)
		fmt.Printf("  Branch: %s\n", t.Branch)
		fmt.Printf("  Base: %s\n", t.Base)
		fmt.Printf("  State: %s\n", t.State)
		fmt.Printf("  Worktree: %s\n", t.WorktreePath)
		fmt.Printf("  Created: %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
		if t.LastCommit != "" {
			fmt.Printf("  Last Commit: %s\n", t.LastCommit)
		}
		if t.PRURL != "" {
			fmt.Printf("  PR URL: %s\n", t.PRURL)
		}
	}

	return nil
}

// extractTaskIDFromBranch extracts the task ID from a branch name
// Branch format: awt/<agent>/<id>
func extractTaskIDFromBranch(branch string) string {
	// Remove refs/heads/ prefix if present
	if len(branch) > 11 && branch[:11] == "refs/heads/" {
		branch = branch[11:]
	}

	// Split by /
	parts := filepath.SplitList(branch)
	if len(parts) < 3 {
		// Try with plain string split
		parts := splitPath(branch)
		if len(parts) >= 3 {
			return parts[2]
		}
		return ""
	}

	return parts[2]
}

// splitPath splits a path by /
func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, c := range path {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// inferTaskIDFromCurrentDirectory tries to infer the task ID from the current directory
func inferTaskIDFromCurrentDirectory(r *repo.Repo) (string, error) {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Check if we're in a worktree
	g := git.New(r.WorkTreeRoot, false)
	worktrees, err := g.WorktreeList()
	if err != nil {
		return "", err
	}

	// Find matching worktree
	for _, wt := range worktrees {
		// Resolve symlinks for comparison
		wtPathAbs, err := filepath.EvalSymlinks(wt.Path)
		if err != nil {
			wtPathAbs = wt.Path
		}
		cwdAbs, err := filepath.EvalSymlinks(cwd)
		if err != nil {
			cwdAbs = cwd
		}

		// Make paths absolute
		wtPathAbs, _ = filepath.Abs(wtPathAbs)
		cwdAbs, _ = filepath.Abs(cwdAbs)

		// Check if current directory is inside this worktree
		// Use filepath.Rel to check if cwd is under wtPath
		rel, err := filepath.Rel(wtPathAbs, cwdAbs)
		if err == nil && !filepath.IsAbs(rel) && rel != ".." && !hasParentDir(rel) {
			// Extract task ID from branch
			taskID := extractTaskIDFromBranch(wt.Branch)
			if taskID != "" {
				return taskID, nil
			}
		}
	}

	return "", fmt.Errorf("not in a task worktree")
}

// hasParentDir checks if a relative path contains ..
func hasParentDir(path string) bool {
	parts := splitPath(path)
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}
	return false
}
