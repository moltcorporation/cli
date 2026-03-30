package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"moltcorp/internal/config"
	"moltcorp/version"
)

// DownloadBase is replaced by sed in the release workflow.
// Do NOT hardcode a URL here.
var DownloadBase = "__DOWNLOAD_BASE__"

const checkInterval = 30 * time.Minute
const networkTimeout = 2 * time.Second

type versionResponse struct {
	Version string `json:"version"`
}

type updateCache struct {
	LastCheck     time.Time `json:"last_check"`
	LatestVersion string    `json:"latest_version"`
}

func cacheFilePath() string {
	return filepath.Join(config.Dir(), "update-check.json")
}

// CheckForUpdateNotice checks for a new version and prints a notice to stderr.
// This is throttled, has a short timeout, and never returns an error.
func CheckForUpdateNotice(cliName string) {
	defer func() { recover() }()

	if shouldSkipCheck() {
		return
	}

	cached, err := loadCache()
	if err == nil && time.Since(cached.LastCheck) < checkInterval {
		if cached.LatestVersion != "" && isNewer(cached.LatestVersion, version.Version) {
			printNotice(cliName, cached.LatestVersion)
		}
		return
	}

	latest, err := fetchLatestVersion()
	if err != nil {
		return
	}

	saveCache(&updateCache{
		LastCheck:     time.Now(),
		LatestVersion: latest,
	})

	if latest != "" && isNewer(latest, version.Version) {
		printNotice(cliName, latest)
	}
}

func shouldSkipCheck() bool {
	if os.Getenv("CI") == "true" || os.Getenv("NO_UPDATE_CHECK") == "true" {
		return true
	}
	if strings.HasPrefix(DownloadBase, "__") {
		return true
	}
	return false
}

func printNotice(cliName, latest string) {
	fmt.Fprintf(os.Stderr, "\nA new version is available: %s → %s\n", version.Version, latest)
	fmt.Fprintf(os.Stderr, "Run `%s update` to update.\n\n", cliName)
}

func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: networkTimeout}
	resp, err := client.Get(DownloadBase + "/latest-version")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var v versionResponse
	if err := json.Unmarshal(body, &v); err != nil {
		return "", err
	}
	return v.Version, nil
}

func loadCache() (*updateCache, error) {
	data, err := os.ReadFile(cacheFilePath())
	if err != nil {
		return nil, err
	}
	var c updateCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func saveCache(c *updateCache) {
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(cacheFilePath()), 0o755)
	os.WriteFile(cacheFilePath(), data, 0o644)
}

// Update downloads and installs the latest version.
func Update() error {
	if strings.HasPrefix(DownloadBase, "__") {
		return fmt.Errorf("update not available in development builds")
	}

	fmt.Fprintln(os.Stderr, "Checking for updates...")

	latest, err := fetchLatestVersion()
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	if !isNewer(latest, version.Version) {
		fmt.Fprintln(os.Stderr, "Already up to date.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Updating %s → %s...\n", version.Version, latest)

	binaryName := fmt.Sprintf("cli-%s-%s", osName(), archName())
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	downloadURL := DownloadBase + "/" + binaryName

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("downloading update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Write to temp file
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	tmpPath := execPath + ".new"
	backupPath := execPath + ".backup"

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("downloading binary: %w", err)
	}
	tmpFile.Close()

	// Make executable
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("setting permissions: %w", err)
	}

	// Atomic replace: current → backup, new → current
	if err := os.Rename(execPath, backupPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("backing up current binary: %w", err)
	}

	if err := os.Rename(tmpPath, execPath); err != nil {
		// Rollback
		os.Rename(backupPath, execPath)
		return fmt.Errorf("installing new binary: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)

	// Update cache
	saveCache(&updateCache{
		LastCheck:     time.Now(),
		LatestVersion: latest,
	})

	fmt.Fprintf(os.Stderr, "Updated to %s.\n", latest)
	return nil
}

// isNewer returns true if remote is a higher semver than local.
// Versions may optionally start with "v". Returns false on parse errors.
func isNewer(remote, local string) bool {
	parse := func(s string) (major, minor, patch int, ok bool) {
		s = strings.TrimPrefix(s, "v")
		parts := strings.SplitN(s, ".", 3)
		if len(parts) != 3 {
			return 0, 0, 0, false
		}
		var err error
		if major, err = strconv.Atoi(parts[0]); err != nil {
			return 0, 0, 0, false
		}
		if minor, err = strconv.Atoi(parts[1]); err != nil {
			return 0, 0, 0, false
		}
		if patch, err = strconv.Atoi(parts[2]); err != nil {
			return 0, 0, 0, false
		}
		return major, minor, patch, true
	}

	rMaj, rMin, rPat, rok := parse(remote)
	lMaj, lMin, lPat, lok := parse(local)
	if !rok || !lok {
		return false
	}
	if rMaj != lMaj {
		return rMaj > lMaj
	}
	if rMin != lMin {
		return rMin > lMin
	}
	return rPat > lPat
}

func osName() string {
	return runtime.GOOS
}

func archName() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	default:
		return runtime.GOARCH
	}
}
