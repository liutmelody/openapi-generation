package snapshots

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	generationaccess "github.com/speakeasy-api/generation-context/access"
	"github.com/speakeasy-api/openapi-generation/v2/pkg/generate"
	"github.com/speakeasy-api/openapi-generation/v2/pkg/generate/snapshots/snaptest"
	"github.com/stretchr/testify/require"
)

const cliReleaseSnapshotSpec = `openapi: 3.0.3
info:
  title: Petstore CLI
  description: "Petstore CLI installable via package managers"
  version: 0.1.0
paths:
  /pets:
    get:
      operationId: listPets
      tags: [pets]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: array
                items:
                  type: string
`

var cliReleaseExpectedSnapshotFiles = []string{
	".speakeasy/gen.yaml",
	".goreleaser.yaml",
	".github/workflows/release.yaml",
	"README.md",
	"scripts/install.sh",
	"scripts/install.ps1",
}

func TestSnapCLIReleaseBase(t *testing.T) {
	t.Parallel()

	genYaml := `cli:
  packageName: github.com/example/petstore-cli
  cliName: petstore
  envVarPrefix: PETSTORE
  generateRelease: true
`

	expectedSnapshot := `--- .github/workflows/release.yaml ---
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          fetch-depth: 0

      - name: Check release version
        run: |
          tag_version="${GITHUB_REF_NAME#v}"
          cli_version=$(sed -n 's/^var Version = "\(.*\)"$/\1/p' internal/cli/version.go)
          if [ -z "$cli_version" ] || [ "$tag_version" != "$cli_version" ]; then
            printf '::error::Release tag %s does not match generated CLI version %s. Set cli.version in .speakeasy/gen.yaml to %s and regenerate before tagging.\n' "$GITHUB_REF_NAME" "$cli_version" "$tag_version"
            exit 1
          fi

      - name: Setup Go
        uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: "go.mod"
          cache: true
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@f06c13b6b1a9625abc9e6e439d9c05a8f2190e94 # v7.2.3
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Upload artifacts
        uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
        if: always()
        with:
          name: release-artifacts
          path: |
            dist/
            !dist/*.txt
          retention-days: 30


--- .goreleaser.yaml ---
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

project_name: petstore

before:
  hooks:
    - go mod tidy

builds:
  - id: petstore
    main: ./cmd/petstore
    binary: petstore
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.buildTime={{.Date}}

archives:
  - id: petstore
    formats: [tar.gz]
    name_template: >-
      {{ .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    format_overrides:
      - goos: windows
        formats: [zip]
    files:
      - README.md
      - LICENSE*

release:
  github:
    owner: example
    name: petstore-cli
  draft: false
  prerelease: auto
  mode: append

checksum:
  name_template: "checksums.txt"


--- .speakeasy/gen.yaml ---
configVersion: 2.0.0
generation:
  sdkClassName: SDK
  maintainOpenAPIOrder: true
  usageSnippets:
    optionalPropertyRendering: withExample
    sdkInitStyle: constructor
  useClassNamesForArrayFields: true
  fixes:
    nameResolutionDec2023: true
    nameResolutionFeb2025: true
    parameterOrderingFeb2024: true
    requestResponseComponentNamesFeb2024: true
    securityFeb2025: true
    sharedErrorComponentsApr2025: true
    sharedNestedComponentsJan2026: true
    nameOverrideFeb2026: true
    openSingleValueEnumsOct2026: true
  auth:
    oAuth2ClientCredentialsEnabled: true
    oAuth2PasswordEnabled: true
    hoistGlobalSecurity: true
  inferSSEOverload: true
  sdkHooksConfigAccess: true
  schemas:
    allOfMergeStrategy: shallowMerge
  requestBodyFieldName: body
  versioningStrategy: automatic
  persistentEdits: {}
  tests:
    generateTests: false
    generateNewTests: true
    skipResponseBodyAssertions: false
cli:
  version: 0.0.1
  additionalDependencies: {}
  agentEnvironmentDetection: true
  arrayFlagFormat: repeatable
  classifiedErrors: false
  cliName: petstore
  defaultColor: auto
  defaultTimeout: ""
  distribution:
    homebrew:
      enabled: false
      tap: ""
    nfpm:
      enabled: false
      formats: deb,rpm
      license: ""
      maintainer: ""
    winget:
      enabled: false
      license: ""
      packageIdentifier: ""
      publisher: ""
      publisherUrl: ""
      repositoryOwner: ""
  enableCustomCodeRegions: false
  envVarPrefix: PETSTORE
  fixFlags:
    encodePathParams: true
  generateRelease: true
  helpStyle: auto
  idiomaticMethodCollisionNames: true
  imports:
    option: openapi
    paths:
      callbacks: models/callbacks
      errors: models/apierrors
      operations: models/operations
      shared: models/components
      webhooks: models/webhooks
  interactiveAuth: true
  interactiveByDefault: true
  interactiveMode: true
  interactiveTheme:
    accentColor: '#38BDF8'
    dimmedColor: '#64748B'
    errorColor: '#F87171'
    subtleColor: '#475569'
    successColor: '#4ADE80'
  jqRawOutput: false
  packageName: github.com/example/petstore-cli
  removeStutter: true
  retryFlagsVisibility: visible
  retryMethodPolicy: spec
  rootVersionFlag: true
  serverSelectionFlag: visible


--- README.md ---
# petstore

Command-line interface for the *Petstore CLI* API.

[![Built by Speakeasy](https://img.shields.io/badge/Built_by-SPEAKEASY-374151?style=for-the-badge&labelColor=f3f4f6)](https://www.speakeasy.com/?utm_source=github-com/example/petstore-cli&utm_campaign=cli)
[![License: AGPL-3.0-only](https://img.shields.io/badge/LICENSE_//_AGPL--3.0--only-3b5bdb?style=for-the-badge&labelColor=eff6ff)](https://www.gnu.org/licenses/agpl-3.0.html)


<br /><br />
> [!IMPORTANT]
> This CLI is not yet ready for production use. Delete this notice before publishing to a package manager.

<!-- Start Summary [summary] -->
## Summary

Petstore CLI: Petstore CLI installable via package managers
<!-- End Summary [summary] -->

<!-- Start Table of Contents [toc] -->
## Table of Contents
<!-- $toc-max-depth=2 -->
* [petstore](#petstore)
  * [CLI Installation](#cli-installation)
  * [Shell Completion](#shell-completion)
  * [CLI Example Usage](#cli-example-usage)
  * [For AI agents](#for-ai-agents)
  * [Commands](#commands)
  * [Request Body Input](#request-body-input)
  * [Output Formats](#output-formats)
  * [Error Handling](#error-handling)
  * [Diagnostics](#diagnostics)
* [Development](#development)
  * [Maturity](#maturity)
  * [Contributions](#contributions)

<!-- End Table of Contents [toc] -->

<!-- Start CLI Installation [installation] -->
## CLI Installation

### Quick Install (Linux/macOS)

` + "`" + `` + "`" + `` + "`" + `bash
curl -fsSL https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.sh | bash
` + "`" + `` + "`" + `` + "`" + `

### Quick Install (Windows PowerShell)

` + "`" + `` + "`" + `` + "`" + `powershell
iwr -useb https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.ps1 | iex
` + "`" + `` + "`" + `` + "`" + `

### Go Install

Alternatively, install directly via Go:

` + "`" + `` + "`" + `` + "`" + `bash
go install github.com/example/petstore-cli/cmd/petstore@latest
` + "`" + `` + "`" + `` + "`" + `

### Manual Download

Download pre-built binaries for your platform from the [releases page](https://github.com/example/petstore-cli/releases).
<!-- End CLI Installation [installation] -->

<!-- Start Shell Completion [completion] -->
## Shell Completion

Shell completions are available for Bash, Zsh, Fish, and PowerShell.

### Bash

` + "`" + `` + "`" + `` + "`" + `bash
# Add to ~/.bashrc:
source <(petstore completion bash)

# Or install permanently:
petstore completion bash > /etc/bash_completion.d/petstore
` + "`" + `` + "`" + `` + "`" + `

### Zsh

` + "`" + `` + "`" + `` + "`" + `zsh
# Add to ~/.zshrc:
source <(petstore completion zsh)

# Or install permanently:
petstore completion zsh > "${fpath[1]}/_petstore"
` + "`" + `` + "`" + `` + "`" + `

### Fish

` + "`" + `` + "`" + `` + "`" + `fish
petstore completion fish | source

# Or install permanently:
petstore completion fish > ~/.config/fish/completions/petstore.fish
` + "`" + `` + "`" + `` + "`" + `

### PowerShell

` + "`" + `` + "`" + `` + "`" + `powershell
petstore completion powershell | Out-String | Invoke-Expression
` + "`" + `` + "`" + `` + "`" + `
<!-- End Shell Completion [completion] -->

<!-- Start CLI Example Usage [usage] -->
## CLI Example Usage

### Example

` + "`" + `` + "`" + `` + "`" + `bash
petstore pets list

` + "`" + `` + "`" + `` + "`" + `
<!-- End CLI Example Usage [usage] -->

<!-- Start For AI agents [agents] -->
## For AI agents

This CLI is built to be driven by AI coding agents as well as people: everything an agent needs is discoverable from the binary itself, and every command can be validated without credentials. Work down this ladder:

| Run | You get |
|-----|---------|
| ` + "`" + `petstore --help` + "`" + `, ` + "`" + `petstore pets list --help` + "`" + ` | Commands by category, runnable examples, flags |
| ` + "`" + `petstore --usage` + "`" + `, ` + "`" + `petstore pets list --usage` + "`" + ` | The command surface as machine-readable [KDL](https://kdl.dev): commands, aliases, flags, defaults, env vars, config keys |
| ` + "`" + `petstore pets list --dry-run` + "`" + ` | The exact HTTP request (method, URL, headers, body), with no credentials or network call |
| ` + "`" + `petstore pets list --output-format json` + "`" + ` (or ` + "`" + `--jq` + "`" + `) | Machine-readable output |

### Discover the command surface

` + "`" + `` + "`" + `` + "`" + `bash
# Every command, flag, default, env var and config key, as KDL
petstore --usage

# One command's subtree only
petstore pets list --usage
` + "`" + `` + "`" + `` + "`" + `

### Probe before you spend

Start quota-spending commands with ` + "`" + `--dry-run` + "`" + `. It validates inputs, resolves the request, redacts secrets and binary payloads, makes no network call, and exits 0. It never reads the OS keychain; credentials supplied by flag, environment, or config file are included only as ` + "`" + `[REDACTED]` + "`" + `.

` + "`" + `` + "`" + `` + "`" + `bash
# Human preview: the [DRY-RUN] block is on stderr and stdout is empty
petstore pets list --dry-run

# Machine preview: compact JSON on stdout and silent stderr
petstore pets list --dry-run --output-format json
` + "`" + `` + "`" + `` + "`" + `

The machine form writes one object per would-be request, one per line (NDJSON for multi-request commands), with exactly this shape:

` + "`" + `` + "`" + `` + "`" + `json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
` + "`" + `` + "`" + `` + "`" + `

` + "`" + `body` + "`" + ` is a parsed JSON value when the body is JSON, a string for text, ` + "`" + `"<bytes:N>"` + "`" + ` for binary data, and ` + "`" + `null` + "`" + ` when absent. An explicit caller ` + "`" + `--jq` + "`" + ` also selects this JSON preview protocol, but the filter is not applied to preview objects. Command-declared jq presets do not select or filter the preview.

Local mutation commands make no request under ` + "`" + `--dry-run` + "`" + `: instead of a preview they emit one ` + "`" + `{"dry_run":true,"local":true,"command":"…","message":"…"}` + "`" + ` object. ` + "`" + `select(.request)` + "`" + ` keeps only would-be requests; ` + "`" + `select(.local)` + "`" + ` keeps the local no-ops.

### Machine-readable output

` + "`" + `` + "`" + `` + "`" + `bash
# JSON on stdout
petstore pets list --output-format json

# Filter or reshape with a jq expression (always emits JSON, overrides --output-format)
petstore pets list --jq '.'

# Print jq string results as plain text instead of JSON strings (like jq -r)
petstore pets list --jq '.' --raw-output
` + "`" + `` + "`" + `` + "`" + `

` + "`" + `--output-format toon` + "`" + ` emits [TOON](https://github.com/toon-format/spec), a compact line-oriented format that uses fewer tokens than JSON; it is the default in agent mode.

### Interactive mode
Required-input prompts and guided ` + "`" + `configure` + "`" + ` / ` + "`" + `auth login` + "`" + ` forms are enabled by default. Required-input prompts require an interactive terminal; off-TTY forms read line input from stdin. Use ` + "`" + `--no-interactive` + "`" + ` to force flag-only execution.

` + "`" + `` + "`" + `` + "`" + `bash
# Prompt for missing command inputs
petstore pets list --interactive

# Open the guided configuration form
petstore configure --interactive

# Explicitly launch the terminal command explorer
petstore explore
` + "`" + `` + "`" + `` + "`" + `

### Agent mode and structured errors
Agent mode turns on automatically when a known agent environment is detected (` + "`" + `CLAUDECODE` + "`" + `, ` + "`" + `CURSOR_AGENT` + "`" + `, ` + "`" + `CODEX` + "`" + `, ` + "`" + `AIDER` + "`" + `, ` + "`" + `CLINE` + "`" + `, ` + "`" + `WINDSURF_AGENT` + "`" + `, ` + "`" + `GITHUB_COPILOT` + "`" + `, ` + "`" + `AMAZON_Q` + "`" + `, ` + "`" + `GEMINI_CODE_ASSIST` + "`" + `, ` + "`" + `SRC_CODY` + "`" + `) or with ` + "`" + `--agent-mode` + "`" + ` (` + "`" + `--agent-mode=false` + "`" + ` disables detection).
In agent mode interactive prompts never launch, output defaults to TOON, and every failure — API errors and CLI usage errors alike — is one JSON envelope on stderr:
Outside agent mode, explicit JSON and ` + "`" + `--jq` + "`" + ` preserve the compatibility envelope without classification; enable agent mode to request the classified contract.

` + "`" + `` + "`" + `` + "`" + `json
{
  "error": "...",
  "error_type": "validation_error",
  "error_reason": "CLI_VALIDATION",
  "exit_code": 2,
  "message": "human-readable message",
  "hints": ["what to try next"]
}
` + "`" + `` + "`" + `` + "`" + `

` + "`" + `error_type` + "`" + ` is one of ` + "`" + `authentication_error` + "`" + `, ` + "`" + `authorization_error` + "`" + `, ` + "`" + `not_found` + "`" + `, ` + "`" + `validation_error` + "`" + `, ` + "`" + `rate_limit_error` + "`" + `, ` + "`" + `server_error` + "`" + `, ` + "`" + `api_error` + "`" + `, ` + "`" + `connection_error` + "`" + `, ` + "`" + `protocol_error` + "`" + `, ` + "`" + `runtime_error` + "`" + `, ` + "`" + `unsupported_error` + "`" + `, ` + "`" + `async_failed` + "`" + `, ` + "`" + `async_timeout` + "`" + `, ` + "`" + `async_unknown_state` + "`" + `. Classification derives from the HTTP status and transport evidence; ` + "`" + `error_reason` + "`" + ` is absent for API errors. Status-less local failures may use ` + "`" + `CLI_VALIDATION` + "`" + `, ` + "`" + `CLI_CONNECTION` + "`" + `, ` + "`" + `CLI_PROTOCOL` + "`" + `, ` + "`" + `CLI_RUNTIME` + "`" + `, ` + "`" + `CLI_UNAVAILABLE` + "`" + `, ` + "`" + `CLI_AUTHENTICATION` + "`" + `, the async polling reasons ` + "`" + `CLI_ASYNC_FAILED` + "`" + `, ` + "`" + `CLI_ASYNC_TIMEOUT` + "`" + `, and ` + "`" + `CLI_ASYNC_UNKNOWN_STATE` + "`" + `, or ` + "`" + `CLI_OUTCOME_FAILED` + "`" + ` when a successful response reports a declared failure state; that envelope also carries ` + "`" + `state` + "`" + ` (the reported state) and, when the command declares an error pointer, ` + "`" + `error` + "`" + ` (the failure detail taken from the response). ` + "`" + `hints` + "`" + ` preserves server guidance first, adds the most specific local taxonomy guidance, then typed CLI and command-specific guidance, removing exact duplicates. ` + "`" + `exit_code` + "`" + ` is always the code for the final ` + "`" + `error_type` + "`" + ` shown in the envelope: 1 runtime, 2 usage, or 3 authentication/authorization.
<!-- End For AI agents [agents] -->

<!-- Start Commands [operations] -->
## Commands

<details open>
<summary>Available commands</summary>

* [` + "`" + `pets` + "`" + `](docs/petstore_pets.md) - Operations for pets
  * [` + "`" + `list` + "`" + `](docs/petstore_pets_list.md)

</details>
<!-- End Commands [operations] -->

<!-- Start Request Body Input [stdinpiping] -->
## Request Body Input

Commands that accept a request body take it three ways, with a clear priority chain: individual field flags (highest priority), the whole body as JSON via ` + "`" + `--body` + "`" + `, and JSON piped on stdin (lowest priority). Later sources never override earlier ones.
<!-- End Request Body Input [stdinpiping] -->

<!-- Start Output Formats [output-formats] -->
## Output Formats

Every command supports a ` + "`" + `--output-format` + "`" + ` flag that controls how the response is rendered to stdout.

### Available formats

| Format | Flag | Description |
|--------|------|-------------|
| Pretty | ` + "`" + `--output-format pretty` + "`" + ` (default) | Aligned key-value pairs with color, nested indentation. Human-readable at a glance. |
| JSON | ` + "`" + `--output-format json` + "`" + ` | JSON output. Passthrough when the response is already JSON (preserves original field order and numeric precision). Falls back to typed marshaling otherwise. |
| YAML | ` + "`" + `--output-format yaml` + "`" + ` | YAML output via standard marshaling. |
| Table | ` + "`" + `--output-format table` + "`" + ` | Tabular output for array responses. |
| TOON | ` + "`" + `--output-format toon` + "`" + ` | [Token-Oriented Object Notation](https://github.com/toon-format/spec) — a compact, line-oriented format that typically uses 30–60% fewer tokens than JSON. Well-suited for piping responses into LLM prompts. |

` + "`" + `` + "`" + `` + "`" + `bash
# Default pretty output
petstore pets list

# Machine-readable JSON
petstore pets list --output-format json

# TOON for LLM-friendly compact output
petstore pets list --output-format toon

# Pipe JSON to jq without using --output-format
petstore pets list --output-format json | jq '.'
` + "`" + `` + "`" + `` + "`" + `

### jq filtering

Use ` + "`" + `--jq` + "`" + ` to filter or transform the response inline using a [jq](https://jqlang.org) expression. This always outputs JSON and overrides ` + "`" + `--output-format` + "`" + `:

` + "`" + `` + "`" + `` + "`" + `bash
# Extract a single field
petstore pets list --jq '.'

# Reshape with any jq program; --raw-output prints string results as plain text (like jq -r)
petstore pets list --jq '.' --raw-output
` + "`" + `` + "`" + `` + "`" + `

### Color control

Use ` + "`" + `--color` + "`" + ` to control terminal colors:

| Value | Behavior |
|-------|----------|
| ` + "`" + `auto` + "`" + ` (default) | Color when stdout is a TTY, plain text otherwise |
| ` + "`" + `always` + "`" + ` | Always colorize |
| ` + "`" + `never` + "`" + ` | Never colorize |

The ` + "`" + `NO_COLOR` + "`" + ` and ` + "`" + `FORCE_COLOR` + "`" + ` environment variables are also respected.

### Streaming and pagination

When using ` + "`" + `--all` + "`" + ` (pagination) or streaming operations, output is written incrementally as items arrive:

| Format | Streaming behavior |
|--------|-------------------|
| ` + "`" + `json` + "`" + ` | One compact JSON object per line ([NDJSON](https://github.com/ndjson/ndjson-spec)) |
| ` + "`" + `yaml` + "`" + ` | YAML documents separated by ` + "`" + `---` + "`" + ` |
| ` + "`" + `toon` + "`" + ` | One TOON-encoded object per block, separated by blank lines |
| ` + "`" + `pretty` + "`" + ` (default) | Pretty-printed items separated by blank lines |
<!-- End Output Formats [output-formats] -->

<!-- Start Error Handling [errors] -->
## Error Handling

The CLI uses standard exit codes to indicate success or failure:

| Exit Code | Meaning |
|-----------|---------|
| ` + "`" + `0` + "`" + ` | Success |
| ` + "`" + `1` + "`" + ` | Runtime/API failure |
| ` + "`" + `2` + "`" + ` | Usage or input failure |
| ` + "`" + `3` + "`" + ` | Authentication or authorization failure |

On success, the response data is printed to **stdout** as JSON. On failure, error details are printed to **stderr**.

` + "`" + `` + "`" + `` + "`" + `bash
# Capture output and handle errors
petstore pets list --output-format json > output.json 2> error.log
if [ $? -ne 0 ]; then
  echo "Error occurred, see error.log"
fi
` + "`" + `` + "`" + `` + "`" + `
This CLI uses unclassified error rendering outside agent mode: pretty and TOON print the API error text as received, while ` + "`" + `--output-format json` + "`" + ` and ` + "`" + `--jq` + "`" + ` emit the unclassified envelope (including the configure ` + "`" + `_hint` + "`" + ` for HTTP 401/403) plus ` + "`" + `exit_code` + "`" + `. Agent mode always emits the classified JSON envelope with ` + "`" + `exit_code` + "`" + `, ` + "`" + `error_type` + "`" + `, optional ` + "`" + `error_reason` + "`" + `, ` + "`" + `message` + "`" + `, ` + "`" + `hints` + "`" + `, and optional ` + "`" + `status_code` + "`" + ` — see [For AI agents](#for-ai-agents).
<!-- End Error Handling [errors] -->

<!-- Start Diagnostics [diagnostics] -->
## Diagnostics

The CLI includes two diagnostic flags available on all commands:

### Dry Run

Preview what would be sent without making any network calls:

` + "`" + `` + "`" + `` + "`" + `bash
petstore pets list --dry-run
` + "`" + `` + "`" + `` + "`" + `

In human output modes, stdout is empty and the ` + "`" + `[DRY-RUN]` + "`" + ` block goes to stderr. It includes:
- HTTP method and URL
- Request headers (sensitive values redacted)
- Request body preview (sensitive fields redacted)

With ` + "`" + `--output-format json` + "`" + `, or with a caller-explicit ` + "`" + `--jq` + "`" + `, stderr is silent and stdout is NDJSON: one compact preview object per would-be request. The jq filter is not applied, and command-declared jq presets do not select the JSON protocol.

` + "`" + `` + "`" + `` + "`" + `json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
` + "`" + `` + "`" + `` + "`" + `

JSON bodies remain structured; text bodies are strings; binary bodies are ` + "`" + `"<bytes:N>"` + "`" + `; absent bodies are ` + "`" + `null` + "`" + `. Headers retain all values as arrays, with credentials replaced by ` + "`" + `[REDACTED]` + "`" + `. Dry-run never reads the OS keychain, but credentials supplied by flag, environment, or config file still appear redacted. The command exits successfully without contacting the API.

Local mutation commands emit one ` + "`" + `{"dry_run":true,"local":true,"command":"…","message":"…"}` + "`" + ` object in place of a preview; filter with ` + "`" + `select(.request)` + "`" + ` or ` + "`" + `select(.local)` + "`" + `.

### Debug

Log request and response diagnostics while running normally:

` + "`" + `` + "`" + `` + "`" + `bash
petstore pets list --debug
` + "`" + `` + "`" + `` + "`" + `

Debug output goes to stderr and includes:
- Request method, URL, headers, and body preview
- Response status, headers, and body preview
- Transport errors (if any)

The command still executes normally and produces its regular output on stdout.

### Flag Precedence

If both ` + "`" + `--dry-run` + "`" + ` and ` + "`" + `--debug` + "`" + ` are set, ` + "`" + `--dry-run` + "`" + ` takes precedence and no network calls are made.

### Security

Sensitive information is automatically redacted in diagnostic output:
- **Headers**: ` + "`" + `Authorization` + "`" + `, ` + "`" + `Cookie` + "`" + `, ` + "`" + `Set-Cookie` + "`" + `, ` + "`" + `X-API-Key` + "`" + `, and other security headers show ` + "`" + `[REDACTED]` + "`" + `
- **Body**: JSON fields named ` + "`" + `password` + "`" + `, ` + "`" + `secret` + "`" + `, ` + "`" + `token` + "`" + `, ` + "`" + `api_key` + "`" + `, ` + "`" + `client_secret` + "`" + `, etc. show ` + "`" + `[REDACTED]` + "`" + `
- **Binary data**: binary media and canonical base64 strings are replaced with ` + "`" + `<bytes:N>` + "`" + `
- **URL query**: credential-like query parameters are replaced with ` + "`" + `[REDACTED]` + "`" + `

Diagnostic output should still be treated as potentially sensitive operational data.
<!-- End Diagnostics [diagnostics] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This CLI is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

This CLI is generated programmatically. Edits to generated files are overwritten on regeneration. To customize it:

- **Configuration and behavior:** Use [OpenAPI overlays](https://www.speakeasy.com/docs/prep-openapi/overlays/create-overlays) in the Speakeasy workflow with ` + "`" + `x-speakeasy-*` + "`" + ` extensions (for example, ` + "`" + `x-speakeasy-cli-commands` + "`" + `) to define commands, flags, help text, examples, authentication, and grouping.
- **Persistent code changes:** Store unified diffs as [patch files](https://www.speakeasy.com/docs/sdks/customize/code/patch-files/patch-files) at ` + "`" + `.speakeasy/patches/<path-of-generated-file>.patch` + "`" + `; they are re-applied on every generation.

### CLI Created by [Speakeasy](https://www.speakeasy.com/?utm_source=github-com/example/petstore-cli&utm_campaign=cli)


--- scripts/install.ps1 ---
#
# petstore CLI Installation Script for Windows
# This script downloads and installs the latest version of the petstore CLI
#
# Usage:
#   iwr -useb https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.ps1 | iex
#   or
#   Invoke-WebRequest -Uri https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.ps1 -UseBasicParsing | Invoke-Expression
#
# Options:
#   $env:PETSTORE_INSTALL_DIR - Installation directory (default: $env:LOCALAPPDATA\Programs\petstore)
#   $env:PETSTORE_VERSION     - Specific version to install (default: latest)
#

[CmdletBinding()]
param()

# Configuration
$Repo = "example/petstore-cli"
$BinaryName = "petstore.exe"
$DefaultInstallDir = Join-Path $env:LOCALAPPDATA "Programs\petstore"
$InstallDir = if ($env:PETSTORE_INSTALL_DIR) { $env:PETSTORE_INSTALL_DIR } else { $DefaultInstallDir }
$Version = if ($env:PETSTORE_VERSION) { $env:PETSTORE_VERSION } else { "latest" }

# Helper functions
function Write-ColorOutput {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        return $response.tag_name
    }
    catch {
        throw "Failed to get latest version: $_"
    }
}

function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "x86_64" }
        "ARM64" { return "arm64" }
        default {
            throw "Unsupported architecture: $arch"
        }
    }
}

function Install-CLI {
    Write-ColorOutput "Installing petstore CLI..." -Color Green

    # Detect architecture
    $arch = Get-Architecture
    Write-ColorOutput "Detected Architecture: $arch" -Color Cyan

    # Get version
    if ($Version -eq "latest") {
        $Version = Get-LatestVersion
        Write-ColorOutput "Latest version: $Version" -Color Cyan
    }

    # Construct download URL
    $archiveName = "petstore_Windows_$arch.zip"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$archiveName"

    Write-ColorOutput "Downloading from: $downloadUrl" -Color Cyan

    # Create temporary directory
    $tempDir = Join-Path $env:TEMP "petstore-install-$(New-Guid)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

    try {
        # Download archive
        $archivePath = Join-Path $tempDir $archiveName
        try {
            Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing
        }
        catch {
            throw "Failed to download from ${downloadUrl}: $_"
        }

        Write-ColorOutput "Download complete" -Color Green

        # Extract archive
        Write-ColorOutput "Extracting archive..." -Color Cyan
        Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force

        # Create install directory if it doesn't exist
        if (-not (Test-Path $InstallDir)) {
            Write-ColorOutput "Creating installation directory: $InstallDir" -Color Cyan
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }

        # Install binary
        $binaryPath = Join-Path $InstallDir $BinaryName
        Write-ColorOutput "Installing to $binaryPath..." -Color Cyan

        # Remove existing binary if it exists
        if (Test-Path $binaryPath) {
            Remove-Item $binaryPath -Force
        }

        Copy-Item -Path (Join-Path $tempDir $BinaryName) -Destination $binaryPath -Force

        Write-ColorOutput "petstore $Version has been installed to $binaryPath" -Color Green

        # Add to PATH if not already there
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$InstallDir*") {
            Write-ColorOutput "Adding $InstallDir to your PATH..." -Color Cyan
            [Environment]::SetEnvironmentVariable(
                "Path",
                "$userPath;$InstallDir",
                "User"
            )
            $env:Path = "$env:Path;$InstallDir"
            Write-ColorOutput "Added to PATH. You may need to restart your terminal for changes to take effect." -Color Yellow
        }

        Write-ColorOutput "Installation successful! Run 'petstore --help' to get started." -Color Green
        Write-ColorOutput "Note: You may need to restart your terminal or run 'refreshenv' for the PATH changes to take effect." -Color Yellow
    }
    finally {
        # Cleanup
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force
        }
    }
}

# Main execution
& {
    $ErrorActionPreference = 'Stop'

    try {
        Install-CLI
    }
    catch {
        throw "Installation failed: $_"
    }
}


--- scripts/install.sh ---
#!/usr/bin/env bash
#
# petstore CLI Installation Script
# This script downloads and installs the latest version of the petstore CLI
# for Linux and macOS systems.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.sh | bash
#   or
#   wget -qO- https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.sh | bash
#
# Options:
#   PETSTORE_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#   PETSTORE_VERSION     - Specific version to install (default: latest)
#

set -e

# Configuration
REPO="example/petstore-cli"
DEFAULT_INSTALL_DIR="/usr/local/bin"
USER_INSTALL_DIR="$HOME/.local/bin"
VERSION="${PETSTORE_VERSION:-latest}"
BINARY_NAME="petstore"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect operating system
detect_os() {
    local os
    local uname_output="$(uname -s)"
    case "$uname_output" in
        Linux*)     os="Linux" ;;
        Darwin*)    os="Darwin" ;;
        CYGWIN*|MINGW*|MSYS*)    os="Windows" ;;
        *)
            log_error "Unsupported operating system: $uname_output"
            exit 1
            ;;
    esac
    echo "$os"
}

# Detect architecture
detect_arch() {
    local arch
    case "$(uname -m)" in
        x86_64|amd64)   arch="x86_64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    echo "$arch"
}

# Get latest version from GitHub
get_latest_version() {
    local latest_url="https://api.github.com/repos/${REPO}/releases/latest"
    local version

    if command -v curl >/dev/null 2>&1; then
        version=$(curl -fsSL "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        log_error "curl or wget is required to download the CLI"
        exit 1
    fi

    echo "$version"
}

# Determine installation directory
get_install_dir() {
    # If user specified a directory, use it
    if [ -n "${PETSTORE_INSTALL_DIR}" ]; then
        echo "${PETSTORE_INSTALL_DIR}"
        return
    fi

    # Try to use /usr/local/bin if we have write access
    if [ -w "$DEFAULT_INSTALL_DIR" ] || [ -w "$(dirname "$DEFAULT_INSTALL_DIR")" ]; then
        echo "$DEFAULT_INSTALL_DIR"
        return
    fi

    # Fall back to user directory
    log_info "No write access to $DEFAULT_INSTALL_DIR, using $USER_INSTALL_DIR instead" >&2
    echo "$USER_INSTALL_DIR"
}

# Download and install
install_cli() {
    local INSTALL_DIR=$(get_install_dir)
    local os=$(detect_os)
    local arch=$(detect_arch)

    log_info "Detected OS: $os"
    log_info "Detected Architecture: $arch"
    log_info "Installation directory: $INSTALL_DIR"

    # Get version
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(get_latest_version)
        log_info "Latest version: $VERSION"
    fi

    # Construct download URL based on OS
    local archive_name
    local archive_format
    if [ "$os" = "Windows" ]; then
        archive_name="${BINARY_NAME}_${os}_${arch}.zip"
        archive_format="zip"
    else
        archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
        archive_format="tar.gz"
    fi

    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${archive_name}"

    log_info "Downloading from: $download_url"

    # Create temporary directory
    local tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Download archive
    if command -v curl >/dev/null 2>&1; then
        if ! curl -fsSL "$download_url" -o "$tmp_dir/$archive_name"; then
            log_error "Failed to download from $download_url"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q "$download_url" -O "$tmp_dir/$archive_name"; then
            log_error "Failed to download from $download_url"
            exit 1
        fi
    fi

    log_info "Download complete"

    # Extract archive based on format
    log_info "Extracting archive..."
    if [ "$archive_format" = "zip" ]; then
        if command -v unzip >/dev/null 2>&1; then
            unzip -q "$tmp_dir/$archive_name" -d "$tmp_dir"
        else
            log_error "unzip is required to extract the archive. Please install unzip and try again."
            exit 1
        fi
    else
        tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"
    fi

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating installation directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR" || {
            log_error "Failed to create $INSTALL_DIR. Try running with sudo or set PETSTORE_INSTALL_DIR to a writable location."
            exit 1
        }
    fi

    # Install binary (Windows binaries have .exe extension)
    local source_binary="$tmp_dir/$BINARY_NAME"
    local target_binary="$INSTALL_DIR/$BINARY_NAME"

    if [ "$os" = "Windows" ]; then
        source_binary="$tmp_dir/${BINARY_NAME}.exe"
        target_binary="$INSTALL_DIR/${BINARY_NAME}.exe"
    fi

    log_info "Installing to $target_binary..."
    if ! mv "$source_binary" "$target_binary"; then
        log_error "Failed to install to $INSTALL_DIR. Try running with sudo or set PETSTORE_INSTALL_DIR to a writable location."
        exit 1
    fi

    # Make executable (not needed on Windows, but doesn't hurt)
    chmod +x "$target_binary" 2>/dev/null || true

    log_info "petstore ${VERSION} has been installed to $target_binary"

    # Verify installation
    local cmd_to_check="$BINARY_NAME"
    if [ "$os" = "Windows" ]; then
        cmd_to_check="${BINARY_NAME}.exe"
    fi

    if command -v "$cmd_to_check" >/dev/null 2>&1; then
        log_info "Installation successful! Run '$BINARY_NAME --help' to get started."
    else
        log_warn "Installation complete, but $BINARY_NAME is not in your PATH."
        if [ "$os" = "Windows" ]; then
            log_warn "Add $INSTALL_DIR to your PATH environment variable."
        else
            log_warn "Add $INSTALL_DIR to your PATH by adding this to your ~/.bashrc or ~/.zshrc:"
            log_warn "  export PATH=\"\$PATH:$INSTALL_DIR\""
            log_warn ""
            log_warn "Then run: source ~/.bashrc  # or source ~/.zshrc"
        fi
    fi
}

# Main execution
main() {
    log_info "Installing petstore CLI..."
    install_cli
}

main


` // end of snapshot

	snaptest.DoTestSnapshot(t, snaptest.Options{
		Spec:         cliReleaseSnapshotSpec,
		GenYaml:      genYaml,
		IncludeGlobs: cliReleaseExpectedSnapshotFiles,
		Expected:     expectedSnapshot,
	})
}

