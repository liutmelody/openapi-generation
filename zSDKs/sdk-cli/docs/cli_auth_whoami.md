## cli auth whoami

Display current authentication and global parameter configuration

### Synopsis

Display the currently configured settings and their sources.

Sources are shown as:
  [flag]    - Set via command line flag
  [env]     - Set via environment variable (CLI_*)
  [keyring] - Set via OS credential store: macOS Keychain, Windows Credential Manager or Secret Service (stored by login/configure command)
  [config]  - Set via config file (~/.config/cli/config.yaml)
  [unset]   - Not configured

Credential values are masked for security.

```
cli auth whoami [flags]
```

### Options

```
  -h, --help   help for whoami
```

### Options inherited from parent commands

```
      --agent-mode                       Enable structured errors and default TOON output for AI coding agents.
      --app-id string                    Custom authentication credential
      --bearer-auth string               HTTP Bearer
      --client-id string                 Client Credentials flow. client identifier
      --client-secret string             Client Credentials flow. client secret
      --color string                     Control colored output: auto (color when output is a TTY), always, or never. Respects NO_COLOR and FORCE_COLOR env vars. (default "auto")
  -d, --debug                            Log request and response diagnostics to stderr
      --deprecated-query-param1 string   A deprecated description (env: CLI_DEPRECATED_QUERY_PARAM1)
      --deprecated-query-param2 string   Global deprecated-query-param2 parameter (env: CLI_DEPRECATED_QUERY_PARAM2)
      --dry-run                          Preview API requests without sending them (no network, no OS keychain). Human preview on stderr; with -o json or --jq, one JSON object per request on stdout. Local mutation commands (auth login, auth logout and configure) make no request: they skip prompts and writes and report a no-op (stderr, or one JSON object on stdout in the machine form)
  -H, --header stringArray               Set a custom HTTP request header (format: "Key: Value"). Can be specified multiple times.
      --host-name string                 Server template variable: HostName
      --include-headers                  Include HTTP response headers in the output
      --interactive                      Prompt for missing inputs and open guided configure/auth forms (forms fall back to line prompts on stdin off-TTY)
  -q, --jq string                        Filter and transform output using a jq expression (e.g., '.name', '.items[] | .id')
      --lone-query-param string          Global lone-query-param parameter (env: CLI_LONE_QUERY_PARAM)
      --mobile-auth string               OAuth2 Password flow, a flow that is too
                                         long to describe in a single line.
      --my-api-key string                API Key
      --no-interactive                   Disable all interactive features (auto-prompting, explorer auto-launch, TUI forms)
      --no-keyring                       Never read or write the OS keychain; store secrets in the config file instead (env: CLI_NO_KEYRING)
      --no-retries                       Disable automatic retries (default: retries enabled with exponential backoff)
      --oauth2 string                    OAuth2 Authorization
  -o, --output-format string             Specify the output format. Options: pretty, json, yaml, table, toon. (default "pretty")
      --password string                  HTTP Basic password
      --port string                      Server template variable: PORT
      --query-param1 string              A long winded, multi-line description (env: CLI_QUERY_PARAM1)
      --raw-output                       Write --jq string results as raw text instead of JSON strings (like jq -r); non-string results stay JSON
      --retry-config string              Full retry config as JSON. Schema: {"strategy":"backoff","backoff":{"initialInterval":500,"maxInterval":10000,"exponent":1.5,"maxElapsedTime":30000},"retryConnectionErrors":false}. Times are in milliseconds.
      --retry-connection-errors          Retry on connection errors (EOF, reset, etc.)
      --retry-max-elapsed-time string    Maximum total time for retries (e.g., 30s, 5m). Default: 30s
      --secret string                    Custom authentication credential
      --server string                    Select a server by index (for indexed servers) or name (for named servers)
      --server-url string                Override the default server URL
      --subdomain string                 Server template variable: subdomain
      --timeout string                   HTTP request timeout (e.g., 30s, 5m, 100ms)
      --token-url string                 Client Credentials flow. token URL
      --usage                            Print the CLI Usage schema in KDL format
      --username string                  HTTP Basic username
      --version-param string             Server template variable: version
```

### SEE ALSO

* [cli auth](cli_auth.md)	 - Manage authentication credentials

Exit codes: 0 ok · 1 runtime · 2 usage · 3 authentication/authorization
