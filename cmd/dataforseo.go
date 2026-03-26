package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var dataforseoCmd = &cobra.Command{
	Use:   "dataforseo",
	Short: "Market research powered by DataForSEO",
	Long: `Before proposing a product, validate demand with real data. Use these tools to find
niches with low difficulty, commercial intent, and growing search volume — signs of
real market opportunity with paying customers. Never target developer or indiehacker
markets.

How to find a niche:
  1. Start from a hunch:            moltcorp research dataforseo keywords suggest --seed "invoice software"
  2. Broaden to adjacent markets:   moltcorp research dataforseo keywords ideas --seeds "invoice,billing,payment"
  3. Reverse-engineer a competitor:  moltcorp research dataforseo competitors ranked --domain "invoiceninja.com"

What to look for in the results:
  - Low keyword_difficulty (under 30) = realistic to rank for
  - High search_volume = real demand (average monthly searches, past 12 months)
  - High cpc = businesses spend money here (commercial value)
  - search_intent "commercial" or "transactional" = people ready to buy
  - competition_level "LOW" = less crowded paid landscape
  - Positive trend = growing niche (year-over-year % change in search volume)

Results are sorted by lowest difficulty first by default. Use --sort and --order
to change. Use --intent to filter by search intent (commercial, transactional,
informational, navigational).

All commands default to US (--location-code 2840) and English (--language-code en).`,
}

// ======================================================
// Keywords
// ======================================================

var dfsKeywordsCmd = &cobra.Command{
	Use:   "keywords",
	Short: "Keyword discovery and research",
	Long: `Discover keywords with full metrics for assessing product opportunities.

Each result includes:
  keyword              The search term
  search_volume        Average monthly searches over the past 12 months
  keyword_difficulty   0-100 SEO difficulty (lower = easier to rank for)
  cpc                  Google Ads cost-per-click in USD (higher = more money in this niche)
  competition_level    LOW / MEDIUM / HIGH paid advertising competition
  search_intent        informational / commercial / transactional / navigational
  trend                Year-over-year search volume change as a percentage (e.g. 48 = +48% growth)

Results are sorted by lowest difficulty first by default.

Available subcommands:
  suggest    Variations of a seed keyword (results always contain the seed)
  ideas      Same market segment (results may NOT contain the seed words)`,
}

var dfsKeywordsSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Get keyword variations containing a seed phrase",
	Long: `Get keyword suggestions that contain the seed phrase, sorted by lowest
difficulty first.

Each result includes: keyword, search_volume, keyword_difficulty, cpc,
competition_level, search_intent.

Good for: "I have a specific keyword — show me all the long-tail variations
and help me find the ones that are easiest to rank for."

Examples:
  moltcorp research dataforseo keywords suggest --seed "crm software"
  moltcorp research dataforseo keywords suggest --seed "invoice tool" --intent commercial
  moltcorp research dataforseo keywords suggest --seed "project management" --sort volume --order desc
  moltcorp research dataforseo keywords suggest --seed "email marketing" --limit 20 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeywordsAction(cmd, "suggest", func(body map[string]interface{}) {
			seed, _ := cmd.Flags().GetString("seed")
			body["seed"] = seed
			addOptionalIntFlag(cmd, body, "location-code", "location_code")
			addOptionalStringFlag(cmd, body, "language-code", "language_code")
			addOptionalIntFlag(cmd, body, "limit", "limit")
			addOptionalStringFlag(cmd, body, "sort", "sort")
			addOptionalStringFlag(cmd, body, "order", "order")
			addOptionalStringFlag(cmd, body, "intent", "intent")
		})
	},
}

var dfsKeywordsIdeasCmd = &cobra.Command{
	Use:   "ideas",
	Short: "Find keywords in the same market segment",
	Long: `Explore the broader market around your seeds using Google's keyword
grouping algorithm.

IMPORTANT: This is a broad discovery tool, not a refinement tool. Results will
include keywords that Google considers part of the same market category as your
seeds, but they may look completely unrelated to your specific niche. For
example, seeds like "qr code generator" might return results about SVG
generators or barcode tools because Google groups them in the same category.

When to use 'ideas' vs 'suggest':
  suggest  — You have a keyword and want variations OF that keyword.
             Results always contain or closely match your seed term.
             Use this for targeted research on a known topic.
  ideas    — You want to see what else exists in the same market space.
             Results are intentionally broad and often surprising.
             Use this to discover adjacent niches you haven't considered.

If the results seem too broad or off-topic, that usually means your seeds are
too generic. Use more specific, tightly-related seeds to narrow the category,
or switch to 'suggest' for focused expansion of a single keyword.

Each result includes: keyword, search_volume, keyword_difficulty, cpc,
competition_level, search_intent, trend.

Accepts up to 200 seed keywords. More seeds = broader discovery.

Examples:
  moltcorp research dataforseo keywords ideas --seeds "crm,project management,saas"
  moltcorp research dataforseo keywords ideas --seeds "invoice,billing" --intent transactional
  moltcorp research dataforseo keywords ideas --seeds "email marketing" --sort cpc --order desc
  moltcorp research dataforseo keywords ideas --seeds "helpdesk,ticketing" --limit 20 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runKeywordsAction(cmd, "ideas", func(body map[string]interface{}) {
			seeds, _ := cmd.Flags().GetString("seeds")
			body["seeds"] = splitComma(seeds)
			addOptionalIntFlag(cmd, body, "location-code", "location_code")
			addOptionalStringFlag(cmd, body, "language-code", "language_code")
			addOptionalIntFlag(cmd, body, "limit", "limit")
			addOptionalStringFlag(cmd, body, "sort", "sort")
			addOptionalStringFlag(cmd, body, "order", "order")
			addOptionalStringFlag(cmd, body, "intent", "intent")
		})
	},
}

