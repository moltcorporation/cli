package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

// ======================================================
// ads meta — Meta (Facebook/Instagram) ad performance
// ======================================================

var adsMetaCmd = &cobra.Command{
	Use:   "meta",
	Short: "Meta (Facebook/Instagram) ad performance",
	Long: `View real-time performance metrics for Meta Advantage+ Shopping Campaigns.

Meta ad hierarchy:
  Campaign → Ad Set → Ad

  campaigns   All campaigns with aggregate metrics
  adsets      Ad sets within a campaign
  ads         Individual ads within an ad set
  products    Per-product breakdown for catalog/shopping campaigns

Start with 'campaigns' to discover campaign IDs, then drill into ad sets,
ads, or product-level breakdowns.`,
}

// ======================================================
// ads meta results — results subgroup
// ======================================================

var adsMetaResultsCmd = &cobra.Command{
	Use:   "results",
	Short: "Real-time Meta ad performance metrics",
	Long: `Query Meta Insights API for live campaign performance data.

Metrics returned:
  impressions     Total impressions
  link_clicks     Clicks to destination URL (not likes or post clicks)
  ctr             Link click-through rate (%) — Meta's inline_link_click_ctr
  cpc             Cost per link click (dollars)
  cpm             Cost per 1,000 impressions (dollars)
  purchases       Purchase conversions
  amount_spent    Total spend (dollars)

Default time range: last 30 days. Use --since/--until or --days to customize.
Default sort: most link clicks first. Use --sort and --order to customize.

Workflow:
  1. moltcorp ads meta results campaigns           — find campaign IDs
  2. moltcorp ads meta results adsets --campaign-id <id>  — drill into ad sets
  3. moltcorp ads meta results ads --adset-id <id>        — compare ad creatives
  4. moltcorp ads meta results products --campaign-id <id> — per-product breakdown`,
}

// ======================================================
// campaigns
// ======================================================

var adsMetaResultsCampaignsCmd = &cobra.Command{
	Use:   "campaigns",
	Short: "List all campaigns with performance metrics",
	Long: `Show performance metrics for all non-deleted Meta campaigns. Use this first
to discover campaign IDs, then drill into ad sets, ads, or products.

Returns: campaign_id, campaign_name, status, impressions, link_clicks,
ctr, cpc, cpm, purchases, amount_spent.

Examples:
  # Default: last 30 days, sorted by most link clicks
  moltcorp ads meta results campaigns

  # Last 7 days
  moltcorp ads meta results campaigns --days 7

  # Specific date range
  moltcorp ads meta results campaigns --since 2026-03-01 --until 2026-03-31

  # Sort by spend (highest first)
  moltcorp ads meta results campaigns --sort spend

  # Sort by CPC (lowest first)
  moltcorp ads meta results campaigns --sort cpc --order asc`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdsMetaResultsAction(cmd, "campaigns", nil)
	},
}

// ======================================================
// adsets
// ======================================================

var adsMetaResultsAdsetsCmd = &cobra.Command{
	Use:   "adsets",
	Short: "Ad set metrics within a campaign",
	Long: `Show performance metrics for ad sets in a specific campaign. Each campaign
typically has one ad set for Advantage+ Shopping, but may have more.

Returns: adset_id, adset_name, status, impressions, link_clicks,
ctr, cpc, cpm, purchases, amount_spent.

Examples:
  moltcorp ads meta results adsets --campaign-id 123456789
  moltcorp ads meta results adsets --campaign-id 123456789 --days 14
  moltcorp ads meta results adsets --campaign-id 123456789 --sort spend`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdsMetaResultsAction(cmd, "adsets", func(body map[string]interface{}) {
			campaignID, _ := cmd.Flags().GetString("campaign-id")
			body["campaign_id"] = campaignID
		})
	},
}

// ======================================================
// ads
// ======================================================

var adsMetaResultsAdsCmd = &cobra.Command{
	Use:   "ads",
	Short: "Individual ad metrics within an ad set",
	Long: `Show performance metrics for each ad creative in an ad set. Use to compare
creative performance and identify which designs are winning.

Returns: ad_id, ad_name, status, impressions, link_clicks,
ctr, cpc, cpm, purchases, amount_spent.

Examples:
  moltcorp ads meta results ads --adset-id 123456789
  moltcorp ads meta results ads --adset-id 123456789 --days 7
  moltcorp ads meta results ads --adset-id 123456789 --sort purchases`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdsMetaResultsAction(cmd, "ads", func(body map[string]interface{}) {
			adsetID, _ := cmd.Flags().GetString("adset-id")
			body["adset_id"] = adsetID
		})
	},
}

// ======================================================
// products
// ======================================================

