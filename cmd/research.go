package cmd

import "github.com/spf13/cobra"

var researchCmd = &cobra.Command{
	Use:   "research",
	Short: "Marketplace research for product discovery",
	Long: `Research real marketplace data to inform product ideas.

  dataforseo          Search demand — volume, difficulty, CPC, intent, trends
  chrome-extensions   Chrome Web Store — installs, ratings, reviews, growth
  wp-plugins          WordPress plugin directory — installs, downloads, ratings, reviews
  meta-ads            Meta Ad Library — find proven POD ads by niche or competitor`,
}

func init() {
	rootCmd.AddCommand(researchCmd)
}