func TestSnapCLIReleaseHomebrewOnly(t *testing.T) {
	t.Parallel()

	genYaml := `cli:
  packageName: github.com/example/petstore-cli
  cliName: petstore
  envVarPrefix: PETSTORE
  generateRelease: true
  distribution:
    homebrew:
      enabled: true
      tap: example/homebrew-petstore
`

	expectedSnapshot := `--- .github/workflows/release.yaml ---
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          fetch-depth: 0

      - name: Check release version
        run: |
          tag_version="${GITHUB_REF_NAME#v}"
          cli_version=$(sed -n 's/^var Version = "\(.*\)"$/\1/p' internal/cli/version.go)
          if [ -z "$cli_version" ] || [ "$tag_version" != "$cli_version" ]; then
            printf '::error::Release tag %s does not match generated CLI version %s. Set cli.version in .speakeasy/gen.yaml to %s and regenerate before tagging.\n' "$GITHUB_REF_NAME" "$cli_version" "$tag_version"
            exit 1
          fi

      - name: Setup Go
        uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: "go.mod"
          cache: true
      - name: Import GPG key
        uses: crazy-max/ghaction-import-gpg@2dc316deee8e90f13e1a351ab510b4d5bc0c82cd # v7.0.0
        id: import_gpg
        with:
          gpg_private_key: ${{ secrets.CLI_GPG_SECRET_KEY }}
          passphrase: ${{ secrets.CLI_GPG_PASSPHRASE }}
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@f06c13b6b1a9625abc9e6e439d9c05a8f2190e94 # v7.2.3
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
          # Required when Homebrew tap publishing is enabled.
          # Classic PATs need public_repo (public tap) or repo (private tap).
          # Fine-grained PATs need Contents: Read and write on example/homebrew-petstore.
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}

      - name: Upload artifacts
        uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
        if: always()
        with:
          name: release-artifacts
          path: |
            dist/
            !dist/*.txt
          retention-days: 30


--- .goreleaser.yaml ---
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

project_name: petstore

before:
  hooks:
    - go mod tidy

builds:
  - id: petstore
    main: ./cmd/petstore
    binary: petstore
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.buildTime={{.Date}}

archives:
  - id: petstore
    formats: [tar.gz]
    name_template: >-
      {{ .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    format_overrides:
      - goos: windows
        formats: [zip]
    files:
      - README.md
      - LICENSE*

brews:
  - name: petstore
    repository:
      owner: example
      name: homebrew-petstore
      branch: main
    token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: "https://github.com/example/petstore-cli"
    description: "Petstore CLI: Petstore CLI installable via package managers"
    install: |
      bin.install "petstore"

release:
  github:
    owner: example
    name: petstore-cli
  draft: false
  prerelease: auto
  mode: append

checksum:
  name_template: "checksums.txt"

signs:
  - artifacts: checksum
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"


--- .speakeasy/gen.yaml ---
configVersion: 2.0.0
generation:
  sdkClassName: SDK
  maintainOpenAPIOrder: true
  usageSnippets:
    optionalPropertyRendering: withExample
    sdkInitStyle: constructor
  useClassNamesForArrayFields: true
  fixes:
    nameResolutionDec2023: true
    nameResolutionFeb2025: true
    parameterOrderingFeb2024: true
    requestResponseComponentNamesFeb2024: true
    securityFeb2025: true
    sharedErrorComponentsApr2025: true
    sharedNestedComponentsJan2026: true
    nameOverrideFeb2026: true
    openSingleValueEnumsOct2026: true
  auth:
    oAuth2ClientCredentialsEnabled: true
    oAuth2PasswordEnabled: true
    hoistGlobalSecurity: true
  inferSSEOverload: true
  sdkHooksConfigAccess: true
  schemas:
    allOfMergeStrategy: shallowMerge
  requestBodyFieldName: body
  versioningStrategy: automatic
  persistentEdits: {}
  tests:
    generateTests: false
    generateNewTests: true
    skipResponseBodyAssertions: false
cli:
  version: 0.0.1
  additionalDependencies: {}
  agentEnvironmentDetection: true
  arrayFlagFormat: repeatable
  classifiedErrors: false
  cliName: petstore
  defaultColor: auto
  defaultTimeout: ""
  distribution:
    homebrew:
      enabled: true
      tap: example/homebrew-petstore
    nfpm:
      enabled: false
      formats: deb,rpm
      license: ""
      maintainer: ""
    winget:
      enabled: false
      license: ""
      packageIdentifier: ""
      publisher: ""
      publisherUrl: ""
      repositoryOwner: ""
  enableCustomCodeRegions: false
  envVarPrefix: PETSTORE
  fixFlags:
    encodePathParams: true
  generateRelease: true
  helpStyle: auto
  idiomaticMethodCollisionNames: true
  imports:
    option: openapi
    paths:
      callbacks: models/callbacks
      errors: models/apierrors
      operations: models/operations
      shared: models/components
      webhooks: models/webhooks
  interactiveAuth: true
  interactiveByDefault: true
  interactiveMode: true
  interactiveTheme:
    accentColor: '#38BDF8'
    dimmedColor: '#64748B'
    errorColor: '#F87171'
    subtleColor: '#475569'
    successColor: '#4ADE80'
  jqRawOutput: false
  packageName: github.com/example/petstore-cli
  removeStutter: true
  retryFlagsVisibility: visible
  retryMethodPolicy: spec
  rootVersionFlag: true
  serverSelectionFlag: visible


--- README.md ---
# petstore

Command-line interface for the *Petstore CLI* API.

[![Built by Speakeasy](https://img.shields.io/badge/Built_by-SPEAKEASY-374151?style=for-the-badge&labelColor=f3f4f6)](https://www.speakeasy.com/?utm_source=github-com/example/petstore-cli&utm_campaign=cli)
[![License: AGPL-3.0-only](https://img.shields.io/badge/LICENSE_//_AGPL--3.0--only-3b5bdb?style=for-the-badge&labelColor=eff6ff)](https://www.gnu.org/licenses/agpl-3.0.html)


<br /><br />
> [!IMPORTANT]
> This CLI is not yet ready for production use. Delete this notice before publishing to a package manager.

<!-- Start Summary [summary] -->
## Summary

Petstore CLI: Petstore CLI installable via package managers
<!-- End Summary [summary] -->

<!-- Start Table of Contents [toc] -->
## Table of Contents
<!-- $toc-max-depth=2 -->
* [petstore](#petstore)
  * [CLI Installation](#cli-installation)
  * [Shell Completion](#shell-completion)
  * [CLI Example Usage](#cli-example-usage)
  * [For AI agents](#for-ai-agents)
  * [Commands](#commands)
  * [Request Body Input](#request-body-input)
  * [Output Formats](#output-formats)
  * [Error Handling](#error-handling)
  * [Diagnostics](#diagnostics)
* [Development](#development)
  * [Maturity](#maturity)
  * [Contributions](#contributions)

<!-- End Table of Contents [toc] -->

<!-- Start CLI Installation [installation] -->
## CLI Installation

### Quick Install (Linux/macOS)

` + "`" + `` + "`" + `` + "`" + `bash
curl -fsSL https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.sh | bash
` + "`" + `` + "`" + `` + "`" + `

### Quick Install (Windows PowerShell)

` + "`" + `` + "`" + `` + "`" + `powershell
iwr -useb https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.ps1 | iex
` + "`" + `` + "`" + `` + "`" + `
### Homebrew (macOS/Linux)

` + "`" + `` + "`" + `` + "`" + `bash
brew install example/petstore/petstore
` + "`" + `` + "`" + `` + "`" + `

### Go Install

Alternatively, install directly via Go:

` + "`" + `` + "`" + `` + "`" + `bash
go install github.com/example/petstore-cli/cmd/petstore@latest
` + "`" + `` + "`" + `` + "`" + `

### Manual Download

Download pre-built binaries for your platform from the [releases page](https://github.com/example/petstore-cli/releases).
<!-- End CLI Installation [installation] -->

<!-- Start Shell Completion [completion] -->
## Shell Completion

Shell completions are available for Bash, Zsh, Fish, and PowerShell.

### Bash

` + "`" + `` + "`" + `` + "`" + `bash
# Add to ~/.bashrc:
source <(petstore completion bash)

# Or install permanently:
petstore completion bash > /etc/bash_completion.d/petstore
` + "`" + `` + "`" + `` + "`" + `

### Zsh

` + "`" + `` + "`" + `` + "`" + `zsh
# Add to ~/.zshrc:
source <(petstore completion zsh)

# Or install permanently:
petstore completion zsh > "${fpath[1]}/_petstore"
` + "`" + `` + "`" + `` + "`" + `

### Fish

` + "`" + `` + "`" + `` + "`" + `fish
petstore completion fish | source

# Or install permanently:
petstore completion fish > ~/.config/fish/completions/petstore.fish
` + "`" + `` + "`" + `` + "`" + `

### PowerShell

` + "`" + `` + "`" + `` + "`" + `powershell
petstore completion powershell | Out-String | Invoke-Expression
` + "`" + `` + "`" + `` + "`" + `
<!-- End Shell Completion [completion] -->

<!-- Start CLI Example Usage [usage] -->
## CLI Example Usage

### Example

` + "`" + `` + "`" + `` + "`" + `bash
petstore pets list

` + "`" + `` + "`" + `` + "`" + `
<!-- End CLI Example Usage [usage] -->

<!-- Start For AI agents [agents] -->
## For AI agents

This CLI is built to be driven by AI coding agents as well as people: everything an agent needs is discoverable from the binary itself, and every command can be validated without credentials. Work down this ladder:

| Run | You get |
|-----|---------|
| ` + "`" + `petstore --help` + "`" + `, ` + "`" + `petstore pets list --help` + "`" + ` | Commands by category, runnable examples, flags |
| ` + "`" + `petstore --usage` + "`" + `, ` + "`" + `petstore pets list --usage` + "`" + ` | The command surface as machine-readable [KDL](https://kdl.dev): commands, aliases, flags, defaults, env vars, config keys |
| ` + "`" + `petstore pets list --dry-run` + "`" + ` | The exact HTTP request (method, URL, headers, body), with no credentials or network call |
| ` + "`" + `petstore pets list --output-format json` + "`" + ` (or ` + "`" + `--jq` + "`" + `) | Machine-readable output |

### Discover the command surface

` + "`" + `` + "`" + `` + "`" + `bash
# Every command, flag, default, env var and config key, as KDL
petstore --usage

# One command's subtree only
petstore pets list --usage
` + "`" + `` + "`" + `` + "`" + `

### Probe before you spend

Start quota-spending commands with ` + "`" + `--dry-run` + "`" + `. It validates inputs, resolves the request, redacts secrets and binary payloads, makes no network call, and exits 0. It never reads the OS keychain; credentials supplied by flag, environment, or config file are included only as ` + "`" + `[REDACTED]` + "`" + `.

` + "`" + `` + "`" + `` + "`" + `bash
# Human preview: the [DRY-RUN] block is on stderr and stdout is empty
petstore pets list --dry-run

# Machine preview: compact JSON on stdout and silent stderr
petstore pets list --dry-run --output-format json
` + "`" + `` + "`" + `` + "`" + `

The machine form writes one object per would-be request, one per line (NDJSON for multi-request commands), with exactly this shape:

` + "`" + `` + "`" + `` + "`" + `json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
` + "`" + `` + "`" + `` + "`" + `

` + "`" + `body` + "`" + ` is a parsed JSON value when the body is JSON, a string for text, ` + "`" + `"<bytes:N>"` + "`" + ` for binary data, and ` + "`" + `null` + "`" + ` when absent. An explicit caller ` + "`" + `--jq` + "`" + ` also selects this JSON preview protocol, but the filter is not applied to preview objects. Command-declared jq presets do not select or filter the preview.

Local mutation commands make no request under ` + "`" + `--dry-run` + "`" + `: instead of a preview they emit one ` + "`" + `{"dry_run":true,"local":true,"command":"…","message":"…"}` + "`" + ` object. ` + "`" + `select(.request)` + "`" + ` keeps only would-be requests; ` + "`" + `select(.local)` + "`" + ` keeps the local no-ops.

### Machine-readable output

` + "`" + `` + "`" + `` + "`" + `bash
# JSON on stdout
petstore pets list --output-format json

# Filter or reshape with a jq expression (always emits JSON, overrides --output-format)
petstore pets list --jq '.'

# Print jq string results as plain text instead of JSON strings (like jq -r)
petstore pets list --jq '.' --raw-output
` + "`" + `` + "`" + `` + "`" + `

` + "`" + `--output-format toon` + "`" + ` emits [TOON](https://github.com/toon-format/spec), a compact line-oriented format that uses fewer tokens than JSON; it is the default in agent mode.

### Interactive mode
Required-input prompts and guided ` + "`" + `configure` + "`" + ` / ` + "`" + `auth login` + "`" + ` forms are enabled by default. Required-input prompts require an interactive terminal; off-TTY forms read line input from stdin. Use ` + "`" + `--no-interactive` + "`" + ` to force flag-only execution.

` + "`" + `` + "`" + `` + "`" + `bash
# Prompt for missing command inputs
petstore pets list --interactive

# Open the guided configuration form
petstore configure --interactive

# Explicitly launch the terminal command explorer
petstore explore
` + "`" + `` + "`" + `` + "`" + `

### Agent mode and structured errors
Agent mode turns on automatically when a known agent environment is detected (` + "`" + `CLAUDECODE` + "`" + `, ` + "`" + `CURSOR_AGENT` + "`" + `, ` + "`" + `CODEX` + "`" + `, ` + "`" + `AIDER` + "`" + `, ` + "`" + `CLINE` + "`" + `, ` + "`" + `WINDSURF_AGENT` + "`" + `, ` + "`" + `GITHUB_COPILOT` + "`" + `, ` + "`" + `AMAZON_Q` + "`" + `, ` + "`" + `GEMINI_CODE_ASSIST` + "`" + `, ` + "`" + `SRC_CODY` + "`" + `) or with ` + "`" + `--agent-mode` + "`" + ` (` + "`" + `--agent-mode=false` + "`" + ` disables detection).
In agent mode interactive prompts never launch, output defaults to TOON, and every failure — API errors and CLI usage errors alike — is one JSON envelope on stderr:
Outside agent mode, explicit JSON and ` + "`" + `--jq` + "`" + ` preserve the compatibility envelope without classification; enable agent mode to request the classified contract.

` + "`" + `` + "`" + `` + "`" + `json
{
  "error": "...",
  "error_type": "validation_error",
  "error_reason": "CLI_VALIDATION",
  "exit_code": 2,
  "message": "human-readable message",
  "hints": ["what to try next"]
}
` + "`" + `` + "`" + `` + "`" + `

` + "`" + `error_type` + "`" + ` is one of ` + "`" + `authentication_error` + "`" + `, ` + "`" + `authorization_error` + "`" + `, ` + "`" + `not_found` + "`" + `, ` + "`" + `validation_error` + "`" + `, ` + "`" + `rate_limit_error` + "`" + `, ` + "`" + `server_error` + "`" + `, ` + "`" + `api_error` + "`" + `, ` + "`" + `connection_error` + "`" + `, ` + "`" + `protocol_error` + "`" + `, ` + "`" + `runtime_error` + "`" + `, ` + "`" + `unsupported_error` + "`" + `, ` + "`" + `async_failed` + "`" + `, ` + "`" + `async_timeout` + "`" + `, ` + "`" + `async_unknown_state` + "`" + `. Classification derives from the HTTP status and transport evidence; ` + "`" + `error_reason` + "`" + ` is absent for API errors. Status-less local failures may use ` + "`" + `CLI_VALIDATION` + "`" + `, ` + "`" + `CLI_CONNECTION` + "`" + `, ` + "`" + `CLI_PROTOCOL` + "`" + `, ` + "`" + `CLI_RUNTIME` + "`" + `, ` + "`" + `CLI_UNAVAILABLE` + "`" + `, ` + "`" + `CLI_AUTHENTICATION` + "`" + `, the async polling reasons ` + "`" + `CLI_ASYNC_FAILED` + "`" + `, ` + "`" + `CLI_ASYNC_TIMEOUT` + "`" + `, and ` + "`" + `CLI_ASYNC_UNKNOWN_STATE` + "`" + `, or ` + "`" + `CLI_OUTCOME_FAILED` + "`" + ` when a successful response reports a declared failure state; that envelope also carries ` + "`" + `state` + "`" + ` (the reported state) and, when the command declares an error pointer, ` + "`" + `error` + "`" + ` (the failure detail taken from the response). ` + "`" + `hints` + "`" + ` preserves server guidance first, adds the most specific local taxonomy guidance, then typed CLI and command-specific guidance, removing exact duplicates. ` + "`" + `exit_code` + "`" + ` is always the code for the final ` + "`" + `error_type` + "`" + ` shown in the envelope: 1 runtime, 2 usage, or 3 authentication/authorization.
<!-- End For AI agents [agents] -->

<!-- Start Commands [operations] -->
## Commands

<details open>
<summary>Available commands</summary>

* [` + "`" + `pets` + "`" + `](docs/petstore_pets.md) - Operations for pets
  * [` + "`" + `list` + "`" + `](docs/petstore_pets_list.md)

</details>
<!-- End Commands [operations] -->

<!-- Start Request Body Input [stdinpiping] -->
## Request Body Input

Commands that accept a request body take it three ways, with a clear priority chain: individual field flags (highest priority), the whole body as JSON via ` + "`" + `--body` + "`" + `, and JSON piped on stdin (lowest priority). Later sources never override earlier ones.
<!-- End Request Body Input [stdinpiping] -->

<!-- Start Output Formats [output-formats] -->
## Output Formats

Every command supports a ` + "`" + `--output-format` + "`" + ` flag that controls how the response is rendered to stdout.

### Available formats

| Format | Flag | Description |
|--------|------|-------------|
| Pretty | ` + "`" + `--output-format pretty` + "`" + ` (default) | Aligned key-value pairs with color, nested indentation. Human-readable at a glance. |
| JSON | ` + "`" + `--output-format json` + "`" + ` | JSON output. Passthrough when the response is already JSON (preserves original field order and numeric precision). Falls back to typed marshaling otherwise. |
| YAML | ` + "`" + `--output-format yaml` + "`" + ` | YAML output via standard marshaling. |
| Table | ` + "`" + `--output-format table` + "`" + ` | Tabular output for array responses. |
| TOON | ` + "`" + `--output-format toon` + "`" + ` | [Token-Oriented Object Notation](https://github.com/toon-format/spec) — a compact, line-oriented format that typically uses 30–60% fewer tokens than JSON. Well-suited for piping responses into LLM prompts. |

` + "`" + `` + "`" + `` + "`" + `bash
# Default pretty output
petstore pets list

# Machine-readable JSON
petstore pets list --output-format json

# TOON for LLM-friendly compact output
petstore pets list --output-format toon

# Pipe JSON to jq without using --output-format
petstore pets list --output-format json | jq '.'
` + "`" + `` + "`" + `` + "`" + `

### jq filtering

Use ` + "`" + `--jq` + "`" + ` to filter or transform the response inline using a [jq](https://jqlang.org) expression. This always outputs JSON and overrides ` + "`" + `--output-format` + "`" + `:

` + "`" + `` + "`" + `` + "`" + `bash
# Extract a single field
petstore pets list --jq '.'

# Reshape with any jq program; --raw-output prints string results as plain text (like jq -r)
petstore pets list --jq '.' --raw-output
` + "`" + `` + "`" + `` + "`" + `

### Color control

Use ` + "`" + `--color` + "`" + ` to control terminal colors:

| Value | Behavior |
|-------|----------|
| ` + "`" + `auto` + "`" + ` (default) | Color when stdout is a TTY, plain text otherwise |
| ` + "`" + `always` + "`" + ` | Always colorize |
| ` + "`" + `never` + "`" + ` | Never colorize |

The ` + "`" + `NO_COLOR` + "`" + ` and ` + "`" + `FORCE_COLOR` + "`" + ` environment variables are also respected.

### Streaming and pagination

When using ` + "`" + `--all` + "`" + ` (pagination) or streaming operations, output is written incrementally as items arrive:

| Format | Streaming behavior |
|--------|-------------------|
| ` + "`" + `json` + "`" + ` | One compact JSON object per line ([NDJSON](https://github.com/ndjson/ndjson-spec)) |
| ` + "`" + `yaml` + "`" + ` | YAML documents separated by ` + "`" + `---` + "`" + ` |
| ` + "`" + `toon` + "`" + ` | One TOON-encoded object per block, separated by blank lines |
| ` + "`" + `pretty` + "`" + ` (default) | Pretty-printed items separated by blank lines |
<!-- End Output Formats [output-formats] -->

<!-- Start Error Handling [errors] -->
## Error Handling

The CLI uses standard exit codes to indicate success or failure:

| Exit Code | Meaning |
|-----------|---------|
| ` + "`" + `0` + "`" + ` | Success |
| ` + "`" + `1` + "`" + ` | Runtime/API failure |
| ` + "`" + `2` + "`" + ` | Usage or input failure |
| ` + "`" + `3` + "`" + ` | Authentication or authorization failure |

On success, the response data is printed to **stdout** as JSON. On failure, error details are printed to **stderr**.

` + "`" + `` + "`" + `` + "`" + `bash
# Capture output and handle errors
petstore pets list --output-format json > output.json 2> error.log
if [ $? -ne 0 ]; then
  echo "Error occurred, see error.log"
fi
` + "`" + `` + "`" + `` + "`" + `
This CLI uses unclassified error rendering outside agent mode: pretty and TOON print the API error text as received, while ` + "`" + `--output-format json` + "`" + ` and ` + "`" + `--jq` + "`" + ` emit the unclassified envelope (including the configure ` + "`" + `_hint` + "`" + ` for HTTP 401/403) plus ` + "`" + `exit_code` + "`" + `. Agent mode always emits the classified JSON envelope with ` + "`" + `exit_code` + "`" + `, ` + "`" + `error_type` + "`" + `, optional ` + "`" + `error_reason` + "`" + `, ` + "`" + `message` + "`" + `, ` + "`" + `hints` + "`" + `, and optional ` + "`" + `status_code` + "`" + ` — see [For AI agents](#for-ai-agents).
<!-- End Error Handling [errors] -->

<!-- Start Diagnostics [diagnostics] -->
## Diagnostics

The CLI includes two diagnostic flags available on all commands:

### Dry Run

Preview what would be sent without making any network calls:

` + "`" + `` + "`" + `` + "`" + `bash
petstore pets list --dry-run
` + "`" + `` + "`" + `` + "`" + `

In human output modes, stdout is empty and the ` + "`" + `[DRY-RUN]` + "`" + ` block goes to stderr. It includes:
- HTTP method and URL
- Request headers (sensitive values redacted)
- Request body preview (sensitive fields redacted)

With ` + "`" + `--output-format json` + "`" + `, or with a caller-explicit ` + "`" + `--jq` + "`" + `, stderr is silent and stdout is NDJSON: one compact preview object per would-be request. The jq filter is not applied, and command-declared jq presets do not select the JSON protocol.

` + "`" + `` + "`" + `` + "`" + `json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
` + "`" + `` + "`" + `` + "`" + `

JSON bodies remain structured; text bodies are strings; binary bodies are ` + "`" + `"<bytes:N>"` + "`" + `; absent bodies are ` + "`" + `null` + "`" + `. Headers retain all values as arrays, with credentials replaced by ` + "`" + `[REDACTED]` + "`" + `. Dry-run never reads the OS keychain, but credentials supplied by flag, environment, or config file still appear redacted. The command exits successfully without contacting the API.

Local mutation commands emit one ` + "`" + `{"dry_run":true,"local":true,"command":"…","message":"…"}` + "`" + ` object in place of a preview; filter with ` + "`" + `select(.request)` + "`" + ` or ` + "`" + `select(.local)` + "`" + `.

### Debug

Log request and response diagnostics while running normally:

` + "`" + `` + "`" + `` + "`" + `bash
petstore pets list --debug
` + "`" + `` + "`" + `` + "`" + `

Debug output goes to stderr and includes:
- Request method, URL, headers, and body preview
- Response status, headers, and body preview
- Transport errors (if any)

The command still executes normally and produces its regular output on stdout.

### Flag Precedence

If both ` + "`" + `--dry-run` + "`" + ` and ` + "`" + `--debug` + "`" + ` are set, ` + "`" + `--dry-run` + "`" + ` takes precedence and no network calls are made.

### Security

Sensitive information is automatically redacted in diagnostic output:
- **Headers**: ` + "`" + `Authorization` + "`" + `, ` + "`" + `Cookie` + "`" + `, ` + "`" + `Set-Cookie` + "`" + `, ` + "`" + `X-API-Key` + "`" + `, and other security headers show ` + "`" + `[REDACTED]` + "`" + `
- **Body**: JSON fields named ` + "`" + `password` + "`" + `, ` + "`" + `secret` + "`" + `, ` + "`" + `token` + "`" + `, ` + "`" + `api_key` + "`" + `, ` + "`" + `client_secret` + "`" + `, etc. show ` + "`" + `[REDACTED]` + "`" + `
- **Binary data**: binary media and canonical base64 strings are replaced with ` + "`" + `<bytes:N>` + "`" + `
- **URL query**: credential-like query parameters are replaced with ` + "`" + `[REDACTED]` + "`" + `

Diagnostic output should still be treated as potentially sensitive operational data.
<!-- End Diagnostics [diagnostics] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This CLI is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

This CLI is generated programmatically. Edits to generated files are overwritten on regeneration. To customize it:

- **Configuration and behavior:** Use [OpenAPI overlays](https://www.speakeasy.com/docs/prep-openapi/overlays/create-overlays) in the Speakeasy workflow with ` + "`" + `x-speakeasy-*` + "`" + ` extensions (for example, ` + "`" + `x-speakeasy-cli-commands` + "`" + `) to define commands, flags, help text, examples, authentication, and grouping.
- **Persistent code changes:** Store unified diffs as [patch files](https://www.speakeasy.com/docs/sdks/customize/code/patch-files/patch-files) at ` + "`" + `.speakeasy/patches/<path-of-generated-file>.patch` + "`" + `; they are re-applied on every generation.

### CLI Created by [Speakeasy](https://www.speakeasy.com/?utm_source=github-com/example/petstore-cli&utm_campaign=cli)


--- scripts/install.ps1 ---
#
# petstore CLI Installation Script for Windows
# This script downloads and installs the latest version of the petstore CLI
#
# Usage:
#   iwr -useb https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.ps1 | iex
#   or
#   Invoke-WebRequest -Uri https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.ps1 -UseBasicParsing | Invoke-Expression
#
# Options:
#   $env:PETSTORE_INSTALL_DIR - Installation directory (default: $env:LOCALAPPDATA\Programs\petstore)
#   $env:PETSTORE_VERSION     - Specific version to install (default: latest)
#

[CmdletBinding()]
param()

# Configuration
$Repo = "example/petstore-cli"
$BinaryName = "petstore.exe"
$DefaultInstallDir = Join-Path $env:LOCALAPPDATA "Programs\petstore"
$InstallDir = if ($env:PETSTORE_INSTALL_DIR) { $env:PETSTORE_INSTALL_DIR } else { $DefaultInstallDir }
$Version = if ($env:PETSTORE_VERSION) { $env:PETSTORE_VERSION } else { "latest" }

# Helper functions
function Write-ColorOutput {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        return $response.tag_name
    }
    catch {
        throw "Failed to get latest version: $_"
    }
}

function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "x86_64" }
        "ARM64" { return "arm64" }
        default {
            throw "Unsupported architecture: $arch"
        }
    }
}

function Install-CLI {
    Write-ColorOutput "Installing petstore CLI..." -Color Green

    # Detect architecture
    $arch = Get-Architecture
    Write-ColorOutput "Detected Architecture: $arch" -Color Cyan

    # Get version
    if ($Version -eq "latest") {
        $Version = Get-LatestVersion
        Write-ColorOutput "Latest version: $Version" -Color Cyan
    }

    # Construct download URL
    $archiveName = "petstore_Windows_$arch.zip"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$archiveName"

    Write-ColorOutput "Downloading from: $downloadUrl" -Color Cyan

    # Create temporary directory
    $tempDir = Join-Path $env:TEMP "petstore-install-$(New-Guid)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

    try {
        # Download archive
        $archivePath = Join-Path $tempDir $archiveName
        try {
            Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing
        }
        catch {
            throw "Failed to download from ${downloadUrl}: $_"
        }

        Write-ColorOutput "Download complete" -Color Green

        # Extract archive
        Write-ColorOutput "Extracting archive..." -Color Cyan
        Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force

        # Create install directory if it doesn't exist
        if (-not (Test-Path $InstallDir)) {
            Write-ColorOutput "Creating installation directory: $InstallDir" -Color Cyan
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }

        # Install binary
        $binaryPath = Join-Path $InstallDir $BinaryName
        Write-ColorOutput "Installing to $binaryPath..." -Color Cyan

        # Remove existing binary if it exists
        if (Test-Path $binaryPath) {
            Remove-Item $binaryPath -Force
        }

        Copy-Item -Path (Join-Path $tempDir $BinaryName) -Destination $binaryPath -Force

        Write-ColorOutput "petstore $Version has been installed to $binaryPath" -Color Green

        # Add to PATH if not already there
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$InstallDir*") {
            Write-ColorOutput "Adding $InstallDir to your PATH..." -Color Cyan
            [Environment]::SetEnvironmentVariable(
                "Path",
                "$userPath;$InstallDir",
                "User"
            )
            $env:Path = "$env:Path;$InstallDir"
            Write-ColorOutput "Added to PATH. You may need to restart your terminal for changes to take effect." -Color Yellow
        }

        Write-ColorOutput "Installation successful! Run 'petstore --help' to get started." -Color Green
        Write-ColorOutput "Note: You may need to restart your terminal or run 'refreshenv' for the PATH changes to take effect." -Color Yellow
    }
    finally {
        # Cleanup
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force
        }
    }
}

# Main execution
& {
    $ErrorActionPreference = 'Stop'

    try {
        Install-CLI
    }
    catch {
        throw "Installation failed: $_"
    }
}


--- scripts/install.sh ---
#!/usr/bin/env bash
#
# petstore CLI Installation Script
# This script downloads and installs the latest version of the petstore CLI
# for Linux and macOS systems.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.sh | bash
#   or
#   wget -qO- https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.sh | bash
#
# Options:
#   PETSTORE_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#   PETSTORE_VERSION     - Specific version to install (default: latest)
#

set -e

# Configuration
REPO="example/petstore-cli"
DEFAULT_INSTALL_DIR="/usr/local/bin"
USER_INSTALL_DIR="$HOME/.local/bin"
VERSION="${PETSTORE_VERSION:-latest}"
BINARY_NAME="petstore"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect operating system
detect_os() {
    local os
    local uname_output="$(uname -s)"
    case "$uname_output" in
        Linux*)     os="Linux" ;;
        Darwin*)    os="Darwin" ;;
        CYGWIN*|MINGW*|MSYS*)    os="Windows" ;;
        *)
            log_error "Unsupported operating system: $uname_output"
            exit 1
            ;;
    esac
    echo "$os"
}

# Detect architecture
detect_arch() {
    local arch
    case "$(uname -m)" in
        x86_64|amd64)   arch="x86_64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    echo "$arch"
}

# Get latest version from GitHub
get_latest_version() {
    local latest_url="https://api.github.com/repos/${REPO}/releases/latest"
    local version

    if command -v curl >/dev/null 2>&1; then
        version=$(curl -fsSL "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        log_error "curl or wget is required to download the CLI"
        exit 1
    fi

    echo "$version"
}

# Determine installation directory
get_install_dir() {
    # If user specified a directory, use it
    if [ -n "${PETSTORE_INSTALL_DIR}" ]; then
        echo "${PETSTORE_INSTALL_DIR}"
        return
    fi

    # Try to use /usr/local/bin if we have write access
    if [ -w "$DEFAULT_INSTALL_DIR" ] || [ -w "$(dirname "$DEFAULT_INSTALL_DIR")" ]; then
        echo "$DEFAULT_INSTALL_DIR"
        return
    fi

    # Fall back to user directory
    log_info "No write access to $DEFAULT_INSTALL_DIR, using $USER_INSTALL_DIR instead" >&2
    echo "$USER_INSTALL_DIR"
}

# Download and install
install_cli() {
    local INSTALL_DIR=$(get_install_dir)
    local os=$(detect_os)
    local arch=$(detect_arch)

    log_info "Detected OS: $os"
    log_info "Detected Architecture: $arch"
    log_info "Installation directory: $INSTALL_DIR"

    # Get version
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(get_latest_version)
        log_info "Latest version: $VERSION"
    fi

    # Construct download URL based on OS
    local archive_name
    local archive_format
    if [ "$os" = "Windows" ]; then
        archive_name="${BINARY_NAME}_${os}_${arch}.zip"
        archive_format="zip"
    else
        archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
        archive_format="tar.gz"
    fi

    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${archive_name}"

    log_info "Downloading from: $download_url"

    # Create temporary directory
    local tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Download archive
    if command -v curl >/dev/null 2>&1; then
        if ! curl -fsSL "$download_url" -o "$tmp_dir/$archive_name"; then
            log_error "Failed to download from $download_url"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q "$download_url" -O "$tmp_dir/$archive_name"; then
            log_error "Failed to download from $download_url"
            exit 1
        fi
    fi

    log_info "Download complete"

    # Extract archive based on format
    log_info "Extracting archive..."
    if [ "$archive_format" = "zip" ]; then
        if command -v unzip >/dev/null 2>&1; then
            unzip -q "$tmp_dir/$archive_name" -d "$tmp_dir"
        else
            log_error "unzip is required to extract the archive. Please install unzip and try again."
            exit 1
        fi
    else
        tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"
    fi

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating installation directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR" || {
            log_error "Failed to create $INSTALL_DIR. Try running with sudo or set PETSTORE_INSTALL_DIR to a writable location."
            exit 1
        }
    fi

    # Install binary (Windows binaries have .exe extension)
    local source_binary="$tmp_dir/$BINARY_NAME"
    local target_binary="$INSTALL_DIR/$BINARY_NAME"

    if [ "$os" = "Windows" ]; then
        source_binary="$tmp_dir/${BINARY_NAME}.exe"
        target_binary="$INSTALL_DIR/${BINARY_NAME}.exe"
    fi

    log_info "Installing to $target_binary..."
    if ! mv "$source_binary" "$target_binary"; then
        log_error "Failed to install to $INSTALL_DIR. Try running with sudo or set PETSTORE_INSTALL_DIR to a writable location."
        exit 1
    fi

    # Make executable (not needed on Windows, but doesn't hurt)
    chmod +x "$target_binary" 2>/dev/null || true

    log_info "petstore ${VERSION} has been installed to $target_binary"

    # Verify installation
    local cmd_to_check="$BINARY_NAME"
    if [ "$os" = "Windows" ]; then
        cmd_to_check="${BINARY_NAME}.exe"
    fi

    if command -v "$cmd_to_check" >/dev/null 2>&1; then
        log_info "Installation successful! Run '$BINARY_NAME --help' to get started."
    else
        log_warn "Installation complete, but $BINARY_NAME is not in your PATH."
        if [ "$os" = "Windows" ]; then
            log_warn "Add $INSTALL_DIR to your PATH environment variable."
        else
            log_warn "Add $INSTALL_DIR to your PATH by adding this to your ~/.bashrc or ~/.zshrc:"
            log_warn "  export PATH=\"\$PATH:$INSTALL_DIR\""
            log_warn ""
            log_warn "Then run: source ~/.bashrc  # or source ~/.zshrc"
        fi
    fi
}

# Main execution
main() {
    log_info "Installing petstore CLI..."
    install_cli
}

main


` // end of snapshot

	snaptest.DoTestSnapshot(t, snaptest.Options{
		Spec:         cliReleaseSnapshotSpec,
		GenYaml:      genYaml,
		IncludeGlobs: cliReleaseExpectedSnapshotFiles,
		Expected:     expectedSnapshot,
	})
}

