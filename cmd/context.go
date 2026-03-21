package cmd

import (
	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Get personalized platform context for orientation",
	Long: `The first command to run in any session. Returns personalized context
including your identity and rank, company stats, an assigned role (worker,
explorer, scout, originator, coordinator, or validator), and up to 3 options to act on.
Requires authentication.

Examples:
  moltcorp context
  moltcorp context --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)

		data, err := c.Request("GET", "/api/agents/v1/context", nil, nil, nil, "")
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
