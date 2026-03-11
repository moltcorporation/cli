package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ProfileConfig holds per-profile settings.
type ProfileConfig struct {
	APIKey  string `json:"api_key,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
}

// Config holds the CLI configuration.
type Config struct {
	APIKey   string                    `json:"api_key,omitempty"`
	BaseURL  string                    `json:"base_url,omitempty"`
	Profiles map[string]*ProfileConfig `json:"profiles,omitempty"`
}

// CLIName is the name of this CLI tool, used for config directory paths.
// The coding agent should update this to match the actual CLI name.
var CLIName = "moltcorp"

// DefaultBaseURL is the default API base URL.
// The coding agent should update this to match the actual API.
var DefaultBaseURL = "https://moltcorporation.com"

// EnvAPIKey is the environment variable name for the API key.
// The coding agent should update this (e.g., "STRIPE_API_KEY").
var EnvAPIKey = "MOLTCORP_API_KEY"

// EnvProfile is the environment variable name for the active profile.
var EnvProfile = "MOLTCORP_PROFILE"

// Dir returns the config directory path.
func Dir() string {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, CLIName)
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "AppData", "Roaming", CLIName)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", CLIName)
}

// Path returns the full path to the config file.
func Path() string {
	return filepath.Join(Dir(), "config.json")
}

// Load reads the config file. Returns an empty Config if the file doesn't exist.
func Load() (*Config, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return os.WriteFile(Path(), data, 0o644)
}

// Clear deletes the config file.
func Clear() error {
	err := os.Remove(Path())
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing config: %w", err)
	}
	return nil
}

// ResolveProfile returns the active profile using precedence: flag > env.
func ResolveProfile(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(EnvProfile)
}

// ResolveAPIKey returns the API key using precedence: flag > env > profile config > default config.
func ResolveAPIKey(flagValue string, profile string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if env := os.Getenv(EnvAPIKey); env != "" {
		return env, nil
	}
	cfg, err := Load()
	if err != nil {
		return "", err
	}
	if profile != "" {
		if p, ok := cfg.Profiles[profile]; ok && p.APIKey != "" {
			return p.APIKey, nil
		}
		return "", fmt.Errorf("profile %q not found or has no API key. Run `%s configure --profile %s --api-key <key>`", profile, CLIName, profile)
	}
	if cfg.APIKey != "" {
		return cfg.APIKey, nil
	}
	// If there's exactly one profile and no default key, use it automatically
	if len(cfg.Profiles) == 1 {
		for _, p := range cfg.Profiles {
			if p.APIKey != "" {
				return p.APIKey, nil
			}
		}
	}
	return "", fmt.Errorf("API key not set. Provide --api-key, set %s, or run `%s configure --api-key <key>`", EnvAPIKey, CLIName)
}

// ResolveBaseURL returns the base URL using precedence: flag > profile config > default config > default.
func ResolveBaseURL(flagValue string, profile string) string {
	if flagValue != "" {
		return strings.TrimRight(flagValue, "/")
	}
	cfg, _ := Load()
	if cfg != nil {
		if profile != "" {
			if p, ok := cfg.Profiles[profile]; ok && p.BaseURL != "" {
				return strings.TrimRight(p.BaseURL, "/")
			}
		}
		if cfg.BaseURL != "" {
			return strings.TrimRight(cfg.BaseURL, "/")
		}
	}
	return strings.TrimRight(DefaultBaseURL, "/")
}

// MaskKey returns a masked version of the key showing only the last 4 characters.
func MaskKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(key)-4) + key[len(key)-4:]
}
