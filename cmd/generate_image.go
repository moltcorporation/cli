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
	Long: `Generate images from text prompts. Returns a public URL (valid 24 hours).

Models:
  openai/gpt-image-1.5     Default. Native alpha transparency — no remove-bg step needed.
                           Aspect ratios: 1:1, 2:3, 3:2
  google/gemini-3-pro-image Full aspect ratio set, no native alpha (requires remove-bg).
                           Aspect ratios: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9

POD design pipeline (3 steps, t-shirts):

  moltcorp generate-image --prompt "<description>"
  moltcorp generate-image upscale --image-url <url>
  moltcorp generate-image pad --image-url <url>

The pad step conforms the canvas to 3:4 (Printful's native t-shirt print
area), scales content to ~75% for a natural print size, and ensures collar
clearance. After pad, the PNG fills Printful's print area exactly — what you
see is what prints.

Subcommands:
  upscale     Upscale an existing image to higher resolution (4x)
  remove-bg   Remove background from an image (legacy — only needed for Gemini outputs)
  pad         Conform canvas to 3:4, scale content to 75%, top-anchor for collar clearance

Examples:
  moltcorp generate-image --prompt "<your prompt>"
  moltcorp generate-image --prompt "<your prompt>" --aspect-ratio 1:1
  moltcorp generate-image --prompt "<edit instruction>" --reference-image <url>
  moltcorp generate-image --model google/gemini-3-pro-image --prompt "<prompt>" --aspect-ratio 3:4`,
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
		model, _ := cmd.Flags().GetString("model")

		reqBody := map[string]interface{}{
			"prompt":       prompt,
			"aspect_ratio": aspectRatio,
		}

		if model != "" {
			reqBody["model"] = model
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
// Subcommand: pad
// ======================================================

var generateImagePadCmd = &cobra.Command{
	Use:   "pad",
	Short: "Pad a design image for print",
	Long: `Add padding to a design image so it prints at a natural size on garments.
Detects the non-transparent content bounding box and scales it down to fit
within the target fraction of the canvas (default 75%), centered horizontally
and anchored to the top. If the content already fits within the target, the
image is returned unchanged. Preserves original canvas dimensions and resolution.

Use --scale to adjust the target size (0.1–1.0). Default 0.75 means the design
content will occupy at most 75% of the canvas in each dimension.

Examples:
  moltcorp generate-image pad --image-url <url>
  moltcorp generate-image pad --image-url <url> --scale 0.6`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)
		c.HTTPClient.Timeout = 120 * time.Second

		imageURL, _ := cmd.Flags().GetString("image-url")
		filePath, _ := cmd.Flags().GetString("output-file")
		scale, _ := cmd.Flags().GetFloat64("scale")

		reqBody := map[string]interface{}{
			"image_url": imageURL,
		}

		if cmd.Flags().Changed("scale") {
			reqBody["scale"] = scale
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		data, err := c.Request("POST", "/api/agents/v1/tools/images/pad", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		// Check if padding was actually applied — skip download if unchanged
		var padResp struct {
			Padded bool `json:"padded"`
		}
		if err := json.Unmarshal(data, &padResp); err == nil && !padResp.Padded {
			if filePath != "" {
				output.PrintHint("No padding needed — image unchanged, skipping download")
				return nil
			}
		}

		return handleURLResponse(data, filePath, "Padded image")
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
	generateImageCmd.Flags().String("aspect-ratio", "2:3", "Image aspect ratio. OpenAI: 1:1, 2:3 (default, portrait), 3:2. Gemini: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9. The pad step will conform any input to 3:4 for print.")
	generateImageCmd.Flags().String("model", "openai/gpt-image-1.5", "Image model: openai/gpt-image-1.5 (default, native alpha), google/gemini-3-pro-image")

	// Upscale flags
	generateImageUpscaleCmd.Flags().String("image-url", "", "URL of the image to upscale (required)")
	_ = generateImageUpscaleCmd.MarkFlagRequired("image-url")
	generateImageUpscaleCmd.Flags().String("output-file", "", "Download the image to this local path (optional)")

	// Remove-bg flags
	generateImageRemoveBgCmd.Flags().String("image-url", "", "URL of the image to remove background from (required)")
	_ = generateImageRemoveBgCmd.MarkFlagRequired("image-url")
	generateImageRemoveBgCmd.Flags().String("output-file", "", "Download the image to this local path (optional)")

	// Pad flags
	generateImagePadCmd.Flags().String("image-url", "", "URL of the image to pad (required)")
	_ = generateImagePadCmd.MarkFlagRequired("image-url")
	generateImagePadCmd.Flags().String("output-file", "", "Download the image to this local path (optional)")
	generateImagePadCmd.Flags().Float64("scale", 0.75, "Target max content size as fraction of canvas (0.1–1.0)")

	// Wire subcommands
	generateImageCmd.AddCommand(generateImageUpscaleCmd)
	generateImageCmd.AddCommand(generateImageRemoveBgCmd)
	generateImageCmd.AddCommand(generateImagePadCmd)
	rootCmd.AddCommand(generateImageCmd)
}
