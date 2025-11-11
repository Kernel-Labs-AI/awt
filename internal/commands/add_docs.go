package commands

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed USAGE.md
var usageMD string

// NewAddDocsCmd creates the add-docs command
func NewAddDocsCmd() *cobra.Command {
	var outputPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "add-docs",
		Short: "Copy AWT usage documentation to your project",
		Long: `Copy the AWT usage documentation (USAGE.md) to your project directory.

This creates a file named AWT_USAGE.md in the current directory (or specified path)
containing comprehensive documentation for all AWT commands.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddDocs(outputPath, force)
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", ".", "output directory or file path")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing file without confirmation")

	return cmd
}

func runAddDocs(outputPath string, force bool) error {
	// Determine the target file path
	targetPath := outputPath
	info, err := os.Stat(outputPath)
	if err == nil && info.IsDir() {
		// If output is a directory, append the filename
		targetPath = filepath.Join(outputPath, "AWT_USAGE.md")
	} else if err == nil {
		// File exists - use as-is
	} else if os.IsNotExist(err) {
		// Path doesn't exist - check if it looks like a directory or file
		if strings.HasSuffix(outputPath, string(filepath.Separator)) || outputPath == "." || outputPath == ".." {
			// It's a directory that doesn't exist
			if err := os.MkdirAll(outputPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", outputPath, err)
			}
			targetPath = filepath.Join(outputPath, "AWT_USAGE.md")
		} else {
			// Check if parent directory exists
			parentDir := filepath.Dir(outputPath)
			if parentDir != "." && parentDir != "" {
				if _, err := os.Stat(parentDir); os.IsNotExist(err) {
					if err := os.MkdirAll(parentDir, 0755); err != nil {
						return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
					}
				}
			}
			// Assume it's a file path
			targetPath = outputPath
		}
	} else {
		return fmt.Errorf("failed to check output path: %w", err)
	}

	// Check if target file already exists
	if _, err := os.Stat(targetPath); err == nil && !force {
		// File exists and force flag is not set - prompt for confirmation
		fmt.Printf("File %s already exists. Overwrite? [y/N]: ", targetPath)
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			// If there's an error reading input (e.g., EOF), treat it as "no"
			fmt.Println("\nOperation cancelled.")
			return nil
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// Write the file
	if err := os.WriteFile(targetPath, []byte(usageMD), 0644); err != nil {
		return fmt.Errorf("failed to write documentation file: %w", err)
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		absPath = targetPath
	}

	fmt.Println("✓ Documentation copied successfully!")
	fmt.Printf("  File: %s\n", absPath)

	return nil
}
