package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/decibelvc/awt/internal/errors"
	"github.com/decibelvc/awt/internal/git"
	"github.com/decibelvc/awt/internal/repo"
	"github.com/decibelvc/awt/internal/task"
	"github.com/spf13/cobra"
)

// CommitOptions contains options for the commit command
type CommitOptions struct {
	RepoPath   string
	TaskID     string
	Branch     string
	Message    string
	All        bool
	Signoff    bool
	GPGSign    string
	OutputJSON bool
}

// CommitResult represents the output of the commit command
type CommitResult struct {
	TaskID     string `json:"task_id"`
	CommitSHA  string `json:"commit_sha"`
	Message    string `json:"message"`
	FilesCount int    `json:"files_count,omitempty"`
}

// NewTaskCommitCmd creates the task commit command
func NewTaskCommitCmd() *cobra.Command {
	opts := &CommitOptions{}

	cmd := &cobra.Command{
		Use:   "commit [task-id]",
		Short: "Commit changes in a task's worktree",
		Long: `Commit changes in the context of a task's worktree.

The task can be specified by:
  1. Providing the task ID as an argument
  2. Using --branch flag
  3. Inferring from current worktree (if in a worktree)

If no message is provided, a default message will be generated:
  feat(task:<id>): <title>

  <metadata body>

Example:
  awt task commit 20250110-120000-abc123 -m "Add feature"
  awt task commit --all -m "Update implementation"
  awt task commit  # infer from current directory`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.TaskID = args[0]
			}
			return runTaskCommit(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "branch name")
	cmd.Flags().StringVarP(&opts.Message, "message", "m", "", "commit message")
	cmd.Flags().BoolVarP(&opts.All, "all", "a", false, "stage all modified files")
	cmd.Flags().BoolVar(&opts.Signoff, "signoff", false, "add Signed-off-by trailer")
	cmd.Flags().StringVar(&opts.GPGSign, "gpg-sign", "", "GPG sign commit (optional key-id)")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	return cmd
}

func runTaskCommit(opts *CommitOptions) error {
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

	// Stage files if --all flag is set
	if opts.All {
		result, err := g.Add(".")
		if err != nil || result.ExitCode != 0 {
			return fmt.Errorf("failed to stage files: %s", result.Stderr)
		}
	}

	// Generate message if not provided
	message := opts.Message
	if message == "" {
		message = generateDefaultCommitMessage(t)
	}

	// Determine GPG signing
	gpgSign := opts.GPGSign != ""

	// Execute commit
	result, err := g.Commit(message, false, opts.Signoff, gpgSign)
	if err != nil || result.ExitCode != 0 {
		// Check for common error cases
		if strings.Contains(result.Stderr, "nothing to commit") {
			return fmt.Errorf("nothing to commit, working tree clean")
		}
		if strings.Contains(result.Stderr, "no changes added to commit") {
			return fmt.Errorf("no changes added to commit\nUse --all to stage all modified files, or stage files manually")
		}
		return fmt.Errorf("failed to commit: %s", result.Stderr)
	}

	// Get the commit SHA from the output
	commitSHA, err := g.RevParse("HEAD")
	if err != nil || commitSHA == "" {
		return fmt.Errorf("failed to get commit SHA")
	}

	// Update task metadata with last commit
	t.LastCommit = commitSHA
	if err := store.Save(t); err != nil {
		return fmt.Errorf("failed to update task metadata: %w", err)
	}

	// Output result
	if opts.OutputJSON {
		output := CommitResult{
			TaskID:    taskID,
			CommitSHA: commitSHA,
			Message:   message,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Committed successfully!\n")
		fmt.Printf("  Task: %s\n", taskID)
		fmt.Printf("  Commit: %s\n", commitSHA)
		if !opts.OutputJSON {
			// Show abbreviated commit output
			fmt.Println()
			fmt.Println(result.Stdout)
		}
	}

	return nil
}

// generateDefaultCommitMessage generates a default commit message for a task
func generateDefaultCommitMessage(t *task.Task) string {
	var sb strings.Builder

	// First line: feat(task:<id>): <title>
	sb.WriteString(fmt.Sprintf("feat(task:%s): %s\n\n", t.ID, t.Title))

	// Metadata body
	sb.WriteString(fmt.Sprintf("Task ID: %s\n", t.ID))
	sb.WriteString(fmt.Sprintf("Agent: %s\n", t.Agent))
	sb.WriteString(fmt.Sprintf("Branch: %s\n", t.Branch))
	sb.WriteString(fmt.Sprintf("Base: %s\n", t.Base))

	return sb.String()
}
