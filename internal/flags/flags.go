package flags

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// ResolveBody returns the content for a body/description flag, supporting:
//   - Direct value from --flag "content"
//   - File path from --flag-file /path/to/file
//   - Stdin from --flag - (reads all of stdin)
//
// The flagName is the base name (e.g. "body" or "description").
func ResolveBody(cmd *cobra.Command, flagName string) (string, error) {
	direct, _ := cmd.Flags().GetString(flagName)
	filePath, _ := cmd.Flags().GetString(flagName + "-file")

	// --flag - means read from stdin
	if direct == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	// --flag-file takes precedence when direct is empty
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("reading file %s: %w", filePath, err)
		}
		return string(data), nil
	}

	return direct, nil
}

// ResolveTarget parses target specification from either:
//   - Combined --target "type:id" format
//   - Separate --target-type and --target-id flags
//
// Returns (targetType, targetID, error).
func ResolveTarget(cmd *cobra.Command) (string, string, error) {
	combined, _ := cmd.Flags().GetString("target")
	targetType, _ := cmd.Flags().GetString("target-type")
	targetID, _ := cmd.Flags().GetString("target-id")

	// --target "type:id" takes precedence
	if combined != "" {
		parts := strings.SplitN(combined, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("--target must be in \"type:id\" format (e.g. \"product:abc123\" or \"forum:xyz789\")")
		}
		return parts[0], parts[1], nil
	}

	return targetType, targetID, nil
}

// AddTargetFlags adds --target, --target-type, and --target-id to a command.
// The typeHelp describes valid types (e.g. "product or forum").
func AddTargetFlags(cmd *cobra.Command, typeHelp string, required bool) {
	cmd.Flags().String("target", "", fmt.Sprintf("Target as type:id — e.g. \"%s\"", exampleTarget(typeHelp)))
	cmd.Flags().String("target-type", "", fmt.Sprintf("(alternative to --target) Target type: %s", typeHelp))
	cmd.Flags().String("target-id", "", "(alternative to --target) Target resource ID")

	if required {
		// Mark the group so Cobra knows at least one target form is needed.
		// We validate in ResolveTarget instead of using MarkFlagRequired since
		// the user can provide either --target OR --target-type + --target-id.
		cmd.MarkFlagsOneRequired("target", "target-type")
	}
}

// AddBodyFlags adds --body, --body-file, and optionally marks --body as required.
func AddBodyFlags(cmd *cobra.Command, flagName, help string, required bool) {
	cmd.Flags().String(flagName, "", help)
	cmd.Flags().String(flagName+"-file", "", fmt.Sprintf("Read %s from a file path instead of the flag value", flagName))

	if required {
		cmd.MarkFlagsOneRequired(flagName, flagName+"-file")
	}
}

func exampleTarget(typeHelp string) string {
	// Extract first type for the example
	parts := strings.SplitN(typeHelp, " or ", 2)
	first := strings.TrimSpace(parts[0])
	return first + ":<id>"
}
