//@ts-ignore
const supportedFeatures = {
  core: "3.16.3",
  getRequestBodies: "2.81.1",
  flattening: "2.81.2",
  globalSecurity: "2.82.15",
  methodSecurity: "2.82.6",
  globalServerURLs: "2.83.1",
  methodServerURLs: "2.82.2",
  globals: "2.82.3",
  enums: "2.82.2",
  serverIDs: "2.81.1",
  nameOverrides: "2.81.4",
  includes: "2.81.1",
  docs: "0.7.5",
  examples: "2.81.7",
  groups: "2.81.3",
  deprecations: "2.81.3",
  retries: "2.84.6",
  pagination: "2.82.9",
  inputOutputModels: "2.83.0",
  ignores: "2.81.1",
  typeOverrides: "2.81.1",
  errors: "2.83.3",
  unions: "2.87.13",
  multiLevelTagging: "2.87.3",
  bigint: "0.0.2",
  decimal: "0.1.1",
  devContainers: "2.90.0",
  constsAndDefaults: "0.1.14",
  additionalProperties: "0.1.3",
  downloadStreams: "0.1.3",
  tests: "0.16.8",
  serverEvents: "0.1.11",
  serverEventsSentinels: "0.1.1",
  oauth2ClientCredentials: "1.1.6",
  oauth2Password: "0.1.3",
  webhooks: "1.0.0",
  callbacks: "1.0.0",
  responseFormat: "0.1.2",
  hiddenGlobals: "0.1.0",
  errorUnions: "0.1.0",
  acceptHeaders: "2.81.2",
  stringNumberFormats: "0.2.2",
  methodArguments: "0.2.1",
  sdkHooks: "0.3.1",
  additionalDependencies: "0.1.0",
  nullables: "0.2.2",
  globalSecurityFlattening: "0.1.0",
  globalSecurityCallbacks: "0.1.0",
  intellisenseMarkdownSupport: "0.1.0",
  openEnums: "0.1.1",
  allowReserved: "0.2.0",
  operationPolling: "0.3.2",
  operationTimeout: "0.2.0",
  defaultEnabledRetries: "0.2.0",
  deepObjectParams: "0.1.1",
  envVarSecurityUsage: "0.3.2",
  envVarGlobals: "0.2.1",
  urlBasedPagination: "0.1.1",
  uploadStreams: "0.1.0",
  customSecuritySchemes: "0.1.0",
  sliceUnions: "0.0.1",
  mockServer: "0.1.4",
  jsonlResponses: "0.2.1",
  transformJq: "0.0.1",
  modelNamespaces: "0.1.1",
  customCodeRegions: "0.0.1",
};

//@ts-ignore
const implementedReadmeSections = [
  "installation",
  "usage",
  "operations",
  "dev-containers",
  "global-parameters",
  "errors",
  "eventstream",
  "http-client",
  "pagination",
  "retries",
  "security",
  "server",
  "types",
];

type templateFeatures =
  | "documentation"
  | "module"
  | "tests"
  | "internalModules";

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
    case "envVarSecurityUsage":
      return config["EnvVarPrefix"] !== undefined;
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
  return [
    "disallowCircularRefs",
    "tagBasedOrdering",
    "sets",
    "reactQueryHooks",
  ].includes(feature);
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
  return implementedReadmeSections.includes(id);
}

//@ts-ignore
function isReadmeSectionIgnored(id: string): boolean {
  return false;
}

//@ts-ignore
function getStaticallyUsedFeatures(): string[] {
  return [
    "sdkHooks",
    "additionalDependencies",
    "globalSecurityCallbacks",
    "intellisenseMarkdownSupport",
  ];
}

