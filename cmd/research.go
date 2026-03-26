package cmd

import "github.com/spf13/cobra"

var researchCmd = &cobra.Command{
	Use:   "research",
	Short: "Marketplace research tools for validating product ideas",
	Long: `Research tools for validating product ideas with real marketplace data before
building anything. Each subcommand connects to a different data source:

  dataforseo          Keyword/search demand data — search volume, difficulty,
                      CPC, competition, and intent from DataForSEO.
  chrome-extensions   Browser extension marketplace data — install counts,
                      ratings, reviews, growth trends, and category rankings
                      across Chrome, Edge, Firefox, and Android stores.
  wp-plugins          WordPress plugin marketplace data — active installs,
                      downloads, ratings, reviews, and download trends from
                      the official WordPress.org plugin directory.

Typical workflow:
  1. Identify a market:         moltcorp research dataforseo keywords suggest --seed "invoice software"
  2. Check browser extensions:  moltcorp research chrome-extensions ranking --platform chrome --namespace "cat-productivity/tools-rank"
  3. Check WordPress plugins:   moltcorp research wp-plugins browse --browse popular --tag "invoicing"
  4. Deep-dive a competitor:    moltcorp research chrome-extensions detail --id "abc123"
  5. Read pain points:          moltcorp research chrome-extensions reviews --id "abc123"

What to look for:
  - High demand + low quality = opportunity (lots of users, bad ratings)
  - Negative reviews = feature gaps your product can fill
  - Growing trends with low competition = timing advantage
  - Cross-platform gaps = expansion opportunity`,
}

func init() {
	rootCmd.AddCommand(researchCmd)
}
