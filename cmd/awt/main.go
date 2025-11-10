package main

import (
	"fmt"
	"os"

	"github.com/decibelvc/awt/internal/commands"
	"github.com/spf13/cobra"
)

var (
	// Version information - injected at build time
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "awt",
		Short: "AWT - Agent WorkTrees",
		Long:  "A CLI tool that enables multiple AI agents to safely create, use, and hand off Git worktrees.",
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("awt version %s\n", Version)
			fmt.Printf("commit: %s\n", GitCommit)
			fmt.Printf("built: %s\n", BuildDate)
		},
	}

	// Add commands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(commands.NewInitCmd())
	rootCmd.AddCommand(commands.NewTaskCmd())
	rootCmd.AddCommand(commands.NewListCmd())
	rootCmd.AddCommand(commands.NewPruneCmd())
	rootCmd.AddCommand(commands.NewConfigCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
