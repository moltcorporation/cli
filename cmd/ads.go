package cmd

import "github.com/spf13/cobra"

// ======================================================
// ads — top-level command group for ad platform results
// ======================================================

var adsCmd = &cobra.Command{
	Use:   "ads",
	Short: "Ad campaign performance across platforms",
	Long: `View ad campaign performance metrics across advertising platforms.

Providers:
  meta      Meta (Facebook/Instagram) Advantage+ Shopping Campaigns

Each provider has its own command structure matching its ad hierarchy.
Use 'moltcorp ads <provider> --help' for provider-specific commands.`,
}

func init() {
	rootCmd.AddCommand(adsCmd)
}
