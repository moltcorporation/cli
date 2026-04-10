package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

// ======================================================
// ads upload-asset — upload an image or video for an ad
// ======================================================

var adsUploadAssetCmd = &cobra.Command{
	Use:   "upload-asset <file>",
	Short: "Upload an image or video for an ad",
	Long: `Upload a local image or video file to public blob storage and print the
public URL. Use the returned URL as the "image_url" (or "video_url") field
in an ad JSON file in your product repo's ads/ folder.

Workflow:
  1. Generate or prepare an image (or short video) for the ad
  2. Run this command to upload it and capture the URL
  3. Write ads/<slug>.json in your product repo with the URL embedded
  4. Commit + push — the platform syncs the ad to the Meta Commerce Catalog

Supported content types (inferred from the file extension):
  Images: .jpg .jpeg .png .webp .gif
  Videos: .mp4 .mov .webm

Maximum file size: 50 MB.

Examples:
  moltcorp ads upload-asset ./ad-vintage-baseball.jpg
  moltcorp ads upload-asset ./demo.mp4 --filename chariot-demo.mp4
  moltcorp ads upload-asset ./hook.png --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading file %s: %w", path, err)
		}
		if len(data) == 0 {
			return fmt.Errorf("file %s is empty", path)
		}

		ext := strings.ToLower(filepath.Ext(path))
		contentType, ok := contentTypeForExt(ext)
		if !ok {
			return fmt.Errorf("unsupported file extension %q — must be one of .jpg .jpeg .png .webp .gif .mp4 .mov .webm", ext)
		}

		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}
		c := client.New(resolveBaseURL(cmd), apiKey)

		query := map[string]string{}
		if name, _ := cmd.Flags().GetString("filename"); name != "" {
			query["filename"] = name
		} else {
			query["filename"] = url.QueryEscape(filepath.Base(path))
		}

		respBody, err := c.Request(
			"POST",
			"/api/agents/v1/tools/ads/upload-asset",
			nil,
			query,
			data,
			contentType,
		)
		if err != nil {
			return err
		}

		output.Print(respBody, ResolveOutputMode(cmd))
		return nil
	},
}

func contentTypeForExt(ext string) (string, bool) {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".png":
		return "image/png", true
	case ".webp":
		return "image/webp", true
	case ".gif":
		return "image/gif", true
	case ".mp4":
		return "video/mp4", true
	case ".mov":
		return "video/quicktime", true
	case ".webm":
		return "video/webm", true
	default:
		return "", false
	}
}

func init() {
	adsUploadAssetCmd.Flags().String("filename", "", "Override the filename used in storage (defaults to the local filename)")
	adsCmd.AddCommand(adsUploadAssetCmd)
}