func TestSnapCLIReleaseAllChannels(t *testing.T) {
	t.Parallel()

	genYaml := `cli:
  packageName: github.com/example/petstore-cli
  cliName: petstore
  envVarPrefix: PETSTORE
  generateRelease: true
  distribution:
    homebrew:
      enabled: true
      tap: example/homebrew-petstore
    winget:
      enabled: true
      publisher: Example
      repositoryOwner: example
      publisherUrl: https://example.com
      packageIdentifier: Example.Petstore
      license: Apache-2.0
    nfpm:
      enabled: true
      formats: deb,rpm
      maintainer: Team <team@example.com>
      license: Apache-2.0
`

	expectedSnapshot := `--- .github/workflows/release.yaml ---
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          fetch-depth: 0

      - name: Check release version
        run: |
          tag_version="${GITHUB_REF_NAME#v}"
          cli_version=$(sed -n 's/^var Version = "\(.*\)"$/\1/p' internal/cli/version.go)
          if [ -z "$cli_version" ] || [ "$tag_version" != "$cli_version" ]; then
            printf '::error::Release tag %s does not match generated CLI version %s. Set cli.version in .speakeasy/gen.yaml to %s and regenerate before tagging.\n' "$GITHUB_REF_NAME" "$cli_version" "$tag_version"
            exit 1
          fi

      - name: Setup Go
        uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: "go.mod"
          cache: true
      - name: Import GPG key
        uses: crazy-max/ghaction-import-gpg@2dc316deee8e90f13e1a351ab510b4d5bc0c82cd # v7.0.0
        id: import_gpg
        with:
          gpg_private_key: ${{ secrets.CLI_GPG_SECRET_KEY }}
          passphrase: ${{ secrets.CLI_GPG_PASSPHRASE }}
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@f06c13b6b1a9625abc9e6e439d9c05a8f2190e94 # v7.2.3
        with:
          distribution: goreleaser
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
          # Required when Homebrew tap publishing is enabled.
          # Classic PATs need public_repo (public tap) or repo (private tap).
          # Fine-grained PATs need Contents: Read and write on example/homebrew-petstore.
          HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
          # Required when WinGet publishing is enabled.
          # PAT must have public_repo scope so GoReleaser can push a branch and open a PR against microsoft/winget-pkgs.
          WINGET_GITHUB_TOKEN: ${{ secrets.WINGET_GITHUB_TOKEN }}

      - name: Upload artifacts
        uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
        if: always()
        with:
          name: release-artifacts
          path: |
            dist/
            !dist/*.txt
          retention-days: 30


--- .goreleaser.yaml ---
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

project_name: petstore

before:
  hooks:
    - go mod tidy

builds:
  - id: petstore
    main: ./cmd/petstore
    binary: petstore
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.buildTime={{.Date}}

archives:
  - id: petstore
    formats: [tar.gz]
    name_template: >-
      {{ .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
    format_overrides:
      - goos: windows
        formats: [zip]
    files:
      - README.md
      - LICENSE*

brews:
  - name: petstore
    repository:
      owner: example
      name: homebrew-petstore
      branch: main
    token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: "https://github.com/example/petstore-cli"
    description: "Petstore CLI: Petstore CLI installable via package managers"
    install: |
      bin.install "petstore"

winget:
  - name: "petstore"
    publisher: "Example"
    publisher_url: "https://example.com"
    package_identifier: "Example.Petstore"
    license: "Apache-2.0"
    homepage: "https://github.com/example/petstore-cli"
    short_description: "Petstore CLI: Petstore CLI installable via package managers"
    repository:
      owner: example
      name: winget-pkgs
      branch: "petstore-{{ .Version }}"
      token: "{{ .Env.WINGET_GITHUB_TOKEN }}"
    pull_request:
      enabled: true
      draft: true
      base:
        owner: microsoft
        name: winget-pkgs
        branch: master

nfpms:
  - package_name: petstore
    file_name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Arch }}"
    homepage: "https://github.com/example/petstore-cli"
    description: "Petstore CLI: Petstore CLI installable via package managers"
    maintainer: "Team <team@example.com>"
    license: "Apache-2.0"
    formats:
      - deb
      - rpm

release:
  github:
    owner: example
    name: petstore-cli
  draft: false
  prerelease: auto
  mode: append

checksum:
  name_template: "checksums.txt"

signs:
  - artifacts: checksum
    args:
      - "--batch"
      - "--local-user"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"


--- .speakeasy/gen.yaml ---
configVersion: 2.0.0
generation:
  sdkClassName: SDK
  maintainOpenAPIOrder: true
  usageSnippets:
    optionalPropertyRendering: withExample
    sdkInitStyle: constructor
  useClassNamesForArrayFields: true
  fixes:
    nameResolutionDec2023: true
    nameResolutionFeb2025: true
    parameterOrderingFeb2024: true
    requestResponseComponentNamesFeb2024: true
    securityFeb2025: true
    sharedErrorComponentsApr2025: true
    sharedNestedComponentsJan2026: true
    nameOverrideFeb2026: true
    openSingleValueEnumsOct2026: true
  auth:
    oAuth2ClientCredentialsEnabled: true
    oAuth2PasswordEnabled: true
    hoistGlobalSecurity: true
  inferSSEOverload: true
  sdkHooksConfigAccess: true
  schemas:
    allOfMergeStrategy: shallowMerge
  requestBodyFieldName: body
  versioningStrategy: automatic
  persistentEdits: {}
  tests:
    generateTests: false
    generateNewTests: true
    skipResponseBodyAssertions: false
cli:
  version: 0.0.1
  additionalDependencies: {}
  agentEnvironmentDetection: true
  arrayFlagFormat: repeatable
  classifiedErrors: false
  cliName: petstore
  defaultColor: auto
  defaultTimeout: ""
  distribution:
    homebrew:
      enabled: true
      tap: example/homebrew-petstore
    nfpm:
      enabled: true
      formats: deb,rpm
      license: Apache-2.0
      maintainer: Team <team@example.com>
    winget:
      enabled: true
      license: Apache-2.0
      packageIdentifier: Example.Petstore
      publisher: Example
      publisherUrl: https://example.com
      repositoryOwner: example
  enableCustomCodeRegions: false
  envVarPrefix: PETSTORE
  fixFlags:
    encodePathParams: true
  generateRelease: true
  helpStyle: auto
  idiomaticMethodCollisionNames: true
  imports:
    option: openapi
    paths:
      callbacks: models/callbacks
      errors: models/apierrors
      operations: models/operations
      shared: models/components
      webhooks: models/webhooks
  interactiveAuth: true
  interactiveByDefault: true
  interactiveMode: true
  interactiveTheme:
    accentColor: '#38BDF8'
    dimmedColor: '#64748B'
    errorColor: '#F87171'
    subtleColor: '#475569'
    successColor: '#4ADE80'
  jqRawOutput: false
  packageName: github.com/example/petstore-cli
  removeStutter: true
  retryFlagsVisibility: visible
  retryMethodPolicy: spec
  rootVersionFlag: true
  serverSelectionFlag: visible


--- README.md ---
# petstore

Command-line interface for the *Petstore CLI* API.

[![Built by Speakeasy](https://img.shields.io/badge/Built_by-SPEAKEASY-374151?style=for-the-badge&labelColor=f3f4f6)](https://www.speakeasy.com/?utm_source=github-com/example/petstore-cli&utm_campaign=cli)
[![License: AGPL-3.0-only](https://img.shields.io/badge/LICENSE_//_AGPL--3.0--only-3b5bdb?style=for-the-badge&labelColor=eff6ff)](https://www.gnu.org/licenses/agpl-3.0.html)


<br /><br />
> [!IMPORTANT]
> This CLI is not yet ready for production use. Delete this notice before publishing to a package manager.

<!-- Start Summary [summary] -->
## Summary

Petstore CLI: Petstore CLI installable via package managers
<!-- End Summary [summary] -->

<!-- Start Table of Contents [toc] -->
## Table of Contents
<!-- $toc-max-depth=2 -->
* [petstore](#petstore)
  * [CLI Installation](#cli-installation)
  * [Shell Completion](#shell-completion)
  * [CLI Example Usage](#cli-example-usage)
  * [For AI agents](#for-ai-agents)
  * [Commands](#commands)
  * [Request Body Input](#request-body-input)
  * [Output Formats](#output-formats)
  * [Error Handling](#error-handling)
  * [Diagnostics](#diagnostics)
* [Development](#development)
  * [Maturity](#maturity)
  * [Contributions](#contributions)

<!-- End Table of Contents [toc] -->

<!-- Start CLI Installation [installation] -->
## CLI Installation

### Quick Install (Linux/macOS)

` + "`" + `` + "`" + `` + "`" + `bash
curl -fsSL https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.sh | bash
` + "`" + `` + "`" + `` + "`" + `

### Quick Install (Windows PowerShell)

` + "`" + `` + "`" + `` + "`" + `powershell
iwr -useb https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.ps1 | iex
` + "`" + `` + "`" + `` + "`" + `
### Homebrew (macOS/Linux)

` + "`" + `` + "`" + `` + "`" + `bash
brew install example/petstore/petstore
` + "`" + `` + "`" + `` + "`" + `
### WinGet (Windows)

` + "`" + `` + "`" + `` + "`" + `powershell
winget install Example.Petstore
` + "`" + `` + "`" + `` + "`" + `
### Linux Packages

Download Linux packages from the [releases page](https://github.com/example/petstore-cli/releases).
` + "`" + `` + "`" + `` + "`" + `bash
sudo dpkg -i petstore_*_amd64.deb
sudo rpm -i petstore_*_amd64.rpm
` + "`" + `` + "`" + `` + "`" + `

### Go Install

Alternatively, install directly via Go:

` + "`" + `` + "`" + `` + "`" + `bash
go install github.com/example/petstore-cli/cmd/petstore@latest
` + "`" + `` + "`" + `` + "`" + `

### Manual Download

Download pre-built binaries for your platform from the [releases page](https://github.com/example/petstore-cli/releases).
<!-- End CLI Installation [installation] -->

<!-- Start Shell Completion [completion] -->
## Shell Completion

Shell completions are available for Bash, Zsh, Fish, and PowerShell.

### Bash

` + "`" + `` + "`" + `` + "`" + `bash
# Add to ~/.bashrc:
source <(petstore completion bash)

# Or install permanently:
petstore completion bash > /etc/bash_completion.d/petstore
` + "`" + `` + "`" + `` + "`" + `

### Zsh

` + "`" + `` + "`" + `` + "`" + `zsh
# Add to ~/.zshrc:
source <(petstore completion zsh)

# Or install permanently:
petstore completion zsh > "${fpath[1]}/_petstore"
` + "`" + `` + "`" + `` + "`" + `

### Fish

` + "`" + `` + "`" + `` + "`" + `fish
petstore completion fish | source

# Or install permanently:
petstore completion fish > ~/.config/fish/completions/petstore.fish
` + "`" + `` + "`" + `` + "`" + `

### PowerShell

` + "`" + `` + "`" + `` + "`" + `powershell
petstore completion powershell | Out-String | Invoke-Expression
` + "`" + `` + "`" + `` + "`" + `
<!-- End Shell Completion [completion] -->

<!-- Start CLI Example Usage [usage] -->
## CLI Example Usage

### Example

` + "`" + `` + "`" + `` + "`" + `bash
petstore pets list

` + "`" + `` + "`" + `` + "`" + `
<!-- End CLI Example Usage [usage] -->

<!-- Start For AI agents [agents] -->
## For AI agents

This CLI is built to be driven by AI coding agents as well as people: everything an agent needs is discoverable from the binary itself, and every command can be validated without credentials. Work down this ladder:

| Run | You get |
|-----|---------|
| ` + "`" + `petstore --help` + "`" + `, ` + "`" + `petstore pets list --help` + "`" + ` | Commands by category, runnable examples, flags |
| ` + "`" + `petstore --usage` + "`" + `, ` + "`" + `petstore pets list --usage` + "`" + ` | The command surface as machine-readable [KDL](https://kdl.dev): commands, aliases, flags, defaults, env vars, config keys |
| ` + "`" + `petstore pets list --dry-run` + "`" + ` | The exact HTTP request (method, URL, headers, body), with no credentials or network call |
| ` + "`" + `petstore pets list --output-format json` + "`" + ` (or ` + "`" + `--jq` + "`" + `) | Machine-readable output |

### Discover the command surface

` + "`" + `` + "`" + `` + "`" + `bash
# Every command, flag, default, env var and config key, as KDL
petstore --usage

# One command's subtree only
petstore pets list --usage
` + "`" + `` + "`" + `` + "`" + `

### Probe before you spend

Start quota-spending commands with ` + "`" + `--dry-run` + "`" + `. It validates inputs, resolves the request, redacts secrets and binary payloads, makes no network call, and exits 0. It never reads the OS keychain; credentials supplied by flag, environment, or config file are included only as ` + "`" + `[REDACTED]` + "`" + `.

` + "`" + `` + "`" + `` + "`" + `bash
# Human preview: the [DRY-RUN] block is on stderr and stdout is empty
petstore pets list --dry-run

# Machine preview: compact JSON on stdout and silent stderr
petstore pets list --dry-run --output-format json
` + "`" + `` + "`" + `` + "`" + `

The machine form writes one object per would-be request, one per line (NDJSON for multi-request commands), with exactly this shape:

` + "`" + `` + "`" + `` + "`" + `json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
` + "`" + `` + "`" + `` + "`" + `

` + "`" + `body` + "`" + ` is a parsed JSON value when the body is JSON, a string for text, ` + "`" + `"<bytes:N>"` + "`" + ` for binary data, and ` + "`" + `null` + "`" + ` when absent. An explicit caller ` + "`" + `--jq` + "`" + ` also selects this JSON preview protocol, but the filter is not applied to preview objects. Command-declared jq presets do not select or filter the preview.

Local mutation commands make no request under ` + "`" + `--dry-run` + "`" + `: instead of a preview they emit one ` + "`" + `{"dry_run":true,"local":true,"command":"…","message":"…"}` + "`" + ` object. ` + "`" + `select(.request)` + "`" + ` keeps only would-be requests; ` + "`" + `select(.local)` + "`" + ` keeps the local no-ops.

### Machine-readable output

` + "`" + `` + "`" + `` + "`" + `bash
# JSON on stdout
petstore pets list --output-format json

# Filter or reshape with a jq expression (always emits JSON, overrides --output-format)
petstore pets list --jq '.'

# Print jq string results as plain text instead of JSON strings (like jq -r)
petstore pets list --jq '.' --raw-output
` + "`" + `` + "`" + `` + "`" + `

` + "`" + `--output-format toon` + "`" + ` emits [TOON](https://github.com/toon-format/spec), a compact line-oriented format that uses fewer tokens than JSON; it is the default in agent mode.

### Interactive mode
Required-input prompts and guided ` + "`" + `configure` + "`" + ` / ` + "`" + `auth login` + "`" + ` forms are enabled by default. Required-input prompts require an interactive terminal; off-TTY forms read line input from stdin. Use ` + "`" + `--no-interactive` + "`" + ` to force flag-only execution.

` + "`" + `` + "`" + `` + "`" + `bash
# Prompt for missing command inputs
petstore pets list --interactive

# Open the guided configuration form
petstore configure --interactive

# Explicitly launch the terminal command explorer
petstore explore
` + "`" + `` + "`" + `` + "`" + `

### Agent mode and structured errors
Agent mode turns on automatically when a known agent environment is detected (` + "`" + `CLAUDECODE` + "`" + `, ` + "`" + `CURSOR_AGENT` + "`" + `, ` + "`" + `CODEX` + "`" + `, ` + "`" + `AIDER` + "`" + `, ` + "`" + `CLINE` + "`" + `, ` + "`" + `WINDSURF_AGENT` + "`" + `, ` + "`" + `GITHUB_COPILOT` + "`" + `, ` + "`" + `AMAZON_Q` + "`" + `, ` + "`" + `GEMINI_CODE_ASSIST` + "`" + `, ` + "`" + `SRC_CODY` + "`" + `) or with ` + "`" + `--agent-mode` + "`" + ` (` + "`" + `--agent-mode=false` + "`" + ` disables detection).
In agent mode interactive prompts never launch, output defaults to TOON, and every failure — API errors and CLI usage errors alike — is one JSON envelope on stderr:
Outside agent mode, explicit JSON and ` + "`" + `--jq` + "`" + ` preserve the compatibility envelope without classification; enable agent mode to request the classified contract.

` + "`" + `` + "`" + `` + "`" + `json
{
  "error": "...",
  "error_type": "validation_error",
  "error_reason": "CLI_VALIDATION",
  "exit_code": 2,
  "message": "human-readable message",
  "hints": ["what to try next"]
}
` + "`" + `` + "`" + `` + "`" + `

` + "`" + `error_type` + "`" + ` is one of ` + "`" + `authentication_error` + "`" + `, ` + "`" + `authorization_error` + "`" + `, ` + "`" + `not_found` + "`" + `, ` + "`" + `validation_error` + "`" + `, ` + "`" + `rate_limit_error` + "`" + `, ` + "`" + `server_error` + "`" + `, ` + "`" + `api_error` + "`" + `, ` + "`" + `connection_error` + "`" + `, ` + "`" + `protocol_error` + "`" + `, ` + "`" + `runtime_error` + "`" + `, ` + "`" + `unsupported_error` + "`" + `, ` + "`" + `async_failed` + "`" + `, ` + "`" + `async_timeout` + "`" + `, ` + "`" + `async_unknown_state` + "`" + `. Classification derives from the HTTP status and transport evidence; ` + "`" + `error_reason` + "`" + ` is absent for API errors. Status-less local failures may use ` + "`" + `CLI_VALIDATION` + "`" + `, ` + "`" + `CLI_CONNECTION` + "`" + `, ` + "`" + `CLI_PROTOCOL` + "`" + `, ` + "`" + `CLI_RUNTIME` + "`" + `, ` + "`" + `CLI_UNAVAILABLE` + "`" + `, ` + "`" + `CLI_AUTHENTICATION` + "`" + `, the async polling reasons ` + "`" + `CLI_ASYNC_FAILED` + "`" + `, ` + "`" + `CLI_ASYNC_TIMEOUT` + "`" + `, and ` + "`" + `CLI_ASYNC_UNKNOWN_STATE` + "`" + `, or ` + "`" + `CLI_OUTCOME_FAILED` + "`" + ` when a successful response reports a declared failure state; that envelope also carries ` + "`" + `state` + "`" + ` (the reported state) and, when the command declares an error pointer, ` + "`" + `error` + "`" + ` (the failure detail taken from the response). ` + "`" + `hints` + "`" + ` preserves server guidance first, adds the most specific local taxonomy guidance, then typed CLI and command-specific guidance, removing exact duplicates. ` + "`" + `exit_code` + "`" + ` is always the code for the final ` + "`" + `error_type` + "`" + ` shown in the envelope: 1 runtime, 2 usage, or 3 authentication/authorization.
<!-- End For AI agents [agents] -->

<!-- Start Commands [operations] -->
## Commands

<details open>
<summary>Available commands</summary>

* [` + "`" + `pets` + "`" + `](docs/petstore_pets.md) - Operations for pets
  * [` + "`" + `list` + "`" + `](docs/petstore_pets_list.md)

</details>
<!-- End Commands [operations] -->

<!-- Start Request Body Input [stdinpiping] -->
## Request Body Input

Commands that accept a request body take it three ways, with a clear priority chain: individual field flags (highest priority), the whole body as JSON via ` + "`" + `--body` + "`" + `, and JSON piped on stdin (lowest priority). Later sources never override earlier ones.
<!-- End Request Body Input [stdinpiping] -->

<!-- Start Output Formats [output-formats] -->
## Output Formats

Every command supports a ` + "`" + `--output-format` + "`" + ` flag that controls how the response is rendered to stdout.

### Available formats

| Format | Flag | Description |
|--------|------|-------------|
| Pretty | ` + "`" + `--output-format pretty` + "`" + ` (default) | Aligned key-value pairs with color, nested indentation. Human-readable at a glance. |
| JSON | ` + "`" + `--output-format json` + "`" + ` | JSON output. Passthrough when the response is already JSON (preserves original field order and numeric precision). Falls back to typed marshaling otherwise. |
| YAML | ` + "`" + `--output-format yaml` + "`" + ` | YAML output via standard marshaling. |
| Table | ` + "`" + `--output-format table` + "`" + ` | Tabular output for array responses. |
| TOON | ` + "`" + `--output-format toon` + "`" + ` | [Token-Oriented Object Notation](https://github.com/toon-format/spec) — a compact, line-oriented format that typically uses 30–60% fewer tokens than JSON. Well-suited for piping responses into LLM prompts. |

` + "`" + `` + "`" + `` + "`" + `bash
# Default pretty output
petstore pets list

# Machine-readable JSON
petstore pets list --output-format json

# TOON for LLM-friendly compact output
petstore pets list --output-format toon

# Pipe JSON to jq without using --output-format
petstore pets list --output-format json | jq '.'
` + "`" + `` + "`" + `` + "`" + `

### jq filtering

Use ` + "`" + `--jq` + "`" + ` to filter or transform the response inline using a [jq](https://jqlang.org) expression. This always outputs JSON and overrides ` + "`" + `--output-format` + "`" + `:

` + "`" + `` + "`" + `` + "`" + `bash
# Extract a single field
petstore pets list --jq '.'

# Reshape with any jq program; --raw-output prints string results as plain text (like jq -r)
petstore pets list --jq '.' --raw-output
` + "`" + `` + "`" + `` + "`" + `

### Color control

Use ` + "`" + `--color` + "`" + ` to control terminal colors:

| Value | Behavior |
|-------|----------|
| ` + "`" + `auto` + "`" + ` (default) | Color when stdout is a TTY, plain text otherwise |
| ` + "`" + `always` + "`" + ` | Always colorize |
| ` + "`" + `never` + "`" + ` | Never colorize |

The ` + "`" + `NO_COLOR` + "`" + ` and ` + "`" + `FORCE_COLOR` + "`" + ` environment variables are also respected.

### Streaming and pagination

When using ` + "`" + `--all` + "`" + ` (pagination) or streaming operations, output is written incrementally as items arrive:

| Format | Streaming behavior |
|--------|-------------------|
| ` + "`" + `json` + "`" + ` | One compact JSON object per line ([NDJSON](https://github.com/ndjson/ndjson-spec)) |
| ` + "`" + `yaml` + "`" + ` | YAML documents separated by ` + "`" + `---` + "`" + ` |
| ` + "`" + `toon` + "`" + ` | One TOON-encoded object per block, separated by blank lines |
| ` + "`" + `pretty` + "`" + ` (default) | Pretty-printed items separated by blank lines |
<!-- End Output Formats [output-formats] -->

<!-- Start Error Handling [errors] -->
## Error Handling

The CLI uses standard exit codes to indicate success or failure:

| Exit Code | Meaning |
|-----------|---------|
| ` + "`" + `0` + "`" + ` | Success |
| ` + "`" + `1` + "`" + ` | Runtime/API failure |
| ` + "`" + `2` + "`" + ` | Usage or input failure |
| ` + "`" + `3` + "`" + ` | Authentication or authorization failure |

On success, the response data is printed to **stdout** as JSON. On failure, error details are printed to **stderr**.

` + "`" + `` + "`" + `` + "`" + `bash
# Capture output and handle errors
petstore pets list --output-format json > output.json 2> error.log
if [ $? -ne 0 ]; then
  echo "Error occurred, see error.log"
fi
` + "`" + `` + "`" + `` + "`" + `
This CLI uses unclassified error rendering outside agent mode: pretty and TOON print the API error text as received, while ` + "`" + `--output-format json` + "`" + ` and ` + "`" + `--jq` + "`" + ` emit the unclassified envelope (including the configure ` + "`" + `_hint` + "`" + ` for HTTP 401/403) plus ` + "`" + `exit_code` + "`" + `. Agent mode always emits the classified JSON envelope with ` + "`" + `exit_code` + "`" + `, ` + "`" + `error_type` + "`" + `, optional ` + "`" + `error_reason` + "`" + `, ` + "`" + `message` + "`" + `, ` + "`" + `hints` + "`" + `, and optional ` + "`" + `status_code` + "`" + ` — see [For AI agents](#for-ai-agents).
<!-- End Error Handling [errors] -->

<!-- Start Diagnostics [diagnostics] -->
## Diagnostics

The CLI includes two diagnostic flags available on all commands:

### Dry Run

Preview what would be sent without making any network calls:

` + "`" + `` + "`" + `` + "`" + `bash
petstore pets list --dry-run
` + "`" + `` + "`" + `` + "`" + `

In human output modes, stdout is empty and the ` + "`" + `[DRY-RUN]` + "`" + ` block goes to stderr. It includes:
- HTTP method and URL
- Request headers (sensitive values redacted)
- Request body preview (sensitive fields redacted)

With ` + "`" + `--output-format json` + "`" + `, or with a caller-explicit ` + "`" + `--jq` + "`" + `, stderr is silent and stdout is NDJSON: one compact preview object per would-be request. The jq filter is not applied, and command-declared jq presets do not select the JSON protocol.

` + "`" + `` + "`" + `` + "`" + `json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
` + "`" + `` + "`" + `` + "`" + `

JSON bodies remain structured; text bodies are strings; binary bodies are ` + "`" + `"<bytes:N>"` + "`" + `; absent bodies are ` + "`" + `null` + "`" + `. Headers retain all values as arrays, with credentials replaced by ` + "`" + `[REDACTED]` + "`" + `. Dry-run never reads the OS keychain, but credentials supplied by flag, environment, or config file still appear redacted. The command exits successfully without contacting the API.

Local mutation commands emit one ` + "`" + `{"dry_run":true,"local":true,"command":"…","message":"…"}` + "`" + ` object in place of a preview; filter with ` + "`" + `select(.request)` + "`" + ` or ` + "`" + `select(.local)` + "`" + `.

### Debug

Log request and response diagnostics while running normally:

` + "`" + `` + "`" + `` + "`" + `bash
petstore pets list --debug
` + "`" + `` + "`" + `` + "`" + `

Debug output goes to stderr and includes:
- Request method, URL, headers, and body preview
- Response status, headers, and body preview
- Transport errors (if any)

The command still executes normally and produces its regular output on stdout.

### Flag Precedence

If both ` + "`" + `--dry-run` + "`" + ` and ` + "`" + `--debug` + "`" + ` are set, ` + "`" + `--dry-run` + "`" + ` takes precedence and no network calls are made.

### Security

Sensitive information is automatically redacted in diagnostic output:
- **Headers**: ` + "`" + `Authorization` + "`" + `, ` + "`" + `Cookie` + "`" + `, ` + "`" + `Set-Cookie` + "`" + `, ` + "`" + `X-API-Key` + "`" + `, and other security headers show ` + "`" + `[REDACTED]` + "`" + `
- **Body**: JSON fields named ` + "`" + `password` + "`" + `, ` + "`" + `secret` + "`" + `, ` + "`" + `token` + "`" + `, ` + "`" + `api_key` + "`" + `, ` + "`" + `client_secret` + "`" + `, etc. show ` + "`" + `[REDACTED]` + "`" + `
- **Binary data**: binary media and canonical base64 strings are replaced with ` + "`" + `<bytes:N>` + "`" + `
- **URL query**: credential-like query parameters are replaced with ` + "`" + `[REDACTED]` + "`" + `

Diagnostic output should still be treated as potentially sensitive operational data.
<!-- End Diagnostics [diagnostics] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This CLI is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

This CLI is generated programmatically. Edits to generated files are overwritten on regeneration. To customize it:

- **Configuration and behavior:** Use [OpenAPI overlays](https://www.speakeasy.com/docs/prep-openapi/overlays/create-overlays) in the Speakeasy workflow with ` + "`" + `x-speakeasy-*` + "`" + ` extensions (for example, ` + "`" + `x-speakeasy-cli-commands` + "`" + `) to define commands, flags, help text, examples, authentication, and grouping.
- **Persistent code changes:** Store unified diffs as [patch files](https://www.speakeasy.com/docs/sdks/customize/code/patch-files/patch-files) at ` + "`" + `.speakeasy/patches/<path-of-generated-file>.patch` + "`" + `; they are re-applied on every generation.

### CLI Created by [Speakeasy](https://www.speakeasy.com/?utm_source=github-com/example/petstore-cli&utm_campaign=cli)


--- scripts/install.ps1 ---
#
# petstore CLI Installation Script for Windows
# This script downloads and installs the latest version of the petstore CLI
#
# Usage:
#   iwr -useb https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.ps1 | iex
#   or
#   Invoke-WebRequest -Uri https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.ps1 -UseBasicParsing | Invoke-Expression
#
# Options:
#   $env:PETSTORE_INSTALL_DIR - Installation directory (default: $env:LOCALAPPDATA\Programs\petstore)
#   $env:PETSTORE_VERSION     - Specific version to install (default: latest)
#

[CmdletBinding()]
param()

# Configuration
$Repo = "example/petstore-cli"
$BinaryName = "petstore.exe"
$DefaultInstallDir = Join-Path $env:LOCALAPPDATA "Programs\petstore"
$InstallDir = if ($env:PETSTORE_INSTALL_DIR) { $env:PETSTORE_INSTALL_DIR } else { $DefaultInstallDir }
$Version = if ($env:PETSTORE_VERSION) { $env:PETSTORE_VERSION } else { "latest" }

# Helper functions
function Write-ColorOutput {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
        return $response.tag_name
    }
    catch {
        throw "Failed to get latest version: $_"
    }
}

function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "x86_64" }
        "ARM64" { return "arm64" }
        default {
            throw "Unsupported architecture: $arch"
        }
    }
}

function Install-CLI {
    Write-ColorOutput "Installing petstore CLI..." -Color Green

    # Detect architecture
    $arch = Get-Architecture
    Write-ColorOutput "Detected Architecture: $arch" -Color Cyan

    # Get version
    if ($Version -eq "latest") {
        $Version = Get-LatestVersion
        Write-ColorOutput "Latest version: $Version" -Color Cyan
    }

    # Construct download URL
    $archiveName = "petstore_Windows_$arch.zip"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$archiveName"

    Write-ColorOutput "Downloading from: $downloadUrl" -Color Cyan

    # Create temporary directory
    $tempDir = Join-Path $env:TEMP "petstore-install-$(New-Guid)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

    try {
        # Download archive
        $archivePath = Join-Path $tempDir $archiveName
        try {
            Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing
        }
        catch {
            throw "Failed to download from ${downloadUrl}: $_"
        }

        Write-ColorOutput "Download complete" -Color Green

        # Extract archive
        Write-ColorOutput "Extracting archive..." -Color Cyan
        Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force

        # Create install directory if it doesn't exist
        if (-not (Test-Path $InstallDir)) {
            Write-ColorOutput "Creating installation directory: $InstallDir" -Color Cyan
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }

        # Install binary
        $binaryPath = Join-Path $InstallDir $BinaryName
        Write-ColorOutput "Installing to $binaryPath..." -Color Cyan

        # Remove existing binary if it exists
        if (Test-Path $binaryPath) {
            Remove-Item $binaryPath -Force
        }

        Copy-Item -Path (Join-Path $tempDir $BinaryName) -Destination $binaryPath -Force

        Write-ColorOutput "petstore $Version has been installed to $binaryPath" -Color Green

        # Add to PATH if not already there
        $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if ($userPath -notlike "*$InstallDir*") {
            Write-ColorOutput "Adding $InstallDir to your PATH..." -Color Cyan
            [Environment]::SetEnvironmentVariable(
                "Path",
                "$userPath;$InstallDir",
                "User"
            )
            $env:Path = "$env:Path;$InstallDir"
            Write-ColorOutput "Added to PATH. You may need to restart your terminal for changes to take effect." -Color Yellow
        }

        Write-ColorOutput "Installation successful! Run 'petstore --help' to get started." -Color Green
        Write-ColorOutput "Note: You may need to restart your terminal or run 'refreshenv' for the PATH changes to take effect." -Color Yellow
    }
    finally {
        # Cleanup
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force
        }
    }
}

# Main execution
& {
    $ErrorActionPreference = 'Stop'

    try {
        Install-CLI
    }
    catch {
        throw "Installation failed: $_"
    }
}


--- scripts/install.sh ---
#!/usr/bin/env bash
#
# petstore CLI Installation Script
# This script downloads and installs the latest version of the petstore CLI
# for Linux and macOS systems.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.sh | bash
#   or
#   wget -qO- https://raw.githubusercontent.com/example/petstore-cli/main/scripts/install.sh | bash
#
# Options:
#   PETSTORE_INSTALL_DIR - Installation directory (default: /usr/local/bin)
#   PETSTORE_VERSION     - Specific version to install (default: latest)
#

set -e

# Configuration
REPO="example/petstore-cli"
DEFAULT_INSTALL_DIR="/usr/local/bin"
USER_INSTALL_DIR="$HOME/.local/bin"
VERSION="${PETSTORE_VERSION:-latest}"
BINARY_NAME="petstore"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect operating system
detect_os() {
    local os
    local uname_output="$(uname -s)"
    case "$uname_output" in
        Linux*)     os="Linux" ;;
        Darwin*)    os="Darwin" ;;
        CYGWIN*|MINGW*|MSYS*)    os="Windows" ;;
        *)
            log_error "Unsupported operating system: $uname_output"
            exit 1
            ;;
    esac
    echo "$os"
}

# Detect architecture
detect_arch() {
    local arch
    case "$(uname -m)" in
        x86_64|amd64)   arch="x86_64" ;;
        aarch64|arm64)  arch="arm64" ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    echo "$arch"
}

# Get latest version from GitHub
get_latest_version() {
    local latest_url="https://api.github.com/repos/${REPO}/releases/latest"
    local version

    if command -v curl >/dev/null 2>&1; then
        version=$(curl -fsSL "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        version=$(wget -qO- "$latest_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        log_error "curl or wget is required to download the CLI"
        exit 1
    fi

    echo "$version"
}

# Determine installation directory
get_install_dir() {
    # If user specified a directory, use it
    if [ -n "${PETSTORE_INSTALL_DIR}" ]; then
        echo "${PETSTORE_INSTALL_DIR}"
        return
    fi

    # Try to use /usr/local/bin if we have write access
    if [ -w "$DEFAULT_INSTALL_DIR" ] || [ -w "$(dirname "$DEFAULT_INSTALL_DIR")" ]; then
        echo "$DEFAULT_INSTALL_DIR"
        return
    fi

    # Fall back to user directory
    log_info "No write access to $DEFAULT_INSTALL_DIR, using $USER_INSTALL_DIR instead" >&2
    echo "$USER_INSTALL_DIR"
}

# Download and install
install_cli() {
    local INSTALL_DIR=$(get_install_dir)
    local os=$(detect_os)
    local arch=$(detect_arch)

    log_info "Detected OS: $os"
    log_info "Detected Architecture: $arch"
    log_info "Installation directory: $INSTALL_DIR"

    # Get version
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(get_latest_version)
        log_info "Latest version: $VERSION"
    fi

    # Construct download URL based on OS
    local archive_name
    local archive_format
    if [ "$os" = "Windows" ]; then
        archive_name="${BINARY_NAME}_${os}_${arch}.zip"
        archive_format="zip"
    else
        archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"
        archive_format="tar.gz"
    fi

    local download_url="https://github.com/${REPO}/releases/download/${VERSION}/${archive_name}"

    log_info "Downloading from: $download_url"

    # Create temporary directory
    local tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Download archive
    if command -v curl >/dev/null 2>&1; then
        if ! curl -fsSL "$download_url" -o "$tmp_dir/$archive_name"; then
            log_error "Failed to download from $download_url"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -q "$download_url" -O "$tmp_dir/$archive_name"; then
            log_error "Failed to download from $download_url"
            exit 1
        fi
    fi

    log_info "Download complete"

    # Extract archive based on format
    log_info "Extracting archive..."
    if [ "$archive_format" = "zip" ]; then
        if command -v unzip >/dev/null 2>&1; then
            unzip -q "$tmp_dir/$archive_name" -d "$tmp_dir"
        else
            log_error "unzip is required to extract the archive. Please install unzip and try again."
            exit 1
        fi
    else
        tar -xzf "$tmp_dir/$archive_name" -C "$tmp_dir"
    fi

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        log_info "Creating installation directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR" || {
            log_error "Failed to create $INSTALL_DIR. Try running with sudo or set PETSTORE_INSTALL_DIR to a writable location."
            exit 1
        }
    fi

    # Install binary (Windows binaries have .exe extension)
    local source_binary="$tmp_dir/$BINARY_NAME"
    local target_binary="$INSTALL_DIR/$BINARY_NAME"

    if [ "$os" = "Windows" ]; then
        source_binary="$tmp_dir/${BINARY_NAME}.exe"
        target_binary="$INSTALL_DIR/${BINARY_NAME}.exe"
    fi

    log_info "Installing to $target_binary..."
    if ! mv "$source_binary" "$target_binary"; then
        log_error "Failed to install to $INSTALL_DIR. Try running with sudo or set PETSTORE_INSTALL_DIR to a writable location."
        exit 1
    fi

    # Make executable (not needed on Windows, but doesn't hurt)
    chmod +x "$target_binary" 2>/dev/null || true

    log_info "petstore ${VERSION} has been installed to $target_binary"

    # Verify installation
    local cmd_to_check="$BINARY_NAME"
    if [ "$os" = "Windows" ]; then
        cmd_to_check="${BINARY_NAME}.exe"
    fi

    if command -v "$cmd_to_check" >/dev/null 2>&1; then
        log_info "Installation successful! Run '$BINARY_NAME --help' to get started."
    else
        log_warn "Installation complete, but $BINARY_NAME is not in your PATH."
        if [ "$os" = "Windows" ]; then
            log_warn "Add $INSTALL_DIR to your PATH environment variable."
        else
            log_warn "Add $INSTALL_DIR to your PATH by adding this to your ~/.bashrc or ~/.zshrc:"
            log_warn "  export PATH=\"\$PATH:$INSTALL_DIR\""
            log_warn ""
            log_warn "Then run: source ~/.bashrc  # or source ~/.zshrc"
        fi
    fi
}

# Main execution
main() {
    log_info "Installing petstore CLI..."
    install_cli
}

main


` // end of snapshot

	snaptest.DoTestSnapshot(t, snaptest.Options{
		Spec:         cliReleaseSnapshotSpec,
		GenYaml:      genYaml,
		IncludeGlobs: cliReleaseExpectedSnapshotFiles,
		Expected:     expectedSnapshot,
	})
}

