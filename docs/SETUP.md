# CLI Generation — Setup Instructions

Read this file first, then `docs/api-overview.md` for the API overview and endpoint TOC.

## The Task

Build a complete CLI from the API documentation. Endpoint specs are in `docs/api-reference/` — organized by resource group.

These CLIs are designed primarily for AI agents, but also for humans. An agent with no prior knowledge of the API should be able to run `--help`, understand what's available, and start using it immediately. The command structure should be logical and intuitive, and help text needs to do the heavy lifting — it's the only documentation users will see.

Use your judgment on how to structure commands, flags, and files based on what makes the best experience for each API.

## API Reference Format

Endpoints in `docs/api-reference/` are documented with a param `in` field:
- **path** → positional argument (goes in URL path)
- **query** → option flag (goes in query string)
- **body** → option flag (goes in request body)
- **header** → option flag (set on request directly — rare)

## Command Example

One command showing how the template utilities fit together:

```go
package cmd

import (
    "encoding/json"
    "fmt"
    "os"

    "api-cli/internal/client"
    "api-cli/internal/config"
    "api-cli/internal/output"

    "github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
    Use:   "users",
    Short: "Manage users",
    Long: "Manage users — user accounts store profile information, " +
        "authentication credentials, roles, and preferences. Users can belong " +
        "to organizations and own resources like projects and API keys.",
}

var usersListCmd = &cobra.Command{
    Use:   "list",
    Short: "List all users",
    Long: `List all users in your organization. Returns paginated results
sorted by creation date. Use filters to narrow by role or account
status. Results include basic profile info — use --json for
full objects including nested settings and permissions.

Examples:
  acme-cli users list
  acme-cli users list --page 2
  acme-cli users list --status active --json`,
    RunE: func(cmd *cobra.Command, args []string) error {
        apiKey, err := config.ResolveAPIKey(cmd.Flag("api-key").Value.String())
        if err != nil {
            return err
        }

        c := client.New(config.ResolveBaseURL(cmd.Flag("base-url").Value.String()), apiKey)

        page, _ := cmd.Flags().GetString("page")
        status, _ := cmd.Flags().GetString("status")

        data, err := c.Request("GET", "/users", nil, map[string]string{
            "page":   page,
            "status": status,
        }, nil, "")
        if err != nil {
            return err
        }

        output.Print(data, ResolveOutputMode(cmd))
        return nil
    },
}

func init() {
    usersListCmd.Flags().String("page", "", "Page number (starts at 1). Default: 1")
    usersListCmd.Flags().String("status", "", "Filter by account status: active, suspended, or invited. Default: active")
    usersCmd.AddCommand(usersListCmd)
    rootCmd.AddCommand(usersCmd)
}
```

Key patterns: `config.ResolveAPIKey()` handles auth resolution, `client.New()` builds the HTTP client, `output.Print()` handles `--json`, `--output`, and `--raw` automatically. See `internal/client/client.go` and `internal/output/output.go` for full APIs.

## Setup Steps

1. **Read `docs/api-overview.md` first.** Understand auth, base URL, conventions, pagination, and scope before writing any code.
2. **Read through `docs/api-reference/`.** Understand the full API surface before deciding on command structure.
3. **Update `internal/config/config.go`** — Set `CLIName`, `DefaultBaseURL`, and `EnvAPIKey`.
4. **Update `cmd/root.go`** — Update `Short` and `Long` descriptions on `rootCmd` (rich, from the API overview).
5. **Create command files** in `cmd/` and register them via `rootCmd.AddCommand()` in each file's `init()`.
6. **Update `README.md`** — Replace `api-cli` with the actual CLI name. Replace `__DOWNLOAD_BASE__` with the download base URL provided in this prompt.
7. **Verify** — see Verification section below.
8. **Commit and release** — commit and push your work as you go (see CLAUDE.md). When done, tag v1.0.0 and push to trigger the release.

## Help Text

Help text is the most important part of this CLI. An AI agent or human encountering this CLI for the first time will run `--help` and nothing else — that output alone needs to tell them what the API does, what each resource is, what each command does, what each flag accepts, and what to expect. Be detailed and specific. Don't pad with filler, but err on the side of more information rather than less.

This applies at every level: the main program description, resource group descriptions, individual command descriptions, and flag descriptions. The api-overview and api-reference docs have rich context — use it.

## Guidance

- **Design for zero-context usage.** Someone running `--help` for the first time should understand what to do without reading anything else.
- **Cover the full API surface.** Don't skip endpoints or parameters unless there's a good reason (e.g., deprecated, redundant, or better exposed differently).
- **No interactive prompts.** AI agents can't use them.
- **The template is a starting point, not a constraint.** If the API needs something the template doesn't support (file uploads, websockets, custom auth flows), extend or modify the code.
- **Pagination:** If the API supports pagination, use the pagination helpers from `internal/pagination/` and add an `--all` flag to list commands.
- **Response envelopes:** Check the API overview Conventions section for response envelope patterns and unwrap accordingly.

## Verification

After implementing all commands:

1. `go vet ./...` — must pass with no errors.
2. `go run . --help` — verify the full command tree is visible and descriptions read well.
3. Spot-check a few commands: `go run . {resource} {command} --help` — verify flags, descriptions, and arguments look correct.

**Do not** attempt actual API calls. There are no API keys in this environment. HTTP errors from missing auth are expected — ignore them.

## Checklist

- [ ] Full API surface is covered
- [ ] Every `--help` screen has detailed, contextual descriptions
- [ ] `go run . --help` shows the complete command tree
- [ ] `go vet ./...` passes
- [ ] README is updated with correct names, env var, and commands
- [ ] Changes committed, tagged v1.0.0, and pushed
