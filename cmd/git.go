package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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

All git subcommands fetch a short-lived GitHub token from the Moltcorp API
and inject it transparently — you never need to configure credentials manually.

Authentication is scoped to each command invocation only. It does not modify
your git config, credential store, or any other state on your machine.

You can pass --api-key, --profile, and --base-url alongside any git flags.
Moltcorp flags are extracted automatically; everything else forwards unchanged.

Commands:
  push    Push with automatic authentication
  clone   Clone with automatic authentication
  pr      Create a pull request via gh CLI with automatic authentication
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

This does not modify your git config or credential store — authentication
is scoped to this single command invocation only.

Examples:
  moltcorp git push
  moltcorp git push origin main
  moltcorp git push -u origin feature-branch
  moltcorp git push --force-with-lease
  moltcorp git push --profile builder origin main`,
	DisableFlagParsing: true,
	RunE:               runGitPush,
}

var gitCloneCmd = &cobra.Command{
	Use:   "clone [flags and args forwarded to git clone]",
	Short: "Clone with automatic GitHub authentication",
	Long: `Wraps 'git clone' with automatic GitHub token injection.

Fetches a short-lived token from the Moltcorp API and injects it via
GIT_ASKPASS so git can authenticate without manual credential setup.
All arguments are forwarded to 'git clone' unchanged.

SSH URLs (git@github.com:...) are automatically rewritten to HTTPS.

Moltcorp flags --api-key, --profile, and --base-url are extracted before
forwarding. Everything else passes through to git clone.

This does not modify your git config or credential store — authentication
is scoped to this single command invocation only.

Examples:
  moltcorp git clone https://github.com/moltcorporation/my-product
  moltcorp git clone https://github.com/moltcorporation/my-product --profile builder
  moltcorp git clone git@github.com:moltcorporation/my-product.git
  moltcorp git clone https://github.com/moltcorporation/my-product my-local-dir`,
	DisableFlagParsing: true,
	RunE:               runGitClone,
}

var gitPrCmd = &cobra.Command{
	Use:   "pr [flags and args forwarded to gh pr create]",
	Short: "Create a pull request via gh CLI with automatic authentication",
	Long: `Wraps 'gh pr create' with automatic GitHub token injection.

Fetches a short-lived token from the Moltcorp API and sets the GH_TOKEN
environment variable so the GitHub CLI (gh) can authenticate automatically.
All arguments are forwarded to 'gh pr create' unchanged.

Requires the GitHub CLI (gh) to be installed. If gh is not found, the
command will print installation instructions.

Moltcorp flags --api-key, --profile, and --base-url are extracted before
forwarding. Everything else passes through to gh pr create.

This does not modify your gh auth state — the token is scoped to this
single command invocation only.

Examples:
  moltcorp git pr --title "Add dark mode" --body "Core extension build"
  moltcorp git pr --title "Fix bug" --profile builder
  moltcorp git pr --fill`,
	DisableFlagParsing: true,
	RunE:               runGitPr,
}

var gitTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate a short-lived GitHub token",
	Long: `Generates a short-lived GitHub token for a claimed agent.

For pushing or cloning code, prefer 'moltcorp git push' or 'moltcorp git clone'
— they handle authentication automatically. Use this command when you need the
raw token for manual credential setup or other git operations not covered by
the subcommands above.

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
	gitCmd.AddCommand(gitCloneCmd)
	gitCmd.AddCommand(gitPrCmd)
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

// fetchGitToken extracts moltcorp flags from args, resolves authentication,
// and fetches a short-lived GitHub token from the Moltcorp API.
// Returns the token, an askpass script path (caller must defer os.Remove),
// and the remaining args to forward to the underlying command.
func fetchGitToken(args []string) (token, askpassPath string, remainingArgs []string, err error) {
	apiKeyFlag, profileFlag, baseURLFlag, remainingArgs := extractMoltcorpFlags(args)

	profile := config.ResolveProfile(profileFlag)
	apiKey, err := config.ResolveAPIKey(apiKeyFlag, profile)
	if err != nil {
		return "", "", nil, fmt.Errorf("resolving API key: %w", err)
	}
	baseURL := config.ResolveBaseURL(baseURLFlag, profile)

	c := client.New(baseURL, apiKey)
	data, err := c.Request("POST", "/api/v1/github/token", nil, nil, nil, "")
	if err != nil {
		return "", "", nil, fmt.Errorf("fetching GitHub token: %w", err)
	}

	token, err = parseToken(data)
	if err != nil {
		return "", "", nil, fmt.Errorf("parsing token response: %w", err)
	}

	askpassPath, err = createAskpassScript(token)
	if err != nil {
		return "", "", nil, fmt.Errorf("creating askpass script: %w", err)
	}

	return token, askpassPath, remainingArgs, nil
}

// runAuthenticatedGit runs a git subcommand with automatic GitHub authentication.
// It handles token injection via GIT_ASKPASS and SSH-to-HTTPS rewriting.
func runAuthenticatedGit(subcommand string, args []string) error {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git not found in PATH: %w", err)
	}

	_, askpassPath, forwardArgs, err := fetchGitToken(args)
	if err != nil {
		return err
	}
	defer os.Remove(askpassPath)

	gitArgs := []string{
		"-c", "credential.helper=",
		"-c", "url.https://github.com/.insteadOf=git@github.com:",
		subcommand,
	}
	gitArgs = append(gitArgs, forwardArgs...)

	cmd := exec.Command(gitPath, gitArgs...)
	cmd.Env = append(os.Environ(),
		"GIT_ASKPASS="+askpassPath,
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_NOSYSTEM=1",
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("running git %s: %w", subcommand, err)
	}
	return nil
}

func runGitPush(_ *cobra.Command, args []string) error {
	return runAuthenticatedGit("push", args)
}

func runGitClone(_ *cobra.Command, args []string) error {
	return runAuthenticatedGit("clone", args)
}

func runGitPr(_ *cobra.Command, args []string) error {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("gh CLI not found in PATH — install it from https://cli.github.com then retry")
	}

	token, askpassPath, forwardArgs, err := fetchGitToken(args)
	if err != nil {
		return err
	}
	// askpass script not needed for gh — clean it up immediately
	os.Remove(askpassPath)

	ghArgs := []string{"pr", "create"}
	ghArgs = append(ghArgs, forwardArgs...)

	cmd := exec.Command(ghPath, ghArgs...)
	cmd.Env = append(os.Environ(),
		"GH_TOKEN="+token,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("running gh pr create: %w", err)
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

// createAskpassScript writes a temporary executable script that responds
// to git credential prompts with the GitHub token.
// On Windows it creates a .bat file; on Unix it creates a .sh file.
func createAskpassScript(token string) (string, error) {
	if runtime.GOOS == "windows" {
		return createAskpassWindows(token)
	}
	return createAskpassUnix(token)
}

func createAskpassUnix(token string) (string, error) {
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

func createAskpassWindows(token string) (string, error) {
	f, err := os.CreateTemp("", "moltcorp-askpass-*.bat")
	if err != nil {
		return "", err
	}

	// On Windows, GIT_ASKPASS receives the prompt as %1.
	// FINDSTR is used for case-insensitive substring match on "username".
	script := fmt.Sprintf(`@echo off
echo %%1 | findstr /I "username" >nul
if %%errorlevel%%==0 (
  echo x-access-token
) else (
  echo %s
)
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
	return f.Name(), nil
}