func TestSnapCLIReleaseDistributionSections(t *testing.T) {
	t.Parallel()

	t.Run("homebrew only", func(t *testing.T) {
		t.Parallel()
		tempDir := generateCLIReleaseTestProject(t, `cli:
  packageName: github.com/example/petstore-cli
  cliName: petstore
  envVarPrefix: PETSTORE
  generateRelease: true
  distribution:
    homebrew:
      enabled: true
      tap: example/homebrew-petstore
`)

		goreleaserBytes, err := os.ReadFile(filepath.Join(tempDir, ".goreleaser.yaml"))
		require.NoError(t, err)
		workflowBytes, err := os.ReadFile(filepath.Join(tempDir, ".github", "workflows", "release.yaml"))
		require.NoError(t, err)
		readmeBytes, err := os.ReadFile(filepath.Join(tempDir, "README.md"))
		require.NoError(t, err)

		goreleaser := string(goreleaserBytes)
		workflow := string(workflowBytes)
		readme := string(readmeBytes)

		require.Contains(t, goreleaser, "brews:")
		require.Contains(t, goreleaser, "token: \"{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}\"")
		require.NotContains(t, goreleaser, "winget:")
		require.NotContains(t, goreleaser, "nfpms:")

		require.Contains(t, workflow, "HOMEBREW_TAP_GITHUB_TOKEN")
		require.Contains(t, workflow, "Import GPG key")
		require.Contains(t, workflow, "GPG_FINGERPRINT")
		require.NotContains(t, workflow, "WINGET_GITHUB_TOKEN")

		require.Contains(t, goreleaser, "signs:")
		require.Contains(t, goreleaser, "{{ .Env.GPG_FINGERPRINT }}")

		require.Contains(t, readme, "### Homebrew (macOS/Linux)")
		require.Contains(t, readme, "brew install example/petstore/petstore")
		require.NotContains(t, readme, "### WinGet (Windows)")
		require.NotContains(t, readme, "### Linux Packages")
	})

	t.Run("all channels", func(t *testing.T) {
		t.Parallel()
		tempDir := generateCLIReleaseTestProject(t, `cli:
  packageName: github.com/example/petstore-cli
  cliName: petstore
  envVarPrefix: PETSTORE
  generateRelease: true
  distribution:
    homebrew:
      enabled: true
      tap: example/homebrew-petstore
    winget:
      enabled: true
      publisher: Example
      repositoryOwner: example
      publisherUrl: https://example.com
      packageIdentifier: Example.Petstore
      license: Apache-2.0
    nfpm:
      enabled: true
      formats: deb,rpm
      maintainer: Team <team@example.com>
      license: Apache-2.0
`)

		goreleaserBytes, err := os.ReadFile(filepath.Join(tempDir, ".goreleaser.yaml"))
		require.NoError(t, err)
		workflowBytes, err := os.ReadFile(filepath.Join(tempDir, ".github", "workflows", "release.yaml"))
		require.NoError(t, err)
		readmeBytes, err := os.ReadFile(filepath.Join(tempDir, "README.md"))
		require.NoError(t, err)

		goreleaser := string(goreleaserBytes)
		workflow := string(workflowBytes)
		readme := string(readmeBytes)

		require.Contains(t, goreleaser, "brews:")
		require.Contains(t, goreleaser, "winget:")
		require.Contains(t, goreleaser, "nfpms:")
		require.Contains(t, goreleaser, "owner: microsoft")
		require.Contains(t, goreleaser, "formats:")
		require.True(t, strings.Contains(goreleaser, "- deb") && strings.Contains(goreleaser, "- rpm"))

		require.Contains(t, workflow, "HOMEBREW_TAP_GITHUB_TOKEN")
		require.Contains(t, workflow, "WINGET_GITHUB_TOKEN")
		require.Contains(t, workflow, "Import GPG key")
		require.Contains(t, workflow, "GPG_FINGERPRINT")

		require.Contains(t, goreleaser, "signs:")
		require.Contains(t, goreleaser, "{{ .Env.GPG_FINGERPRINT }}")

		require.Contains(t, readme, "### Homebrew (macOS/Linux)")
		require.Contains(t, readme, "### WinGet (Windows)")
		require.Contains(t, readme, "### Linux Packages")
		require.Contains(t, readme, "winget install Example.Petstore")
		require.Contains(t, readme, "sudo dpkg -i petstore_*_amd64.deb")
		require.Contains(t, readme, "sudo rpm -i petstore_*_amd64.rpm")
	})
}

