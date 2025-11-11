package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kernel-labs-ai/awt/internal/errors"
	"github.com/kernel-labs-ai/awt/internal/logger"
	"github.com/kernel-labs-ai/awt/internal/repo"
	"github.com/kernel-labs-ai/awt/internal/task"
	"github.com/spf13/cobra"
)

// CopyOptions contains options for the copy command
type CopyOptions struct {
	RepoPath   string
	TaskID     string
	Files      []string
	Source     string
	OutputJSON bool
}

// CopyResult represents the output of the copy command
type CopyResult struct {
	TaskID       string   `json:"task_id"`
	FilesCopied  []string `json:"files_copied"`
	WorktreePath string   `json:"worktree_path"`
}

// NewTaskCopyCmd creates the task copy command
func NewTaskCopyCmd() *cobra.Command {
	opts := &CopyOptions{}

	cmd := &cobra.Command{
		Use:   "copy <task-id> <file> [file...]",
		Short: "Copy files into a task's worktree",
		Long: `Copy files from the current directory (or --source) into a task's worktree.

This is useful for copying files that are git-ignored (like .env files)
into a task's worktree so agents can use them.

The command will:
  1. Find the task by ID
  2. Locate the task's worktree
  3. Copy the specified files, preserving directory structure

Example:
  awt task copy my-task .env
  awt task copy my-task .env config/local.json
  awt task copy my-task .env --source=/path/to/source`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.TaskID = args[0]
			opts.Files = args[1:]
			return runTaskCopy(opts)
		},
	}

	cmd.Flags().StringVar(&opts.RepoPath, "repo", "", "path to Git repository")
	cmd.Flags().StringVar(&opts.Source, "source", "", "source directory (default: current directory)")
	cmd.Flags().BoolVar(&opts.OutputJSON, "json", false, "output result as JSON")

	return cmd
}

func runTaskCopy(opts *CopyOptions) error {
	log := logger.WithFields(map[string]string{
		"command": "task copy",
		"task_id": opts.TaskID,
	})
	log.Info("Copying files to task worktree")

	// Discover repository
	r, err := repo.DiscoverRepo(opts.RepoPath)
	if err != nil {
		return errors.RepoNotFound(opts.RepoPath)
	}

	store := task.NewTaskStore(r.GitCommonDir)

	// Load task
	t, err := store.Load(opts.TaskID)
	if err != nil {
		return errors.InvalidTaskID(opts.TaskID)
	}

	// Determine source directory
	sourceDir := opts.Source
	if sourceDir == "" {
		sourceDir = r.WorkTreeRoot
	} else {
		// Make source path absolute
		if !filepath.IsAbs(sourceDir) {
			sourceDir = filepath.Join(r.WorkTreeRoot, sourceDir)
		}
	}

	// Verify source directory exists
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		return fmt.Errorf("source directory does not exist: %s", sourceDir)
	}

	// Verify task worktree exists
	if _, err := os.Stat(t.WorktreePath); os.IsNotExist(err) {
		return fmt.Errorf("task worktree does not exist: %s\nUse 'awt task checkout %s' to create it", t.WorktreePath, opts.TaskID)
	}

	// Copy each file
	copiedFiles := []string{}
	for _, file := range opts.Files {
		sourcePath := filepath.Join(sourceDir, file)
		destPath := filepath.Join(t.WorktreePath, file)

		// Verify source file exists
		sourceInfo, err := os.Stat(sourcePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("source file does not exist: %s", file)
		}
		if err != nil {
			return fmt.Errorf("failed to stat source file %s: %w", file, err)
		}

		// Don't allow copying directories (for now - keep it simple)
		if sourceInfo.IsDir() {
			return fmt.Errorf("cannot copy directories (yet): %s", file)
		}

		// Create destination directory if needed
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}

		// Copy the file
		if err := copyFile(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", file, err)
		}

		copiedFiles = append(copiedFiles, file)
		log.Info("Copied file: %s", file)
	}

	// Output result
	if opts.OutputJSON {
		output := CopyResult{
			TaskID:       opts.TaskID,
			FilesCopied:  copiedFiles,
			WorktreePath: t.WorktreePath,
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Successfully copied %d file(s) to task %s:\n", len(copiedFiles), opts.TaskID)
		for _, file := range copiedFiles {
			fmt.Printf("  - %s\n", file)
		}
		fmt.Printf("\nWorktree: %s\n", t.WorktreePath)
	}

	return nil
}

// copyFile copies a file from src to dst, preserving permissions
func copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Get source file info for permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Create destination file
	destFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy contents
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	return nil
}
