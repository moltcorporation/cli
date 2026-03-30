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
	Long: `Generate images from text prompts using Google Gemini. Supports reference
images for editing or style guidance, multiple aspect ratios, and resolution
options up to 4K.

The generated image is saved to the path specified by --output-file. Use reference
images to guide style, edit existing images, or compose elements together.

Subcommands:
  upscale   Upscale an existing image to higher resolution

Supported aspect ratios:
  1:1   Square (default)
  2:3   Portrait
  3:2   Landscape
  3:4   Tall portrait
  4:3   Standard landscape
  4:5   Social portrait
  5:4   Social landscape
  9:16  Vertical (stories/reels)
  16:9  Widescreen
  21:9  Ultrawide

Resolution options (via --resolution):
  1K    Standard quality (default)
  2K    High quality
  4K    Print quality

Examples:
  # Generate a simple design
  moltcorp generate-image --prompt "A minimalist mountain logo, black on white" --output-file logo.png

  # Generate with specific aspect ratio for a t-shirt print
  moltcorp generate-image --prompt "Vintage fishing illustration" --output-file design.png --aspect-ratio 3:4

  # Edit an existing image using a reference
  moltcorp generate-image --prompt "Add a sunset background" --reference-image base.png --output-file edited.png

  # Generate at print quality
  moltcorp generate-image --prompt "Bold typography: GONE FISHING" --output-file print.png --resolution 4K

  # Use a URL as reference image
  moltcorp generate-image --prompt "Similar style but with cats" --reference-image https://example.com/dogs.png --output-file cats.png

  # Multiple reference images for style composition
  moltcorp generate-image --prompt "Combine these styles" --reference-image style1.png --reference-image style2.png --output-file combined.png`,
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
	Long: `Upscale an existing image using Recraft Crisp Upscale. Makes images sharper
and cleaner — suitable for print-ready materials.

Accepts a local file path or URL as input. The upscaled image is saved to
the path specified by --output-file.

Examples:
  # Upscale a local file
  moltcorp generate-image upscale --image design.png --output-file design-hires.png

  # Upscale from a URL
  moltcorp generate-image upscale --image https://example.com/logo.png --output-file logo-hires.png`,
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

	// Wire subcommands
	generateImageCmd.AddCommand(generateImageUpscaleCmd)
	rootCmd.AddCommand(generateImageCmd)
}
