package flags

import (
	"fmt"
	"io"
	"os"

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

// AddParentFlags registers explicit parent-type flags on a command.
// parentTypes is e.g. []string{"post"} or []string{"product", "forum"}.
// If required is true, cobra enforces that exactly one is provided.
func AddParentFlags(cmd *cobra.Command, parentTypes []string, required bool) {
	for _, pt := range parentTypes {
		cmd.Flags().String(pt, "", fmt.Sprintf("The %s id to target", pt))
	}

	if len(parentTypes) > 1 {
		cmd.MarkFlagsMutuallyExclusive(parentTypes...)
	}

	if required {
		cmd.MarkFlagsOneRequired(parentTypes...)
	}
}

// ResolveParent reads the explicit parent flags and returns (type, id, error).
// Errors if more than one parent flag is set (should be caught by cobra's
// mutual exclusion, but checked defensively).
func ResolveParent(cmd *cobra.Command, parentTypes []string) (string, string, error) {
	var foundType, foundID string
	for _, pt := range parentTypes {
		val, _ := cmd.Flags().GetString(pt)
		if val != "" {
			if foundType != "" {
				return "", "", fmt.Errorf("only one parent flag allowed, got --%s and --%s", foundType, pt)
			}
			foundType = pt
			foundID = val
		}
	}
	return foundType, foundID, nil
}

// AddBodyFlags adds --body, --body-file, and optionally marks --body as required.
func AddBodyFlags(cmd *cobra.Command, flagName, help string, required bool) {
	cmd.Flags().String(flagName, "", help)
	cmd.Flags().String(flagName+"-file", "", fmt.Sprintf("Read %s from a file path instead of the flag value", flagName))

	if required {
		cmd.MarkFlagsOneRequired(flagName, flagName+"-file")
	}
}
