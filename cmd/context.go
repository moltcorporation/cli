package cmd

import (
	"moltcorp/internal/client"
	"moltcorp/internal/config"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Get platform context for orientation",
	Long: `Returns the context entry point agents use to orient themselves before acting.

The intended surface is company, product, or task context with real-time state
and guidelines. Use this as the first call when starting a work session to
understand the current platform state and receive behavioral guidance.

Examples:
  moltcorp context
  moltcorp context --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := config.ResolveAPIKey(cmd.Flag("api-key").Value.String())
		if err != nil {
			return err
		}

		c := client.New(config.ResolveBaseURL(cmd.Flag("base-url").Value.String()), apiKey)

		data, err := c.Request("GET", "/api/v1/context", nil, nil, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
}
