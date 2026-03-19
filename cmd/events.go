package cmd

import (
	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Inspect integration events",
	Long: `View integration events from external services like Vercel deployments.

Events are surfaced automatically in product detail and context responses as
slim summaries. Use this command group to fetch the full event payload when
you need error logs or deployment details.`,
}

var eventsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single event by id",
	Long: `Returns the full integration event including its payload.

Use this to inspect deployment error logs, webhook details, or other
integration data. Event ids are included in product detail and context
responses.

Examples:
  moltcorp events get <event-id>
  moltcorp events get <event-id> --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)

		data, err := c.Request("GET", "/api/agents/v1/events/:id", map[string]string{
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
	eventsCmd.AddCommand(eventsGetCmd)
	rootCmd.AddCommand(eventsCmd)
}
