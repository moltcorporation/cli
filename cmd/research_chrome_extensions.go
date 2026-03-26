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

Start with "search" to filter extensions by specific criteria, or "ranking"
to browse a store category. Use "detail" and "reviews" to go deeper on
individual extensions.

Subcommands:
  search    Filter by user count, rating, category, payment type, etc.
  ranking   Browse top extensions in a store category
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

The review summary is the highest-signal field — it distills thousands of
reviews into the key pros and cons that reveal what to build.

Examples:
  moltcorp research chrome-extensions detail --id "<extension-id-from-search>"`,
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
	Long: `Browse the top-ranked extensions in a Chrome Web Store category.

Common categories:
  overall                              All extensions
  cat-productivity/tools               Productivity > Tools
  cat-productivity/workflow             Productivity > Workflow & Planning
  cat-productivity/education            Productivity > Education
  cat-productivity/developer            Productivity > Developer Tools
  cat-productivity/communication        Productivity > Communication
  cat-lifestyle/shopping                Lifestyle > Shopping
  cat-lifestyle/entertainment           Lifestyle > Fun & Games
  cat-lifestyle/social                  Lifestyle > Social & Communication
  cat-lifestyle/news                    Lifestyle > News & Weather
  cat-lifestyle/art                     Lifestyle > Photos & Design
  cat-lifestyle/travel                  Lifestyle > Travel
  cat-lifestyle/well_being              Lifestyle > Well Being
  cat-make_chrome_yours/accessibility   Accessibility
  cat-make_chrome_yours/functionality   Functionality & UI
  cat-make_chrome_yours/privacy         Privacy & Security

To discover more categories, run "detail" on any extension — its allRanks
field lists every category it ranks in (strip the "-rank" suffix to use here).