//@ts-ignore
function isTestSkipped(test: string): boolean {
  return [
    "smart-union-nullable-collection-item",
    "smart-union-wrapped-complex-fields",
    "servers-select-global-server-by-name-valid-using-builder",
    "request-bodies-file-upload-extract-content-type",
    "request-bodies-get-inferred-optional-request-wrapper",
    "request-bodies-form-encoded-string-array",
    "response-bodies-empty-with-headers",
    "retries-connect-error",
    "test-hooks-before-create-request",
    "unions-nested-enums-multipart",
    "pagination-body-wrapped-request",
    "pagination-body-flattened-with-security",
    "pagination-body-flattened-optional-security",
    "pagination-cursor-response-envelope",
    "pagination-cursor-non-numeric-with-limit",
    "pagination-ambiguous-input",
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
    "pagination-params-wrapped-request",
    "cancellation-token-cancelled-before-request",
    "cancellation-token-cancelled-during-request",
    "cancellation-token-cancelled-inside-hook",
    "cancellation-token-no-cancellation",
    "customclient-request-parameters-retained-legacy",
    "request-bodies-form-encoded-string-array",
    "union-enum-array-deserialization",
    "custom-client-http-proxy",
    "open-union-smart-union-interop",
    "unions-discriminated-multiple-memberships",
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
    "errors-error-body-validation-disabled",
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
    "smart-union-nullable-union-nullable-fields",
    "pagination-cursor-nullable-results",
    "retries-binary-request-body",
    "retries-header-http-date-beyond-max-interval",
    "retries-header-rate-limit-reset",
    "errors-error-body-validation-lenient",
    "errors-response-body-validation-lenient-non-json",
    "errors-response-body-validation-lenient-union-variant",
    "event-stream-malformed-frame-lenient",
    "event-stream-malformed-frame-strict",
    "agent-mode-does-not-leak-between-invocations",
    "catalog-lists-values",
    "catalog-rejects-positional-argument",
    "catalog-structured-output",
    "cli-exit-code-bare-intent",
    "cli-exit-code-connection-and-agent-block",
    "cli-exit-code-mapping-tables",
    "cli-exit-code-reason-first",
    "cli-exit-code-root-unknown-message",
    "cli-exit-code-single-envelope",
    "cli-exit-code-success-and-discovery",
    "cli-exit-codes-rendering-modes",
    "client-credentials-dry-run-token-and-api",
    "clierrors-agent-mode-explicitly-disabled",
    "clierrors-agent-mode-from-environment",
    "clierrors-agent-mode-matrix",
    "clierrors-api-error-not-double-wrapped",
    "clierrors-declared-hints-fire",
    "clierrors-flag-values-not-rendering-flags",
    "clierrors-invalid-values-do-not-stop-preparse",
    "clierrors-json-output-envelope",
    "clierrors-plain-mode-classified",
    "clierrors-typed-reasons",
    "error-taxonomy-declared-rules-and-ordering",
    "error-taxonomy-every-closed-type-has-a-hint",
    "error-taxonomy-fallback-message-omits-raw-body",
    "error-taxonomy-hint-precedence-and-dedupe",
    "error-taxonomy-jq-uses-structured-envelope",
    "error-taxonomy-odd-bodies-use-fallback-envelope",
    "error-taxonomy-odd-reflection-fields-do-not-panic",
    "error-taxonomy-origin-semantics",
    "error-taxonomy-pretty-http-status-without-reason",
    "error-taxonomy-pretty-json-agent-parity",
    "error-taxonomy-pretty-residual-details",
    "error-taxonomy-reserved-keys-and-rendered-classification",
    "event-stream-connection-refused-stays-connection-error",
    "event-stream-decode-failure-is-protocol-error",
    "event-stream-decode-failure-without-json-error-type",
    "event-stream-error-event-agent-mode",
    "event-stream-error-event-agent-mode-fallback",
    "event-stream-error-event-exits-non-zero",
    "event-stream-error-event-pretty",
    "event-stream-numbers-keep-precision",
    "event-stream-server-error-mentioning-decode-is-not-protocol-error",
    "event-stream-undiscriminated-error-shaped-variant-is-data",
    "event-stream-union-events-render-wire-shape",
    "intent-dispatch-resolves-stdin-once",
    "intent-dispatch-select-matrix",
    "intent-interactive-piped-body-skips-prompts",
    "intent-route-dispatch-agent-envelope",
    "intent-route-dispatch-bare-help",
    "intent-route-dispatch-body-evidence",
    "intent-route-dispatch-conflicting-evidence",
    "intent-route-dispatch-default-required",
    "intent-route-dispatch-flag-paths",
    "intent-route-dispatch-help-usage",
    "intent-route-dispatch-interactive-plan",
    "intent-route-dispatch-override-once",
    "intents-discriminated-partial-body",
    "intents-foreign-selector-agent-envelope",
    "intents-foreign-selector-usage-error",
    "intents-non-object-body-falls-through",
    "intents-partial-body-keeps-presets",
    "intents-partial-body-user-keys-win",
    "intents-preset-from-args",
    "intents-preset-merge-matrix",
    "intents-undeclared-nested-group-order",
    "pre-request-connection-classifier-typed-table",
    "pre-request-discriminated-root-union-messages",
    "pre-request-dry-run-unresolvable-host",
    "pre-request-generated-required-union-paths-and-optionality",
    "pre-request-generated-required-union-positive-echoes",
    "pre-request-real-connection-failures",
    "pre-request-silent-stdin-with-flag-input-reports-missing-root-union-body",
    "pre-request-validation-envelope-matrix",
    "request-bodies-complex-number-types-bigint-overflow",
    "request-shape-body-agent-variant",
    "request-shape-body-and-whole-body-flag-rejected",
    "request-shape-body-both-selectors-rejected",
    "request-shape-body-default-selector-merges",
    "request-shape-body-null-optional-key-allowed",
    "request-shape-body-typo-key-rejected",
    "request-shape-expanded-body-typo-key-lenient-warning",
    "request-shape-expanded-body-typo-key-rejected",
    "request-shape-intent-bare-usage",
    "request-shape-intent-empty-stdin-usage",
    "request-shape-intent-flag-and-body-conflict",
    "request-shape-intent-flag-merged-into-body",
    "request-shape-intent-positional-and-body-conflict",
    "request-shape-intent-positional-merged-into-body",
    "request-shape-intent-positional-merged-into-whole-body-flag",
    "request-shape-intent-stdin-runs-request",
    "request-shape-intent-stdin-typo-key-rejected",
    "request-shape-intent-stdin-with-positional",
    "request-shape-intent-synthesized-body",
    "request-shape-intent-zero-value-flag-merged",
    "request-shape-nested-union-agent-selector-no-default",
    "request-shape-nested-union-body-default-selector-merges",
    "request-shape-nested-union-own-flag-default-selector-merges",
    "request-shape-nested-union-stdin-default-selector-merges",
    "request-shape-open-schema-keeps-unknown-keys",
    "request-shape-stdin-typo-key-rejected",
    "request-shape-whole-body-flag-typo-key-rejected",
    "request-shape-whole-body-flag-wins-over-stdin",
    "security-ranking-both-flags",
    "security-ranking-env-only",
    "security-ranking-env-tie",
    "security-ranking-flag-beats-env",
    "security-ranking-operation-restriction",
    "security-ranking-pick-matrix",
    "security-ranking-source-rank",
    "open-union-strict-request-validation",
    "parameters-path-parameter-format-union",
    "parameters-path-parameter-formats",
    "pagination-object-results-page-params",
    "clierrors-unknown-flag-suggestion",
    "shared-enum-constructor-default",
    "cli-forbidden-without-credentials",
    "cli-exit-codes-invalid-jq-before-request",
    "clierrors-dry-run-hint-server-validation-only",
    "clierrors-unknown-command-suggestion",
    "event-stream-absent-required-union",
    "event-stream-equivalent-number-lexemes",
    "event-stream-null-and-undeclared-fields",
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
function isTemplateFeatureEnabled(feature: templateFeatures): boolean {
  const enabledFeatures = context.Global.EnabledTemplateFeatures;

  if (enabledFeatures && feature in enabledFeatures) {
    return enabledFeatures[feature] === true;
  }

  return templateFeaturesDefaults[feature] === true;
}
typeof registerTemplateFunc === "function" &&
  registerTemplateFunc("isTemplateFeatureEnabled", isTemplateFeatureEnabled);

// @ts-ignore
function collectGlobalsAndServersModels(): boolean {
  return true;
}
