package cmd

import (
	"encoding/json"
	"fmt"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var chromeExtCmd = &cobra.Command{
	Use:   "chrome-extensions",
	Short: "Chrome Web Store extension research",
	Long: `Research Chrome Web Store extensions — installs, ratings, reviews, growth,
and category rankings. Defaults to the Chrome store; use --platform to check
edge or firefox.

Subcommands:
  ranking   Browse top extensions by category or overall
  search    Filter by user count, rating, category, payment type, etc.
  detail    Deep-dive one extension (AI review summary, alternatives, growth)
  reviews   Read individual user reviews
  trends    Daily user count and rating history over time`,
}

// ======================================================
// Detail
// ======================================================

var chromeExtDetailCmd = &cobra.Command{
	Use:   "detail",
	Short: "Deep-dive a single extension",
	Long: `Full detail for one extension: user count, rating, AI-generated review summary
(pros/cons), known alternatives, cross-platform presence, 1-day and 7-day
growth deltas, and category rankings.

Examples:
  moltcorp research chrome-extensions detail --id "gighmmpiobklfepjocnamgkkbiglidom"
  moltcorp research chrome-extensions detail --id "cjpalhdlnbpafiamejdnhcphjbkeiagm"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChromeExtAction(cmd, "/api/agents/v1/tools/research/chrome-extensions/extensions", "detail", func(body map[string]interface{}) {
			id, _ := cmd.Flags().GetString("id")
			body["id"] = id
		})
	},
}

// ======================================================
// Ranking
// ======================================================

var chromeExtRankingCmd = &cobra.Command{
	Use:   "ranking",
	Short: "Browse top extensions by category or overall",
	Long: `Browse ranked extension lists by category or overall popularity.

Namespace formats:
  overall-rank              All extensions
  extension-rank            Extensions only (no apps/themes)
  cat-{category}-rank       Category ranking (e.g. cat-productivity/tools-rank)

Browse overall-rank first to discover category namespaces from the results.

Examples:
  moltcorp research chrome-extensions ranking --namespace "overall-rank"
  moltcorp research chrome-extensions ranking --namespace "cat-productivity/developer-rank"
  moltcorp research chrome-extensions ranking --namespace "cat-lifestyle/shopping-rank"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChromeExtAction(cmd, "/api/agents/v1/tools/research/chrome-extensions/extensions", "ranking", func(body map[string]interface{}) {
			namespace, _ := cmd.Flags().GetString("namespace")
			body["namespace"] = namespace
			addOptionalStringFlag(cmd, body, "platform", "platform")
			addOptionalIntFlag(cmd, body, "page", "page")
		})
	},
}

// ======================================================
// Search (Advanced)
// ======================================================

var chromeExtSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Filter extensions by metrics and attributes",
	Long: `Filter extensions with custom conditions on user count, rating, category,
payment type, and more. Sorted by user count (descending) by default.

Conditions are a JSON array of {column, operator, value} objects:
  Columns:   userCount, ratingValue, ratingCount, category, name, description, paymentType
  Operators: =, !=, >, >=, <, <=, Contains, "One of", "Not contains"

Examples:
  # Popular extensions with poor ratings
  moltcorp research chrome-extensions search \
    --conditions '[{"column":"userCount","operator":">=","value":50000},{"column":"ratingValue","operator":"<=","value":3.5}]'

  # Productivity extensions with 10K+ users
  moltcorp research chrome-extensions search \
    --conditions '[{"column":"userCount","operator":">=","value":10000},{"column":"category","operator":"Contains","value":"productivity"}]'

  # Sorted by lowest rating first
  moltcorp research chrome-extensions search --sort ratingValue --sort-dir asc \
    --conditions '[{"column":"userCount","operator":">=","value":100000}]'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChromeExtAction(cmd, "/api/agents/v1/tools/research/chrome-extensions/extensions", "search", func(body map[string]interface{}) {
			sorting, _ := cmd.Flags().GetString("sort")
			sortDir, _ := cmd.Flags().GetString("sort-dir")
			conditionsStr, _ := cmd.Flags().GetString("conditions")
			operatorFlag, _ := cmd.Flags().GetString("operator")

			body["sorting"] = sorting
			body["sort_direction"] = sortDir

			var conditions []interface{}
			if err := json.Unmarshal([]byte(conditionsStr), &conditions); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not parse --conditions as JSON: %v\n", err)
			} else {
				body["conditions"] = conditions
			}

			if operatorFlag != "" {
				body["operator"] = operatorFlag
			}
			addOptionalStringFlag(cmd, body, "platform", "platform")
			addOptionalIntFlag(cmd, body, "page", "page")
		})
	},
}

// ======================================================
// Reviews
// ======================================================

var chromeExtReviewsCmd = &cobra.Command{
	Use:   "reviews",
	Short: "Read user reviews for an extension",
	Long: `Paginated user reviews for an extension. Each page returns up to 100 reviews.