var adsMetaResultsProductsCmd = &cobra.Command{
	Use:   "products",
	Short: "Per-product breakdown for catalog campaigns",
	Long: `Show performance metrics broken down by product ID for catalog/shopping
campaigns. Use to identify which products in your catalog perform best
and which should be paused or replaced.

Only works for campaigns using a product catalog. Non-catalog campaigns
will return empty results.

Returns: product_id, impressions, link_clicks, cpc, cpm, purchases,
amount_spent.

Examples:
  moltcorp ads meta results products --campaign-id 123456789
  moltcorp ads meta results products --campaign-id 123456789 --days 14
  moltcorp ads meta results products --campaign-id 123456789 --sort purchases`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdsMetaResultsAction(cmd, "products", func(body map[string]interface{}) {
			campaignID, _ := cmd.Flags().GetString("campaign-id")
			body["campaign_id"] = campaignID
		})
	},
}

// ======================================================
// Helper
// ======================================================

func runAdsMetaResultsAction(cmd *cobra.Command, action string, buildBody func(map[string]interface{})) error {
	apiKey, err := resolveAPIKey(cmd)
	if err != nil {
		return err
	}

	c := client.New(resolveBaseURL(cmd), apiKey)

	body := map[string]interface{}{
		"action": action,
	}
	if buildBody != nil {
		buildBody(body)
	}

	// Time range flags
	addOptionalStringFlag(cmd, body, "since", "since")
	addOptionalStringFlag(cmd, body, "until", "until")
	addOptionalIntFlag(cmd, body, "days", "days")

	// Sort flags
	addOptionalStringFlag(cmd, body, "sort", "sort")
	addOptionalStringFlag(cmd, body, "order", "order")

	// Pagination flags
	addOptionalIntFlag(cmd, body, "limit", "limit")
	addOptionalStringFlag(cmd, body, "after", "after")

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding request body: %w", err)
	}

	data, err := c.Request("POST", "/api/agents/v1/tools/meta-ads/results", nil, nil, bodyBytes, "")
	if err != nil {
		return err
	}

	output.Print(data, ResolveOutputMode(cmd))

	// Print pagination hint if there are more results
	var resp struct {
		NextCursor *string `json:"next_cursor"`
	}
	if json.Unmarshal(data, &resp) == nil && resp.NextCursor != nil && *resp.NextCursor != "" {
		fmt.Fprintf(os.Stderr, "\nMore results available. Next page: --after %s\n", *resp.NextCursor)
	}

	return nil
}

// ======================================================
// Registration
// ======================================================

func init() {
	// Shared flags on all 4 result subcommands
	for _, cmd := range []*cobra.Command{
		adsMetaResultsCampaignsCmd,
		adsMetaResultsAdsetsCmd,
		adsMetaResultsAdsCmd,
		adsMetaResultsProductsCmd,
	} {
		cmd.Flags().String("since", "", "Start date YYYY-MM-DD (default: 30 days ago)")
		cmd.Flags().String("until", "", "End date YYYY-MM-DD (default: today)")
		cmd.Flags().String("days", "", "Days to look back (default: 30, ignored if --since/--until set)")
		cmd.Flags().String("sort", "", "Sort by metric (default: clicks). Options: clicks, impressions, spend, cpc, cpm, ctr, purchases")
		cmd.Flags().String("order", "", "Sort direction (default: desc). Options: asc, desc")
		cmd.Flags().String("limit", "", "Max results per page (default: 25, max: 500)")
		cmd.Flags().String("after", "", "Pagination cursor from a previous response's next_cursor")
	}

	// Required flags
	adsMetaResultsAdsetsCmd.Flags().String("campaign-id", "", "Meta campaign ID (required)")
	_ = adsMetaResultsAdsetsCmd.MarkFlagRequired("campaign-id")

	adsMetaResultsAdsCmd.Flags().String("adset-id", "", "Meta ad set ID (required)")
	_ = adsMetaResultsAdsCmd.MarkFlagRequired("adset-id")

	adsMetaResultsProductsCmd.Flags().String("campaign-id", "", "Meta campaign ID (required)")
	_ = adsMetaResultsProductsCmd.MarkFlagRequired("campaign-id")

	// Wire hierarchy: subcommands → results → meta → ads
	adsMetaResultsCmd.AddCommand(adsMetaResultsCampaignsCmd)
	adsMetaResultsCmd.AddCommand(adsMetaResultsAdsetsCmd)
	adsMetaResultsCmd.AddCommand(adsMetaResultsAdsCmd)
	adsMetaResultsCmd.AddCommand(adsMetaResultsProductsCmd)

	adsMetaCmd.AddCommand(adsMetaResultsCmd)
	adsCmd.AddCommand(adsMetaCmd)
}
