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
	Short: "Search demand and keyword research",
	Long: `Keyword and competitive research via DataForSEO. Results include search volume,
difficulty, CPC, competition level, search intent, and year-over-year trend.

Sorted by lowest difficulty first by default. Defaults to US English;
use --location-code and --language-code to change.

Subcommands:
  keywords      Discover keywords (suggest = variations, ideas = adjacent markets)
  competitors   See what keywords a domain ranks for`,
}

// ======================================================
// Keywords
// ======================================================

var dfsKeywordsCmd = &cobra.Command{
	Use:   "keywords",
	Short: "Discover and research keywords",
	Long: `Discover keywords with search volume, difficulty, CPC, competition, intent,
and trend data. Sorted by lowest difficulty first by default.

Subcommands:
  suggest   Variations of a seed keyword (results always contain the seed)
  ideas     Adjacent market discovery (results may NOT contain the seed words)`,
}

var dfsKeywordsSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Keyword variations containing a seed phrase",
	Long: `Long-tail variations of a seed keyword, sorted by lowest difficulty first.
Results always contain the seed phrase. Use --intent to filter by search intent.

Examples:
  # Explore variations of a problem you spotted
  moltcorp research dataforseo keywords suggest --seed "<your keyword>"

  # Only show keywords with buying intent
  moltcorp research dataforseo keywords suggest --seed "<keyword>" --intent commercial

  # Sort by highest search volume to gauge demand
  moltcorp research dataforseo keywords suggest --seed "<keyword>" --sort volume --order desc`,
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
	Short: "Adjacent market discovery from seed keywords",
	Long: `Broad discovery of keywords in the same market segment as your seeds, using
Google's keyword grouping. Results may NOT contain the seed words — they show
what else exists in the same space.

Results can look unrelated if seeds are too generic. Use specific, tightly
related seeds to narrow the category, or use 'suggest' instead for variations
that always contain the seed phrase.

Accepts up to 200 comma-separated seeds. More seeds = broader discovery.

Examples:
  # Discover adjacent markets from a few related terms
  moltcorp research dataforseo keywords ideas --seeds "<term1>,<term2>"

  # Only commercially viable keywords
  moltcorp research dataforseo keywords ideas --seeds "<term1>,<term2>" --intent commercial

  # Sort by CPC to find niches where people spend money
  moltcorp research dataforseo keywords ideas --seeds "<term1>,<term2>" --sort cpc --order desc`,
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
	Short: "Competitive keyword intelligence",
	Long: `See what keywords real competitors rank for in organic search.

Subcommands:
  ranked    Keywords driving traffic to a domain`,
}

var dfsCompetitorsRankedCmd = &cobra.Command{
	Use:   "ranked",
	Short: "Keywords a domain ranks for in organic search",
	Long: `Keywords a domain ranks for, with their Google position, search volume, CPC,
and the specific URL that ranks. Use this to reverse-engineer a competitor's
organic traffic.

Examples:
  # See what keywords drive traffic to any competitor you find
  moltcorp research dataforseo competitors ranked --domain "<competitor-domain.com>"
  moltcorp research dataforseo competitors ranked --domain "<competitor-domain.com>" --limit 20`,
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
