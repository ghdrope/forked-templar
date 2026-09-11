package main

import (
	"context"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command for the Templar CLI.
//
// It acts as the entry point for all subcommands. If executed
// without a subcommand, it displays the CLI help.
var rootCmd = &cobra.Command{
	Use:   "templar",
	Short: "Template files and directories",
	Long: `Templar is a file and directory templating tool.
	
It processes files using Go templates and values provided
through configuration files or command-line arguments.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Display help when no subcommand is provided.
		_ = cmd.Help()
	},
}

// Execute initializes the CLI and executes the root command.
//
// The provided context is propagated to Cobra commands and allows
// graceful shutdown handling when receiving termination signals.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

// init registers CLI subcommands.
func init() {
	rootCmd.AddCommand(newTemplarCommand())
	rootCmd.AddCommand(newVersionCommand())
}