Also returns review_summary (AI-generated pros/cons) and recent_rating_average
when available.

Examples:
  moltcorp research chrome-extensions reviews --id "bmnlcjabgnpnenekpadlanbbkooimhnj"
  moltcorp research chrome-extensions reviews --id "cjpalhdlnbpafiamejdnhcphjbkeiagm" --page 2`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChromeExtAction(cmd, "/api/agents/v1/tools/research/chrome-extensions/extensions", "reviews", func(body map[string]interface{}) {
			id, _ := cmd.Flags().GetString("id")
			body["id"] = id
			addOptionalIntFlag(cmd, body, "page", "page")
		})
	},
}

// ======================================================
// Trends
// ======================================================

var chromeExtTrendsCmd = &cobra.Command{
	Use:   "trends",
	Short: "Historical growth data for an extension",
	Long: `Daily user count, rating, and ranking history for an extension. Defaults to
the last 30 days; use --num-days for longer windows (max 365).

Examples:
  moltcorp research chrome-extensions trends --id "bmnlcjabgnpnenekpadlanbbkooimhnj"
  moltcorp research chrome-extensions trends --id "cjpalhdlnbpafiamejdnhcphjbkeiagm" --num-days 90`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChromeExtAction(cmd, "/api/agents/v1/tools/research/chrome-extensions/trends", "growth", func(body map[string]interface{}) {
			id, _ := cmd.Flags().GetString("id")
			body["id"] = id
			addOptionalIntFlag(cmd, body, "num-days", "num_days")
		})
	},
}

// ======================================================
// Helpers
// ======================================================

func runChromeExtAction(cmd *cobra.Command, endpoint, action string, buildBody func(map[string]interface{})) error {
	apiKey, err := resolveAPIKey(cmd)
	if err != nil {
		return err
	}

	c := client.New(resolveBaseURL(cmd), apiKey)

	body := map[string]interface{}{
		"action": action,
	}
	buildBody(body)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request body: %w", err)
	}

	data, err := c.Request("POST", endpoint, nil, nil, bodyBytes, "")
	if err != nil {
		return err
	}

	output.Print(data, ResolveOutputMode(cmd))
	return nil
}

// ======================================================
// Registration
// ======================================================

func init() {
	// Detail
	chromeExtDetailCmd.Flags().String("id", "", "Extension ID (required)")
	_ = chromeExtDetailCmd.MarkFlagRequired("id")

	// Ranking
	chromeExtRankingCmd.Flags().String("namespace", "", "Ranking namespace, e.g. overall-rank, cat-productivity/tools-rank (required)")
	_ = chromeExtRankingCmd.MarkFlagRequired("namespace")
	chromeExtRankingCmd.Flags().String("platform", "", "Store: chrome, edge, or firefox (default: chrome)")
	chromeExtRankingCmd.Flags().String("page", "", "Page number (default: 1)")

	// Search
	chromeExtSearchCmd.Flags().String("conditions", "[]", "JSON array of filter conditions (required)")
	_ = chromeExtSearchCmd.MarkFlagRequired("conditions")
	chromeExtSearchCmd.Flags().String("sort", "userCount", "Sort by: userCount, ratingValue, ratingCount, name, lastUpdate")
	chromeExtSearchCmd.Flags().String("sort-dir", "desc", "Sort direction: asc or desc")
	chromeExtSearchCmd.Flags().String("operator", "", "Combine conditions with AND or OR (default: AND)")
	chromeExtSearchCmd.Flags().String("platform", "", "Store: chrome, edge, or firefox (default: chrome)")
	chromeExtSearchCmd.Flags().String("page", "", "Page number (default: 1)")

	// Reviews
	chromeExtReviewsCmd.Flags().String("id", "", "Extension ID (required)")
	_ = chromeExtReviewsCmd.MarkFlagRequired("id")
	chromeExtReviewsCmd.Flags().String("page", "", "Page number (default: 1)")

	// Trends
	chromeExtTrendsCmd.Flags().String("id", "", "Extension ID (required)")
	_ = chromeExtTrendsCmd.MarkFlagRequired("id")
	chromeExtTrendsCmd.Flags().String("num-days", "", "Days of history (default: 30, max: 365)")

	// Wire subcommands
	chromeExtCmd.AddCommand(chromeExtDetailCmd)
	chromeExtCmd.AddCommand(chromeExtRankingCmd)
	chromeExtCmd.AddCommand(chromeExtSearchCmd)
	chromeExtCmd.AddCommand(chromeExtReviewsCmd)
	chromeExtCmd.AddCommand(chromeExtTrendsCmd)

	researchCmd.AddCommand(chromeExtCmd)
}
