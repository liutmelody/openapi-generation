# cli

Command-line interface for the *SDK Review* API.

[![Built by Speakeasy](https://img.shields.io/badge/Built_by-SPEAKEASY-374151?style=for-the-badge&labelColor=f3f4f6)](https://www.speakeasy.com/?utm_source=openapi&utm_campaign=cli)
[![License: MIT](https://img.shields.io/badge/LICENSE_//_MIT-3b5bdb?style=for-the-badge&labelColor=eff6ff)](https://opensource.org/licenses/MIT)


<br /><br />
> [!IMPORTANT]
> This CLI is not yet ready for production use. Delete this notice before publishing to a package manager.

<!-- Start Summary [summary] -->
## Summary

SDK Review: A test document for reviewing the SDK.

This document will show case as many of our features as possible in as little operations/models as possible.
This will then generate a SDK that we can more easily review than the test SDKs based on uber.yaml spec.

For more information about the API: [Speakeasy Docs](https://speakeasy.com/docs)
<!-- End Summary [summary] -->

<!-- Start Table of Contents [toc] -->
## Table of Contents
<!-- $toc-max-depth=2 -->
* [cli](#cli)
  * [CLI Installation](#cli-installation)
  * [Shell Completion](#shell-completion)
  * [CLI Example Usage](#cli-example-usage)
  * [For AI agents](#for-ai-agents)
  * [Authentication](#authentication)
  * [Configuration](#configuration)
  * [Commands](#commands)
  * [Request Body Input](#request-body-input)
  * [Server Selection](#server-selection)
  * [Output Formats](#output-formats)
  * [Server-Sent Event Streaming](#server-sent-event-streaming)
  * [Pagination](#pagination)
  * [Retries](#retries)
  * [Error Handling](#error-handling)
  * [Diagnostics](#diagnostics)
* [Development](#development)
  * [Maturity](#maturity)
  * [Contributions](#contributions)

<!-- End Table of Contents [toc] -->

<!-- Start CLI Installation [installation] -->
## CLI Installation

To install the CLI, use `go install`:
```bash
go install openapi/cmd/cli@latest
```

Or download a pre-built binary from the [releases page](https://openapi/releases) if available.
<!-- End CLI Installation [installation] -->

<!-- Start Shell Completion [completion] -->
## Shell Completion

Shell completions are available for Bash, Zsh, Fish, and PowerShell.

### Bash

```bash
# Add to ~/.bashrc:
source <(cli completion bash)

# Or install permanently:
cli completion bash > /etc/bash_completion.d/cli
```

### Zsh

```zsh
# Add to ~/.zshrc:
source <(cli completion zsh)

# Or install permanently:
cli completion zsh > "${fpath[1]}/_cli"
```

### Fish

```fish
cli completion fish | source

# Or install permanently:
cli completion fish > ~/.config/fish/completions/cli.fish
```

### PowerShell

```powershell
cli completion powershell | Out-String | Invoke-Expression
```
<!-- End Shell Completion [completion] -->

<!-- Start CLI Example Usage [usage] -->
## CLI Example Usage

### Quick start

```bash
# Invite by email
cli invite someone@example.com

cli enroll --enrollment-email someone@example.com

cli tag1 recent --page "100" --query-param2 "1" --header-param1 "some example header param"
```

### Example 1

```bash
cli post-file --upload ../.speakeasy/testfiles/example.file

```

### Example 2

```bash
cli tag1 post-file-with-encoding --file ../.speakeasy/testfiles/example.file

```

### Example 3

```bash
cli test-group tag2 post-test --deprecated-query-param2 'some example query param' --obj '{"str":"example","bool":true,"int":999999,"int32":1,"num":1.1,"float32":2940.96,"enumProp":"First","date":"2020-01-01","dateTime":"2020-01-01T00:00:00Z","anything":"<value>","boolOpt":true,"intOptNull":999999,"numOptNull":1.1,"intEnum":3,"int32Enum":69,"bigint":702830,"bigintStr":"12345678901234567890","decimal":3.141592653589,"decimalStr":"3858.6","obj":{"str":"example"},"map":{"key":{"str":"example"}},"arr":[{"str":"example"}],"any":"<value>","type":"0","nullableIntEnum":3,"nullableStringEnum":"Second","color":"green","icon":"tick","heroWidth":480}' --type type1

```

### A custom readme heading

A custom usage description

```bash
cli tag1 list-test1 --page 100 --query-param1 'some example query param' --query-param2 1 --header-param1 'some example header param'

```
<!-- End CLI Example Usage [usage] -->

<!-- Start For AI agents [agents] -->
## For AI agents

This CLI is built to be driven by AI coding agents as well as people: everything an agent needs is discoverable from the binary itself, and every command can be validated without credentials. Work down this ladder:

| Run | You get |
|-----|---------|
| `cli --help`, `cli get-duplicate-export-collision --help` | Commands by category, runnable examples, flags |
| `cli --usage`, `cli get-duplicate-export-collision --usage` | The command surface as machine-readable [KDL](https://kdl.dev): commands, aliases, flags, defaults, env vars, config keys |
| `cli invite --schema` | The exact JSON Schema of the command's request body (all `$ref`s bundled) — build a valid `--body` from it |
| `cli get-duplicate-export-collision --dry-run` | The exact HTTP request (method, URL, headers, body), with no credentials or network call |
| `cli get-duplicate-export-collision --output-format json` (or `--jq`) | Machine-readable output |

### Discover the command surface

```bash
# Every command, flag, default, env var and config key, as KDL
cli --usage

# One command's subtree only
cli get-duplicate-export-collision --usage
```

### Read the exact request schema

`--schema` is available on every command that accepts a request body (`--body`, stdin, or a whole-body flag where the command has one), including intent commands. It prints the JSON Schema the request is validated against and exits without calling the API.

```bash
# JSON Schema (draft 2020-12) of the request body, with every $ref bundled under $defs
cli invite --schema
```

### Probe before you spend

Start quota-spending commands with `--dry-run`. It validates inputs, resolves the request, redacts secrets and binary payloads, makes no network call, and exits 0. It never reads the OS keychain; credentials supplied by flag, environment, or config file are included only as `[REDACTED]`.

```bash
# Human preview: the [DRY-RUN] block is on stderr and stdout is empty
cli get-duplicate-export-collision --dry-run
cli invite someone@example.com --dry-run

# Machine preview: compact JSON on stdout and silent stderr
cli get-duplicate-export-collision --dry-run --output-format json
```

The machine form writes one object per would-be request, one per line (NDJSON for multi-request commands), with exactly this shape:

```json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
```

`body` is a parsed JSON value when the body is JSON, a string for text, `"<bytes:N>"` for binary data, and `null` when absent. An explicit caller `--jq` also selects this JSON preview protocol, but the filter is not applied to preview objects. Command-declared jq presets do not select or filter the preview.

Local mutation commands make no request under `--dry-run`: instead of a preview they emit one `{"dry_run":true,"local":true,"command":"…","message":"…"}` object. `select(.request)` keeps only would-be requests; `select(.local)` keeps the local no-ops.

### Machine-readable output

```bash
# JSON on stdout
cli get-duplicate-export-collision --output-format json

# Filter or reshape with a jq expression (always emits JSON, overrides --output-format)
cli get-duplicate-export-collision --jq '.'

# Print jq string results as plain text instead of JSON strings (like jq -r)
cli get-duplicate-export-collision --jq '.' --raw-output
```

`--output-format toon` emits [TOON](https://github.com/toon-format/spec), a compact line-oriented format that uses fewer tokens than JSON; it is the default in agent mode.

### Interactive mode
This CLI is non-interactive by default. Pass `--interactive` to prompt for missing inputs or open guided `configure` / `auth login` forms. Required-input prompts require an interactive terminal; off-TTY forms read line input from stdin.

```bash
# Prompt for missing command inputs
cli invite --interactive

# Open the guided configuration form
cli configure --interactive

# Explicitly launch the terminal command explorer
cli explore
```

### Agent mode and structured errors
Agent mode turns on only when explicitly requested with `--agent-mode`; environment variables do not identify the caller.
In agent mode interactive prompts never launch, output defaults to TOON, and every failure — API errors and CLI usage errors alike — is one JSON envelope on stderr:
`--output-format json` and `--jq` use the same error envelope without requiring agent mode.

```json
{
  "error": "...",
  "error_type": "validation_error",
  "error_reason": "CLI_VALIDATION",
  "exit_code": 2,
  "message": "human-readable message",
  "hints": ["what to try next"]
}
```

`error_type` is one of `authentication_error`, `authorization_error`, `not_found`, `validation_error`, `rate_limit_error`, `server_error`, `api_error`, `connection_error`, `protocol_error`, `service_disabled`, `billing_disabled`, `runtime_error`, `unsupported_error`, `async_failed`, `async_timeout`, `async_unknown_state`. Classification reads the structured reason code at `$.details[*].reason`, then `$.status` in the error body (resolved against the nested `error` object when the body has one) before HTTP status, so a declared credential reason sent with HTTP 400 is not mistaken for request validation. `error_reason` carries the reason code found there, verbatim from a declared carrier when no declared rule matches it; it is absent for status-only API errors. Status-less local failures may use `CLI_VALIDATION`, `CLI_CONNECTION`, `CLI_PROTOCOL`, `CLI_RUNTIME`, `CLI_UNAVAILABLE`, `CLI_AUTHENTICATION`, the async polling reasons `CLI_ASYNC_FAILED`, `CLI_ASYNC_TIMEOUT`, and `CLI_ASYNC_UNKNOWN_STATE`, or `CLI_OUTCOME_FAILED` when a successful response reports a declared failure state. `hints` preserves server guidance first, adds the most specific local taxonomy guidance, then typed CLI and command-specific guidance, removing exact duplicates. `exit_code` is always the code for the final `error_type` shown in the envelope: 1 runtime, 2 usage, or 3 authentication/authorization.

```bash
# "invite" declares hints for CLI_VALIDATION; a failing request returns them in the envelope
cli invite someone@example.com
```

### Lists, streams, and files

List commands accept `--all` to fetch every page and stream results as they arrive (one JSON value per line with `--output-format json`; `--max-pages N` bounds the walk).

Structured output and agent mode never write pagination hints to stderr; if a later page fails or the server repeats a cursor, the command exits non-zero after the pages already written.

```bash
cli get-union-errors --page 12 --all --output-format json
```

Streaming commands write each event as it arrives (one JSON object per line with `--output-format json`; a declared streamed projection prints just the selected text, e.g. `/data/content`):

```bash
cli say "hello there" --output-format json
```

Commands that produce media write the file and print only its path on stdout (`--out <path-or-dir>` chooses the location, default `./render-{timestamp}-{rand}.{ext}`; `--raw-response` prints the API response instead):

```bash
cli render "a lighthouse at sunset" --out ./output/
```

Long-running commands poll to a terminal response; human progress goes to stderr and machine-mode success keeps stderr silent. Add `--async` to `cli produce "a lighthouse at sunrise"` to return its handle immediately, or tune foreground polling with `--poll-interval <duration>` and `--poll-timeout <duration>`. Resume an escaped or timed-out operation with `cli produce --resume <id>`.
<!-- End For AI agents [agents] -->

<!-- Start Authentication [security] -->
## Authentication

Authentication credentials can be configured in four ways (in order of priority):

### 1. Command-line flags

Pass credentials directly as flags to any command:

```bash
cli --username "$CLI_USERNAME" --password "$CLI_PASSWORD" --bearer-auth "$CLI_BEARER_AUTH" get-duplicate-export-collision
```

### 2. Environment variables

Set credentials via environment variables:

| Variable | Description |
|----------|-------------|
| `CLI_USERNAME` | HTTP Basic username |
| `CLI_PASSWORD` | HTTP Basic password |
| `CLI_BEARER_AUTH` | HTTP Bearer |
| `CLI_MY_API_KEY` | API Key |
| `CLI_OAUTH2` | OAuth2 Authorization |
| `CLI_APP_ID` | Custom authentication credential |
| `CLI_SECRET` | Custom authentication credential |
| `CLI_MOBILE_AUTH` | OAuth2 Password flow, a flow that is too long to describe in a single line. |
| `CLI_CLIENT_ID` | Client Credentials flow. client identifier |
| `CLI_CLIENT_SECRET` | Client Credentials flow. client secret |
| `CLI_TOKEN_URL` | Client Credentials flow. token URL |

### 3. OS Keychain (recommended for workstations)

Credentials are stored securely in your operating system's keychain when you run:

```bash
cli configure
```

Secret credentials (tokens, API keys, passwords) are automatically stored in:
- **macOS**: Keychain
- **Linux**: GNOME Keyring / KWallet (via D-Bus Secret Service)
- **Windows**: Windows Credential Locker

If no keychain is available (e.g., in CI environments), credentials fall back to the config file.

Where the keychain cannot be unlocked (e.g., headless Linux or SSH sessions with a locked GNOME Keyring), skip it entirely with `--no-keyring`, `CLI_NO_KEYRING=true`, or `no_keyring: true` in the config file. The keychain is then never read or written, and secrets are stored in the config file instead; flags and environment variables still take precedence.

### 4. Configuration file

Run the interactive `configure` command to store non-secret settings:

```bash
cli configure
```

Configuration is stored in `~/.config/cli/config.yaml`.
<!-- End Authentication [security] -->

<!-- Start Configuration [global-parameters] -->
## Configuration

`cli configure` stores your settings in `~/.config/cli/config.yaml`. You can run it interactively to set credentials and persistent preferences, or edit the config file directly.

For authentication credentials specifically, see [Authentication](#authentication).

### Global Parameters

Certain parameters are configured globally and applied to all commands that use them. These parameters can be set via CLI flags, environment variables, or the config file. Individual commands can override global values with their own flags when needed.

Priority: CLI flags > environment variables > config file

| Source | Example |
|--------|---------|
| CLI flag | `cli --query-param1 value get-duplicate-export-collision` |
| Environment variable | `CLI_QUERY_PARAM1=value cli get-duplicate-export-collision` |
| Config file | `cli configure` |

#### Available Global Parameters

| Flag                        | Type   | Description                                                                        | Environment                 |
| --------------------------- | ------ | ---------------------------------------------------------------------------------- | --------------------------- |
| `--query-param1`            | string | A long winded, multi-line description<br/>for the query parameter number one.<br/> | CLI_QUERY_PARAM1            |
| `--deprecated-query-param1` | string | A deprecated description                                                           | CLI_DEPRECATED_QUERY_PARAM1 |
| `--deprecated-query-param2` | string | The DeprecatedQueryParam2 parameter.                                               | CLI_DEPRECATED_QUERY_PARAM2 |
| `--lone-query-param`        | string | The LoneQueryParam parameter.                                                      | CLI_LONE_QUERY_PARAM        |

### Example

```bash
# Set a global parameter via flag
cli --query-param1 value get-duplicate-export-collision

# Or set via environment variable
CLI_QUERY_PARAM1=value cli get-duplicate-export-collision

# Or configure globally (persisted to config file)
cli configure
```
<!-- End Configuration [global-parameters] -->

<!-- Start Commands [operations] -->
## Commands

Commands are grouped the way `cli --help` shows them. Every command accepts `--help`; body-bearing commands also accept `--schema` (exact request JSON Schema) and `--dry-run` (preview the request without sending it) — see [For AI agents](#for-ai-agents).

### Create

* [`invite`](docs/cli_invite.md) - Invite a user by email

  ```bash
  # Invite by email
  cli invite someone@example.com
  cli invite someone@example.com --given-name John
  ```

* [`render`](docs/cli_render.md) - Render an image asset to a file

  ```bash
  cli render "a lighthouse at sunset"
  ```

* [`produce`](docs/cli_produce.md) - Produce an image asset and wait for completion

  ```bash
  cli produce "a lighthouse at sunrise"
  ```

* [`say`](docs/cli_say.md) - Stream a chat reply

  ```bash
  cli say "hello there"
  ```

### Manage

* [`enroll`](docs/cli_enroll.md) - Enroll a user by email

  ```bash
  cli enroll --enrollment-email someone@example.com
  ```

* [`observe`](docs/cli_observe.md) - Produce an asset and print its terminal status

  ```bash
  cli observe "<prompt>"
  ```

* [`archive`](docs/cli_archive.md) - Archive a user account — _not in this build_: "archive" needs an account-lifecycle API surface that is not part of this build. Meanwhile use "cli delete-user"
* [`test-group`](docs/cli_test-group.md) - Operations for test-group
  * [`tag2`](docs/cli_test-group_tag2.md) - Operations for tag2
    * [`post-test`](docs/cli_test-group_tag2_post-test.md) - Post Test2
  * [`tag3`](docs/cli_test-group_tag3.md) - Operations for tag3
    * [`post-test`](docs/cli_test-group_tag3_post-test.md) - Post Test2
* [`hello`](docs/cli_hello.md) - Hand-written command that survives regeneration

### Additional commands

* [`operation-with-leading-and-trailing-underscores`](docs/cli_operation-with-leading-and-trailing-underscores.md)
* [`post-file`](docs/cli_post-file.md) - Post File
* [`get-polymorphism`](docs/cli_get-polymorphism.md)
* [`get-union-errors`](docs/cli_get-union-errors.md)
* [`get-request-body-flattened-away`](docs/cli_get-request-body-flattened-away.md)
* [`get-fully-flattened-request`](docs/cli_get-fully-flattened-request.md)
* [`create-with-union`](docs/cli_create-with-union.md) - Create with discriminated union request body
* [`test-endpoint`](docs/cli_test-endpoint.md)
* [`create-user`](docs/cli_create-user.md) - Create User
* [`get-user`](docs/cli_get-user.md) - Get User
* [`update-user`](docs/cli_update-user.md) - Update User
* [`delete-user`](docs/cli_delete-user.md) - Delete User
* [`login`](docs/cli_login.md) - Login
* [`validate`](docs/cli_validate.md) - Validate
* [`chat`](docs/cli_chat.md)
* [`get-binary-default-response`](docs/cli_get-binary-default-response.md)
* [`test-enum-formats`](docs/cli_test-enum-formats.md) - Test x-speakeasy-enums in different formats
* [`binary-and-string-upload`](docs/cli_binary-and-string-upload.md)
* [`get-error-in-union`](docs/cli_get-error-in-union.md)
* [`get-duplicate-export-collision`](docs/cli_get-duplicate-export-collision.md) - Tests that a spec-defined error type colliding with a built-in SDK error name does not cause TS2308
* [`get-named-primitive-union`](docs/cli_get-named-primitive-union.md) - Test named primitive union options using title and x-speakeasy-name-override
* [`get-empty-object-error`](docs/cli_get-empty-object-error.md) - Get Empty Object Error
* [`url-validation-stress-test`](docs/cli_url-validation-stress-test.md)
* [`parentheses-in-path-allowed`](docs/cli_parentheses-in-path-allowed.md) - A string with {{ double braces }} and { single braces }
and \{\{ escaped curlies \}\} and `backticks`.
and \`escaped backticks\` and double slashes\\
and 'single quotes' and "double quotes".
and  \'escaped single quotes\' and \"escaped double quotes\".

* [`get-nested-integer-string`](docs/cli_get-nested-integer-string.md) - Test nested struct with integer:string tag
* [`render-asset`](docs/cli_render-asset.md) - Render Asset
* [`get-asset`](docs/cli_get-asset.md) - Get Asset
* [`get-positional-default`](docs/cli_get-positional-default.md) - Get an item with a default path identifier
* [`get-error-only-example`](docs/cli_get-error-only-example.md) - Operation with example only on error response
* [`tag1`](docs/cli_tag1.md) - The first tag
  * [`recent`](docs/cli_tag1_recent.md) - List recent test1 pages via tag1

    ```bash
    cli tag1 recent --page "100" --query-param2 "1" --header-param1 "some example header param"
    ```

  * [`~~deprecated1~~`](docs/cli_tag1_deprecated1.md) - Deprecated Operation :warning: **Deprecated**
  * [`auth`](docs/cli_tag1_auth.md) - This operation aims at testing available OAuth2 scopes collection:
 - only operation with oauth2 authorizationCode security flow
 - belongs to a subSDK

  * [`list-test1`](docs/cli_tag1_list-test1.md) - Get Test1
  * [`post-file-with-encoding`](docs/cli_tag1_post-file-with-encoding.md) - Post File With Encoding
* [`obsolete`](docs/cli_obsolete.md) - A subSDK in which all operations are deprecated
  * [`~~deprecated1~~`](docs/cli_obsolete_deprecated1.md) - Deprecated Operation :warning: **Deprecated**
* [`group`](docs/cli_group.md) - Operations for group
  * [`root-group-op`](docs/cli_group_root-group-op.md) - An operation at the group's root level
  * [`sub`](docs/cli_group_sub.md) - Operations for sub-group
    * [`op`](docs/cli_group_sub_op.md) - An operation at the group's top level
    * [`empty`](docs/cli_group_sub_empty.md) - Operations for empty
      * [`tail`](docs/cli_group_sub_empty_tail.md) - Operations for tail
        * [`nested-group-op`](docs/cli_group_sub_empty_tail_nested-group-op.md) - An operation at the group's deepest level
* [`namespace-tests`](docs/cli_namespace-tests.md) - Operations for namespace-tests
  * [`conflicts`](docs/cli_namespace-tests_conflicts.md) - Operations for conflicts
    * [`get-namespace`](docs/cli_namespace-tests_conflicts_get-namespace.md) - Get Namespace Conflict Test
    * [`put-namespace`](docs/cli_namespace-tests_conflicts_put-namespace.md) - Put Property Name Conflicts Behind
    * [`create-namespace`](docs/cli_namespace-tests_conflicts_create-namespace.md) - Create Namespace Conflict Test
    * [`get-triple-namespace`](docs/cli_namespace-tests_conflicts_get-triple-namespace.md) - Get Triple Namespace Conflict Test
    * [`get-pet-owners`](docs/cli_namespace-tests_conflicts_get-pet-owners.md) - Get Pet Owners
  * [`single-foo`](docs/cli_namespace-tests_single-foo.md) - Operations for single-foo
    * [`get-single-namespace-foo-pet`](docs/cli_namespace-tests_single-foo_get-single-namespace-foo-pet.md) - Get Single Namespace Foo Pet
    * [`create-single-namespace-foo-pet`](docs/cli_namespace-tests_single-foo_create-single-namespace-foo-pet.md) - Create Single Namespace Foo Pet
  * [`single-bar`](docs/cli_namespace-tests_single-bar.md) - Operations for single-bar
    * [`get-single-namespace-bar-pet`](docs/cli_namespace-tests_single-bar_get-single-namespace-bar-pet.md) - Get Single Namespace Bar Pet
  * [`types`](docs/cli_namespace-tests_types.md) - Operations for types
    * [`get-namespace`](docs/cli_namespace-tests_types_get-namespace.md) - Get Namespace Types Test
    * [`get-namespace-animal`](docs/cli_namespace-tests_types_get-namespace-animal.md) - Get Namespace Animal (Discriminated Union)
    * [`get-namespace-vehicle`](docs/cli_namespace-tests_types_get-namespace-vehicle.md) - Get Namespace Vehicle (Non-Discriminated Union)
    * [`get-namespace-organization`](docs/cli_namespace-tests_types_get-namespace-organization.md) - Get Namespace Organization (Nested Inline Schemas)
* [`stream-metadata`](docs/cli_stream-metadata.md) - Streamed events can contain text and an optional resource identifier in separate events.
<!-- End Commands [operations] -->

<!-- Start Request Body Input [stdinpiping] -->
## Request Body Input

Commands that accept a request body take it three ways, with a clear priority chain. The examples use `cli namespace-tests conflicts create-namespace`; every body-bearing command works the same way and prints its exact request schema with `--schema`.

### Individual flags (highest priority)

Each top-level body field is a flag:

```bash
cli namespace-tests conflicts create-namespace --id 'pet-foo-123' --name 'Fluffy' --species 'cat'
```

### `--body` flag

Provide the entire request body as a JSON string:

```bash
cli namespace-tests conflicts create-namespace --body '{"id":"pet-foo-123","name":"Fluffy","species":"cat"}'
```

Individual flags override `--body` values:

```bash
# Sends {"id":"pet-foo-123 (updated)","name":"Fluffy","species":"cat"}
cli namespace-tests conflicts create-namespace --body '{"id":"pet-foo-123","name":"Fluffy","species":"cat"}' --id 'pet-foo-123 (updated)'
```

### Stdin piping (lowest priority)

Pipe JSON into any command that accepts a request body:

```bash
echo '{"id":"pet-foo-123","name":"Fluffy","species":"cat"}' | cli namespace-tests conflicts create-namespace
```

Individual flags override stdin values:

```bash
# Sends {"id":"pet-foo-123 (updated)","name":"Fluffy","species":"cat"}
echo '{"id":"pet-foo-123","name":"Fluffy","species":"cat"}' | cli namespace-tests conflicts create-namespace --id 'pet-foo-123 (updated)'
```

This is useful for chaining commands, reading from files, or scripting:

```bash
# Read body from a file
cli namespace-tests conflicts create-namespace < request.json

# Pipe from another command
curl -s https://example.com/request.json | cli namespace-tests conflicts create-namespace
```

### Priority

When multiple input methods are used, the priority is:

| Priority | Source | Description |
|----------|--------|-------------|
| 1 (highest) | Individual flags | `--id ...` always wins |
| 2 | `--body` flag | Whole-body JSON via flag |
| 3 (lowest) | Stdin | Piped JSON input |
<!-- End Request Body Input [stdinpiping] -->

<!-- Start Server Selection [server] -->
## Server Selection

### Select Server by Index

Use `--server <index>` to select a server by its zero-based index (default: `0`):

| #   | Server                                     | Variables                 | Description                     |
| --- | ------------------------------------------ | ------------------------- | ------------------------------- |
| 0   | `http://localhost:35123`                   |                           | The default server.             |
| 1   | `http://{subdomain}.domain.com/v{version}` | `subdomain`<br/>`version` |                                 |
| 2   | `http://{HostName}:{PORT}`                 | `HostName`<br/>`PORT`     | A server with an enum variable. |

```bash
cli --server 0 get-duplicate-export-collision
```

### Server Variables

Some server URLs contain template variables (e.g., `https://{hostname}:{port}/v1`). Set these via dedicated flags:

| Variable    | Flag                      | Supported Values                      | Default       | Description                              |
| ----------- | ------------------------- | ------------------------------------- | ------------- | ---------------------------------------- |
| `subdomain` | `--subdomain <value>`     | string                                | `"api"`       |                                          |
| `version`   | `--version-param <value>` | string                                | `"1"`         |                                          |
| `HostName`  | `--host-name <value>`     | string                                | `"localhost"` | The hostname of the server.              |
| `PORT`      | `--port <value>`          | - `"80"`<br/>- `"8080"`<br/>- `"443"` | `"8080"`      | The port on which the server is running. |

```bash
cli --subdomain value --version-param value --host-name value --port value get-duplicate-export-collision
```

Server variable flags are combined with the selected server URL to produce the final endpoint.
### Override Server URL

Use `--server-url` to override the server URL entirely, bypassing any named or indexed server selection:

```bash
cli --server-url https://custom-api.example.com get-duplicate-export-collision
```

**Precedence**: `--server-url` > `--server` > default
<!-- End Server Selection [server] -->

<!-- Start Output Formats [output-formats] -->
## Output Formats

Every command supports a `--output-format` flag that controls how the response is rendered to stdout.

### Available formats

| Format | Flag | Description |
|--------|------|-------------|
| Pretty | `--output-format pretty` (default) | Aligned key-value pairs with color, nested indentation. Human-readable at a glance. |
| JSON | `--output-format json` | JSON output. Passthrough when the response is already JSON (preserves original field order and numeric precision). Falls back to typed marshaling otherwise. |
| YAML | `--output-format yaml` | YAML output via standard marshaling. |
| Table | `--output-format table` | Tabular output for array responses. |
| TOON | `--output-format toon` | [Token-Oriented Object Notation](https://github.com/toon-format/spec) — a compact, line-oriented format that typically uses 30–60% fewer tokens than JSON. Well-suited for piping responses into LLM prompts. |

```bash
# Default pretty output
cli get-duplicate-export-collision

# Machine-readable JSON
cli get-duplicate-export-collision --output-format json

# TOON for LLM-friendly compact output
cli get-duplicate-export-collision --output-format toon

# Pipe JSON to jq without using --output-format
cli get-duplicate-export-collision --output-format json | jq '.'
```

### jq filtering

Use `--jq` to filter or transform the response inline using a [jq](https://jqlang.org) expression. This always outputs JSON and overrides `--output-format`:

```bash
# Extract a single field
cli get-duplicate-export-collision --jq '.'

# Reshape with any jq program; --raw-output prints string results as plain text (like jq -r)
cli get-duplicate-export-collision --jq '.' --raw-output
```

### Color control

Use `--color` to control terminal colors:

| Value | Behavior |
|-------|----------|
| `auto` (default) | Color when stdout is a TTY, plain text otherwise |
| `always` | Always colorize |
| `never` | Never colorize |

The `NO_COLOR` and `FORCE_COLOR` environment variables are also respected.

### Streaming and pagination

When using `--all` (pagination) or streaming operations, output is written incrementally as items arrive:

| Format | Streaming behavior |
|--------|-------------------|
| `json` | One compact JSON object per line ([NDJSON](https://github.com/ndjson/ndjson-spec)) |
| `yaml` | YAML documents separated by `---` |
| `toon` | One TOON-encoded object per block, separated by blank lines |
| `pretty` (default) | Pretty-printed items separated by blank lines |
<!-- End Output Formats [output-formats] -->

<!-- Start Server-Sent Event Streaming [eventstreaming] -->
## Server-Sent Event Streaming

Some operations return server-sent events (SSE). These are streamed to the terminal in real-time, with each event output as a separate JSON object (one per line).

```bash
# Stream events in JSON format
cli say "hello there" --output-format json

# Filter streaming events with jq
cli say "hello there" --output-format json --jq '.'
```

Events are output as they arrive. Use `Ctrl+C` to stop streaming.

For operation commands with a declared streamed projection, the selected string is written raw as it arrives. When the command exposes a stream toggle flag, its default decides the response shape — the command's help says whether to pass `--stream=false` for one complete JSON response (streaming on by default) or `--stream` to request a streamed response (off by default). Use `-o json` to keep each full streamed event. On a command that streams by default, `--jq` without an explicit output format or stream setting requests one complete JSON response and filters that.
<!-- End Server-Sent Event Streaming [eventstreaming] -->

<!-- Start Pagination [pagination] -->
## Pagination

Some operations in this CLI support automatic pagination. These operations accept `--all` to automatically fetch all pages and stream results incrementally.

### Basic usage

```bash
# Fetch a single page (default behavior)
cli get-union-errors --page 12

# Automatically fetch all pages
cli get-union-errors --page 12 --all
```

### Limiting pages

Use `--max-pages` with `--all` to cap the number of pages fetched. A negative value is invalid; `0` means unlimited. Passing `--max-pages` without `--all` is an error.

```bash
# Fetch at most 5 pages
cli get-union-errors --page 12 --all --max-pages 5
```

### Output formats

When using `--all`, output is streamed as each page is fetched. Operations whose pagination declaration names an `outputs.results` array emit one item at a time. Other operations emit one complete page object at a time, preserving the single-page response shape and any continuation cursor.

| Format | Behavior |
|--------|----------|
| `--output-format json` | One JSON object per line ([NDJSON](https://github.com/ndjson/ndjson-spec)) |
| `--output-format yaml` | YAML documents separated by `---` |
| `--output-format toon` | One TOON-encoded block per item, separated by blank lines |
| Default (pretty) | Pretty-printed items separated by blank lines |

```bash
# Stream all results as NDJSON
cli get-union-errors --page 12 --all --output-format json

# Pipe to jq for further processing
cli get-union-errors --page 12 --all --output-format json | jq '.'

# Use the built-in --jq flag
cli get-union-errors --page 12 --all --jq '.'
```

### How it works

Under the hood, `--all` calls the operation once, then follows the underlying `Next()` pagination closure to fetch subsequent pages. Results are written to stdout as they arrive rather than buffered in memory, so this works well even with large result sets.

Without `--all`, paginated operations behave like any other command — pass cursor, page, offset, or limit flags manually and get a single page of results. In pretty or table output, a cursor response that proves another page exists prints a hint on stderr. JSON, YAML, TOON, `--jq`, and agent mode keep stderr silent on success. Offset/limit responses do not guess from a full result page. Cursor operations that declare both a results array and a mutable limit also suppress the hint because the client cannot safely reproduce the SDK's runtime limit check.

Pagination can fail after earlier pages have already been written. A later-page API failure or a repeated/cyclic continuation cursor stops with a non-zero exit status; callers should treat stdout as partial whenever the command exits non-zero. `--all` tracks cursor values and stops before issuing another request when the server repeats one; the same applies to next URLs when the target generator supports them.
<!-- End Pagination [pagination] -->

<!-- Start Retries [retries] -->
## Retries

Some operations in this CLI support automatic retries with exponential backoff.

### Retry controls

```bash
# Disable retries for one command
cli get-duplicate-export-collision --no-retries

# Replace the retry policy with backoff capped by elapsed time
cli get-duplicate-export-collision --retry-max-elapsed-time 5s

# Retry eligible connection errors
cli get-duplicate-export-collision --retry-connection-errors

# Supply the complete retry configuration
cli get-duplicate-export-collision --retry-config '{"strategy":"backoff","backoff":{"initialInterval":500,"maxInterval":60000,"exponent":1.5,"maxElapsedTime":300000}}'
```

### Persist retry settings

Add retry configuration with `cli configure` or edit `~/.config/cli/config.yaml`:

```yaml
timeout: 30s
no_retries: false
retry_connection_errors: true
retry_max_elapsed_time: 1m
# retry_config replaces the whole policy (overrides retry_max_elapsed_time):
# retry_config: '{"strategy":"backoff","backoff":{"initialInterval":500,"maxInterval":60000,"exponent":1.5,"maxElapsedTime":300000}}'
```

### Retry-After

`Retry-After` (integer seconds or an RFC1123 date) and `retry-after-ms` override the next computed interval. With the `backoff` strategy, a server-directed wait that exceeds the remaining `maxElapsedTime` budget is not slept; the last response is returned.

### Timeout

`timeout` and `--timeout` bound the whole operation, including retry sleeps:

```bash
cli get-duplicate-export-collision --timeout 30s
```

**Precedence**: `--no-retries` > `--retry-config` > individual flags > config file > API specification defaults.
<!-- End Retries [retries] -->

<!-- Start Error Handling [errors] -->
## Error Handling

The CLI uses standard exit codes to indicate success or failure:

| Exit Code | Meaning |
|-----------|---------|
| `0` | Success |
| `1` | Runtime/API failure |
| `2` | Usage or input failure |
| `3` | Authentication or authorization failure |

On success, the response data is printed to **stdout** as JSON. On failure, error details are printed to **stderr**.

```bash
# Capture output and handle errors
cli get-duplicate-export-collision --output-format json > output.json 2> error.log
if [ $? -ne 0 ]; then
  echo "Error occurred, see error.log"
fi
```
In pretty mode, each error is printed once as `Error (<type>): <message>`, followed by its reason/HTTP status, actionable `Fix:` bullets, and only non-duplicative residual details.

In agent mode, or with explicit `--output-format json`, `--output-format toon`, or `--jq` machine output, stderr is one classified JSON envelope with `exit_code`, `error_type`, optional `error_reason`, `message`, `hints`, and optional `status_code` — see [For AI agents](#for-ai-agents).

`error_reason` is the structured reason code read from the error body at `$.details[*].reason`, then `$.status` (resolved against the nested `error` object when the body has one).
<!-- End Error Handling [errors] -->

<!-- Start Diagnostics [diagnostics] -->
## Diagnostics

The CLI includes two diagnostic flags available on all commands:

### Dry Run

Preview what would be sent without making any network calls:

```bash
cli get-duplicate-export-collision --dry-run
```

In human output modes, stdout is empty and the `[DRY-RUN]` block goes to stderr. It includes:
- HTTP method and URL
- Request headers (sensitive values redacted)
- Request body preview (sensitive fields redacted)

With `--output-format json`, or with a caller-explicit `--jq`, stderr is silent and stdout is NDJSON: one compact preview object per would-be request. The jq filter is not applied, and command-declared jq presets do not select the JSON protocol.

```json
{"dry_run":true,"request":{"method":"POST","url":"https://…","headers":{"Accept":["application/json"],…},"body":<JSON value | string | null>}}
```

JSON bodies remain structured; text bodies are strings; binary bodies are `"<bytes:N>"`; absent bodies are `null`. Headers retain all values as arrays, with credentials replaced by `[REDACTED]`. Dry-run never reads the OS keychain, but credentials supplied by flag, environment, or config file still appear redacted. The command exits successfully without contacting the API.

Local mutation commands emit one `{"dry_run":true,"local":true,"command":"…","message":"…"}` object in place of a preview; filter with `select(.request)` or `select(.local)`.

### Debug

Log request and response diagnostics while running normally:

```bash
cli get-duplicate-export-collision --debug
```

Debug output goes to stderr and includes:
- Request method, URL, headers, and body preview
- Response status, headers, and body preview
- Transport errors (if any)

The command still executes normally and produces its regular output on stdout.

### Flag Precedence

If both `--dry-run` and `--debug` are set, `--dry-run` takes precedence and no network calls are made.

### Security

Sensitive information is automatically redacted in diagnostic output:
- **Headers**: `Authorization`, `Cookie`, `Set-Cookie`, `X-API-Key`, and other security headers show `[REDACTED]`
- **Body**: JSON fields named `password`, `secret`, `token`, `api_key`, `client_secret`, etc. show `[REDACTED]`
- **Binary data**: binary media and canonical base64 strings are replaced with `<bytes:N>`
- **URL query**: credential-like query parameters are replaced with `[REDACTED]`

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

- **Configuration and behavior:** Use [OpenAPI overlays](https://www.speakeasy.com/docs/prep-openapi/overlays/create-overlays) in the Speakeasy workflow with `x-speakeasy-*` extensions (for example, `x-speakeasy-cli-commands`) to define commands, flags, help text, examples, authentication, and grouping.
- **Persistent code changes:** Store unified diffs as [patch files](https://www.speakeasy.com/docs/sdks/customize/code/patch-files/patch-files) at `.speakeasy/patches/<path-of-generated-file>.patch`; they are re-applied on every generation.
- **Hand-written commands:** Add them under `internal/cli/custom/`; the scaffold is generated once and never overwritten.

### CLI Created by [Speakeasy](https://www.speakeasy.com/?utm_source=openapi&utm_campaign=cli)
