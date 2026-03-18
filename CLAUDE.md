# CLI Template

A Go CLI template built with Cobra. Designed primarily for AI agents — help text is the only documentation, so it needs to be detailed enough for zero-context usage.

## Architecture

```
main.go              — Entry point. Calls cmd.Execute().
cmd/
  root.go            — Root command, global flags, output mode resolution, command registration.
  configure.go       — Config management. Store/show/clear API key and base URL.
  version.go         — Print version.
  update.go          — Self-update command.
  <resource>.go      — API command files. Each registers via init().
internal/
  client/client.go   — HTTP client. Auth, requests, retry, error handling. Shared by all commands.
  config/config.go   — Config system. Reads/writes ~/.config/{cli-name}/config.json. Also exports API key/base URL resolution.
  output/output.go   — Output formatting. Table (default), JSON, and raw modes. Commands call output.Print().
  pagination/        — Optional pagination helpers (cursor, page, offset).
  updater/updater.go — Self-update. Checks download proxy for new versions. Startup update notice.
version/version.go   — Version variable (injected at build time via ldflags).
docs/
  api-overview.md    — API overview: auth, base URL, conventions, and endpoint TOC.
  SETUP.md           — One-time setup instructions used during initial CLI generation. Not needed for ongoing work.
scripts/
  install.sh         — Unix install script template (processed by release workflow).
  install.ps1        — Windows install script template (processed by release workflow).
```

## Key Conventions

- **Global flags** (`--api-key`, `--base-url`, `--output`, `--json`, `--raw`) are defined in `cmd/root.go` and available to all commands via `cmd.Flag("flag-name").Value.String()`.
- **API key resolution:** flag > environment variable > config file. Handled by `config.ResolveAPIKey()`.
- **Base URL resolution:** flag > config file > default. Handled by `config.ResolveBaseURL()`.
- **Output:** Always call `output.Print(data, ResolveOutputMode(cmd))` from `internal/output/output.go`. The `--output`, `--json`, and `--raw` flags are handled automatically — commands don't need to check them.
- **Errors:** `internal/client/client.go` handles all API errors — prints to stderr and returns an error. Commands just return the error.
- **Data goes to stdout, everything else to stderr.** Critical for piping and agent consumption.
- **No interactive prompts.** All input via flags and positional arguments.
- **Flag naming:** `--kebab-case` in the CLI, mapped back to the API's naming convention (usually snake_case) when constructing requests.

## Help Text

This CLI is designed primarily for AI agents, but also for humans. Help text is the
only documentation — an agent with no prior context should be able to run `--help` and
fully understand what the API does and how to use it. Be detailed and specific at every
level: program description, resource groups, commands, and flags. Don't pad with filler,
but err on the side of more information rather than less.

## Commands Structure

- API commands live in `cmd/` — typically one file per resource group
- Built-in commands (configure, update, version) live in their respective files in `cmd/`
- All commands are registered via `rootCmd.AddCommand()` in each file's `init()` function

## Development

```bash
go run . --help               # Run locally
go run . users list           # Test a specific command
go vet ./...                  # Lint / vet
go build .                    # Build for production
```

## Release

Push a version tag to trigger the release workflow:
```bash
git tag v1.0.0
git push origin main --tags
```
The GitHub Action cross-compiles static binaries for all platforms and creates a release.

## Placeholders

**Do NOT modify (handled by release workflow):**
- `__DOWNLOAD_BASE__` in `internal/updater/updater.go`
- `__CLI_NAME__` and `__DOWNLOAD_BASE__` in `scripts/install.sh` and `scripts/install.ps1`

**Agent replaces during generation:**
- `api-cli` everywhere — replace with the project name
- `__DOWNLOAD_BASE__` in `README.md` only — replace with the download base URL provided in the prompt

## Saving Work

Commit and push your changes as you go — after finishing a batch of resource groups,
after a significant fix, or any time you've made meaningful progress. This preserves
your work if the workflow times out. Only tag a release when the CLI is ready for users.

## If You Get Stuck

If you're fully stuck on something and continued attempts aren't making progress, commit
and push what you have, note what's unresolved in the commit message, and move on. It's
better to ship a CLI missing a few commands than to lose all progress looping on one issue.

## After Making Changes

When everything is implemented and verification passes (`go vet ./...` + `--help` looks good),
tag the release and push:

```bash
git add -A && git commit -m "Build CLI from API reference" && git tag v1.0.0 && git push origin main --tags
```

The version tag triggers the release workflow that builds binaries. Without the tag, no release.

## Adding or Modifying Commands

- To add a new endpoint: add it to the appropriate file in `cmd/` following the existing pattern
- To add a new resource group: create a new file in `cmd/`, define commands, and register via `rootCmd.AddCommand()` in `init()`
- Refer to `docs/api-overview.md` for the endpoint TOC
- Run `go vet ./...` after changes
