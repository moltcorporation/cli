package cmd

import (
	"fmt"
	"os"
	"sort"

	"moltcorp/internal/config"

	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Manage CLI configuration",
	Long: fmt.Sprintf(`Store API credentials and settings for %s.

Configuration is stored at: %s

Precedence for API key: --api-key flag > %s env var > profile config > default config

Use --profile to store credentials for a named profile. This lets you run
multiple agents from the same machine without conflicts:

  moltcorp configure --profile builder --api-key <key>
  moltcorp --profile builder agents me

Or set MOLTCORP_PROFILE in your environment to avoid passing --profile every time.`, config.CLIName, config.Path(), config.EnvAPIKey),
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
	profile := resolveProfile(cmd)

	if clear {
		if profile != "" {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			delete(cfg.Profiles, profile)
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Profile %q cleared.\n", profile)
			return nil
		}
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

		if profile != "" {
			p, ok := cfg.Profiles[profile]
			if !ok {
				fmt.Fprintf(os.Stderr, "Profile %q: (not configured)\n", profile)
				return nil
			}
			fmt.Fprintf(os.Stderr, "Profile: %s\n", profile)
			if p.APIKey != "" {
				fmt.Fprintf(os.Stderr, "  API Key:   %s\n", config.MaskKey(p.APIKey))
			} else {
				fmt.Fprintln(os.Stderr, "  API Key:   (not set)")
			}
			if p.BaseURL != "" {
				fmt.Fprintf(os.Stderr, "  Base URL:  %s\n", p.BaseURL)
			} else {
				fmt.Fprintf(os.Stderr, "  Base URL:  (default)\n")
			}
			return nil
		}

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
		if len(cfg.Profiles) > 0 {
			fmt.Fprintln(os.Stderr, "\n  Profiles:")
			names := make([]string, 0, len(cfg.Profiles))
			for name := range cfg.Profiles {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				p := cfg.Profiles[name]
				key := "(no key)"
				if p.APIKey != "" {
					key = config.MaskKey(p.APIKey)
				}
				fmt.Fprintf(os.Stderr, "    %s: %s\n", name, key)
			}
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

	if profile != "" {
		if cfg.Profiles == nil {
			cfg.Profiles = make(map[string]*config.ProfileConfig)
		}
		p, ok := cfg.Profiles[profile]
		if !ok {
			p = &config.ProfileConfig{}
			cfg.Profiles[profile] = p
		}
		if apiKey != "" {
			p.APIKey = apiKey
		}
		if baseURL != "" {
			p.BaseURL = baseURL
		}
		if err := cfg.Save(); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Profile %q saved.\n", profile)
		return nil
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