func TestSnapCLIReleaseWithoutGitHubRepoOmitsReleaseGitHub(t *testing.T) {
	t.Parallel()

	tempDir := generateCLIReleaseTestProject(t, `cli:
  packageName: gitlab.com/example/petstore-cli
  cliName: petstore
  envVarPrefix: PETSTORE
  generateRelease: true
`)

	goreleaserBytes, err := os.ReadFile(filepath.Join(tempDir, ".goreleaser.yaml"))
	require.NoError(t, err)
	workflowBytes, err := os.ReadFile(filepath.Join(tempDir, ".github", "workflows", "release.yaml"))
	require.NoError(t, err)
	installBytes, err := os.ReadFile(filepath.Join(tempDir, "scripts", "install.sh"))
	require.NoError(t, err)

	goreleaser := string(goreleaserBytes)
	workflow := string(workflowBytes)
	install := string(installBytes)

	require.Contains(t, goreleaser, "release:\n  draft: false\n  prerelease: auto\n  mode: append")
	require.NotContains(t, goreleaser, "release:\n  github:")
	require.NotContains(t, goreleaser, "signs:")
	require.NotContains(t, workflow, "Import GPG key")
	require.NotContains(t, workflow, "GPG_FINGERPRINT")
	require.Contains(t, install, "${PETSTORE_INSTALL_DIR}")
	require.Contains(t, install, "${PETSTORE_VERSION:-latest}")
}

