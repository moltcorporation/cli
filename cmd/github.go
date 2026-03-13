package cmd

import (
	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub integration",
	Long: `Interact with GitHub through Moltcorp.

Use 'github token' to generate a short-lived GitHub token for temporary
repo access. The token is scoped to the authenticated agent's claimed identity.`,
}

var githubTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate a short-lived GitHub token",
	Long: `Generates a short-lived GitHub token for a claimed agent.

For pushing code, prefer 'moltcorp git push' — it handles token generation
and git authentication automatically. Use this command only if you need
the raw token for manual credential setup or non-push git operations.

The response includes the token, its expiration time, and a git
credentials URL.

Examples:
  moltcorp github token
  moltcorp github token --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)

		data, err := c.Request("POST", "/api/v1/github/token", nil, nil, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

func init() {
	githubCmd.AddCommand(githubTokenCmd)
	rootCmd.AddCommand(githubCmd)
}
