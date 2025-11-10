package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kernel-labs-ai/awt/internal/errors"
	"github.com/kernel-labs-ai/awt/internal/git"
	"github.com/kernel-labs-ai/awt/internal/idgen"
	"github.com/kernel-labs-ai/awt/internal/repo"
	"github.com/kernel-labs-ai/awt/internal/task"
	"github.com/spf13/cobra"
)

// AdoptOptions contains options for the adopt command
type AdoptOptions struct {
	RepoPath   string
	Branch     string
	ID         string
	Agent      string
	Base       string
	Title      string
	OutputJSON bool
}

// AdoptResult represents the output of the adopt command
type AdoptResult struct {
	TaskID string `json:"task_id"`
	Branch string `json:"branch"`
	Base   string `json:"base"`
	Agent  string `json:"agent"`
	Title  string `json:"title"`
}

// NewTaskAdoptCmd creates the task adopt command
func NewTaskAdoptCmd() *cobra.Command {
	opts := &AdoptOptions{}

	cmd := &cobra.Command{
		Use:   "adopt --branch=<branch>",
		Short: "Adopt an existing branch as a task",
		Long: `Adopt an existing Git branch as an AWT task.

This command creates task metadata for an existing branch, allowing it
to be managed with AWT commands.

Example:
  awt task adopt --branch=feature/new-api --agent=claude --title="New API"
  awt task adopt --branch=feature/fix --agent=human --base=develop`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTaskAdopt(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "branch name (required)")
	cmd.Flags().StringVar(&opts.ID, "id", "", "task ID (auto-generated if not provided)")
	cmd.Flags().StringVar(&opts.Agent, "agent", "", "agent name (required)")
	cmd.Flags().StringVar(&opts.Base, "base", "", "base branch (auto-detected if not provided)")
	cmd.Flags().StringVar(&opts.Title, "title", "", "task title (uses branch name if not provided)")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	cmd.MarkFlagRequired("branch")
	cmd.MarkFlagRequired("agent")

	return cmd
}

func runTaskAdopt(opts *AdoptOptions) error {
	// Discover repository
	r, err := repo.DiscoverRepo(opts.RepoPath)
	if err != nil {
		return errors.RepoNotFound(opts.RepoPath)
	}

	store := task.NewTaskStore(r.GitCommonDir)

	// Create Git wrapper
	g := git.New(r.WorkTreeRoot, false)

	// Verify branch exists
	branch := opts.Branch
	if !strings.HasPrefix(branch, "refs/heads/") {
		branch = "refs/heads/" + branch
	}

	exists, err := g.BranchExists(strings.TrimPrefix(branch, "refs/heads/"))
	if err != nil {
		return fmt.Errorf("failed to check branch existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("branch does not exist: %s", opts.Branch)
	}

	// Detect base branch if not provided
	base := opts.Base
	if base == "" {
		// Try to detect base via merge-base
		// Common base branches to try
		baseCandidates := []string{"origin/main", "origin/master", "main", "master", "origin/develop", "develop"}
		for _, candidate := range baseCandidates {
			candidateExists, err := g.BranchExists(strings.TrimPrefix(candidate, "origin/"))
			if err == nil && candidateExists {
				base = candidate
				break
			}
		}
		if base == "" {
			return fmt.Errorf("could not detect base branch, please specify with --base")
		}
	}

	// Generate task ID if not provided
	taskID := opts.ID
	if taskID == "" {
		taskID, err = idgen.GenerateTaskID()
		if err != nil {
			return fmt.Errorf("failed to generate task ID: %w", err)
		}
	} else if !idgen.ValidateTaskID(taskID) {
		return errors.InvalidTaskID(taskID)
	}

	// Use branch name as title if not provided
	title := opts.Title
	if title == "" {
		// Remove refs/heads/ prefix
		title = strings.TrimPrefix(branch, "refs/heads/")
		// Remove common prefixes
		title = strings.TrimPrefix(title, "feature/")
		title = strings.TrimPrefix(title, "fix/")
		title = strings.TrimPrefix(title, "bugfix/")
		// Replace slashes and dashes with spaces
		title = strings.ReplaceAll(title, "/", " ")
		title = strings.ReplaceAll(title, "-", " ")
		title = strings.ReplaceAll(title, "_", " ")
	}

	// Create task metadata
	t := &task.Task{
		ID:           taskID,
		Agent:        opts.Agent,
		Title:        title,
		Branch:       branch,
		Base:         base,
		CreatedAt:    time.Now(),
		State:        task.StateActive,
		WorktreePath: "", // Empty until checkout
	}

	// Get last commit if branch exists
	commitSHA, err := g.RevParse(strings.TrimPrefix(branch, "refs/heads/"))
	if err == nil && commitSHA != "" {
		t.LastCommit = commitSHA
	}

	// Save task
	if err := store.Save(t); err != nil {
		return fmt.Errorf("failed to save task: %w", err)
	}

	// Output result
	if opts.OutputJSON {
		output := AdoptResult{
			TaskID: taskID,
			Branch: branch,
			Base:   base,
			Agent:  opts.Agent,
			Title:  title,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Adopted branch successfully!\n")
		fmt.Printf("  Task ID: %s\n", taskID)
		fmt.Printf("  Branch: %s\n", branch)
		fmt.Printf("  Base: %s\n", base)
		fmt.Printf("  Agent: %s\n", opts.Agent)
		fmt.Printf("  Title: %s\n", title)
		fmt.Printf("\nUse 'awt task checkout %s' to create a worktree for this task.\n", taskID)
	}

	return nil
}
