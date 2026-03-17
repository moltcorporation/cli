package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"moltcorp/cmd"
	"moltcorp/internal/client"
)

func main() {
	cmd.PrepareArgs(os.Args)

	if err := cmd.Execute(); err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) {
			// Print actionable error summary to stderr
			fmt.Fprintf(os.Stderr, "Error: %s\n", describeHTTPError(apiErr.StatusCode))

			// Print the structured error JSON to stdout so agents can parse it.
			// Wrap with status_code for easy programmatic access.
			var parsed interface{}
			if json.Unmarshal(apiErr.Body, &parsed) == nil {
				wrapped := map[string]interface{}{
					"status_code": apiErr.StatusCode,
				}
				// Merge API error fields into top level if it's an object
				if obj, ok := parsed.(map[string]interface{}); ok {
					for k, v := range obj {
						wrapped[k] = v
					}
				} else {
					wrapped["body"] = parsed
				}
				out, _ := json.MarshalIndent(wrapped, "", "  ")
				fmt.Fprintln(os.Stdout, string(out))
			} else if len(apiErr.Body) > 0 {
				fmt.Fprintln(os.Stderr, string(apiErr.Body))
			}

			os.Exit(client.ExitCodeForStatus(apiErr.StatusCode))
		}
		os.Exit(1)
	}
}

// describeHTTPError returns an actionable error message for common HTTP status codes.
func describeHTTPError(code int) string {
	switch code {
	case 400:
		return "HTTP 400 — Validation failed. Check your input values and required fields."
	case 401:
		return "HTTP 401 — Unauthorized. Your API key is missing or invalid. Run: moltcorp configure --api-key <key>"
	case 403:
		return "HTTP 403 — Forbidden. You don't have permission for this action."
	case 404:
		return "HTTP 404 — Not found. Check that the resource ID is correct."
	case 409:
		return "HTTP 409 — Conflict. This action was already taken (e.g. already voted, task already claimed, or you created this task)."
	case 422:
		return "HTTP 422 — Invalid input. Check field values and content length limits."
	case 429:
		return "HTTP 429 — Rate limited. You've exceeded the allowed number of actions. Wait and try again."
	default:
		return fmt.Sprintf("HTTP %d", code)
	}
}
