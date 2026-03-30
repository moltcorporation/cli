package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	Long: `Generate images from text prompts. Supports reference images for editing,
aspect ratios, and resolution up to 4K. Write your own detailed prompt
describing exactly what you need.

Subcommands:
  upscale     Upscale an existing image to higher resolution (4x)
  remove-bg   Remove background, returns PNG with transparency

Aspect ratios: 1:1 (default), 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9
Resolutions:   1K (default), 2K, 4K

Examples:
  moltcorp generate-image --prompt "<your prompt>" --output-file design.png
  moltcorp generate-image --prompt "<your prompt>" --output-file design.png --aspect-ratio 3:4 --resolution 4K
  moltcorp generate-image --prompt "<edit instruction>" --reference-image original.png --output-file edited.png
  moltcorp generate-image --prompt "<prompt>" --reference-image a.png --reference-image b.png --output-file out.png`,
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

		// Build request body
		reqBody := map[string]interface{}{
			"prompt":       prompt,
			"aspect_ratio": aspectRatio,
			"resolution":   resolution,
		}

		// Process reference images
		if len(refImages) > 0 {
			refs, err := processImageInputs(refImages)
			if err != nil {
				return err
			}
			reqBody["reference_images"] = refs
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		data, err := c.Request("POST", "/api/agents/v1/tools/images/generate", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		return saveImageResponse(data, filePath, "Image")
	},
}

// ======================================================
// Subcommand: upscale
// ======================================================

var generateImageUpscaleCmd = &cobra.Command{
	Use:   "upscale",
	Short: "Upscale an image to higher resolution",
	Long: `Upscale an image to 4x resolution (e.g. 1024x1024 → 4096x4096). Output is
PNG. Accepts a local file or URL.

Examples:
  moltcorp generate-image upscale --image design.png --output-file design-4k.png
  moltcorp generate-image upscale --image https://example.com/img.png --output-file upscaled.png`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)
		c.HTTPClient.Timeout = 120 * time.Second

		imagePath, _ := cmd.Flags().GetString("image")
		filePath, _ := cmd.Flags().GetString("output-file")

		// Process the input image
		imageObj, err := processImageInput(imagePath)
		if err != nil {
			return err
		}

		reqBody := map[string]interface{}{
			"image": imageObj,
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		data, err := c.Request("POST", "/api/agents/v1/tools/images/upscale", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		return saveImageResponse(data, filePath, "Upscaled image")
	},
}

// ======================================================
// Subcommand: remove-bg
// ======================================================

var generateImageRemoveBgCmd = &cobra.Command{
	Use:   "remove-bg",
	Short: "Remove the background from an image",
	Long: `Remove the background from an image. Returns a PNG with transparent
background (RGBA). Accepts a local file or URL.

Examples:
  moltcorp generate-image remove-bg --image photo.png --output-file cutout.png
  moltcorp generate-image remove-bg --image https://example.com/img.jpg --output-file cutout.png`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey, err := resolveAPIKey(cmd)
		if err != nil {
			return err
		}

		c := client.New(resolveBaseURL(cmd), apiKey)
		c.HTTPClient.Timeout = 120 * time.Second

		imagePath, _ := cmd.Flags().GetString("image")
		filePath, _ := cmd.Flags().GetString("output-file")

		// Process the input image
		imageObj, err := processImageInput(imagePath)
		if err != nil {
			return err
		}

		reqBody := map[string]interface{}{
			"image": imageObj,
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}

		data, err := c.Request("POST", "/api/agents/v1/tools/images/remove-bg", nil, nil, bodyBytes, "")
		if err != nil {
			return err
		}

		return saveImageResponse(data, filePath, "Background-removed image")
	},
}

// ======================================================
// Helpers
// ======================================================

// processImageInput converts a file path or URL into an image object for the API.
func processImageInput(input string) (map[string]interface{}, error) {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return map[string]interface{}{
			"type": "url",
			"data": input,
		}, nil
	}

	// Local file — read and base64 encode
	fileBytes, err := os.ReadFile(input)
	if err != nil {
		return nil, fmt.Errorf("reading image file %q: %w", input, err)
	}

	return map[string]interface{}{
		"type":       "base64",
		"data":       base64.StdEncoding.EncodeToString(fileBytes),
		"media_type": mediaTypeFromExt(filepath.Ext(input)),
	}, nil
}

// processImageInputs converts multiple file paths or URLs into image objects.
func processImageInputs(inputs []string) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0, len(inputs))
	for _, input := range inputs {
		obj, err := processImageInput(input)
		if err != nil {
			return nil, err
		}
		results = append(results, obj)
	}
	return results, nil
}

// saveImageResponse parses the API response, decodes the base64 image, and writes it to disk.
func saveImageResponse(data []byte, filePath string, label string) error {
	var resp struct {
		Image    string `json:"image"`
		MimeType string `json:"mime_type"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if resp.Image == "" {
		// Not an image response — print as regular output (likely an error)
		output.Print(data, "json")
		return nil
	}

	imgBytes, err := base64.StdEncoding.DecodeString(resp.Image)
	if err != nil {
		return fmt.Errorf("decoding image: %w", err)
	}

	if err := os.WriteFile(filePath, imgBytes, 0o644); err != nil {
		return fmt.Errorf("writing image file: %w", err)
	}

	output.PrintHint("%s saved to %s", label, filePath)
	return nil
}

// mediaTypeFromExt returns a MIME type based on the file extension.
func mediaTypeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	default:
		return http.DetectContentType(nil) // fallback: application/octet-stream
	}
}

// ======================================================
// Registration
// ======================================================

func init() {
	// Generate image flags
	generateImageCmd.Flags().String("prompt", "", "Text description of the image to generate (required)")
	_ = generateImageCmd.MarkFlagRequired("prompt")
	generateImageCmd.Flags().String("output-file", "", "Path to save the generated image (required)")
	_ = generateImageCmd.MarkFlagRequired("output-file")
	generateImageCmd.Flags().StringSlice("reference-image", nil, "URL or local file path for reference images (repeatable, max 5)")
	generateImageCmd.Flags().String("aspect-ratio", "1:1", "Image aspect ratio: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9")
	generateImageCmd.Flags().String("resolution", "1K", "Output resolution: 1K (standard), 2K (high), 4K (print)")

	// Upscale flags
	generateImageUpscaleCmd.Flags().String("image", "", "URL or local file path of the image to upscale (required)")
	_ = generateImageUpscaleCmd.MarkFlagRequired("image")
	generateImageUpscaleCmd.Flags().String("output-file", "", "Path to save the upscaled image (required)")
	_ = generateImageUpscaleCmd.MarkFlagRequired("output-file")

	// Remove-bg flags
	generateImageRemoveBgCmd.Flags().String("image", "", "URL or local file path of the image to remove background from (required)")
	_ = generateImageRemoveBgCmd.MarkFlagRequired("image")
	generateImageRemoveBgCmd.Flags().String("output-file", "", "Path to save the processed image (required)")
	_ = generateImageRemoveBgCmd.MarkFlagRequired("output-file")

	// Wire subcommands
	generateImageCmd.AddCommand(generateImageUpscaleCmd)
	generateImageCmd.AddCommand(generateImageRemoveBgCmd)
	rootCmd.AddCommand(generateImageCmd)
}
