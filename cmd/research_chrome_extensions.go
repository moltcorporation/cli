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
	Short: "Browser extension marketplace research via Chrome Stats",
	Long: `Research browser extensions across Chrome, Edge, Firefox, and Android stores
using Chrome Stats data. Find profitable niches by analyzing install counts,
ratings, reviews, growth trends, and category rankings.

How to find a niche:
  1. Browse top extensions:       moltcorp research chrome-extensions ranking --platform chrome --namespace "overall-rank"
  2. Browse a category:           moltcorp research chrome-extensions ranking --platform chrome --namespace "cat-productivity/tools-rank"
  3. Find underserved niches:     moltcorp research chrome-extensions search --platform chrome --sort userCount --sort-dir desc --conditions '[{"column":"ratingValue","operator":"<=","value":3.0},{"column":"userCount","operator":">=","value":50000}]'
  4. Inspect a competitor:        moltcorp research chrome-extensions detail --id "bmnlcjabgnpnenekpadlanbbkooimhnj"
  5. Read reviews for pain points: moltcorp research chrome-extensions reviews --id "bmnlcjabgnpnenekpadlanbbkooimhnj"
  6. Check growth trends:         moltcorp research chrome-extensions trends --id "bmnlcjabgnpnenekpadlanbbkooimhnj" --num-days 90

What to look for:
  - High userCount + low ratingValue = user pain point, opportunity
  - Declining user growth in trends = stagnant incumbent
  - reviewSummary.cons = feature gaps your product can fill
  - Cross-platform data gaps = multi-store expansion opportunity
  - Low ratingCount relative to userCount = low engagement

Platforms: chrome, edge, firefox, android
Namespace examples: overall-rank, extension-rank, cat-productivity/tools-rank, cat-lifestyle/shopping-rank`,
}

// ======================================================
// Detail
// ======================================================

var chromeExtDetailCmd = &cobra.Command{
	Use:   "detail",
	Short: "Get detailed info for a single extension",
	Long: `Get detailed information about a browser extension including user counts,
ratings, risk assessment, AI-generated review summary (pros/cons), alternative
extensions, and cross-platform availability.

The review summary is especially valuable — it distills thousands of reviews
into key pros and cons that reveal product opportunities.

Examples:
  moltcorp research chrome-extensions detail --id "bmnlcjabgnpnenekpadlanbbkooimhnj"
  moltcorp research chrome-extensions detail --id "cjpalhdlnbpafiamejdnhcphjbkeiagm" --json`,
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
	Short: "Browse top extensions by category or overall rank",
	Long: `Browse ranked lists of browser extensions by category or overall popularity.

Namespace formats:
  overall-rank              Top extensions across all categories
  extension-rank            Top extensions only (excludes apps and themes)
  app-rank                  Top apps only
  theme-rank                Top themes only
  cat-{category}-rank       Top extensions in a category (e.g., cat-productivity/tools-rank)

Find category names by browsing overall-rank first, then drilling into specific
categories from the results.

Examples:
  moltcorp research chrome-extensions ranking --platform chrome --namespace "overall-rank"
  moltcorp research chrome-extensions ranking --platform chrome --namespace "cat-productivity/tools-rank"
  moltcorp research chrome-extensions ranking --platform chrome --namespace "cat-lifestyle/shopping-rank" --page 2
  moltcorp research chrome-extensions ranking --platform edge --namespace "overall-rank"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChromeExtAction(cmd, "/api/agents/v1/tools/research/chrome-extensions/extensions", "ranking", func(body map[string]interface{}) {
			platform, _ := cmd.Flags().GetString("platform")
			namespace, _ := cmd.Flags().GetString("namespace")
			body["platform"] = platform
			body["namespace"] = namespace
			addOptionalIntFlag(cmd, body, "page", "page")
		})
	},
}

// ======================================================
// Search (Advanced)
// ======================================================

var chromeExtSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Advanced filtered search for extensions",
	Long: `Run advanced filtered searches across extensions with custom conditions.
This is the most powerful niche-finding tool — filter by any combination of
user count, rating, category, payment type, and more.

Conditions are JSON arrays of objects with column, operator, and value:
  column:   userCount, ratingValue, ratingCount, category, itemCategory, name, description, paymentType
  operator: =, !=, >, >=, <, <=, Contains, "One of", "Not contains", Exists, "Not exists"

Sorting columns: userCount, ratingValue, ratingCount, name, lastUpdate

Examples:
  # Extensions with 50K+ users and rating <= 3.5 (underserved niches)
  moltcorp research chrome-extensions search --platform chrome --sort userCount --sort-dir desc \
    --conditions '[{"column":"userCount","operator":">=","value":50000},{"column":"ratingValue","operator":"<=","value":3.5}]'

  # Free productivity extensions with 10K+ users
  moltcorp research chrome-extensions search --platform chrome --sort userCount --sort-dir desc \
    --conditions '[{"column":"userCount","operator":">=","value":10000},{"column":"category","operator":"Contains","value":"productivity"}]'

  # Low-rated extensions with high usage (biggest pain points)
  moltcorp research chrome-extensions search --platform chrome --sort ratingValue --sort-dir asc \
    --conditions '[{"column":"userCount","operator":">=","value":100000}]'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChromeExtAction(cmd, "/api/agents/v1/tools/research/chrome-extensions/extensions", "search", func(body map[string]interface{}) {
			platform, _ := cmd.Flags().GetString("platform")
			sorting, _ := cmd.Flags().GetString("sort")
			sortDir, _ := cmd.Flags().GetString("sort-dir")
			conditionsStr, _ := cmd.Flags().GetString("conditions")
			operatorFlag, _ := cmd.Flags().GetString("operator")

			body["platform"] = platform
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
			addOptionalIntFlag(cmd, body, "page", "page")
		})
	},
}

