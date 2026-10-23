//@ts-ignore
// TODO: determine if this needs to be insync with the go version or can define these differently
const supportedFeatures = {
  core: "0.9.11",
  allowReserved: "0.2.0",
  getRequestBodies: "0.0.1",
  flattening: "0.0.0",
  globalSecurity: "0.1.0",
  methodSecurity: "0.0.0",
  globalServerURLs: "0.1.0",
  methodServerURLs: "0.0.0",
  globals: "0.1.1",
  enums: "0.0.2",
  openEnums: "0.0.1",
  serverIDs: "0.0.0",
  nameOverrides: "0.0.0",
  includes: "0.0.0",
  docs: "0.3.2",
  examples: "0.1.1",
  groups: "0.1.0",
  deprecations: "0.0.0",
  retries: "0.1.1",
  pagination: "0.2.0",
  inputOutputModels: "0.0.0",
  ignores: "0.0.0",
  typeOverrides: "0.0.0",
  errors: "0.2.1",
  errorUnions: "0.0.0",
  unions: "0.1.3",
  sliceUnions: "0.0.0",
  multiLevelTagging: "0.0.0",
  bigint: "0.0.0",
  decimal: "0.0.0",
  devContainers: "0.0.0",
  constsAndDefaults: "0.0.0",
  additionalProperties: "0.0.1",
  deepObjectParams: "0.0.0",
  downloadStreams: "0.0.2",
  serverEvents: "0.1.4",
  oauth2ClientCredentials: "0.0.1",
  webhooks: "0.0.0",
  callbacks: "0.0.0",
  responseFormat: "0.1.2",
  sdkHooks: "0.0.1",
  customSecuritySchemes: "0.0.0",
  oauth2Password: "0.0.1",
  tests: "0.0.0",
  mockServer: "0.0.1",
  transformJq: "0.0.0",
  jsonlResponses: "0.0.1",
  customCodeRegions: "0.0.1",
  cliCommands: "0.2.0",
};

// @ts-ignore
type templateFeatures =
  | "documentation"
  | "module"
  | "tests"
  | "internalModules";

// @ts-ignore
const templateFeaturesDefaults = {
  documentation: true,
  module: true,
  tests: true,
  internalModules: true,
};

//@ts-ignore
function isFeatureSupported(feature: string): boolean {
  return feature in supportedFeatures;
}

//@ts-ignore
function isFeatureEnabled(
  feature: string,
  config: Record<string, any>,
): boolean {
  switch (feature) {
    default:
      return false;
  }
}

//@ts-ignore
function allSupportedFeatures(): string[] {
  const allFeatures = Object.keys(supportedFeatures);
  return allFeatures.filter((feature) => !isFeatureIgnored(feature));
}

//@ts-ignore
function isFeatureIgnored(feature: string): boolean {
  return false;
}

//@ts-ignore
function getFeatureVersion(feature: string): string {
  return supportedFeatures[feature];
}

//@ts-ignore
function isUnionWrapper(): boolean {
  return true;
}

//@ts-ignore
function isReadmeSectionImplemented(id: string): boolean {
  const implemented = [
    "summary",
    "toc",
    "installation",
    "usage",
    "operations",
    "security",
    "server",
    "errors",
    "global-parameters",
    "pagination",
    "dev-containers",
  ];
  return implemented.includes(id);
}

//@ts-ignore
function isReadmeSectionIgnored(id: string): boolean {
  return false;
}

//@ts-ignore
function getStaticallyUsedFeatures(): string[] {
  return [];
}

