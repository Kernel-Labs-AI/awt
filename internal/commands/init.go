package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kernel-labs-ai/awt/internal/errors"
	"github.com/kernel-labs-ai/awt/internal/repo"
	"github.com/spf13/cobra"
)

const (
	// AWTVersion is the AWT metadata version
	AWTVersion = "1"
)

// NewInitCmd creates the init command
func NewInitCmd() *cobra.Command {
	var repoPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize AWT in a Git repository",
		Long: `Initialize AWT (Agent WorkTrees) in the current Git repository.

This creates the necessary directory structure and metadata files:
  $GIT_COMMON/awt/tasks/   - Task metadata
  $GIT_COMMON/awt/locks/   - Lock files
  $GIT_COMMON/awt/version  - Version file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(repoPath)
		},
	}

	cmd.Flags().StringVar(&repoPath, "repo", "", "path to Git repository (default: current directory)")

	return cmd
}

func runInit(repoPath string) error {
	// Discover the Git repository
	r, err := repo.DiscoverRepo(repoPath)
	if err != nil {
		return errors.RepoNotFound(repoPath)
	}

	// Create AWT directory structure
	awtDir := filepath.Join(r.GitCommonDir, "awt")
	tasksDir := filepath.Join(awtDir, "tasks")
	locksDir := filepath.Join(awtDir, "locks")

	// Check if already initialized
	versionFile := filepath.Join(awtDir, "version")
	if _, err := os.Stat(versionFile); err == nil {
		fmt.Println("AWT is already initialized in this repository")
		fmt.Printf("  AWT directory: %s\n", awtDir)
		return nil
	}

	// Create directories
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}
	if err := os.MkdirAll(locksDir, 0755); err != nil {
		return fmt.Errorf("failed to create locks directory: %w", err)
	}

	// Write version file
	if err := os.WriteFile(versionFile, []byte(AWTVersion+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write version file: %w", err)
	}

	// Success
	fmt.Println("AWT initialized successfully!")
	fmt.Printf("  Repository: %s\n", r.WorkTreeRoot)
	fmt.Printf("  AWT directory: %s\n", awtDir)
	fmt.Printf("  Version: %s\n", AWTVersion)
	fmt.Println()
	fmt.Println("Tip: Run 'awt add-docs' to copy usage instructions to your project directory.")

	return nil
}