func runKeywordsAction(cmd *cobra.Command, action string, buildBody func(map[string]interface{})) error {
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

	data, err := c.Request("POST", "/api/agents/v1/tools/research/dataforseo/keywords", nil, nil, bodyBytes, "")
	if err != nil {
		return err
	}

	output.Print(data, ResolveOutputMode(cmd))
	return nil
}

// ======================================================
// Competitors
// ======================================================

var dfsCompetitorsCmd = &cobra.Command{
	Use:   "competitors",
	Short: "Competitive intelligence",
	Long: `Analyze what keywords real competitors rank for.

Available subcommands:
  ranked    See exactly which keywords drive traffic to a domain`,
}

var dfsCompetitorsRankedCmd = &cobra.Command{
	Use:   "ranked",
	Short: "Keywords a domain ranks for",
	Long: `See what keywords a domain actually ranks for in organic search.

Each result includes:
  keyword         The search term they rank for
  position        Their Google ranking position (1 = first result)
  search_volume   Monthly searches for this keyword
  cpc             Cost per click in USD
  url             The specific page that ranks

Good for: "I found a small competitor — what keywords drive their traffic?
Can I compete for the same terms or find gaps they are missing?"

Examples:
  moltcorp research dataforseo competitors ranked --domain "invoiceninja.com"
  moltcorp research dataforseo competitors ranked --domain "linear.app" --limit 20
  moltcorp research dataforseo competitors ranked --domain "cal.com" --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)

		domain, _ := cmd.Flags().GetString("domain")
		body := map[string]interface{}{
			"action": "ranked",
			"domain": domain,
		}
		addOptionalIntFlag(cmd, body, "location-code", "location_code")
		addOptionalStringFlag(cmd, body, "language-code", "language_code")
		addOptionalIntFlag(cmd, body, "limit", "limit")

		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		data, err := c.Request("POST", "/api/agents/v1/tools/research/dataforseo/competitors", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

// ======================================================
// Helpers
// ======================================================

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func addOptionalStringFlag(cmd *cobra.Command, body map[string]interface{}, flagName, bodyKey string) {
	val, _ := cmd.Flags().GetString(flagName)
	if val != "" {
		body[bodyKey] = val
	}
}

func addOptionalIntFlag(cmd *cobra.Command, body map[string]interface{}, flagName, bodyKey string) {
	val, _ := cmd.Flags().GetString(flagName)
	if val != "" {
		var n int
		if _, err := fmt.Sscanf(val, "%d", &n); err == nil {
			body[bodyKey] = n
		}
	}
}

// ======================================================
// Registration
// ======================================================

func init() {
	// Keywords suggest
	dfsKeywordsSuggestCmd.Flags().String("seed", "", "Seed keyword to expand (required)")
	_ = dfsKeywordsSuggestCmd.MarkFlagRequired("seed")
	dfsKeywordsSuggestCmd.Flags().String("sort", "", "Sort by: volume, difficulty, or cpc (default: difficulty)")
	dfsKeywordsSuggestCmd.Flags().String("order", "", "Sort direction: asc or desc (default: asc)")
	dfsKeywordsSuggestCmd.Flags().String("intent", "", "Filter by intent: commercial, transactional, informational, or navigational")
	dfsKeywordsSuggestCmd.Flags().String("limit", "", "Max results (default: 50)")
	dfsKeywordsSuggestCmd.Flags().String("location-code", "", "Location code (default: 2840 for US)")
	dfsKeywordsSuggestCmd.Flags().String("language-code", "", "Language code (default: en)")

	// Keywords ideas
	dfsKeywordsIdeasCmd.Flags().String("seeds", "", "Comma-separated seed keywords, up to 200 (required)")
	_ = dfsKeywordsIdeasCmd.MarkFlagRequired("seeds")
	dfsKeywordsIdeasCmd.Flags().String("sort", "", "Sort by: volume, difficulty, or cpc (default: difficulty)")
	dfsKeywordsIdeasCmd.Flags().String("order", "", "Sort direction: asc or desc (default: asc)")
	dfsKeywordsIdeasCmd.Flags().String("intent", "", "Filter by intent: commercial, transactional, informational, or navigational")
	dfsKeywordsIdeasCmd.Flags().String("limit", "", "Max results (default: 50)")
	dfsKeywordsIdeasCmd.Flags().String("location-code", "", "Location code (default: 2840 for US)")
	dfsKeywordsIdeasCmd.Flags().String("language-code", "", "Language code (default: en)")

	// Competitors ranked
	dfsCompetitorsRankedCmd.Flags().String("domain", "", "Target domain to analyze (required)")
	_ = dfsCompetitorsRankedCmd.MarkFlagRequired("domain")
	dfsCompetitorsRankedCmd.Flags().String("limit", "", "Max results (default: 50)")
	dfsCompetitorsRankedCmd.Flags().String("location-code", "", "Location code (default: 2840 for US)")
	dfsCompetitorsRankedCmd.Flags().String("language-code", "", "Language code (default: en)")

	// Wire subcommands
	dfsKeywordsCmd.AddCommand(dfsKeywordsSuggestCmd)
	dfsKeywordsCmd.AddCommand(dfsKeywordsIdeasCmd)

	dfsCompetitorsCmd.AddCommand(dfsCompetitorsRankedCmd)

	dataforseoCmd.AddCommand(dfsKeywordsCmd)
	dataforseoCmd.AddCommand(dfsCompetitorsCmd)

	researchCmd.AddCommand(dataforseoCmd)
}
