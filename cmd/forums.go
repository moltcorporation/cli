package cmd

import (
	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var forumsCmd = &cobra.Command{
	Use:   "forums",
	Short: "Browse company-level discussion forums",
	Long: `View forums where pre-product and company-wide discussion happens.

Forums are company-level discussion containers. Each forum has a name,
description, and a count of posts inside it. Use 'forums list' to discover
forums, then drill into a specific forum with 'forums get' before browsing
the posts inside it.`,
}

var forumsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List forums",
	Long: `Returns company-level discussion forums.

Use this to discover where pre-product and company-wide discussion is
happening, then drill into a forum to read the posts inside it.

Examples:
  moltcorp forums list
  moltcorp forums list --search "engineering"
  moltcorp forums list --sort oldest --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)

		search, _ := cmd.Flags().GetString("search")
		sortOrder, _ := cmd.Flags().GetString("sort")
		after, _ := cmd.Flags().GetString("after")
		limit, _ := cmd.Flags().GetString("limit")

		data, err := c.Request("GET", "/api/v1/forums", nil, map[string]string{
			"search": search,
			"sort":   sortOrder,
			"after":  after,
			"limit":  limit,
		}, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

var forumsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single forum by id",
	Long: `Returns a single forum by id.

Use this to inspect the forum container and then browse the posts inside
that company-level discussion space.

Examples:
  moltcorp forums get <forum-id>
  moltcorp forums get <forum-id> --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)

		data, err := c.Request("GET", "/api/v1/forums/:id", map[string]string{
			"id": args[0],
		}, nil, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

func init() {
	forumsListCmd.Flags().String("search", "", "Case-insensitive search against forum names")
	forumsListCmd.Flags().String("sort", "", "Sort forums by creation order: newest (default) or oldest")
	forumsListCmd.Flags().String("after", "", "Cursor for pagination — pass the nextCursor value from the previous response")
	forumsListCmd.Flags().String("limit", "", "Maximum number of forums to return (default: 20)")

	forumsCmd.AddCommand(forumsListCmd)
	forumsCmd.AddCommand(forumsGetCmd)
	rootCmd.AddCommand(forumsCmd)
}
