# api-cli

Command-line interface for interacting with the API.

## Installation

**macOS / Linux:**

```sh
curl -fsSL __DOWNLOAD_BASE__/install.sh | sh
```

**Windows (PowerShell):**

```powershell
irm __DOWNLOAD_BASE__/install.ps1 | iex
```

## Configuration

Set your API key as an environment variable:

```sh
export API_CLI_API_KEY="your-api-key"
```

Or configure it persistently:

```sh
api-cli configure --api-key your-api-key
```

Or pass it directly with `--api-key`:

```sh
api-cli --api-key your-api-key <command>
```

## Usage

```sh
# Show all commands
api-cli --help

# Use JSON output
api-cli <command> --json

# Print raw API response
api-cli <command> --raw
```

## Global Options

| Option           | Description                          |
| ---------------- | ------------------------------------ |
| `--api-key`      | API key (or set via env variable)    |
| `--base-url`     | Override API base URL                |
| `--output`       | Output format: `json` or `table`     |
| `--json`         | Shorthand for `--output json`        |
| `--raw`          | Print raw API response               |
| `--version`      | Show version number                  |
| `--help`         | Show help                            |

## Updating

To update to the latest version:

```sh
api-cli update
```

To check your current version:

```sh
api-cli version
```

## Available Commands

Commands will be listed here after generation from the API reference.

## Development

```sh
go run . --help
go vet ./...
go build .
```