//@ts-ignore
function isTestSkipped(test: string): boolean {
  return [
    // Below tests are skipped in Go so ignoring for now
    //-----------------------------------------------------------------------------------------

    "servers-select-global-server-by-name-valid-using-builder",
    "request-bodies-file-upload-extract-content-type",
    "response-bodies-empty-with-headers",
    "test-hooks-before-create-request",
    "unions-nested-enums-form",
    "unions-nested-enums-multipart",
    "pagination-body-wrapped-request",
    "pagination-body-flattened-with-security",
    "pagination-body-flattened-optional-security",
    "pagination-cursor-response-envelope",
    "pagination-cursor-non-numeric-with-limit",
    "pagination-ambiguous-input",
    "pagination-wrapped-optional-body",
    "pagination-encapsulated-parameter",
    "pagination-limit-offset-default-offset-params",
    "flattening-component-body-and-param-no-conflict-flatten-requests",
    "flattening-component-body-and-param-conflict-flatten-requests",
    "flattening-inline-body-and-param-conflict-flatten-requests",
    "flattening-inline-body-and-param-no-conflict-flatten-requests",
    "flattening-conflicting-params-flatten-requests",
    "flattening-required-body-all-optional-flatten-requests",
    "hooks-access-retry-config",
    "hooks-terminate-retry-loop",
    "hooks-trigger-retries",
    "hooks-client-credentials-basic-success",
    "hooks-client-credentials-basic-success-alt-token-url",
    "hooks-client-credentials-basic-success-global-server",
    "webhooks-consume",
    "webhooks-consume-custom-security",
    "webhooks-consume-bad-data",
    "webhooks-consume-bad-signature",
    "jsonl-stream-data-async-envelope-http-responses",
    "jsonl-stream-data-async-chunks-flat-response",
    "jsonl-stream-data-async-flat-response",
    "errors-additional-properties",
    "x-ndjson-stream-data-async-envelope-http-responses",
    "x-ndjson-stream-data-async-chunks-envelope-http-responses",
    "servers-select-global-server-by-id-default-using-builder",
    "servers-select-global-server-by-name-broken-using-builder",
    "servers-select-global-server-by-name-default-using-builder",
    "servers-select-global-server-by-name-with-templates-broken-using-builder",
    "servers-select-global-server-by-name-with-templates-defaults-using-builder",
    "servers-select-global-server-by-name-with-templates-valid-using-builder",
    "servers-select-global-server-by-id-valid-using-builder",
    "servers-select-global-server-by-id-invalid-using-builder",
    "servers-select-global-server-by-id-broken-using-builder",
    "custom-code-region-model-method-with-imports",
    "parameters-open-enum",
    "parameters-ordering-body-first",
    "parameters-ordering-parameters-first",
    "parameters-ordering-with-legacy-flattening-order",
    "parameters-ordering-using-optional-request-body-body-first",
    "parameters-ordering-using-optional-request-body-parameters-first",
    "parameters-ordering-using-optional-request-body-with-legacy-flattening-order",
    "pagination-cursor-params-snake",
    "pagination-cursor-snake",
    "pagination-limit-offset-params-snake",
    "pagination-limit-offset-snake",
    "pagination-simple-object-snake",
    "cancellation-token-cancelled-before-request",
    "cancellation-token-cancelled-during-request",
    "cancellation-token-cancelled-inside-hook",
    "cancellation-token-no-cancellation",
    "customclient-request-parameters-retained-legacy",
    "union-enum-array-deserialization",
    "pagination-params-wrapped-request",

    // Not applicable to CLI (SDK-internal behavior)
    //-----------------------------------------------------------------------------------------

    // SDK hooks — scope constants test, not applicable to CLI (tests Go type assertions)
    "hooks-available-oauth2-scopes",
    // SDK hooks — require RequestRecorderClient (Go-internal, not applicable to CLI)
    "hooks-client-credentials-option-success",
    "hooks-client-credentials-option-success-alt-token-url",
    // Auth: operation-level OAuth2 token reuse — requires request recorder (SDK internal behavior)
    "auth-operation-level-oauth2",

    // Errors: errorUnions not applicable to CLI
    "errors-union-of-errors",
    "errors-union-of-errors-discriminated",

    // Union enum deserialization — serialization-only tests, no HTTP round-trip
    "union-enum-nested-in-array-in-union",

    // Custom HTTP client — not applicable to CLI but in the future we probably want to test things like the CLI respects proxies etc
    "customclient-request-parameters-retained",

    // Request bodies: tests streaming vs buffered upload — both map to the same CLI command
    // (request-body-put-multipart-file --file), so there's no way to distinguish them from CLI flags.
    // Already covered by request-bodies-put-multipart-file.
    "request-bodies-put-multipart-file-streaming",

    // SDK hooks — oauth2-password: token-renewal + operation-scope require persistent state across multiple CLI invocations
    "hooks-oauth2-password-token-renewal",
    "hooks-oauth2-password-operation-scope",

    // SSE: break-early tests SDK-internal Go iterator break keyword behavior, no CLI equivalent
    "event-stream-stay-open-break-early",
    // SSE: abort-signal tests context cancellation mid-stream, not testable in CLI harness
    "event-stream-with-abort-signal",
    // JSONL flat-response variants: CLI always uses envelope-http format
    "jsonl-stream-data-flat-responses",
    "jsonl-stream-data-chunks-flat-response",

    // Headers: flat response format tests — CLI always uses envelope-http, not flat
    "headers-empty-response-body-with-headers-flat",
    "headers-response-body-with-headers-flat",

    // Applicable to CLI but currently unsupported / deferred
    //-----------------------------------------------------------------------------------------

    // Custom code regions — SDK-internal feature, CLI custom code is different (Cobra subcommands)
    "custom-code-region-sdk-method-with-imports",
    "custom-code-region-sub-sdk-method-with-imports",

    // Custom HTTP client proxy — skipped in most targets, not yet supported in CLI
    "custom-client-http-proxy",

    // Open unions — partially supported; tests below require non-CLI mechanisms
    "open-union-embedded", // needs array-of-Vehicle endpoint (none in spec)
    "open-union-invalid-payload", // can't send string/number/null as union body via CLI flags
    "open-union-smart-union-interop", // also skipped in Go

    // SSE: union with standalone comments — recently added
    "event-stream-union-with-standalone-comments",
    "timeout-ms-override-allows-completion",
    "timeout-ms-override-is-respected",
    "request-bodies-complex-number-types-optional-unpopulated",
    "request-bodies-complex-number-types-required-missing",
    "flattening-nullable-body-with-required-param-ordering",
    "retries-timeout-fresh-signal",
    "auth-global-security-flattening-env-var-fallback",
    "request-bodies-base64-input-mode-file-shared-component",
    "request-bodies-base64-input-mode-file-split-components",
    "event-stream-chat-json",
    "unions-strongly-typed-one-of-discriminated-post",
    "event-stream-sse-overload-json-response",
    "event-stream-sse-overload-streaming-response",
    "raw-response-helpers-strip-internal-header",
    "unions-typed-object-one-of-post-body-less-throws",
    "request-bodies-put-multipart-file-bytesio",
    "request-bodies-put-multipart-file-handle",
    "request-bodies-put-multipart-file-stringio",
    "request-extras-extra-body-merged",
    "request-extras-extra-body-overrides",
    "request-extras-extra-body-rejects-non-json",
    "request-extras-extra-query-appended",
    "request-extras-extra-query-merged",
    "request-extras-extra-query-nullish-arrays",
    "request-extras-extra-query-object-json",
    "request-extras-extra-query-overrides",
    "request-extras-extra-query-preserves-server-url-query",
    "request-extras-extra-query-security-overrides",
    "collections-map-input-accepts-non-dict-mapping",
    "collections-parameter-annotations-advertise-iterables",
    "collections-parameters-accept-iterables",
    "collections-parameters-accept-iterables-direct",
    "pagination-cursor-deep-nested-outputs",
    "pagination-cursor-deep-nested-outputs-iterator",
    "event-stream-split-boundaries",
    "request-bodies-base64-file-input-idempotent",
    "errors-error-body-validation-disabled",
    "errors-error-body-validation-enabled",
    "errors-response-body-validation-disabled",
    "errors-custom-error-inheritance",
    "errors-response-body-validation-enabled",
    "errors-schema-validation-disabled",
    "errors-schema-validation-enabled",
    "errors-optional-nullable-error-message",
    "errors-optional-nullable-error-message-nested",
    "retries-status-codes-override",
    "retries-status-codes-override-global",
    "retries-status-codes-override-default",
    "errors-response-body-validation-lenient",
    "hooks-access-operation-metadata",
    "request-bodies-duration-format",
    "request-bodies-uuid-format",
    "react-query-builders-key-distinct-bodies",
    "react-query-key-includes-parameters-and-request-body",
    "react-query-key-includes-request-body",
    "react-query-key-opt-in-infers-query-type",
    "object-with-optional-true-nullable-false-field-null",
    "object-with-optional-false-nullable-false-field-null",
    "smart-union-nested-union-vs-nested-union",
    "request-bodies-put-multipart-optional-nullable",
    "request-bodies-post-form-optional-nullable",
    "request-bodies-post-json-optional-nullable",
    "smart-union-nullable-union-nullable-fields",
    "request-bodies-post-form-optional-nullable-json-shared",
    "request-bodies-put-multipart-optional-nullable-json-shared",
    "request-bodies-post-json-optional-nullable-body-and-param",
    "pagination-limit-offset-page-body-nullable",
    "pagination-limit-offset-page-body-optional-nullable",
    "smart-union-nullable-collection-item",
    "smart-union-wrapped-complex-fields",
    "smart-union-wrapped-complex-fields",
    "pagination-cursor-nullable-results",
    "retries-binary-request-body",
    "retries-header-http-date-beyond-max-interval",
    "retries-header-rate-limit-reset",
    "errors-error-body-validation-lenient",
    "errors-response-body-validation-lenient-non-json",
    "errors-response-body-validation-lenient-union-variant",
    "event-stream-malformed-frame-lenient",
    "event-stream-malformed-frame-strict",
    "optional-nullable-array-params-null-and-unset",
    "optional-nullable-array-params-set",
    "optional-nullable-deep-object-param",
    "optional-nullable-special-header-params",
    "optional-nullable-urlencoded-body",
    "request-bodies-complex-number-types-bigint-overflow",
    "event-stream-with-operation-timeout-streams-to-completion",
    "jsonl-stream-timeout-still-bounds-slow-stream",
    "jsonl-stream-with-timeout-streams-to-completion",
    "open-union-strict-request-validation",
    "optional-nullable-object-property-params",
    "parameters-path-parameter-format-union",
    "parameters-path-parameter-formats",
    "shared-enum-constructor-default",
    "open-union-invalid-json-is-an-error",
    "unknown-fields-json-round-trip",
  ].includes(test);
}

//@ts-ignore
function getAutoGeneratedHeader(
  generatedId: string,
  config: Record<string, any>,
): string {
  let header =
    "// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.\n";
  if (generatedId) {
    header += `// @generated-id: ${generatedId}\n`;
  }
  if (config.GeneratedLicense === "agpl") {
    header +=
      "// Generated under the AGPL-3.0-only license.\n" +
      "// SPDX-License-Identifier: AGPL-3.0-only\n";
  }
  header += "\n";
  return header;
}

// @ts-ignore
function collectGlobalsAndServersModels(): boolean {
  return true;
}

// @ts-ignore
function isTemplateFeatureEnabled(feature: templateFeatures): boolean {
  const enabledFeatures = context.Global.EnabledTemplateFeatures;

  if (enabledFeatures && feature in enabledFeatures) {
    return enabledFeatures[feature] === true;
  }

  return templateFeaturesDefaults[feature] === true;
}
typeof registerTemplateFunc === "function" &&
  registerTemplateFunc("isTemplateFeatureEnabled", isTemplateFeatureEnabled);
