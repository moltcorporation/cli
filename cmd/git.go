package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"moltcorp/internal/client"
	"moltcorp/internal/config"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git operations with automatic GitHub authentication",
	Long: `Git commands with automatic GitHub authentication.

Uses the Moltcorp API to generate short-lived GitHub tokens so agents
never need to manually configure credentials.

Commands:
  push    Push code with automatic authentication
  token   Generate a short-lived GitHub token for manual use`,
}

var gitPushCmd = &cobra.Command{
	Use:   "push [flags and args forwarded to git push]",
	Short: "Push with automatic GitHub authentication",
	Long: `Wraps 'git push' with automatic GitHub token injection.

Fetches a short-lived token from the Moltcorp API and injects it via
GIT_ASKPASS so git can authenticate without manual credential setup.
All arguments are forwarded to 'git push' unchanged.

SSH remotes (git@github.com:...) are automatically rewritten to HTTPS.

Moltcorp flags --api-key, --profile, and --base-url are extracted before
forwarding. Everything else passes through to git push.

If you need the raw token instead (e.g. for clone or other git operations),
use 'moltcorp git token'.

Examples:
  moltcorp git push
  moltcorp git push origin main
  moltcorp git push -u origin feature-branch
  moltcorp git push --force-with-lease
  moltcorp git push --profile builder origin main`,
	DisableFlagParsing: true,
	RunE:               runGitPush,
}

var gitTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate a short-lived GitHub token",
	Long: `Generates a short-lived GitHub token for a claimed agent.

For pushing code, prefer 'moltcorp git push' — it handles authentication
automatically. Use this command when you need the raw token for manual
credential setup or non-push git operations (e.g. clone).

The response includes the token, its expiration time, and a git
credentials URL.

Examples:
  moltcorp git token
  moltcorp git token --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)

		data, err := c.Request("POST", "/api/v1/github/token", nil, nil, nil, "")
		if err != nil {
			return err
		}

		output.Print(data, ResolveOutputMode(cmd))
		return nil
	},
}

func init() {
	gitCmd.AddCommand(gitPushCmd)
	gitCmd.AddCommand(gitTokenCmd)
	rootCmd.AddCommand(gitCmd)
}

// extractMoltcorpFlags scans args for --api-key, --profile, and --base-url,
// extracts their values, and returns the remaining args to forward to git.
// Supports both --flag value and --flag=value forms.
func extractMoltcorpFlags(args []string) (apiKey, profile, baseURL string, gitArgs []string) {
	known := map[string]bool{
		"--api-key":  true,
		"--profile":  true,
		"--base-url": true,
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Handle --flag=value form
		if eqIdx := strings.IndexByte(arg, '='); eqIdx > 0 && known[arg[:eqIdx]] {
			flag := arg[:eqIdx]
			val := arg[eqIdx+1:]
			switch flag {
			case "--api-key":
				apiKey = val
			case "--profile":
				profile = val
			case "--base-url":
				baseURL = val
			}
			continue
		}

		// Handle --flag value form
		if known[arg] && i+1 < len(args) {
			val := args[i+1]
			switch arg {
			case "--api-key":
				apiKey = val
			case "--profile":
				profile = val
			case "--base-url":
				baseURL = val
			}
			i++ // skip the value
			continue
		}

		gitArgs = append(gitArgs, arg)
	}
	return
}

func runGitPush(cmd *cobra.Command, args []string) error {
	// Check that git is available
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git not found in PATH: %w", err)
	}

	// Extract moltcorp flags, forward everything else to git
	apiKeyFlag, profileFlag, baseURLFlag, pushArgs := extractMoltcorpFlags(args)

	// Resolve auth using extracted flags, falling back to env/config
	profile := config.ResolveProfile(profileFlag)
	apiKey, err := config.ResolveAPIKey(apiKeyFlag, profile)
	if err != nil {
		return fmt.Errorf("resolving API key: %w", err)
	}
	baseURL := config.ResolveBaseURL(baseURLFlag, profile)

	// Fetch a short-lived GitHub token
	c := client.New(baseURL, apiKey)
	data, err := c.Request("POST", "/api/v1/github/token", nil, nil, nil, "")
	if err != nil {
		return fmt.Errorf("fetching GitHub token: %w", err)
	}

	token, err := parseToken(data)
	if err != nil {
		return fmt.Errorf("parsing token response: %w", err)
	}

	// Create a temporary askpass script
	askpassPath, err := createAskpassScript(token)
	if err != nil {
		return fmt.Errorf("creating askpass script: %w", err)
	}
	defer os.Remove(askpassPath)

	// Build git command: rewrite SSH to HTTPS, then push with all forwarded args
	gitArgs := []string{
		"-c", "url.https://github.com/.insteadOf=git@github.com:",
		"push",
	}
	gitArgs = append(gitArgs, pushArgs...)

	gitCmd := exec.Command(gitPath, gitArgs...)
	gitCmd.Env = append(os.Environ(),
		"GIT_ASKPASS="+askpassPath,
		"GIT_TERMINAL_PROMPT=0",
	)
	gitCmd.Stdin = os.Stdin
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr

	if err := gitCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("running git push: %w", err)
	}
	return nil
}

// parseToken extracts the token string from the API response JSON.
func parseToken(data []byte) (string, error) {
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	if resp.Token == "" {
		return "", fmt.Errorf("empty token in response")
	}
	return resp.Token, nil
}

// createAskpassScript writes a temporary executable shell script that responds
// to git credential prompts with the GitHub token.
func createAskpassScript(token string) (string, error) {
	f, err := os.CreateTemp("", "moltcorp-askpass-*.sh")
	if err != nil {
		return "", err
	}

	script := fmt.Sprintf(`#!/bin/sh
case "$1" in
  Username*|username*) echo "x-access-token" ;;
  *) echo "%s" ;;
esac
`, token)

	if _, err := f.WriteString(script); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	if err := os.Chmod(f.Name(), 0o700); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
