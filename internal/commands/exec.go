package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/decibelvc/awt/internal/errors"
	"github.com/decibelvc/awt/internal/repo"
	"github.com/decibelvc/awt/internal/task"
	"github.com/spf13/cobra"
)

// ExecOptions contains options for the exec command
type ExecOptions struct {
	RepoPath string
	TaskID   string
	Branch   string
	Command  []string
}

// NewTaskExecCmd creates the task exec command
func NewTaskExecCmd() *cobra.Command {
	opts := &ExecOptions{}

	cmd := &cobra.Command{
		Use:   "exec [task-id] -- <command> [args...]",
		Short: "Execute a command in a task's worktree",
		Long: `Execute a command in the context of a task's worktree.

The task can be specified by:
  1. Providing the task ID as an argument
  2. Using --branch flag
  3. Inferring from current worktree (if in a worktree)

Commands are executed with:
  - Working directory set to the task's worktree root
  - Stdin/stdout/stderr connected to the parent process
  - Signals (SIGINT, SIGTERM) propagated to the child process
  - Exit code returned from the child process

Example:
  awt task exec 20250110-120000-abc123 -- make test
  awt task exec --branch=awt/claude/20250110-120000-abc123 -- git status
  awt task exec -- ls -la  # infer from current directory`,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse flags manually since we disabled flag parsing
			var taskID string
			var branch string
			var repoPath string
			var cmdArgs []string

			i := 0
			for i < len(args) {
				arg := args[i]

				if arg == "--" {
					// Everything after -- is the command
					if i+1 >= len(args) {
						return fmt.Errorf("no command specified after '--'")
					}
					cmdArgs = args[i+1:]
					break
				} else if arg == "--branch" {
					if i+1 >= len(args) {
						return fmt.Errorf("--branch requires a value")
					}
					branch = args[i+1]
					i += 2
				} else if arg == "--repo" {
					if i+1 >= len(args) {
						return fmt.Errorf("--repo requires a value")
					}
					repoPath = args[i+1]
					i += 2
				} else if arg == "-h" || arg == "--help" {
					cmd.Help()
					return nil
				} else {
					// Assume it's the task ID
					taskID = arg
					i++
				}
			}

			if len(cmdArgs) == 0 {
				return fmt.Errorf("missing '--' separator before command\nUsage: awt task exec [task-id] -- <command> [args...]")
			}

			opts.TaskID = taskID
			opts.Branch = branch
			opts.RepoPath = repoPath
			opts.Command = cmdArgs

			return runTaskExec(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "branch name")

	return cmd
}

func runTaskExec(opts *ExecOptions) error {
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

	// Execute command in worktree
	exitCode, err := executeCommand(worktreePathAbs, opts.Command)
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	// Exit with child process exit code
	if exitCode != 0 {
		os.Exit(exitCode)
	}

	return nil
}

// executeCommand executes a command in the specified directory with signal handling
func executeCommand(workDir string, cmdArgs []string) (int, error) {
	// Create command
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Dir = workDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start command
	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("failed to start command: %w", err)
	}

	// Context for cleanup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals in goroutine
	go func() {
		select {
		case sig := <-sigChan:
			// Propagate signal to child process
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		case <-ctx.Done():
			return
		}
	}()

	// Wait for command to complete
	err := cmd.Wait()

	// Stop signal handling
	signal.Stop(sigChan)
	cancel()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return 1, err
		}
	}

	return exitCode, nil
}
