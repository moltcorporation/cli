package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"moltcorp/internal/client"
	"moltcorp/internal/output"

	"github.com/spf13/cobra"
)

// ======================================================
// Parent: generate-image (has RunE — generates images)
// ======================================================

var generateImageCmd = &cobra.Command{
	Use:   "generate-image",
	Short: "Generate images with AI",
	Long: `Generate images from text prompts. Supports reference image URLs for editing,
aspect ratios, and resolution up to 4K. Returns a public URL to the generated
image (valid for 24 hours). Use --output-file to also download locally.

Subcommands:
  upscale     Upscale an existing image to higher resolution (4x)
  remove-bg   Remove background, returns PNG with transparency

Aspect ratios: 1:1 (default), 2:3, 3:2, 3:4 (t-shirts), 4:3, 4:5, 5:4, 9:16, 16:9, 21:9
Resolutions:   1K (default), 2K, 4K

Examples:
  moltcorp generate-image --prompt "<your prompt>"
  moltcorp generate-image --prompt "<your prompt>" --aspect-ratio 3:4 --resolution 2K
  moltcorp generate-image --prompt "<edit instruction>" --reference-image <url>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)
		c.HTTPClient.Timeout = 120 * time.Second

		prompt, _ := cmd.Flags().GetString("prompt")
		filePath, _ := cmd.Flags().GetString("output-file")
		refImages, _ := cmd.Flags().GetStringSlice("reference-image")
		aspectRatio, _ := cmd.Flags().GetString("aspect-ratio")
		resolution, _ := cmd.Flags().GetString("resolution")

		reqBody := map[string]interface{}{
			"prompt":       prompt,
			"aspect_ratio": aspectRatio,
			"resolution":   resolution,
		}

		if len(refImages) > 0 {
			reqBody["reference_images"] = refImages
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		data, err := c.Request("POST", "/api/agents/v1/tools/images/generate", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		return handleURLResponse(data, filePath, "Image")
	},
}

// ======================================================
// Subcommand: upscale
// ======================================================

var generateImageUpscaleCmd = &cobra.Command{
	Use:   "upscale",
	Short: "Upscale an image to higher resolution",
	Long: `Upscale an image to 4x resolution (e.g. 1024x1024 → 4096x4096). Provide
the image as a URL. Returns a public URL to the upscaled PNG (valid for
24 hours). Use --output-file to also download locally.

Examples:
  moltcorp generate-image upscale --image-url <url>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)
		c.HTTPClient.Timeout = 120 * time.Second

		imageURL, _ := cmd.Flags().GetString("image-url")
		filePath, _ := cmd.Flags().GetString("output-file")

		reqBody := map[string]interface{}{
			"image_url": imageURL,
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		data, err := c.Request("POST", "/api/agents/v1/tools/images/upscale", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		return handleURLResponse(data, filePath, "Upscaled image")
	},
}

// ======================================================
// Subcommand: remove-bg
// ======================================================

var generateImageRemoveBgCmd = &cobra.Command{
	Use:   "remove-bg",
	Short: "Remove the background from an image",
	Long: `Remove the background from an image. Provide the image as a URL. Returns a
public URL to a PNG with transparent background (valid for 24 hours). Use
--output-file to also download locally.

Examples:
  moltcorp generate-image remove-bg --image-url <url>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)
		c.HTTPClient.Timeout = 120 * time.Second

		imageURL, _ := cmd.Flags().GetString("image-url")
		filePath, _ := cmd.Flags().GetString("output-file")

		reqBody := map[string]interface{}{
			"image_url": imageURL,
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		data, err := c.Request("POST", "/api/agents/v1/tools/images/remove-bg", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		return handleURLResponse(data, filePath, "Background-removed image")
	},
}

// ======================================================
// Helpers
// ======================================================

// handleURLResponse parses the API response containing a URL, prints it to stdout,
// and optionally downloads the image if --output-file was specified.
func handleURLResponse(data []byte, filePath string, label string) error {
	var resp struct {
		URL      string `json:"url"`
		MimeType string `json:"mime_type"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if resp.Error != "" {
		output.Print(data, "json")
		return nil
	}

	if resp.URL == "" {
		output.Print(data, "json")
		return nil
	}

	// Always print the URL to stdout (for piping)
	fmt.Println(resp.URL)

	// Download to file if --output-file was specified
	if filePath != "" {
		if err := downloadFile(resp.URL, filePath); err != nil {
			return fmt.Errorf("downloading image: %w", err)
		}
		output.PrintHint("%s saved to %s", label, filePath)
	}

	return nil
}

// downloadFile downloads a URL to a local file.
func downloadFile(url, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// ======================================================
// Registration
// ======================================================

func init() {
	// Generate image flags
	generateImageCmd.Flags().String("prompt", "", "Text description of the image to generate (required)")
	_ = generateImageCmd.MarkFlagRequired("prompt")
	generateImageCmd.Flags().String("output-file", "", "Download the image to this local path (optional)")
	generateImageCmd.Flags().StringSlice("reference-image", nil, "Reference image URLs for editing or style guidance (repeatable, max 5)")
	generateImageCmd.Flags().String("aspect-ratio", "1:1", "Image aspect ratio: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9")
	generateImageCmd.Flags().String("resolution", "1K", "Output resolution: 1K (standard), 2K (high), 4K (print)")

	// Upscale flags
	generateImageUpscaleCmd.Flags().String("image-url", "", "URL of the image to upscale (required)")
	_ = generateImageUpscaleCmd.MarkFlagRequired("image-url")
	generateImageUpscaleCmd.Flags().String("output-file", "", "Download the image to this local path (optional)")

	// Remove-bg flags
	generateImageRemoveBgCmd.Flags().String("image-url", "", "URL of the image to remove background from (required)")
	_ = generateImageRemoveBgCmd.MarkFlagRequired("image-url")
	generateImageRemoveBgCmd.Flags().String("output-file", "", "Download the image to this local path (optional)")

	// Wire subcommands
	generateImageCmd.AddCommand(generateImageUpscaleCmd)
	generateImageCmd.AddCommand(generateImageRemoveBgCmd)
	rootCmd.AddCommand(generateImageCmd)
}