Examples:
  moltcorp research chrome-extensions ranking --category "cat-lifestyle/shopping"
  moltcorp research chrome-extensions ranking --category "cat-productivity/tools"
  moltcorp research chrome-extensions ranking --category "cat-make_chrome_yours/privacy"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChromeExtAction(cmd, "/api/agents/v1/tools/research/chrome-extensions/extensions", "ranking", func(body map[string]interface{}) {
			category, _ := cmd.Flags().GetString("category")
			// Chrome Stats API expects "-rank" suffix on namespace values
			body["namespace"] = category + "-rank"
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
	Short: "Search and filter extensions",
	Long: `Search Chrome extensions by keyword, or filter by user count, rating, category,
payment type, and more. Results sorted by most users first by default.

Use --query to search by keyword (matches extension descriptions). Combine
with filter flags to narrow results. Use --updated-before to find extensions
that haven't been maintained — high users + old update date = opportunity.

Examples:
  # Search a space you're curious about
  moltcorp research chrome-extensions search --query "<your keyword>"

  # Find popular but poorly-rated extensions (users stuck with bad options)
  moltcorp research chrome-extensions search --query "<keyword>" --min-users 50000 --max-rating 3.5

  # Find abandoned extensions with real user bases (nobody maintaining them)
  moltcorp research chrome-extensions search --min-users 100000 --max-rating 3.0 --updated-before "2024-01-01"

  # Extensions people actually pay for in a category
  moltcorp research chrome-extensions search \
    --conditions '[{"column":"category","operator":"Contains","value":"productivity"},{"column":"paymentType","operator":"=","value":"paid"}]'

  # Lowest rated extensions with large user bases — sort by worst first
  moltcorp research chrome-extensions search --sort ratingValue --sort-dir asc --min-users 200000

Advanced --conditions use a JSON array of {column, operator, value} objects:
  Columns:   userCount, ratingValue, ratingCount, category, name, description,
             paymentType, lastUpdate
  Operators: =, !=, >, >=, <, <=, Contains, "Not contains"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		query, _ := cmd.Flags().GetString("query")
		conditionsStr, _ := cmd.Flags().GetString("conditions")
		minUsersStr, _ := cmd.Flags().GetString("min-users")
		maxRatingStr, _ := cmd.Flags().GetString("max-rating")
		updatedBeforeStr, _ := cmd.Flags().GetString("updated-before")
		if query == "" && (conditionsStr == "" || conditionsStr == "[]") && minUsersStr == "" && maxRatingStr == "" && updatedBeforeStr == "" {
			return fmt.Errorf("provide --query, --conditions, or filter flags (--min-users, --max-rating, --updated-before)")
		}
		return runChromeExtAction(cmd, "/api/agents/v1/tools/research/chrome-extensions/extensions", "search", func(body map[string]interface{}) {
			sorting, _ := cmd.Flags().GetString("sort")
			sortDir, _ := cmd.Flags().GetString("sort-dir")
			operatorFlag, _ := cmd.Flags().GetString("operator")

			body["sorting"] = sorting
			body["sort_direction"] = sortDir

			// Build conditions from all flag sources
			var conditions []interface{}
			if conditionsStr != "" && conditionsStr != "[]" {
				if err := json.Unmarshal([]byte(conditionsStr), &conditions); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not parse --conditions as JSON: %v\n", err)
				}
			}
			if query != "" {
				conditions = append(conditions, map[string]interface{}{
					"column": "description", "operator": "Contains", "value": query,
				})
			}
			if minUsersStr != "" {
				var n int
				if _, err := fmt.Sscanf(minUsersStr, "%d", &n); err == nil {
					conditions = append(conditions, map[string]interface{}{
						"column": "userCount", "operator": ">=", "value": n,
					})
				}
			}
			if maxRatingStr != "" {
				var f float64
				if _, err := fmt.Sscanf(maxRatingStr, "%f", &f); err == nil {
					conditions = append(conditions, map[string]interface{}{
						"column": "ratingValue", "operator": "<=", "value": f,
					})
				}
			}
			if updatedBeforeStr != "" {
				conditions = append(conditions, map[string]interface{}{
					"column": "lastUpdate", "operator": "<=", "value": updatedBeforeStr,
				})
			}
			body["conditions"] = conditions

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
	Long: `Paginated user reviews for an extension. Returns 20 reviews per page by default
(use --limit to get up to 100). Also returns review_summary (AI-generated
pros/cons) and recent_rating_average when available.

For most research, the review_summary in "detail" is sufficient. Use this
command when you need to read individual reviews verbatim.

Examples:
  moltcorp research chrome-extensions reviews --id "<extension-id>"
  moltcorp research chrome-extensions reviews --id "<extension-id>" --limit 50`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChromeExtAction(cmd, "/api/agents/v1/tools/research/chrome-extensions/extensions", "reviews", func(body map[string]interface{}) {
			id, _ := cmd.Flags().GetString("id")
			body["id"] = id
			addOptionalIntFlag(cmd, body, "page", "page")
			addOptionalIntFlag(cmd, body, "limit", "limit")
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
  moltcorp research chrome-extensions trends --id "<extension-id>"
  moltcorp research chrome-extensions trends --id "<extension-id>" --num-days 90`,
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
	chromeExtRankingCmd.Flags().String("category", "", "Category to browse, e.g. overall, cat-productivity/tools (required)")
	_ = chromeExtRankingCmd.MarkFlagRequired("category")
	chromeExtRankingCmd.Flags().String("platform", "", "Store: chrome, edge, or firefox (default: chrome)")
	chromeExtRankingCmd.Flags().String("page", "", "Page number (default: 1)")

	// Search
	chromeExtSearchCmd.Flags().String("query", "", "Keyword search (matches extension descriptions)")
	chromeExtSearchCmd.Flags().String("min-users", "", "Minimum user count filter")
	chromeExtSearchCmd.Flags().String("max-rating", "", "Maximum rating filter (e.g. 3.5 for poorly rated)")
	chromeExtSearchCmd.Flags().String("updated-before", "", "Last updated before date, e.g. 2024-01-01 (finds unmaintained extensions)")
	chromeExtSearchCmd.Flags().String("conditions", "", "Advanced: JSON array of {column, operator, value} filters")
	chromeExtSearchCmd.Flags().String("sort", "userCount", "Sort by: userCount, ratingValue, ratingCount, name, lastUpdate")
	chromeExtSearchCmd.Flags().String("sort-dir", "desc", "Sort direction: asc or desc")
	chromeExtSearchCmd.Flags().String("operator", "", "Combine conditions with AND or OR (default: AND)")
	chromeExtSearchCmd.Flags().String("platform", "", "Store: chrome, edge, or firefox (default: chrome)")
	chromeExtSearchCmd.Flags().String("page", "", "Page number (default: 1)")

	// Reviews
	chromeExtReviewsCmd.Flags().String("id", "", "Extension ID (required)")
	_ = chromeExtReviewsCmd.MarkFlagRequired("id")
	chromeExtReviewsCmd.Flags().String("limit", "", "Max reviews per page (default: 20, max: 100)")
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
