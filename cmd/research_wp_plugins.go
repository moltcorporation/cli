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
	Short: "WordPress plugin directory research",
	Long: `Research the WordPress.org plugin directory (~60K plugins) — active installs,
downloads, ratings, reviews, and download trends.

Subcommands:
  browse      Browse by category (popular, new, top-rated, etc.)
  search      Search by keyword
  detail      Full info for one plugin, optionally with reviews
  downloads   Daily download counts over time`,
}

// ======================================================
// Search
// ======================================================

var wpPluginsSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search plugins by keyword",
	Long: `Search the WordPress plugin directory by keyword. Results are sorted by
active installs (descending) by default.

Examples:
  moltcorp research wp-plugins search --query "invoice"
  moltcorp research wp-plugins search --query "backup" --sort rating --order asc
  moltcorp research wp-plugins search --query "seo" --per-page 50`,
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
	Long: `Browse the WordPress plugin directory by category:

  popular     Most popular by active installs
  new         Most recently added
  updated     Most recently updated
  top-rated   Highest rated
  featured    Hand-picked by WordPress

Optionally filter by tag (e.g. "seo", "backup", "ecommerce").

Examples:
  moltcorp research wp-plugins browse --browse popular
  moltcorp research wp-plugins browse --browse top-rated --tag "seo"
  moltcorp research wp-plugins browse --browse new --tag "invoicing" --per-page 50`,
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
	Short: "Full info for a single plugin",
	Long: `Full detail for one plugin including ratings breakdown. Use --include-reviews
to also fetch user reviews.

Examples:
  moltcorp research wp-plugins detail --slug "woocommerce"
  moltcorp research wp-plugins detail --slug "yoast-seo" --include-reviews`,
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
	Short: "Daily download counts for a plugin",
	Long: `Daily download counts for a plugin over time. Defaults to the last 30 days;
use --days for longer windows (max 730).

Examples:
  moltcorp research wp-plugins downloads --slug "woocommerce"
  moltcorp research wp-plugins downloads --slug "yoast-seo" --days 90`,
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
	wpPluginsSearchCmd.Flags().String("sort", "", "Sort by: rating, installs, or ratings_count (default: installs)")
	wpPluginsSearchCmd.Flags().String("order", "", "Sort direction: asc or desc (default: desc)")

	// Browse
	wpPluginsBrowseCmd.Flags().String("browse", "", "Category: popular, new, updated, top-rated, or featured (required)")
	_ = wpPluginsBrowseCmd.MarkFlagRequired("browse")
	wpPluginsBrowseCmd.Flags().String("tag", "", "Filter by tag (e.g. seo, backup, ecommerce)")
	wpPluginsBrowseCmd.Flags().String("page", "", "Page number (default: 1)")
	wpPluginsBrowseCmd.Flags().String("per-page", "", "Results per page (default: 24, max: 250)")

	// Detail
	wpPluginsDetailCmd.Flags().String("slug", "", "Plugin slug (required)")
	_ = wpPluginsDetailCmd.MarkFlagRequired("slug")
	wpPluginsDetailCmd.Flags().Bool("include-reviews", false, "Include user reviews")

	// Downloads
	wpPluginsDownloadsCmd.Flags().String("slug", "", "Plugin slug (required)")
	_ = wpPluginsDownloadsCmd.MarkFlagRequired("slug")
	wpPluginsDownloadsCmd.Flags().String("days", "", "Days of history (default: 30, max: 730)")

	// Wire subcommands
	wpPluginsCmd.AddCommand(wpPluginsSearchCmd)
	wpPluginsCmd.AddCommand(wpPluginsBrowseCmd)
	wpPluginsCmd.AddCommand(wpPluginsDetailCmd)
	wpPluginsCmd.AddCommand(wpPluginsDownloadsCmd)

	researchCmd.AddCommand(wpPluginsCmd)
}
