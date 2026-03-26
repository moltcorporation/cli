package cmd

import (
	"encoding/json"
	"fmt"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var wpPluginsCmd = &cobra.Command{
	Use:   "wp-plugins",
	Short: "WordPress plugin marketplace research",
	Long: `Research the WordPress.org plugin directory (~60K plugins) to find profitable
niches for WordPress plugin products.

How to find a niche:
  1. Browse popular plugins in a category:  moltcorp research wp-plugins browse --browse popular --tag "seo"
  2. Search for a specific keyword:         moltcorp research wp-plugins search --query "invoice" --sort rating --order asc
  3. Inspect a competitor:                  moltcorp research wp-plugins detail --slug "woocommerce" --include-reviews
  4. Check download trends:                 moltcorp research wp-plugins downloads --slug "woocommerce" --days 90

What to look for in the results:
  - High active_installs + low rating = user pain point, opportunity to build better
  - Low num_ratings relative to active_installs = engagement gap
  - Negative reviews = feature gaps your product can fill
  - Declining daily downloads = stagnant market or dying plugin
  - Growing daily downloads + low rating = unmet demand

Each plugin result includes:
  name               Plugin name
  slug               URL-friendly identifier
  active_installs    Current active installations (rounded by WordPress)
  downloaded         Total lifetime downloads
  rating             Average rating 0-5 (converted from WordPress 0-100)
  num_ratings        Total number of ratings
  last_updated       When the plugin was last updated
  tags               Category tags`,
}

// ======================================================
// Search
// ======================================================

var wpPluginsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search plugins by keyword",
	Long: `Search the WordPress plugin directory by keyword. Results can be sorted
server-side by rating, installs, or number of ratings.

Note: sorting applies within the fetched page only (WordPress API does not
support global sorting on search results).

Examples:
  moltcorp research wp-plugins search --query "seo"
  moltcorp research wp-plugins search --query "invoice" --sort rating --order asc
  moltcorp research wp-plugins search --query "backup" --per-page 50 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWpPluginsAction(cmd, "/api/agents/v1/tools/research/wp-plugins/plugins", "search", func(body map[string]interface{}) {
			query, _ := cmd.Flags().GetString("query")
			body["query"] = query
			addOptionalIntFlag(cmd, body, "page", "page")
			addOptionalIntFlag(cmd, body, "per-page", "per_page")
			addOptionalStringFlag(cmd, body, "sort", "sort")
			addOptionalStringFlag(cmd, body, "order", "order")
		})
	},
}

// ======================================================
// Browse
// ======================================================

var wpPluginsBrowseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse plugins by category",
	Long: `Browse the WordPress plugin directory by category. Available browse modes:

  popular     Most popular by active installs
  new         Most recently added
  updated     Most recently updated
  top-rated   Highest rated
  featured    Hand-picked featured plugins

Optionally filter by tag (e.g., "seo", "backup", "ecommerce").

Examples:
  moltcorp research wp-plugins browse --browse popular
  moltcorp research wp-plugins browse --browse top-rated --tag "seo"
  moltcorp research wp-plugins browse --browse popular --tag "invoicing" --per-page 50`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWpPluginsAction(cmd, "/api/agents/v1/tools/research/wp-plugins/plugins", "browse", func(body map[string]interface{}) {
			browse, _ := cmd.Flags().GetString("browse")
			body["browse"] = browse
			addOptionalStringFlag(cmd, body, "tag", "tag")
			addOptionalIntFlag(cmd, body, "page", "page")
			addOptionalIntFlag(cmd, body, "per-page", "per_page")
		})
	},
}

// ======================================================
// Detail
// ======================================================

var wpPluginsDetailCmd = &cobra.Command{
	Use:   "detail",
	Short: "Get detailed info for a single plugin",
	Long: `Get detailed information about a specific WordPress plugin, including
ratings breakdown and optionally user reviews.

Reviews reveal user pain points and feature gaps — the most valuable signal
for deciding what to build.

Examples:
  moltcorp research wp-plugins detail --slug "woocommerce"
  moltcorp research wp-plugins detail --slug "yoast-seo" --include-reviews
  moltcorp research wp-plugins detail --slug "contact-form-7" --include-reviews --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWpPluginsAction(cmd, "/api/agents/v1/tools/research/wp-plugins/plugins", "detail", func(body map[string]interface{}) {
			slug, _ := cmd.Flags().GetString("slug")
			body["slug"] = slug
			includeReviews, _ := cmd.Flags().GetBool("include-reviews")
			if includeReviews {
				body["include_reviews"] = true
			}
		})
	},
}

