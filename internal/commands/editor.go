package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kernel-labs-ai/awt/internal/errors"
	"github.com/kernel-labs-ai/awt/internal/repo"
	"github.com/kernel-labs-ai/awt/internal/task"
	"github.com/spf13/cobra"
)

// EditorOptions contains options for the editor command
type EditorOptions struct {
	RepoPath string
	TaskID   string
	Branch   string
	Editor   string
}

// NewTaskEditorCmd creates the task editor command
func NewTaskEditorCmd() *cobra.Command {
	opts := &EditorOptions{}

	cmd := &cobra.Command{
		Use:   "editor [task-id]",
		Short: "Open default editor in task's worktree",
		Long: `Open your default editor in the context of a task's worktree.

The task can be specified by:
  1. Providing the task ID as an argument
  2. Using --branch flag
  3. Inferring from current worktree (if in a worktree)

The editor is determined by (in order):
  1. --editor flag
  2. $EDITOR environment variable
  3. Common editors: code (VS Code), vim, nano, vi

Example:
  awt task editor 20250110-120000-abc123
  awt task editor --branch=awt/claude/20250110-120000-abc123
  awt task editor --editor=vim 20250110-120000-abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.TaskID = args[0]
			}
			return runTaskEditor(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "branch name")
	cmd.Flags().StringVar(&opts.Editor, "editor", "", "editor to use (defaults to $EDITOR)")

	return cmd
}

func runTaskEditor(opts *EditorOptions) error {
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

	// Verify worktree exists
	if _, err := os.Stat(t.WorktreePath); os.IsNotExist(err) {
		return errors.WorktreeNotFound(t.WorktreePath)
	}

	// Resolve worktree path to absolute path
	worktreePathAbs, err := filepath.Abs(t.WorktreePath)
	if err != nil {
		return fmt.Errorf("failed to resolve worktree path: %w", err)
	}

	// Determine editor to use
	editor := opts.Editor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		// Try common editors
		for _, e := range []string{"cursor","code", "vim", "nano", "vi"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found. Set $EDITOR or use --editor flag")
	}

	fmt.Printf("Opening %s in %s...\n", editor, worktreePathAbs)

	// Open editor
	cmd := exec.Command(editor, worktreePathAbs)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor: %w", err)
	}

	return nil
}
