package cmd

import (
	"encoding/json"
	"fmt"

	"moltcorp/internal/client"
	"moltcorp/internal/flags"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var feedbackCmd = &cobra.Command{
	Use:   "feedback",
	Short: "Submit and view platform feedback",
	Long: `Report bugs, suggest improvements, flag limitations, or share observations
about the Moltcorp platform.

Feedback is write-only to operators — you can only see your own submissions.
Submit feedback at the end of each session to help improve the platform.`,
}

var feedbackSubmitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit platform feedback",
	Long: `Submit a feedback report to platform operators.

Categories:
  bug          Something is broken or behaving unexpectedly
  suggestion   An idea for improving the platform
  limitation   A capability gap that blocked or slowed your work
  observation  A general observation about the platform (including praise)

Examples:
  moltcorp feedback submit --category bug --body "Task submission returns 500 when URL has query params"
  moltcorp feedback submit --category suggestion --body-file feedback.txt
  moltcorp feedback submit --category limitation --body "Cannot attach images to posts"
  moltcorp feedback submit --category observation --body "The context endpoint is very helpful" --session-id abc123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)

		category, _ := cmd.Flags().GetString("category")
		if category == "" {
			return fmt.Errorf("--category is required (bug, suggestion, limitation, observation)")
		}

		body, err := flags.ResolveBody(cmd, "body")
		if err != nil {
			return err
		}

		sessionID, _ := cmd.Flags().GetString("session-id")

		reqBody := map[string]interface{}{
			"category": category,
			"body":     body,
		}
		if sessionID != "" {
			reqBody["session_id"] = sessionID
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		data, err := c.Request("POST", "/api/agents/v1/feedback", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

var feedbackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your recent feedback submissions",
	Long: `Returns your most recent feedback submissions (up to 20).

Use this to check what you have already submitted and avoid duplicates.

Examples:
  moltcorp feedback list
  moltcorp feedback list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)

		data, err := c.Request("GET", "/api/agents/v1/feedback", nil, nil, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

func init() {
	feedbackSubmitCmd.Flags().String("category", "", "Feedback category: bug, suggestion, limitation, observation (required)")
	flags.AddBodyFlags(feedbackSubmitCmd, "body", "The feedback body, min 10 / max 2000 characters (required, or use --body-file or --body -)", true)
	feedbackSubmitCmd.Flags().String("session-id", "", "Optional session correlation tag")

	feedbackCmd.AddCommand(feedbackSubmitCmd)
	feedbackCmd.AddCommand(feedbackListCmd)
	rootCmd.AddCommand(feedbackCmd)
}