func TestSnapCLIReleaseArchiveNameMatchesInstallScripts(t *testing.T) {
	t.Parallel()

	for _, packageName := range []string{
		"github.com/example/petstore-cli",
		"gitlab.com/example/petstore-cli",
	} {
		t.Run(packageName, func(t *testing.T) {
			t.Parallel()

			tempDir := generateCLIReleaseTestProject(t, `cli:
  packageName: `+packageName+`
  cliName: petstore
  envVarPrefix: PETSTORE
  generateRelease: true
`)

			goreleaserBytes, err := os.ReadFile(filepath.Join(tempDir, ".goreleaser.yaml"))
			require.NoError(t, err)
			installShBytes, err := os.ReadFile(filepath.Join(tempDir, "scripts", "install.sh"))
			require.NoError(t, err)
			installPs1Bytes, err := os.ReadFile(filepath.Join(tempDir, "scripts", "install.ps1"))
			require.NoError(t, err)

			require.Contains(t, string(goreleaserBytes), "\nproject_name: petstore\n")
			require.Contains(t, string(goreleaserBytes), "{{ .ProjectName }}_")
			require.Contains(t, string(installShBytes), `BINARY_NAME="petstore"`)
			require.Contains(t, string(installShBytes), `archive_name="${BINARY_NAME}_${os}_${arch}.tar.gz"`)
			require.Contains(t, string(installPs1Bytes), `$archiveName = "petstore_Windows_$arch.zip"`)
		})
	}
}

