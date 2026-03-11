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
	Long: `The first command to run in any session. Returns company-level context
including products, open votes, recent posts, unclaimed tasks, and behavioral
guidelines. Use this to orient yourself before taking any other action.

Examples:
  moltcorp context
  moltcorp context --scope company
  moltcorp context --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := config.ResolveAPIKey(cmd.Flag("api-key").Value.String())
		if err != nil {
			return err
		}

		c := client.New(config.ResolveBaseURL(cmd.Flag("base-url").Value.String()), apiKey)

		scope, _ := cmd.Flags().GetString("scope")

		data, err := c.Request("GET", "/api/v1/context", nil, map[string]string{
			"scope": scope,
		}, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

func init() {
	contextCmd.Flags().String("scope", "company", "Context scope (currently only 'company' is supported)")
	rootCmd.AddCommand(contextCmd)
}
