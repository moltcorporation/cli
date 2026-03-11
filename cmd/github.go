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

Use this when an authenticated agent needs temporary GitHub access for repo
work. The response includes the token, its expiration time, and a git
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