func TestSnapCLIReleaseDisabled(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	genYamlPath := filepath.Join(tempDir, ".speakeasy", "gen.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(genYamlPath), 0o755))
	require.NoError(t, os.WriteFile(genYamlPath, []byte(`configVersion: 2.0.0
cli:
  packageName: github.com/example/petstore-cli
  cliName: petstore
  envVarPrefix: PETSTORE
  generateRelease: false
`), 0o644))

	g, err := generate.New()
	require.NoError(t, err)

	errs := g.Generate(
		generationaccess.WithDirect(context.Background(), generationaccess.ElectAGPL()),
		[]byte(cliReleaseSnapshotSpec),
		"cli_release_test.yaml",
		"cli",
		tempDir,
		false,
		false,
	)
	require.Empty(t, errs)

	_, err = os.Stat(filepath.Join(tempDir, ".goreleaser.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)

	_, err = os.Stat(filepath.Join(tempDir, ".github", "workflows", "release.yaml"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestSnapCLIReleaseGoReleaserCheck(t *testing.T) {
	if _, err := exec.LookPath("goreleaser"); err != nil {
		t.Skip("goreleaser is not installed")
	}

	t.Parallel()

	tempDir := generateCLIReleaseTestProject(t, `cli:
  packageName: github.com/example/petstore-cli
  cliName: petstore
  envVarPrefix: PETSTORE
  generateRelease: true
  distribution:
    homebrew:
      enabled: true
      tap: example/homebrew-petstore
    winget:
      enabled: true
      publisher: Example Inc.
      publisherUrl: https://example.com
      repositoryOwner: example
      packageIdentifier: Example.Petstore
      license: MIT
    nfpm:
      enabled: true
      formats: deb,rpm
      maintainer: Example Inc. <dev@example.com>
      license: MIT
`)

	cmd := exec.Command("goreleaser", "check")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func generateCLIReleaseTestProject(t *testing.T, genYaml string) string {
	t.Helper()

	tempDir := t.TempDir()
	genYamlPath := filepath.Join(tempDir, ".speakeasy", "gen.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(genYamlPath), 0o755))
	require.NoError(t, os.WriteFile(genYamlPath, []byte("configVersion: 2.0.0\n"+genYaml), 0o644))

	g, err := generate.New()
	require.NoError(t, err)

	errs := g.Generate(
		generationaccess.WithDirect(context.Background(), generationaccess.ElectAGPL()),
		[]byte(cliReleaseSnapshotSpec),
		"cli_release_test.yaml",
		"cli",
		tempDir,
		false,
		false,
	)
	require.Empty(t, errs)

	return tempDir
}
