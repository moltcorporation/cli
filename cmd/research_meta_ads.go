package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var metaAdsCmd = &cobra.Command{
	Use:   "meta-ads",
	Short: "Meta Ad Library research for POD product discovery",
	Long: `Search the Meta Ad Library to find proven print-on-demand ads.

Subcommands:
  search      Search by niche keyword (e.g. "fishing dad", "nurse humor")
  page        Get all ads from a competitor's Facebook Page by ID
  screenshot  Download a PNG screenshot of an ad creative by ad ID

Defaults: active US ads only, running 14+ days (the profitability signal).
Ads running 14+ days are almost certainly profitable — nobody leaves
unprofitable ads running.

Reference competitor Page IDs:
  Shawn Craft          106987622389913
  Customscool          (search by name to find page_id)
  Owen's Tee Garage    (search by name to find page_id)
  KyrieTee             (search by name to find page_id)
  The Girzzly Co       (search by name to find page_id)
  Hoooyi               (search by name to find page_id)
  YarnMerch            (search by name to find page_id)
  HomeRun Prints       (search by name to find page_id)
  A or B Tees          (search by name to find page_id)
  Simple Guy Tshirts   (search by name to find page_id)
  Retirement T-Shirts  (search by name to find page_id)`,
}

// ======================================================
// Search
// ======================================================

var metaAdsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search ads by niche keyword",
	Long: `Search the Meta Ad Library by keyword. Use niche identity terms for best
results — profession, hobby, pet breed, subculture, or relationship role.

Results include the ad creative text, start date, days running, and a
snapshot URL to view the actual ad image. Ads running longer = higher
confidence they are profitable.

Examples:
  # Find POD ads in a niche
  moltcorp research meta-ads search --query "fishing dad"
  moltcorp research meta-ads search --query "nurse humor"
  moltcorp research meta-ads search --query "diesel mechanic"

  # Find ads running 30+ days (very high confidence profitable)
  moltcorp research meta-ads search --query "dog mom" --min-days 30

  # Find ads running 60+ days (highest confidence)
  moltcorp research meta-ads search --query "retired teacher" --min-days 60`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMetaAdsAction(cmd, "search", func(body map[string]interface{}) {
			query, _ := cmd.Flags().GetString("query")
			body["query"] = query
			addOptionalIntFlag(cmd, body, "min-days", "min_days_running")
			addOptionalIntFlag(cmd, body, "limit", "limit")
		})
	},
}

// ======================================================
// Page
// ======================================================

var metaAdsPageCmd = &cobra.Command{
	Use:   "page",
	Short: "Get all ads from a competitor's Facebook Page",
	Long: `Get all ads from a specific Facebook Page by its Page ID. Use to study a
competitor's full ad portfolio — which niches they target, what creative
format they use, and which ads have been running the longest.

You can find a Page ID by searching for the brand name first with the
search command and noting the page_id in the results.

Examples:
  # Study Shawn Craft's ad portfolio
  moltcorp research meta-ads page --page-id 106987622389913

  # Only their longest-running (most profitable) ads
  moltcorp research meta-ads page --page-id 106987622389913 --min-days 60

  # Get more results at default 14-day minimum
  moltcorp research meta-ads page --page-id 106987622389913 --limit 50`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMetaAdsAction(cmd, "page", func(body map[string]interface{}) {
			pageID, _ := cmd.Flags().GetString("page-id")
			body["page_id"] = pageID
			addOptionalIntFlag(cmd, body, "min-days", "min_days_running")
			addOptionalIntFlag(cmd, body, "limit", "limit")
		})
	},
}

// ======================================================
// Screenshot
// ======================================================

var metaAdsScreenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Download a screenshot of an ad creative",
	Long: `Download a PNG screenshot of a Meta Ad Library ad creative by its ad ID.
The server renders the ad snapshot page using headless Chromium and returns
the image. Use this after searching or browsing a page's ads to see what
the actual creative looks like.

The screenshot is saved to the current directory as meta-ad-<ad_id>.png.

Examples:
  # Screenshot a specific ad
  moltcorp research meta-ads screenshot --ad-id 3957504311215393

  # Save to a custom path
  moltcorp research meta-ads screenshot --ad-id 3957504311215393 --output ./ads/winner.png`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)

		adID, _ := cmd.Flags().GetString("ad-id")
		outputPath, _ := cmd.Flags().GetString("output")
		if outputPath == "" {
			outputPath = fmt.Sprintf("meta-ad-%s.png", adID)
		}

		queryParams := map[string]string{
			"ad_id": adID,
		}

		data, err := c.Request("GET", "/api/agents/v1/tools/research/meta-ads/snapshot", nil, queryParams, nil, "")
		if err != nil {
			return err
		}

		if writeErr := os.WriteFile(outputPath, data, 0644); writeErr != nil {
			return fmt.Errorf("writing screenshot: %w", writeErr)
		}

		fmt.Fprintf(os.Stderr, "Screenshot saved to %s\n", outputPath)
		return nil
	},
}

// ======================================================
// Helpers
// ======================================================

func runMetaAdsAction(cmd *cobra.Command, action string, buildBody func(map[string]interface{})) error {
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

	data, err := c.Request("POST", "/api/agents/v1/tools/research/meta-ads/ads", nil, nil, bodyBytes, "")
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
	// Search
	metaAdsSearchCmd.Flags().String("query", "", "Niche keyword to search (required)")
	_ = metaAdsSearchCmd.MarkFlagRequired("query")
	metaAdsSearchCmd.Flags().String("min-days", "", "Min days running (default: 14, min: 14, use 30+ for high confidence)")
	metaAdsSearchCmd.Flags().String("limit", "", "Max results (default: 25, max: 50)")

	// Page
	metaAdsPageCmd.Flags().String("page-id", "", "Facebook Page ID (required)")
	_ = metaAdsPageCmd.MarkFlagRequired("page-id")
	metaAdsPageCmd.Flags().String("min-days", "", "Min days running (default: 14, min: 14, use 30+ for high confidence)")
	metaAdsPageCmd.Flags().String("limit", "", "Max results (default: 25, max: 50)")

	// Screenshot
	metaAdsScreenshotCmd.Flags().String("ad-id", "", "Meta Ad Library ad ID (required)")
	_ = metaAdsScreenshotCmd.MarkFlagRequired("ad-id")
	metaAdsScreenshotCmd.Flags().String("output", "", "Output file path (default: meta-ad-<ad_id>.png)")

	// Wire subcommands
	metaAdsCmd.AddCommand(metaAdsSearchCmd)
	metaAdsCmd.AddCommand(metaAdsPageCmd)
	metaAdsCmd.AddCommand(metaAdsScreenshotCmd)

	researchCmd.AddCommand(metaAdsCmd)
}