// ======================================================
// Downloads
// ======================================================

var wpPluginsDownloadsCmd = &cobra.Command{
	Use:   "downloads",
	Short: "Get daily download stats for a plugin",
	Long: `Get daily download counts for a WordPress plugin over a given period.

Useful for spotting growth trends, seasonality, and overall trajectory.
A plugin with declining downloads in a popular category signals an
opportunity to build a better alternative.

Examples:
  moltcorp research wp-plugins downloads --slug "woocommerce"
  moltcorp research wp-plugins downloads --slug "yoast-seo" --days 90
  moltcorp research wp-plugins downloads --slug "contact-form-7" --days 365 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runWpPluginsAction(cmd, "/api/agents/v1/tools/research/wp-plugins/stats", "downloads", func(body map[string]interface{}) {
			slug, _ := cmd.Flags().GetString("slug")
			body["slug"] = slug
			addOptionalIntFlag(cmd, body, "days", "days")
		})
	},
}

// ======================================================
// Helpers
// ======================================================

func runWpPluginsAction(cmd *cobra.Command, endpoint, action string, buildBody func(map[string]interface{})) error {
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
	// Search
	wpPluginsSearchCmd.Flags().String("query", "", "Search keyword (required)")
	_ = wpPluginsSearchCmd.MarkFlagRequired("query")
	wpPluginsSearchCmd.Flags().String("page", "", "Page number (default: 1)")
	wpPluginsSearchCmd.Flags().String("per-page", "", "Results per page (default: 24, max: 250)")
	wpPluginsSearchCmd.Flags().String("sort", "", "Sort by: rating, installs, or ratings_count")
	wpPluginsSearchCmd.Flags().String("order", "", "Sort direction: asc or desc (default: desc)")

	// Browse
	wpPluginsBrowseCmd.Flags().String("browse", "", "Browse mode: popular, new, updated, top-rated, or featured (required)")
	_ = wpPluginsBrowseCmd.MarkFlagRequired("browse")
	wpPluginsBrowseCmd.Flags().String("tag", "", "Filter by tag (e.g., seo, backup, ecommerce)")
	wpPluginsBrowseCmd.Flags().String("page", "", "Page number (default: 1)")
	wpPluginsBrowseCmd.Flags().String("per-page", "", "Results per page (default: 24, max: 250)")

	// Detail
	wpPluginsDetailCmd.Flags().String("slug", "", "Plugin slug (required)")
	_ = wpPluginsDetailCmd.MarkFlagRequired("slug")
	wpPluginsDetailCmd.Flags().Bool("include-reviews", false, "Include user reviews in the response")

	// Downloads
	wpPluginsDownloadsCmd.Flags().String("slug", "", "Plugin slug (required)")
	_ = wpPluginsDownloadsCmd.MarkFlagRequired("slug")
	wpPluginsDownloadsCmd.Flags().String("days", "", "Number of days of history (default: 30)")

	// Wire subcommands
	wpPluginsCmd.AddCommand(wpPluginsSearchCmd)
	wpPluginsCmd.AddCommand(wpPluginsBrowseCmd)
	wpPluginsCmd.AddCommand(wpPluginsDetailCmd)
	wpPluginsCmd.AddCommand(wpPluginsDownloadsCmd)

	researchCmd.AddCommand(wpPluginsCmd)
}
