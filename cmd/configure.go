package cmd

import (
	"fmt"
	"os"

	"api-cli/internal/config"

	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Manage CLI configuration",
	Long: fmt.Sprintf(`Store API credentials and settings for %s.

Configuration is stored at: %s

Precedence for API key: --api-key flag > %s env var > config file`, config.CLIName, config.Path(), config.EnvAPIKey),
	RunE: runConfigure,
}

func init() {
	configureCmd.Flags().String("api-key", "", "Store API key in config")
	configureCmd.Flags().String("base-url", "", "Store default base URL in config")
	configureCmd.Flags().Bool("show", false, "Display current configuration")
	configureCmd.Flags().Bool("clear", false, "Delete configuration file")
}

func runConfigure(cmd *cobra.Command, args []string) error {
	show, _ := cmd.Flags().GetBool("show")
	clear, _ := cmd.Flags().GetBool("clear")

	if clear {
		if err := config.Clear(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Configuration cleared.")
		return nil
	}

	if show {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Config file: %s\n\n", config.Path())
		if cfg.APIKey != "" {
			fmt.Fprintf(os.Stderr, "  API Key:   %s\n", config.MaskKey(cfg.APIKey))
		} else {
			fmt.Fprintln(os.Stderr, "  API Key:   (not set)")
		}
		if cfg.BaseURL != "" {
			fmt.Fprintf(os.Stderr, "  Base URL:  %s\n", cfg.BaseURL)
		} else {
			fmt.Fprintf(os.Stderr, "  Base URL:  %s (default)\n", config.DefaultBaseURL)
		}
		return nil
	}

	apiKey, _ := cmd.Flags().GetString("api-key")
	baseURL, _ := cmd.Flags().GetString("base-url")

	if apiKey == "" && baseURL == "" {
		return cmd.Help()
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if apiKey != "" {
		cfg.APIKey = apiKey
	}
	if baseURL != "" {
		cfg.BaseURL = baseURL
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, "Configuration saved.")
	return nil
}
