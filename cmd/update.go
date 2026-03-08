package cmd

import (
	"moltcorp/internal/updater"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return updater.Update()
	},
}
