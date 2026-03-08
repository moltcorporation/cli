package cmd

import (
	"os"

	"api-cli/internal/config"
	"api-cli/internal/updater"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   config.CLIName,
	Short: "CLI for the API",
	Long:  "A command-line interface for interacting with the API.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Startup update check — skip for certain commands
		name := cmd.Name()
		if name == "update" || name == "version" || name == "configure" || name == "help" {
			return
		}
		updater.CheckForUpdateNotice(cmd.Root().Name())
	},
	SilenceUsage: true,
}

func init() {
	rootCmd.PersistentFlags().String("api-key", "", "API key (overrides env and config)")
	rootCmd.PersistentFlags().String("base-url", "", "Override the API base URL")
	rootCmd.PersistentFlags().String("output", "table", "Output format: table or json")
	rootCmd.PersistentFlags().Bool("json", false, "Output as JSON (shorthand for --output json)")
	rootCmd.PersistentFlags().Bool("raw", false, "Print raw API response without formatting")

	rootCmd.AddCommand(configureCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// ResolveOutputMode returns the output mode based on flags.
func ResolveOutputMode(cmd *cobra.Command) string {
	raw, _ := cmd.Flags().GetBool("raw")
	if raw {
		return "raw"
	}
	jsonFlag, _ := cmd.Flags().GetBool("json")
	if jsonFlag {
		return "json"
	}
	mode, _ := cmd.Flags().GetString("output")
	return mode
}

// AddCommand registers a command on the root command.
// This is the entry point for the coding agent to register generated commands.
func AddCommand(cmds ...*cobra.Command) {
	rootCmd.AddCommand(cmds...)
}

// ExitError prints an error to stderr and exits with code 1.
func ExitError(msg string) {
	os.Stderr.WriteString("Error: " + msg + "\n")
	os.Exit(1)
}