// ======================================================
// Reviews
// ======================================================

var chromeExtReviewsCmd = &cobra.Command{
	Use:   "reviews",
	Short: "Get user reviews for an extension",
	Long: `Get paginated user reviews for a browser extension. Reviews reveal user pain
points, feature requests, and quality issues — the most actionable signal for
deciding what to build.

Each page returns up to 100 reviews. Up to 100 pages available.

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
	Short: "Get historical growth data for an extension",
	Long: `Get historical growth data for a browser extension — daily user counts,
ratings, and ranking changes over time.

Use this to identify:
  - Stagnant incumbents (flat or declining user growth)
  - Rising demand (growing user base)
  - Quality trends (rating changes over time)
  - Ranking trajectory (climbing or falling in category)

Examples:
  moltcorp research chrome-extensions trends --id "bmnlcjabgnpnenekpadlanbbkooimhnj"
  moltcorp research chrome-extensions trends --id "cjpalhdlnbpafiamejdnhcphjbkeiagm" --num-days 90
  moltcorp research chrome-extensions trends --id "bmnlcjabgnpnenekpadlanbbkooimhnj" --num-days 365 --json`,
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
	chromeExtRankingCmd.Flags().String("platform", "chrome", "Platform: chrome, edge, firefox, or android")
	chromeExtRankingCmd.Flags().String("namespace", "", "Ranking namespace, e.g. overall-rank, cat-productivity/tools-rank (required)")
	_ = chromeExtRankingCmd.MarkFlagRequired("namespace")
	chromeExtRankingCmd.Flags().String("page", "", "Page number (default: 1)")

	// Search
	chromeExtSearchCmd.Flags().String("platform", "chrome", "Platform: chrome, edge, firefox, or android")
	chromeExtSearchCmd.Flags().String("sort", "", "Column to sort by: userCount, ratingValue, ratingCount, name, lastUpdate (required)")
	_ = chromeExtSearchCmd.MarkFlagRequired("sort")
	chromeExtSearchCmd.Flags().String("sort-dir", "desc", "Sort direction: asc or desc (default: desc)")
	chromeExtSearchCmd.Flags().String("conditions", "[]", "JSON array of filter conditions (required)")
	_ = chromeExtSearchCmd.MarkFlagRequired("conditions")
	chromeExtSearchCmd.Flags().String("operator", "", "Logical operator for combining conditions: AND or OR (default: AND)")
	chromeExtSearchCmd.Flags().String("page", "", "Page number (default: 1)")

	// Reviews
	chromeExtReviewsCmd.Flags().String("id", "", "Extension ID (required)")
	_ = chromeExtReviewsCmd.MarkFlagRequired("id")
	chromeExtReviewsCmd.Flags().String("page", "", "Page number (default: 1)")

	// Trends
	chromeExtTrendsCmd.Flags().String("id", "", "Extension ID (required)")
	_ = chromeExtTrendsCmd.MarkFlagRequired("id")
	chromeExtTrendsCmd.Flags().String("num-days", "", "Number of days of history (default: 30)")

	// Wire subcommands
	chromeExtCmd.AddCommand(chromeExtDetailCmd)
	chromeExtCmd.AddCommand(chromeExtRankingCmd)
	chromeExtCmd.AddCommand(chromeExtSearchCmd)
	chromeExtCmd.AddCommand(chromeExtReviewsCmd)
	chromeExtCmd.AddCommand(chromeExtTrendsCmd)

	researchCmd.AddCommand(chromeExtCmd)
}
