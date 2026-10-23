/**
 * CLI Security configuration helpers.
 * Handles global security flags, config file loading, and environment variables.
 *
 * Uses security annotations from the OpenAPI spec to determine:
 * - SecType: apiKey, http, oauth2
 * - SubType: basic, bearer, custom, client_credentials
 * - Whether fields are secrets (based on security type)
 */

/**
 * Security field information extracted from the SDK's Security type.
 */
interface CLISecurityFieldInfo {
  field: FieldDef; // Original field definition
  flagName: string; // kebab-case flag name
  envVarSuffix: string; // UPPER_SNAKE_CASE env var suffix
  configKey: string; // snake_case config key
  description: string; // Help text for the flag
  isSecret: boolean; // Whether to use secure input
  isArray: boolean; // Whether this is a string array field (e.g., Scopes)
  secType: string; // apiKey, http, oauth2, etc.
  secSubType: string; // basic, bearer, custom, etc.
  composite: boolean; // Part of a requirement that needs several schemes
  schemeKey: string; // Security scheme name the field belongs to
}

/**
 * Flatten the Security type to extract all primitive security fields.
 * Handles the Option structure and nested class types.
 * Uses security annotations to determine field properties.
 */
function flattenCLISecurityObject(
  typeDef: TypeDef,
  parentIsOptional: boolean = false,
): CLISecurityFieldInfo[] {
  const results: CLISecurityFieldInfo[] = [];
  if (!typeDef || !typeDef.Fields || !typeDef.Fields.length) {
    return [];
  }

  const seen = new Set<string>(); // Dedupe by flag name

  for (const field of typeDef.Fields) {
    if (field.Const) {
      continue;
    }

    // Get the security annotation
    const secAnno = field.Annotations?.Get("security") as
      | SecurityAnnotation
      | undefined;

    // If this is a class type, recurse into it to get the actual fields
    if (field.Type.Type.toString() === "class") {
      const subResults = flattenCLISecurityObject(
        field.Type,
        parentIsOptional || field.Optional,
      );
      for (const subResult of subResults) {
        if (!seen.has(subResult.flagName)) {
          seen.add(subResult.flagName);
          results.push(subResult);
        }
      }
      continue;
    }

    // If this is a union type (e.g., Oauth2Input = Credentials | Token),
    // flatten all variants by recursing into each AssociatedType.
    if (field.Type.Type.toString() === "union") {
      const associatedTypes = field.Type.AssociatedTypes || [];
      for (const assocType of associatedTypes) {
        if (assocType.Type.toString() === "class") {
          // Class variant (e.g., Oauth2Credentials) — recurse into its fields
          const subResults = flattenCLISecurityObject(
            assocType,
            parentIsOptional || field.Optional,
          );
          for (const subResult of subResults) {
            if (!seen.has(subResult.flagName)) {
              seen.add(subResult.flagName);
              results.push(subResult);
            }
          }
        } else if (assocType.Type.toString() === "string") {
          // String variant (e.g., Oauth2Token) — create field info from the TypeDef
          const flagName = sanitizeFlagNameWithReserved(assocType.Name);
          if (!seen.has(flagName)) {
            seen.add(flagName);
            const description =
              assocType.Comments?.Description?.trim() ||
              getDefaultSecurityDescription(assocType.Name, secAnno);
            results.push({
              field: {
                Name: assocType.Name,
                Type: assocType,
                Optional: true,
              } as any,
              flagName: flagName,
              envVarSuffix: toEnvVarSuffix(assocType.Name),
              configKey: toConfigKey(assocType.Name),
              description: description.trim(),
              isSecret: false,
              isArray: false,
              secType: secAnno?.SecType || "",
              secSubType: secAnno?.SubType || "",
              composite: !!secAnno?.Composite,
              schemeKey: secAnno?.SchemeKey || "",
            });
          }
        }
      }
      continue;
    }

    // Collect string and string-array fields
    const fieldTypeStr = field.Type.Type.toString();
    if (fieldTypeStr !== "string" && fieldTypeStr !== "array") {
      continue;
    }

    // For arrays, only support string element types
    const isArray = fieldTypeStr === "array";

    // This is a primitive security field (string or []string)
    const flagName = sanitizeFlagNameWithReserved(field.Name);
    if (seen.has(flagName)) {
      continue;
    }
    seen.add(flagName);

    // Get description from field comments or type comments
    let description = "";
    if (field.Comments?.Description) {
      description = field.Comments.Description;
    } else if (field.Type.Comments?.Description) {
      description = field.Type.Comments.Description;
    } else {
      // Generate default description based on security type
      description = getDefaultSecurityDescription(field.Name, secAnno);
    }

    const isSecret = !isArray && isSecuritySecret(secAnno);

    results.push({
      field: {
        ...field,
        Optional: parentIsOptional || field.Optional,
      },
      flagName: flagName,
      envVarSuffix: toEnvVarSuffix(field.Name),
      configKey: toConfigKey(field.Name),
      description: description.trim(),
      isSecret: isSecret,
      isArray: isArray,
      secType: secAnno?.SecType || "",
      secSubType: secAnno?.SubType || "",
      composite: !!secAnno?.Composite,
      schemeKey: secAnno?.SchemeKey || "",
    });
  }

  return results;
}
registerTemplateFunc("flattenCLISecurityObject", flattenCLISecurityObject);

/**
 * Convert a field name to UPPER_SNAKE_CASE for env var suffix.
 */
function toEnvVarSuffix(name: string): string {
  return name
    .replace(/([a-z])([A-Z])/g, "$1_$2")
    .replace(/[^a-zA-Z0-9]/g, "_")
    .toUpperCase();
}

/**
 * Convert a field name to snake_case for config key.
 */
function toConfigKey(name: string): string {
  return name
    .replace(/([a-z])([A-Z])/g, "$1_$2")
    .replace(/[^a-zA-Z0-9]/g, "_")
    .toLowerCase();
}

/**
 * Get default description based on security annotation type.
 */
function getDefaultSecurityDescription(
  fieldName: string,
  secAnno: SecurityAnnotation | undefined,
): string {
  if (!secAnno) {
    return `${caser()
      .ToPascal(fieldName)
      .replace(/([a-z])([A-Z])/g, "$1 $2")} for authentication`;
  }

  switch (secAnno.SecType) {
    case "apiKey":
      return "API key for authentication";
    case "http":
      switch (secAnno.SubType) {
        case "bearer":
          return "Bearer token for authentication";
        case "basic":
          // Check if it's username or password field
          if (fieldName.toLowerCase().includes("password")) {
            return "Password for HTTP basic authentication";
          }
          if (fieldName.toLowerCase().includes("username")) {
            return "Username for HTTP basic authentication";
          }
          return "HTTP authentication credential";
        case "custom":
          return "Custom authentication credential";
        default:
          return "HTTP authentication credential";
      }
    case "oauth2":
      switch (secAnno.SubType) {
        case "client_credentials":
          if (fieldName.toLowerCase().includes("secret")) {
            return "OAuth2 client secret";
          }
          if (fieldName.toLowerCase().includes("id")) {
            return "OAuth2 client ID";
          }
          return "OAuth2 credential";
        default:
          return "OAuth2 token";
      }
    case "openIdConnect":
      return "OpenID Connect token";
    default:
      return `${caser()
        .ToPascal(fieldName)
        .replace(/([a-z])([A-Z])/g, "$1 $2")} for authentication`;
  }
}

/**
 * Determine if a security field should be treated as a secret based on its annotation.
 */
function isSecuritySecret(secAnno: SecurityAnnotation | undefined): boolean {
  if (!secAnno) {
    return false;
  }
  if (
    secAnno.FieldName === "username" &&
    ((secAnno.SecType === "http" && secAnno.SubType === "basic") ||
      (secAnno.SecType === "oauth2" && secAnno.SubType === "password"))
  ) {
    return false;
  }

  switch (secAnno.SecType) {
    case "apiKey":
      return true; // API keys are always secrets
    case "http":
      switch (secAnno.SubType) {
        case "bearer":
          return true; // Bearer tokens are secrets
        case "basic":
          return true;
        case "custom":
          return true;
        default:
          return true;
      }
    case "oauth2":
      return true; // OAuth tokens and secrets are always secrets
    case "openIdConnect":
      return true;
    default:
      return true; // Default to secret for safety
  }
}

/**
 * Check if the SDK has global security configuration.
 */
function hasGlobalSecurity(): boolean {
  return context.Global.AST.MainSDK.Security != null;
}
registerTemplateFunc("hasGlobalSecurity", hasGlobalSecurity);

function hasSecretCLIFields(): boolean {
  return getCLISecurityFields().some((f) => f.isSecret);
}
registerTemplateFunc("hasSecretCLIFields", hasSecretCLIFields);

/**
 * Get all global security fields for the SDK.
 */
function getCLISecurityFields(): CLISecurityFieldInfo[] {
  if (!context.Global.AST.MainSDK.Security) {
    return [];
  }
  return flattenCLISecurityObject(context.Global.AST.MainSDK.Security.Type);
}
registerTemplateFunc("getCLISecurityFields", getCLISecurityFields);

/**
 * Wire header names of every apiKey security scheme delivered via an HTTP
 * header, across global and operation-level security. These are the API's own
 * credential headers, injected into the diagnostics redaction list so dry-run
 * and debug output never print live credentials. Names are lowercased and
 * deduped; entries already covered verbatim by the static vendor-neutral list
 * in diagnostics.go.stmpl are omitted.
 */
function templateSensitiveSecurityHeaders(): string[] {
  // Mirrors the static entries in diagnostics.go.stmpl sensitiveHeaderKeys /
  // sensitiveHeaderSuffixes; keep in sync. A stale mirror is safe: worst case
  // a redundant (or duplicate-covered) entry is emitted.
  const staticKeys = new Set([
    "authorization",
    "proxy-authorization",
    "x-api-key",
    "api-key",
    "x-session-token",
    "cookie",
    "set-cookie",
  ]);
  const staticSuffixes = ["-secret", "-token"];
  const names = new Set<string>();

  const collect = (fields: FieldDef[] | undefined): void => {
    for (const field of fields || []) {
      if (field.Const) continue;
      const typeStr = field.Type?.Type.toString() ?? "";
      if (typeStr === "class") {
        collect(field.Type.Fields || []);
        continue;
      }
      if (typeStr === "union") {
        for (const assocType of field.Type.AssociatedTypes || []) {
          if (assocType.Type.toString() === "class") {
            collect(assocType.Fields || []);
          }
        }
        continue;
      }
      const secAnno = field.Annotations?.Get("security") as
        | SecurityAnnotation
        | undefined;
      if (
        secAnno?.SecType === "apiKey" &&
        secAnno.SubType === "header" &&
        secAnno.FieldName
      ) {
        const lower = secAnno.FieldName.toLowerCase();
        if (staticKeys.has(lower)) continue;
        if (staticSuffixes.some((s) => lower.endsWith(s))) continue;
        names.add(lower);
      }
    }
  };

  collect(context.Global.AST.MainSDK.Security?.Type?.Fields);
  for (const op of allOperations()) {
    collect(op.Security?.Type?.Fields);
  }
  return [...names].sort();
}
registerTemplateFunc(
  "templateSensitiveSecurityHeaders",
  templateSensitiveSecurityHeaders,
);

function primaryCLISecurityField(
  fields: CLISecurityFieldInfo[],
): CLISecurityFieldInfo | undefined {
  const schemeGroups = new Set(
    fields.map((f) => `${f.secType}\u0000${f.secSubType}`),
  );
  if (schemeGroups.size > 1) {
    return (
      fields.find((f) => f.secSubType === "apiKey" || f.secType === "apiKey") ||
      fields.find((f) => f.isSecret) ||
      fields[0]
    );
  }
  return fields.find((f) => f.isSecret) || fields[0];
}

function cliEnvVarPrefix(): string {
  return context.Global.Config.EnvVarPrefix
    ? String(context.Global.Config.EnvVarPrefix).toUpperCase()
    : "";
}

function prefixedEnvVarName(suffix: string): string {
  const prefix = cliEnvVarPrefix();
  return prefix ? `${prefix}_${suffix}` : suffix;
}

function noKeyringEnvVar(): string {
  return prefixedEnvVarName("NO_KEYRING");
}
registerTemplateFunc("noKeyringEnvVar", noKeyringEnvVar);

function credentialEnvVarName(field: CLISecurityFieldInfo): string {
  return prefixedEnvVarName(field.envVarSuffix);
}

// Concrete credential variable for compact root setup guidance. Empty means
// the document has no credential field and the Setup line must be omitted.
function templatePrimaryAuthEnvVar(): string {
  const primary = primaryCLISecurityField(getCLISecurityFields());
  return primary ? credentialEnvVarName(primary) : "";
}
registerTemplateFunc("templatePrimaryAuthEnvVar", templatePrimaryAuthEnvVar);

// Go string literals of every security field's env var, secret or not, so a
// test can clear the whole credential environment.
function templateSecurityFieldEnvVars(): string {
  return getCLISecurityFields()
    .map((f) => goStringLiteral(credentialEnvVarName(f)))
    .join(", ");
}
registerTemplateFunc(
  "templateSecurityFieldEnvVars",
  templateSecurityFieldEnvVars,
);

/**
 * One-line credentials hint for agent-mode authentication errors, derived
 * from the API's actual security fields. Single-scheme APIs name the primary
 * environment variable and every credential flag; multi-scheme APIs name the
 * primary credential as an actionable starting point, then point to the
 * complete environment/--help surfaces for the remaining alternatives.
 */
function templateAuthErrorHint(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) {
    return "Set credentials via environment variables or CLI flags";
  }

  const schemeGroups = new Set(
    fields.map((f) => `${f.secType}\u0000${f.secSubType}`),
  );
  const primary = primaryCLISecurityField(fields)!;
  const primaryEnv = credentialEnvVarName(primary);
  if (schemeGroups.size > 1) {
    // Even with several schemes, steer to the primary credential concretely —
    // agents act on a named variable, not a category. Prefer an API key.
    const envVarPrefix = cliEnvVarPrefix();
    const environment = envVarPrefix
      ? `${envVarPrefix}_* environment variables`
      : "environment variables";
    return `Set ${primaryEnv} (or another credential via the ${environment} / the credential flags listed in --help)`;
  }

  const flagNames = fields.map((f) => `--${f.flagName}`).join(" / ");
  return `Set ${primaryEnv} (or use ${flagNames})`;
}
registerTemplateFunc("templateAuthErrorHint", templateAuthErrorHint);

/**
 * Generate global security flag registration code for root.go.
 * These flags are persistent (inherited by all subcommands).
 */
function templateGlobalSecurityFlagRegistration(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) {
    return "";
  }

  const lines: string[] = [];
  lines.push("// Global security flags");

  for (const field of fields) {
    if (field.isArray) {
      lines.push(
        `rootCmd.PersistentFlags().StringSlice("${
          field.flagName
        }", nil, "${escapeGoString(field.description)}")`,
      );
    } else {
      lines.push(
        `rootCmd.PersistentFlags().String("${
          field.flagName
        }", "", "${escapeGoString(field.description)}")`,
      );
    }
    lines.push(
      `_ = rootCmd.PersistentFlags().SetAnnotation("${field.flagName}", "speakeasy:group", []string{"Authentication"})`,
    );
  }

  lines.push(
    `rootCmd.PersistentFlags().Bool("no-keyring", false, "Never read or write the OS keychain; store secrets in the config file instead (env: ${noKeyringEnvVar()})")`,
    `_ = rootCmd.PersistentFlags().SetAnnotation("no-keyring", "speakeasy:group", []string{"Authentication"})`,
  );

  return lines.join("\n    ");
}
registerTemplateFunc(
  "templateGlobalSecurityFlagRegistration",
  templateGlobalSecurityFlagRegistration,
);

/**
 * Generate nested config struct type definitions (SecurityConfig, GlobalsConfig).
 * These are emitted before the main Config struct in config.go.
 */
function templateConfigNestedTypes(): string {
  const lines: string[] = [];

  // SecurityConfig struct (only if security fields exist)
  const secFields = getCLISecurityFields();
  if (secFields.length > 0) {
    lines.push("// SecurityConfig holds authentication credentials.");
    lines.push("type SecurityConfig struct {");
    for (const field of secFields) {
      const fieldName = caser().ToPascal(field.field.Name);
      const yamlTag = field.configKey;
      const goType = field.isArray ? "[]string" : "string";
      lines.push(`\t${fieldName} ${goType} \`yaml:"${yamlTag},omitempty"\``);
    }
    lines.push("}");
    lines.push("");
  }

  // GlobalsConfig struct (only if globals exist)
  if (hasGlobals()) {
    const globals = context.Global.AST.MainSDK.Globals;
    lines.push("// GlobalsConfig holds global parameter values.");
    lines.push("type GlobalsConfig struct {");
    for (const field of globals.Fields) {
      const fieldName = caser().ToPascal(field.Name);
      const yamlTag = toConfigKey(field.Name);
      lines.push(`\t${fieldName} string \`yaml:"${yamlTag},omitempty"\``);
    }
    lines.push("}");
    lines.push("");
  }

  return lines.join("\n");
}
registerTemplateFunc("templateConfigNestedTypes", templateConfigNestedTypes);

/**
 * Generate config struct fields for config.go.
 * Security and global fields are nested under SecurityConfig/GlobalsConfig.
 * Operational fields (timeout, retries) stay at the top level.
 */
function templateConfigStructFields(): string {
  const lines: string[] = [];

  // Nested security config
  const secFields = getCLISecurityFields();
  if (secFields.length > 0) {
    lines.push(`Security SecurityConfig \`yaml:"security,omitempty"\``);
  }

  // Nested globals config
  if (hasGlobals()) {
    lines.push(`Globals GlobalsConfig \`yaml:"globals,omitempty"\``);
  }

  // Preferences
  lines.push(`OutputFormat string \`yaml:"output_format,omitempty"\``);

  // Operational fields stay at top level
  lines.push(`Timeout string \`yaml:"timeout,omitempty"\``);

  if (hasGlobalSecurity()) {
    lines.push(`NoKeyring string \`yaml:"no_keyring,omitempty"\``);
  }

  // Retry config (when retries feature is used)
  if (isFeatureUsed("retries")) {
    lines.push(`NoRetries string \`yaml:"no_retries,omitempty"\``);
    lines.push(
      `RetryMaxElapsedTime string \`yaml:"retry_max_elapsed_time,omitempty"\``,
    );
    lines.push(
      `RetryConnectionErrors string \`yaml:"retry_connection_errors,omitempty"\``,
    );
    lines.push(`RetryConfig string \`yaml:"retry_config,omitempty"\``);
  }

  return lines.join("\n\t");
}
registerTemplateFunc("templateConfigStructFields", templateConfigStructFields);

/**
 * Generate config GetString switch cases for config.go.
 */
function templateConfigGetStringCases(): string {
  const fields = getCLISecurityFields();
  const lines: string[] = [];

  for (const field of fields) {
    if (field.isArray) continue; // Arrays use GetConfigStringSlice
    const fieldName = caser().ToPascal(field.field.Name);
    lines.push(`case "${field.flagName}":`);
    lines.push(`\t\treturn cfg.Security.${fieldName}`);
  }

  // Output format preference
  lines.push(`case "output-format":`);
  lines.push(`\t\treturn cfg.OutputFormat`);

  // Timeout (always available)
  lines.push(`case "timeout":`);
  lines.push(`\t\treturn cfg.Timeout`);

  if (hasGlobalSecurity()) {
    lines.push(`case "no-keyring":`);
    lines.push(`\t\treturn cfg.NoKeyring`);
  }

  // Retry keys (when retries feature is used)
  if (isFeatureUsed("retries")) {
    lines.push(`case "no-retries":`);
    lines.push(`\t\treturn cfg.NoRetries`);
    lines.push(`case "retry-max-elapsed-time":`);
    lines.push(`\t\treturn cfg.RetryMaxElapsedTime`);
    lines.push(`case "retry-connection-errors":`);
    lines.push(`\t\treturn cfg.RetryConnectionErrors`);
    lines.push(`case "retry-config":`);
    lines.push(`\t\treturn cfg.RetryConfig`);
  }

  // Global parameter keys
  if (hasGlobals()) {
    const globals = context.Global.AST.MainSDK.Globals;
    for (const field of globals.Fields) {
      const flagName = sanitizeFlagNameWithReserved(field.Name);
      const fieldName = caser().ToPascal(field.Name);
      lines.push(`case "${flagName}":`);
      lines.push(`\t\treturn cfg.Globals.${fieldName}`);
    }
  }

  return lines.join("\n\t");
}
registerTemplateFunc(
  "templateConfigGetStringCases",
  templateConfigGetStringCases,
);

/**
 * Generate config GetConfigStringSlice switch cases for config.go.
 */
function templateConfigGetStringSliceCases(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) {
    return "";
  }

  const lines: string[] = [];

  for (const field of fields) {
    if (!field.isArray) continue; // Only arrays
    const fieldName = caser().ToPascal(field.field.Name);
    lines.push(`case "${field.flagName}":`);
    lines.push(`\t\treturn cfg.Security.${fieldName}`);
  }

  return lines.join("\n\t");
}
registerTemplateFunc(
  "templateConfigGetStringSliceCases",
  templateConfigGetStringSliceCases,
);

/**
 * Helper to get the config accessor prefix for security fields.
 * Returns "cfg.Security." for nested config structure.
 */
function securityConfigAccessor(): string {
  return "cfg.Security.";
}

/**
 * Helper to get the config accessor prefix for globals fields.
 * Returns "cfg.Globals." for nested config structure.
 */
function globalsConfigAccessor(): string {
  return "cfg.Globals.";
}

// securityFieldResolver returns the call resolving a security field. Only
// secret fields are ever stored in the OS keychain, so only they go through
// secretResolver; the others resolve flag > env > config.
function securityFieldResolver(
  field: CLISecurityFieldInfo,
  secretResolver: string,
): string {
  if (field.isArray) {
    return `config.ResolveStringSliceCredential(cmd, "${field.flagName}")`;
  }
  if (!field.isSecret) {
    return `config.ResolveCredential(cmd, "${field.flagName}", false)`;
  }
  return `${secretResolver}(cmd, "${field.flagName}")`;
}

/**
 * Generate security field variable declarations.
 * Reads from flags, env vars, and config (priority: flag > env > config).
 */
function templateSecurityFieldParsing(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) {
    return "";
  }

  addImport("internal/config", true);

  const lines: string[] = [];
  lines.push(
    "// Resolve request credentials: flag > env var > keyring > config file (keyring skipped for dry-run)",
  );

  if (hasMultipleSecuritySchemes()) {
    // Several alternatives compete: keep each credential's source so the
    // scheme selection below can rank alternatives by how explicitly the
    // caller supplied them (see config.PickCredential). The values are
    // resolved by templateRankedSecurityConstruction, flag/env first.
    lines.push("var (");
    for (const field of fields) {
      const varName = sanitizePrivateFieldName(field.field.Name);
      lines.push(`    ${varName} ${field.isArray ? "[]string" : "string"}`);
    }
    lines.push(")");
    lines.push(`${securitySourcesVar()} := map[string]string{}`);
    return lines.join("\n    ");
  }

  for (const field of fields) {
    const varName = sanitizePrivateFieldName(field.field.Name);
    lines.push(
      `${varName}, _ := ${securityFieldResolver(
        field,
        "config.ResolveRequestSecurityCredential",
      )}`,
    );
  }

  return lines.join("\n    ");
}

// securityIdentifier allocates a Go identifier for the generated
// cross-scheme ranking code that cannot collide with a security field's
// local variable (fields become camelCase identifiers, so a scheme could in
// principle be named "credentialSources"): the base name gets a numeric
// suffix until it is free. Deterministic for a given document.
function securityIdentifier(base: string): string {
  const used = new Set<string>(["cmd", "globalSecurity"]);
  for (const field of getCLISecurityFields()) {
    used.add(sanitizePrivateFieldName(field.field.Name));
  }
  let name = base;
  for (let i = 2; used.has(name); i++) {
    name = `${base}${i}`;
  }
  return name;
}
registerTemplateFunc("securityIdentifier", securityIdentifier);

// securitySourcesVar names the generated map of flag name → credential
// source used for cross-scheme ranking.
function securitySourcesVar(): string {
  return securityIdentifier("credentialSources");
}
registerTemplateFunc(
  "templateSecurityFieldParsing",
  templateSecurityFieldParsing,
);

/**
 * Get the fully qualified Go type name for the global security type.
 * Used by client.go.stmpl for the buildGlobalSecurity return type.
 */
function templateGlobalSecurityTypeName(): string {
  if (!context.Global.AST.MainSDK.Security) {
    return "";
  }
  return sanitizeType(context.Global.AST.MainSDK.Security.Type, false, "");
}
registerTemplateFunc(
  "templateGlobalSecurityTypeName",
  templateGlobalSecurityTypeName,
);

/**
 * Generate security object construction code.
 * Builds the SDK Security struct from parsed field values.
 */
function templateGlobalSecurityConstruction(): string {
  if (!context.Global.AST.MainSDK.Security) {
    return "";
  }

  const security = context.Global.AST.MainSDK.Security;
  const securityType = security.Type;
  const typeName = sanitizeType(securityType, false, "");

  // Note: sanitizeType below will add the correct import for the security type
  const lines: string[] = [];
  lines.push(`globalSecurity := ${typeName}{}`);

  if (hasMultipleSecuritySchemes()) {
    return lines
      .concat(
        templateRankedSecurityConstruction(securityType, "globalSecurity"),
      )
      .join("\n    ");
  }

  // Iterate through the Security type's fields (Option1, Option2, etc.)
  for (const optionField of securityType.Fields || []) {
    if (optionField.Const) continue;

    const secAnno = optionField.Annotations?.Get("security") as
      | SecurityAnnotation
      | undefined;

    // Check if this is an option container or a class-type scheme
    if (secAnno?.Option || optionField.Type.Type.toString() === "class") {
      lines.push(
        templateSecurityOptionAssignment(optionField, "globalSecurity"),
      );
    } else if (optionField.Type.Type.toString() === "union") {
      lines.push(
        templateUnionSecurityConstruction(optionField, "globalSecurity"),
      );
    } else if (optionField.Type.Type.toString() === "string") {
      // Direct string field on security
      const varName = sanitizePrivateFieldName(optionField.Name);
      const fieldName = sanitizeFieldName(optionField.Name);
      lines.push(`if ${varName} != "" {`);
      if (optionField.Optional) {
        lines.push(`    globalSecurity.${fieldName} = &${varName}`);
      } else {
        lines.push(`    globalSecurity.${fieldName} = ${varName}`);
      }
      lines.push(`}`);
    } else if (optionField.Type.Type.toString() === "array") {
      // Direct array field on security (e.g., Scopes)
      const varName = sanitizePrivateFieldName(optionField.Name);
      const fieldName = sanitizeFieldName(optionField.Name);
      lines.push(`if len(${varName}) > 0 {`);
      lines.push(`    globalSecurity.${fieldName} = ${varName}`);
      lines.push(`}`);
    }
  }

  return lines.join("\n    ");
}
registerTemplateFunc(
  "templateGlobalSecurityConstruction",
  templateGlobalSecurityConstruction,
);

// ─── Cross-scheme credential ranking ────────────────────────────────────────
// With several OR alternatives, every alternative that has any credential used
// to be populated and the SDK sent the first declared one — so an access token
// from the environment silently beat an API key passed as a flag whenever the
// token's scheme was declared first. Alternatives are now candidates ranked
// by config.PickCredential (operation-allowed schemes, completeness, then the
// most explicit source), and exactly one is populated.

interface SecurityCandidate {
  field: string; // Go field name on the Security struct
  complete: string; // Go bool expression: every mandatory member is set
  sources: string[]; // flag names whose sources rank the candidate
  assignment: string[]; // Go lines populating the alternative
}

interface SecurityMember {
  cond: string; // Go bool expression: the member is set
  mandatory: boolean;
  flagName: string;
  assignment: string; // struct-literal entry ("Field: value")
}

// A member is mandatory for the alternative to be "complete" when it is a
// scalar without a schema default — except in the OAuth2 client-credentials
// scheme, whose generated struct also carries optional extras (audience,
// scopes, a defaulted token URL) as plain fields: there only the client id
// and secret make the credential usable. Completeness only breaks ties
// between equally explicit sources (see config.PickCredential).
function isMandatorySecurityMember(field: FieldDef, subType: string): boolean {
  if (field.Optional) return false;
  if (field.Type.Type.toString() !== "string") return false;
  if (field.Default?.Value !== undefined && field.Default?.Value !== null) {
    return false;
  }
  if (subType === "client_credentials") {
    const name = field.Name.toLowerCase().replace(/[^a-z]/g, "");
    return name === "clientid" || name === "clientsecret";
  }
  return true;
}

function securityMemberForField(
  field: FieldDef,
  subType: string,
): SecurityMember | null {
  const varName = sanitizePrivateFieldName(field.Name);
  const fieldName = sanitizeFieldName(field.Name);
  const flagName = sanitizeFlagNameWithReserved(field.Name);
  const typeStr = field.Type.Type.toString();
  if (typeStr === "string") {
    return {
      cond: `${varName} != ""`,
      mandatory: isMandatorySecurityMember(field, subType),
      flagName,
      assignment: field.Optional
        ? `${fieldName}: &${varName}`
        : `${fieldName}: ${varName}`,
    };
  }
  if (typeStr === "array") {
    return {
      cond: `len(${varName}) > 0`,
      mandatory: false,
      flagName,
      assignment: `${fieldName}: ${varName}`,
    };
  }
  return null;
}

function securitySubType(
  field: FieldDef | undefined,
  fallback: string,
): string {
  const secAnno = field?.Annotations?.Get("security") as
    | SecurityAnnotation
    | undefined;
  return secAnno?.SubType || fallback;
}

// Members of a class-typed alternative (an Option container or a scheme
// struct), flattening one level of nested class (e.g. BasicAuth) into a
// nested struct literal exactly as the single-scheme construction does.
function securityMembersForClass(
  classType: TypeDef,
  subType: string,
): SecurityMember[] {
  const members: SecurityMember[] = [];
  for (const inner of classType.Fields || []) {
    if (inner.Const) continue;
    if (inner.Type.Type.toString() === "class" && inner.Type.Fields?.length) {
      const nestedTypeName = sanitizeType(inner.Type, false, "");
      const nestedSubType = securitySubType(inner, subType);
      const nestedMembers: SecurityMember[] = [];
      for (const nested of inner.Type.Fields || []) {
        if (nested.Const) continue;
        const m = securityMemberForField(nested, nestedSubType);
        if (m) nestedMembers.push(m);
      }
      if (nestedMembers.length === 0) continue;
      const literal = `${sanitizeFieldName(
        inner.Name,
      )}: ${nestedTypeName}{${nestedMembers
        .map((m) => m.assignment)
        .join(", ")}}`;
      // The nested struct is one assignment; each nested field keeps its own
      // set-condition, mandatory-ness, and source.
      nestedMembers.forEach((m, i) =>
        members.push({ ...m, assignment: i === 0 ? literal : "" }),
      );
      continue;
    }
    const m = securityMemberForField(inner, securitySubType(inner, subType));
    if (m) members.push(m);
  }
  return members;
}

function completeExpr(members: SecurityMember[]): string {
  const mandatory = members.filter((m) => m.mandatory).map((m) => m.cond);
  if (mandatory.length > 0) return mandatory.join(" && ");
  return members.map((m) => m.cond).join(" || ") || "false";
}

function candidatesForSecurityField(
  optionField: FieldDef,
  accessor: string,
): SecurityCandidate[] {
  const goField = sanitizeFieldName(optionField.Name);
  const typeStr = optionField.Type.Type.toString();
  const secAnno = optionField.Annotations?.Get("security") as
    | SecurityAnnotation
    | undefined;

  const subType = secAnno?.SubType || "";
  if (secAnno?.Option || typeStr === "class") {
    const members = securityMembersForClass(optionField.Type, subType);
    if (members.length === 0) return [];
    const optionTypeName = sanitizeType(optionField.Type, false, "");
    const assignment = [`${accessor}.${goField} = &${optionTypeName}{`];
    for (const m of members) {
      if (m.assignment) assignment.push(`    ${m.assignment},`);
    }
    assignment.push(`}`);
    return [
      {
        field: goField,
        complete: completeExpr(members),
        sources: members.map((m) => m.flagName),
        assignment,
      },
    ];
  }

  if (typeStr === "union") {
    // Each variant is its own candidate so a token from the environment
    // cannot outrank credentials passed as flags inside the same alternative.
    const unionTypeName = sanitizeType(optionField.Type, false, "");
    const out: SecurityCandidate[] = [];
    for (const assocType of optionField.Type.AssociatedTypes || []) {
      if (assocType.Type.toString() === "string") {
        const varName = sanitizePrivateFieldName(assocType.Name);
        out.push({
          field: goField,
          complete: `${varName} != ""`,
          sources: [sanitizeFlagNameWithReserved(assocType.Name)],
          assignment: [
            `${accessor}.${goField} = &${unionTypeName}{`,
            `    ${sanitizeFieldName(assocType.Name)}: &${varName},`,
            `}`,
          ],
        });
      } else if (assocType.Type.toString() === "class") {
        const members = securityMembersForClass(assocType, subType);
        if (members.length === 0) continue;
        const credTypeName = sanitizeType(assocType, false, "");
        const assignment = [
          `${accessor}.${goField} = &${unionTypeName}{`,
          `    ${sanitizeFieldName(assocType.Name)}: &${credTypeName}{`,
        ];
        for (const m of members) {
          if (m.assignment) assignment.push(`        ${m.assignment},`);
        }
        assignment.push(`    },`, `}`);
        out.push({
          field: goField,
          complete: completeExpr(members),
          sources: members.map((m) => m.flagName),
          assignment,
        });
      }
    }
    return out;
  }

  const m = securityMemberForField(optionField, subType);
  if (!m) return [];
  const varName = sanitizePrivateFieldName(optionField.Name);
  const value =
    typeStr === "string" && optionField.Optional ? `&${varName}` : varName;
  return [
    {
      field: goField,
      complete: m.cond,
      sources: [m.flagName],
      assignment: [`${accessor}.${goField} = ${value}`],
    },
  ];
}

// templateRankedSecurityConstruction emits the candidate table and the
// switch that populates exactly the alternative config.PickCredential picks.
// allowedSecurityFields is the operation's hoisted scheme restriction (Go
// field names, declared order), empty when the operation accepts them all.
function templateRankedSecurityConstruction(
  securityType: TypeDef,
  accessor: string,
): string[] {
  const candidates: SecurityCandidate[] = [];
  // Sibling top-level flattened fields that belong to the same security
  // scheme (an http:basic username/password pair, an oauth2
  // client_credentials id/secret pair) are fields of ONE scheme, not
  // competing alternatives: they must form a single candidate so the
  // exactly-one switch assigns them together — splitting them would send a
  // username with no password.
  const flattenedSchemeKey = (f: FieldDef): string | null => {
    const typeStr = f.Type.Type.toString();
    if (typeStr !== "string" && typeStr !== "array") return null;
    const secAnno = f.Annotations?.Get("security") as
      | SecurityAnnotation
      | undefined;
    if (!secAnno || secAnno.Option) return null;
    // SchemeKey is the scheme's identity: two distinct schemes can share a
    // type/subtype pair, and their fields must not merge into one candidate.
    return secAnno.SchemeKey || `${secAnno.SecType}\u0000${secAnno.SubType}`;
  };
  // Unannotated top-level scalar/array fields (a client-credentials token
  // URL or scopes list) belong to the document's sole flattened scheme when
  // exactly one exists; with several schemes their ownership is ambiguous
  // and they stay independent candidates.
  const isUnannotatedAux = (f: FieldDef): boolean => {
    const typeStr = f.Type.Type.toString();
    if (typeStr !== "string" && typeStr !== "array") return false;
    const secAnno = f.Annotations?.Get("security") as
      | SecurityAnnotation
      | undefined;
    return !secAnno;
  };
  const fields = (securityType.Fields || []).filter((f) => !f.Const);
  const schemeCounts = new Map<string, number>();
  for (const f of fields) {
    const key = flattenedSchemeKey(f);
    if (key !== null) {
      schemeCounts.set(key, (schemeCounts.get(key) || 0) + 1);
    }
  }
  const soleKey = schemeCounts.size === 1 ? [...schemeCounts.keys()][0] : null;
  const effectiveKey = (f: FieldDef): string | null => {
    const key = flattenedSchemeKey(f);
    if (key !== null) return key;
    return soleKey !== null && isUnannotatedAux(f) ? soleKey : null;
  };
  const effectiveCounts = new Map<string, number>();
  for (const f of fields) {
    const key = effectiveKey(f);
    if (key !== null) {
      effectiveCounts.set(key, (effectiveCounts.get(key) || 0) + 1);
    }
  }
  const emittedGroups = new Set<string>();
  for (const optionField of fields) {
    const key = effectiveKey(optionField);
    if (key === null || (effectiveCounts.get(key) || 0) < 2) {
      candidates.push(...candidatesForSecurityField(optionField, accessor));
      continue;
    }
    if (emittedGroups.has(key)) continue;
    emittedGroups.add(key);
    const group = fields.filter((f) => effectiveKey(f) === key);
    const members = group
      .map((f) => {
        const secAnno = f.Annotations?.Get("security") as
          | SecurityAnnotation
          | undefined;
        return { f, m: securityMemberForField(f, secAnno?.SubType || "") };
      })
      .filter((x) => x.m !== null);
    if (members.length === 0) continue;
    // Each member assigns only when supplied: an unset auxiliary (a token
    // URL with an SDK-side default, say) must not overwrite that default
    // with an empty value.
    const assignment: string[] = [];
    for (const { f, m } of members) {
      const varName = sanitizePrivateFieldName(f.Name);
      const value =
        f.Type.Type.toString() === "string" && f.Optional
          ? `&${varName}`
          : varName;
      assignment.push(`if ${m!.cond} {`);
      assignment.push(
        `    ${accessor}.${sanitizeFieldName(f.Name)} = ${value}`,
      );
      assignment.push(`}`);
    }
    candidates.push({
      field: sanitizeFieldName(members[0].f.Name),
      // Mandatory members decide completeness: an unset optional member
      // (scopes or a defaulted token URL, say) must not demote a usable
      // credential in the ranking.
      complete: completeExpr(members.map(({ m }) => m!)),
      sources: members.map(({ m }) => m!.flagName),
      assignment,
    });
  }
  if (candidates.length === 0) return [];

  const lines: string[] = [];
  lines.push(
    "// Rank the alternatives by how explicitly the caller supplied them",
    "// (flag > env > keyring > config; complete before partial at the same",
    "// tier) and send exactly one: an explicit credential picks its scheme",
    "// regardless of the declared order.",
  );
  const candidatesVar = securityIdentifier("credentialCandidates");
  const allowedVar = securityIdentifier("allowedSecurityFields");
  const resolveCandidatesVar = securityIdentifier("resolveCandidates");
  const resolveVar = securityIdentifier("resolve");
  const pickedVar = securityIdentifier("picked");
  lines.push(
    `${resolveCandidatesVar} := func(${resolveVar} func(*cobra.Command, string) (string, string)) []config.CredentialCandidate {`,
  );
  for (const field of getCLISecurityFields()) {
    const varName = sanitizePrivateFieldName(field.field.Name);
    lines.push(
      `    ${varName}, ${securitySourcesVar()}["${
        field.flagName
      }"] = ${securityFieldResolver(field, resolveVar)}`,
    );
  }
  lines.push(`    return []config.CredentialCandidate{`);
  for (const c of candidates) {
    const sources = c.sources
      .map((flagName) => `${securitySourcesVar()}["${flagName}"]`)
      .join(", ");
    lines.push(
      `        {Field: "${c.field}", Complete: ${c.complete}, Sources: []string{${sources}}},`,
    );
  }
  lines.push("    }");
  lines.push("}");
  lines.push(
    "// A complete flag or env credential outranks anything in the keychain,",
    "// so only fall back to it (and risk a slow or locked keychain) when no",
    "// allowed alternative was supplied explicitly.",
  );
  lines.push(
    `${candidatesVar} := ${resolveCandidatesVar}(config.ResolveExplicitSecurityCredential)`,
  );
  lines.push(
    `${pickedVar} := config.PickExplicitCredential(${candidatesVar}, ${allowedVar})`,
  );
  lines.push(`if ${pickedVar} == -1 {`);
  lines.push(
    `    ${candidatesVar} = ${resolveCandidatesVar}(config.ResolveRequestSecurityCredential)`,
  );
  lines.push(
    `    ${pickedVar} = config.PickCredential(${candidatesVar}, ${allowedVar})`,
  );
  lines.push("}");
  lines.push(
    `flagutil.RecordCredentialsMissing(cmd, ${pickedVar} == -1 && config.PickCredential(${candidatesVar}, nil) == -1, "")`,
  );
  lines.push(`switch ${pickedVar} {`);
  candidates.forEach((c, i) => {
    lines.push(`case ${i}:`);
    for (const line of c.assignment) {
      lines.push(`    ${line}`);
    }
  });
  lines.push("}");
  return lines;
}

/**
 * Generate code to assign a security option based on available fields.
 */
function templateSecurityOptionAssignment(
  optionField: FieldDef,
  accessor: string,
): string {
  const optionType = optionField.Type;
  const optionTypeName = sanitizeType(optionType, false, "");
  const optionFieldName = sanitizeFieldName(optionField.Name);
  const lines: string[] = [];

  // Note: sanitizeType below will add the correct import for the option type

  // Get the inner fields of this option
  const innerFields = optionType.Fields || [];
  if (innerFields.length === 0) {
    return "";
  }

  // Build conditions and assignments for this option
  const conditions: string[] = [];
  const assignments: string[] = [];

  for (const innerField of innerFields) {
    if (innerField.Const) continue;

    // Check if this inner field is itself a class (e.g., BasicAuth with Username/Password)
    if (
      innerField.Type.Type.toString() === "class" &&
      innerField.Type.Fields?.length
    ) {
      const nestedResult = templateNestedSecurityConstruction(innerField);
      if (nestedResult.conditions.length > 0) {
        conditions.push(...nestedResult.conditions);
        assignments.push(...nestedResult.assignments);
      }
    } else if (innerField.Type.Type.toString() === "string") {
      const varName = sanitizePrivateFieldName(innerField.Name);
      const innerFieldName = sanitizeFieldName(innerField.Name);
      conditions.push(`${varName} != ""`);
      assignments.push(`${innerFieldName}: ${varName}`);
    } else if (innerField.Type.Type.toString() === "array") {
      const varName = sanitizePrivateFieldName(innerField.Name);
      const innerFieldName = sanitizeFieldName(innerField.Name);
      conditions.push(`len(${varName}) > 0`);
      assignments.push(`${innerFieldName}: ${varName}`);
    }
  }

  if (conditions.length > 0 && assignments.length > 0) {
    // Use OR logic - if any field is set, construct this option
    lines.push(`if ${conditions.join(" || ")} {`);
    lines.push(`    ${accessor}.${optionFieldName} = &${optionTypeName}{`);
    for (const assignment of assignments) {
      lines.push(`        ${assignment},`);
    }
    lines.push(`    }`);
    lines.push(`}`);
  }

  return lines.join("\n    ");
}

/**
 * Generate construction code for nested security types (e.g., BasicAuth).
 */
function templateNestedSecurityConstruction(field: FieldDef): {
  conditions: string[];
  assignments: string[];
} {
  const nestedType = field.Type;
  const nestedTypeName = sanitizeType(nestedType, false, "");
  const fieldName = sanitizeFieldName(field.Name);

  // Note: sanitizeType below will add the correct import for the nested type

  const conditions: string[] = [];
  const nestedAssignments: string[] = [];

  for (const nestedField of nestedType.Fields || []) {
    if (nestedField.Const) continue;

    const fieldTypeStr = nestedField.Type.Type.toString();
    const varName = sanitizePrivateFieldName(nestedField.Name);
    const nestedFieldName = sanitizeFieldName(nestedField.Name);

    if (fieldTypeStr === "string") {
      conditions.push(`${varName} != ""`);
      nestedAssignments.push(`${nestedFieldName}: ${varName}`);
    } else if (fieldTypeStr === "array") {
      conditions.push(`len(${varName}) > 0`);
      nestedAssignments.push(`${nestedFieldName}: ${varName}`);
    }
  }

  if (nestedAssignments.length === 0) {
    return { conditions: [], assignments: [] };
  }

  // Create the nested struct assignment
  const assignment = `${fieldName}: ${nestedTypeName}{${nestedAssignments.join(
    ", ",
  )}}`;

  return { conditions, assignments: [assignment] };
}

/**
 * Generate construction code for union-typed security fields (e.g., Oauth2Input).
 * Builds the correct union variant based on which flags are provided.
 * Priority: string variant (direct token) first, then class variant (credentials).
 */
function templateUnionSecurityConstruction(
  optionField: FieldDef,
  accessor: string,
): string {
  const unionType = optionField.Type;
  const unionTypeName = sanitizeType(unionType, false, "");
  const optionFieldName = sanitizeFieldName(optionField.Name);
  const associatedTypes = unionType.AssociatedTypes || [];

  const lines: string[] = [];

  // Separate variants into string (token) and class (credentials)
  const stringVariants: TypeDef[] = [];
  const classVariants: TypeDef[] = [];
  for (const assocType of associatedTypes) {
    if (assocType.Type.toString() === "string") {
      stringVariants.push(assocType);
    } else if (assocType.Type.toString() === "class") {
      classVariants.push(assocType);
    }
  }

  // Generate string variant (token) first — it takes priority if set
  for (const tokenVariant of stringVariants) {
    const varName = sanitizePrivateFieldName(tokenVariant.Name);
    lines.push(`if ${varName} != "" {`);
    lines.push(`    ${accessor}.${optionFieldName} = &${unionTypeName}{`);
    const tokenFieldName = sanitizeFieldName(tokenVariant.Name);
    lines.push(`        ${tokenFieldName}: &${varName},`);
    lines.push(`    }`);

    // If there are class variants, leave the closing brace open for "} else if"
    if (classVariants.length === 0) {
      lines.push(`}`);
    }
  }

  // Generate class variant (credentials) — used as else-if after token
  for (const credVariant of classVariants) {
    const credTypeName = sanitizeType(credVariant, false, "");
    const credFieldName = sanitizeFieldName(credVariant.Name);

    // Collect conditions and assignments from credential fields
    const conditions: string[] = [];
    const credAssignments: string[] = [];

    for (const credField of credVariant.Fields || []) {
      if (credField.Const) continue;
      const fieldTypeStr = credField.Type.Type.toString();
      const varName = sanitizePrivateFieldName(credField.Name);
      const fieldName = sanitizeFieldName(credField.Name);

      if (fieldTypeStr === "string") {
        conditions.push(`${varName} != ""`);
        if (credField.Optional) {
          credAssignments.push(`${fieldName}: &${varName}`);
        } else {
          credAssignments.push(`${fieldName}: ${varName}`);
        }
      } else if (fieldTypeStr === "array") {
        conditions.push(`len(${varName}) > 0`);
        credAssignments.push(`${fieldName}: ${varName}`);
      }
    }

    if (conditions.length > 0 && credAssignments.length > 0) {
      // Use "} else if" when there's a token variant before this
      const prefix = stringVariants.length > 0 ? `} else if` : `if`;
      lines.push(`${prefix} ${conditions.join(" || ")} {`);
      lines.push(`    ${accessor}.${optionFieldName} = &${unionTypeName}{`);
      lines.push(`        ${credFieldName}: &${credTypeName}{`);
      for (const assignment of credAssignments) {
        lines.push(`            ${assignment},`);
      }
      lines.push(`        },`);
      lines.push(`    }`);
      lines.push(`}`);
    }
  }

  return lines.join("\n    ");
}

/**
 * Check if all security credentials are optional for the configure command.
 *
 * Credentials are optional when:
 * - There are multiple OR alternatives (user picks whichever they use)
 * - The security block is optional (has {} empty requirement)
 *
 * Credentials are required when there is a single security option with no alternatives.
 */
function isConfigureAllOptional(): boolean {
  const security = context.Global.AST.MainSDK.Security;
  if (!security) return false;

  const fields = (security.Type.Fields || []).filter((f: FieldDef) => !f.Const);
  if (fields.length === 0) return false;

  // Check for OR alternatives: SecurityOption annotation is set when there
  // are multiple security requirements (e.g., apiKey OR bearerAuth OR oauth2).
  // Note: multiple top-level fields does NOT mean OR alternatives — an AND
  // group (e.g., clientCredentials AND basicAuth) also produces multiple fields.
  for (const field of fields) {
    const secAnno = field.Annotations?.Get("security") as
      | SecurityAnnotation
      | undefined;
    if (secAnno?.SecurityOption) {
      return true; // Multiple OR alternatives exist
    }
    // Also check structural optionality (from {} empty requirement)
    if (field.Optional) {
      return true;
    }
  }

  return false;
}
registerTemplateFunc("isConfigureAllOptional", isConfigureAllOptional);

// ─── Security Scheme Grouping ────────────────────────────────────────────────
// Groups security fields by their top-level OR alternative, preserving the
// AND relationship within each group.

interface SecuritySchemeGroup {
  key: string; // Unique key for switch/case (e.g., "my-api-key", "option2")
  label: string; // Human-readable label for the select menu
  fields: CLISecurityFieldInfo[]; // Fields belonging to this scheme
}

/**
 * Build security scheme groups from the top-level Security type.
 * Each group represents one OR alternative. Groups with class-type fields
 * contain all the AND-ed inner fields.
 */
function getSecuritySchemeGroups(): SecuritySchemeGroup[] {
  const security = context.Global.AST.MainSDK.Security;
  if (!security) return [];

  const securityType = security.Type;
  const groups: SecuritySchemeGroup[] = [];

  for (const optionField of securityType.Fields || []) {
    if (optionField.Const) continue;

    const secAnno = optionField.Annotations?.Get("security") as
      | SecurityAnnotation
      | undefined;
    const fieldType = optionField.Type.Type.toString();
    const key = sanitizeFlagName(optionField.Name);

    if (fieldType === "class" && optionField.Type.Fields?.length) {
      // AND group — flatten inner fields for this option
      const fields = flattenCLISecurityObject(
        optionField.Type,
        optionField.Optional,
      );
      if (fields.length === 0) continue;

      // Build label from the inner field descriptions
      const label = buildSchemeLabel(fields, secAnno);
      groups.push({ key, label, fields });
    } else if (fieldType === "union") {
      // Union type — flatten all variants
      const fields = flattenCLISecurityObject(
        optionField.Type,
        optionField.Optional,
      );
      if (fields.length === 0) continue;

      const label = buildSchemeLabel(fields, secAnno);
      groups.push({ key, label, fields });
    } else if (fieldType === "string" || fieldType === "array") {
      // Single primitive field
      const fields = flattenCLISecurityObject(
        // Wrap in a pseudo-type to reuse flattenCLISecurityObject
        { Fields: [optionField], Type: securityType.Type } as any,
        optionField.Optional,
      );
      if (fields.length === 0) continue;

      const label = buildSchemeLabel(fields, secAnno);
      groups.push({ key, label, fields });
    }
  }

  return groups;
}

/**
 * Build a human-readable label for a scheme group.
 */
function buildSchemeLabel(
  fields: CLISecurityFieldInfo[],
  secAnno: SecurityAnnotation | undefined,
): string {
  if (fields.length === 1) {
    // Single field — use its description
    return fields[0].description;
  }

  // Multiple AND fields — build combined label
  // Try to use sec type for a category
  const secType = secAnno?.SecType || fields[0]?.secType || "";
  const subType = secAnno?.SubType || fields[0]?.secSubType || "";

  if (secType === "http" && subType === "basic") {
    return "HTTP Basic (Username + Password)";
  }
  if (secType === "http" && subType === "custom") {
    return "Custom (" + fields.map((f) => f.description).join(" + ") + ")";
  }
  if (secType === "oauth2" && subType === "client_credentials") {
    return "OAuth2 Client Credentials";
  }
  if (secType === "apiKey") {
    if (fields.length > 1) {
      return (
        "API Key + " +
        fields
          .filter((f) => f.secType !== "apiKey")
          .map((f) => f.description)
          .join(" + ")
      );
    }
    return "API Key";
  }

  // Fallback: join field names
  return fields.map((f) => f.description).join(" + ");
}

/**
 * Returns true when there are multiple OR security alternatives,
 * meaning the auth login form should show a scheme selector first.
 */
function hasMultipleSecuritySchemes(): boolean {
  return getSecuritySchemeGroups().length > 1;
}
registerTemplateFunc("hasMultipleSecuritySchemes", hasMultipleSecuritySchemes);

/**
 * Generate configure command prompts for interactive security setup.
 *
 * When security has multiple OR alternatives, all prompts allow skipping
 * (empty Enter = leave unset). When there is a single required security
 * option, prompts loop until a value is entered.
 */
function templateConfigurePrompts(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) {
    return "";
  }

  const allOptional = isConfigureAllOptional();
  const acc = securityConfigAccessor();

  const lines: string[] = [];

  for (const field of fields) {
    const fieldName = caser().ToPascal(field.field.Name);
    const cfgField = `${acc}${fieldName}`;

    if (field.isSecret) {
      // Helper to store secret value — keyring preferred, config file fallback.
      // Warns only when the keychain is available but the write fails; a missing
      // keychain falls back silently.
      const storeSecret = (varName: string, indent: string): string[] => {
        addImport("errors");
        const storeLines: string[] = [];
        storeLines.push(
          `${indent}if err := config.StoreSecret("${field.flagName}", string(${varName}), &${cfgField}); err != nil {`,
        );
        storeLines.push(
          `${indent}    if !errors.Is(err, config.ErrKeyringUnavailable) {`,
        );
        storeLines.push(
          `${indent}        fmt.Fprintf(out, "WARNING: failed to store --${field.flagName} in OS keychain (%v); falling back to config file storage.\\n", err)`,
        );
        storeLines.push(`${indent}    }`);
        storeLines.push(`${indent}} else {`);
        storeLines.push(`${indent}    keychainStored = true`);
        storeLines.push(`${indent}}`);
        return storeLines;
      };

      // Helper to clear a secret — remove from keyring + config
      const clearSecret = (indent: string): string[] => {
        const clearLines: string[] = [];
        clearLines.push(`${indent}if config.KeyringAvailable() {`);
        clearLines.push(
          `${indent}    _ = config.DeleteKeyringValue("${field.flagName}")`,
        );
        clearLines.push(`${indent}}`);
        clearLines.push(`${indent}${cfgField} = ""`);
        clearLines.push(
          `${indent}fmt.Fprintln(out, "Cleared --${field.flagName}")`,
        );
        return clearLines;
      };

      if (allOptional) {
        // Optional: prompt once, allow empty Enter to skip.
        // Wrapped in a block for Go variable scoping.
        lines.push(`{`);
        lines.push(
          `    fmt.Fprintf(out, "${escapeGoString(
            field.description,
          )} [%s]: ", maskSecret(config.GetStoredSecret("${
            field.flagName
          }", ${cfgField})))`,
        );
        lines.push(`    input, err := readSecret(cmd)`);
        lines.push(`    fmt.Fprintln(out) // newline after hidden input`);
        lines.push(`    if err == nil && string(input) == "-" {`);
        lines.push(...clearSecret("        "));
        lines.push(`    } else if err == nil && len(input) > 0 {`);
        lines.push(...storeSecret("input", "        "));
        lines.push(`    }`);
        lines.push(`}`);
      } else {
        // Required: loop until non-empty value entered
        lines.push(`for {`);
        lines.push(
          `    fmt.Fprintf(out, "${escapeGoString(
            field.description,
          )} [%s]: ", maskSecret(config.GetStoredSecret("${
            field.flagName
          }", ${cfgField})))`,
        );
        lines.push(`    input, err := readSecret(cmd)`);
        lines.push(`    fmt.Fprintln(out) // newline after hidden input`);
        lines.push(`    if err == nil && string(input) == "-" {`);
        lines.push(...clearSecret("        "));
        lines.push(`        break`);
        lines.push(`    } else if err == nil && len(input) > 0 {`);
        lines.push(...storeSecret("input", "        "));
        lines.push(`        break`);
        lines.push(`    }`);
        lines.push(
          `    if config.GetStoredSecret("${field.flagName}", ${cfgField}) != "" {`,
        );
        lines.push(`        break // keep existing value`);
        lines.push(`    }`);
        lines.push(
          `    fmt.Fprintln(out, "Value cannot be empty. Please enter a value.")`,
        );
        lines.push(`}`);
      }
    } else if (field.isArray) {
      // Array fields: always allow skipping (comma-separated input)
      lines.push(
        `fmt.Fprintf(out, "${escapeGoString(
          field.description,
        )} (comma-separated) [%s]: ", strings.Join(${cfgField}, ","))`,
      );
      lines.push(
        `if input, _ := reader.ReadString('\\n'); strings.TrimSpace(input) == "-" {`,
      );
      lines.push(`    ${cfgField} = nil`);
      lines.push(`    fmt.Fprintln(out, "Cleared --${field.flagName}")`);
      lines.push(`} else if strings.TrimSpace(input) != "" {`);
      lines.push(
        `    ${cfgField} = strings.Split(strings.TrimSpace(input), ",")`,
      );
      lines.push(`}`);
    } else {
      if (allOptional) {
        // Optional: prompt once, allow empty Enter to skip
        lines.push(
          `fmt.Fprintf(out, "${escapeGoString(
            field.description,
          )} [%s]: ", ${cfgField})`,
        );
        lines.push(`{`);
        lines.push(`    input, _ := reader.ReadString('\\n')`);
        lines.push(`    v := strings.TrimSpace(input)`);
        lines.push(`    if v == "-" {`);
        lines.push(`        ${cfgField} = ""`);
        lines.push(`        fmt.Fprintln(out, "Cleared --${field.flagName}")`);
        lines.push(`    } else if v != "" {`);
        lines.push(`        ${cfgField} = v`);
        lines.push(`    }`);
        lines.push(`}`);
      } else {
        // Required: loop until non-empty value entered
        lines.push(`for {`);
        lines.push(
          `    fmt.Fprintf(out, "${escapeGoString(
            field.description,
          )} [%s]: ", ${cfgField})`,
        );
        lines.push(`    input, _ := reader.ReadString('\\n')`);
        lines.push(`    v := strings.TrimSpace(input)`);
        lines.push(`    if v == "-" {`);
        lines.push(`        ${cfgField} = ""`);
        lines.push(`        fmt.Fprintln(out, "Cleared --${field.flagName}")`);
        lines.push(`        break`);
        lines.push(`    } else if v != "" {`);
        lines.push(`        ${cfgField} = v`);
        lines.push(`        break`);
        lines.push(`    }`);
        lines.push(`    if ${cfgField} != "" {`);
        lines.push(`        break // keep existing value`);
        lines.push(`    }`);
        lines.push(
          `    fmt.Fprintln(out, "Value cannot be empty. Please enter a value.")`,
        );
        lines.push(`}`);
      }
    }
    lines.push("");
  }

  return lines.join("\n        ");
}
registerTemplateFunc("templateConfigurePrompts", templateConfigurePrompts);

/**
 * Generate whoami command credential display code.
 */
function templateWhoamiCredentials(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) {
    return "";
  }

  // Add strings import if any security field is an array (uses strings.Join)
  if (fields.some((f) => f.isArray)) {
    addImport("strings");
  }

  const lines: string[] = [];

  for (const field of fields) {
    // Sanitize description for single-line comment
    const singleLineDescription = field.description
      .replace(/\n/g, " ")
      .replace(/\s+/g, " ")
      .trim();
    lines.push(`// ${singleLineDescription}`);
    lines.push(`{`);
    if (field.isArray) {
      lines.push(
        `    value, source := ${securityFieldResolver(
          field,
          "config.ResolveRequestSecurityCredential",
        )}`,
      );
      lines.push(
        `    fmt.Fprintf(out, "  --%-25s [%-7s] %s\\n", "${field.flagName}", source, strings.Join(value, ","))`,
      );
    } else {
      lines.push(
        `    value, source := ${securityFieldResolver(
          field,
          "config.ResolveRequestSecurityCredential",
        )}`,
      );
      lines.push(
        `    fmt.Fprintf(out, "  --%-25s [%-7s] %s\\n", "${field.flagName}", source, maskSecret(value))`,
      );
    }
    lines.push(`}`);
    lines.push(``);
  }

  return lines.join("\n    ");
}
registerTemplateFunc("templateWhoamiCredentials", templateWhoamiCredentials);

/**
 * Machine-mode companion to templateWhoamiCredentials: fills a credentials
 * map for the structured whoami object instead of printing table rows.
 * Emits nothing when the document declares no global security.
 */
function templateWhoamiCredentialsStructured(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) {
    return "";
  }
  const lines: string[] = [`credentials := map[string]any{}`];
  for (const field of fields) {
    lines.push(`{`);
    if (field.isArray) {
      lines.push(
        `    value, source := ${securityFieldResolver(
          field,
          "config.ResolveRequestSecurityCredential",
        )}`,
      );
      lines.push(
        `    credentials["${field.flagName}"] = map[string]any{"source": source, "value": strings.Join(value, ",")}`,
      );
    } else {
      lines.push(
        `    value, source := ${securityFieldResolver(
          field,
          "config.ResolveRequestSecurityCredential",
        )}`,
      );
      lines.push(
        `    credentials["${field.flagName}"] = map[string]any{"source": source, "value": maskSecret(value)}`,
      );
    }
    lines.push(`}`);
  }
  lines.push(`info["credentials"] = credentials`);
  return lines.join("\n        ");
}
registerTemplateFunc(
  "templateWhoamiCredentialsStructured",
  templateWhoamiCredentialsStructured,
);

/**
 * Check if a security field is a token URL field that should be skipped in test harness flags.
 * Token URL fields use the spec default, and the test client intercepts the actual token exchange.
 */
function isTokenURLField(field: CLISecurityFieldInfo): boolean {
  const name = field.field.Name.toLowerCase();
  return name.includes("tokenurl") || name.includes("token_url");
}

/**
 * Get a default test value for a security field based on its type.
 * These values match the standard test credentials used in Arazzo test specs.
 */
function getDefaultSecurityTestValue(field: CLISecurityFieldInfo): string {
  return getTestValueForSchemeField(
    field.field.Name,
    field.secType,
    field.secSubType,
  );
}

/**
 * Get the test value for a security field based on the parent scheme's type.
 * Used for multi-scheme security where inner fields don't carry secType/secSubType.
 */
function getTestValueForSchemeField(
  fieldName: string,
  secType: string,
  secSubType: string,
): string {
  const name = fieldName.toLowerCase();
  switch (secType) {
    case "apiKey":
      return "test_api_key";
    case "http":
      switch (secSubType) {
        case "bearer":
          return "Bearer test_token";
        case "basic":
          if (name.includes("username") || name.includes("user")) {
            return "testUser";
          }
          return "testPass";
        default:
          return "test";
      }
    case "oauth2":
      if (name.includes("secret")) return "test_client_secret";
      if (name.includes("id")) return "test_client_id";
      if (name.includes("username") || name.includes("user")) return "testUser";
      if (name.includes("password") || name.includes("pass")) return "testPass";
      return "Bearer testToken";
    case "openIdConnect":
      return "Bearer testToken";
    default:
      return "test";
  }
}

// =============================================================================
// README Security Section Helpers
// =============================================================================

/**
 * Generate the CLI flag example for the README security section. Values are
 * shell references to the credential's environment variable, so the line is
 * runnable as written and never carries a literal secret:
 * `--api-key "$ACME_API_KEY" --api-secret "$ACME_API_SECRET"`
 */
function templateCLISecurityFlagExample(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) return "";
  const envPrefix = context.Global.Config.EnvVarPrefix || "";
  return fields
    .slice(0, 3)
    .map((f) => {
      const envVar = envPrefix
        ? `${envPrefix}_${f.envVarSuffix}`
        : f.envVarSuffix;
      return `--${f.flagName} "$${envVar}"`;
    })
    .join(" ");
}
registerTemplateFunc(
  "templateCLISecurityFlagExample",
  templateCLISecurityFlagExample,
);

/**
 * Generate the env var table rows for the README security section.
 * Returns markdown table rows (without header).
 */
function templateCLISecurityEnvTable(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) return "";
  const envPrefix = context.Global.Config.EnvVarPrefix || "";
  return fields
    .map((f) => {
      const envVar = envPrefix
        ? `${envPrefix}_${f.envVarSuffix}`
        : f.envVarSuffix;
      // A multi-line description would break the markdown table row.
      const desc = (f.description || "Authentication credential").replace(
        /\s*\r?\n\s*/g,
        " ",
      );
      return `| \`${envVar}\` | ${desc} |`;
    })
    .join("\n");
}
registerTemplateFunc(
  "templateCLISecurityEnvTable",
  templateCLISecurityEnvTable,
);

// =============================================================================
// Global Parameters in Configure/Whoami
// =============================================================================

/**
 * Check if either global security or global parameters exist,
 * meaning the configure/whoami commands should be generated.
 */
function hasConfigurableSettings(): boolean {
  return hasGlobalSecurity() || hasGlobals();
}
registerTemplateFunc("hasConfigurableSettings", hasConfigurableSettings);

/**
 * Generate the Short description for the configure command.
 * Adapts based on whether security and/or globals are present.
 */
function templateConfigureShort(): string {
  const hasSec = hasGlobalSecurity();
  const hasGlob = hasGlobals();
  if (hasSec && hasGlob) {
    return "Configure authentication, global parameters, and preferences";
  }
  if (hasSec) {
    return "Configure authentication credentials and preferences";
  }
  if (hasGlob) {
    return "Configure global parameters and preferences";
  }
  return "Configure CLI preferences";
}
registerTemplateFunc("templateConfigureShort", templateConfigureShort);

/**
 * Generate the Long description for the configure command.
 * Adapts based on whether security and/or globals are present.
 */
function templateConfigureLong(): string {
  const hasSec = hasGlobalSecurity();
  const hasGlob = hasGlobals();
  const cliName = sanitizeCliName();
  const envPrefix = context.Global.Config.EnvVarPrefix || "";

  let what: string;
  if (hasSec && hasGlob) {
    what = "authentication credentials, global parameters, and preferences";
  } else if (hasSec) {
    what = "authentication credentials and preferences";
  } else if (hasGlob) {
    what = "global parameters and preferences";
  } else {
    what = "preferences (such as default output format)";
  }

  let desc = `Interactively configure ${what} for the CLI.\n`;
  desc += `Settings are stored in ~/.config/${cliName}/config.yaml.\n`;
  desc += `Secret credentials are stored in the OS keychain when available.\n`;
  desc += `\n`;
  if (envPrefix) {
    // Derive the example from the document's actual configurable values: the
    // primary credential's environment variable, else the first global
    // parameter's. Never fabricate a variable the CLI does not read.
    const primary = hasSec
      ? primaryCLISecurityField(getCLISecurityFields())
      : undefined;
    let exampleVar = "";
    if (primary) {
      exampleVar = `${envPrefix}_${primary.envVarSuffix}`;
    } else if (hasGlob) {
      const globalFields = context.Global.AST.MainSDK.Globals?.Fields;
      if (globalFields?.length) {
        exampleVar = `${envPrefix}_${toEnvVarSuffix(globalFields[0].Name)}`;
      }
    }
    desc += `You can also set values via environment variables with the ${envPrefix}_ prefix\n`;
    if (exampleVar) {
      desc += `(e.g., ${exampleVar}) or pass them as flags to individual commands.\n`;
    } else {
      desc += `or pass them as flags to individual commands.\n`;
    }
    desc += `\n`;
    desc += `Priority: CLI flags > environment variables > OS keychain > config file`;
  } else {
    desc += `You can also pass values as flags to individual commands.\n`;
    desc += `\n`;
    desc += `Priority: CLI flags > OS keychain > config file`;
  }

  return desc;
}
registerTemplateFunc("templateConfigureLong", templateConfigureLong);

/**
 * Generate configure command prompts for global parameters.
 * All global parameter prompts are optional (allow skipping with Enter).
 */
function templateConfigureGlobalPrompts(): string {
  if (!hasGlobals()) return "";

  const globals = context.Global.AST.MainSDK.Globals;
  const acc = globalsConfigAccessor();
  const lines: string[] = [];

  for (const field of globals.Fields) {
    const fieldName = caser().ToPascal(field.Name);
    const cfgField = `${acc}${fieldName}`;
    const desc =
      field.Comments?.Summary?.trim() ||
      field.Comments?.Description?.trim()?.split(/[.\n]/)[0]?.trim() ||
      `Global ${sanitizeFlagName(field.Name)} parameter`;

    // All global parameters are optional — prompt once, allow empty Enter to skip
    lines.push(
      `fmt.Fprintf(out, "${escapeGoString(desc)} [%s]: ", ${cfgField})`,
    );
    lines.push(`{`);
    lines.push(`    input, _ := reader.ReadString('\\n')`);
    lines.push(`    v := strings.TrimSpace(input)`);
    lines.push(`    if v == "-" {`);
    lines.push(`        ${cfgField} = ""`);
    lines.push(
      `        fmt.Fprintln(out, "Cleared --${sanitizeFlagName(field.Name)}")`,
    );
    lines.push(`    } else if v != "" {`);
    lines.push(`        ${cfgField} = v`);
    lines.push(`    }`);
    lines.push(`}`);
    lines.push(``);
  }

  return lines.join("\n        ");
}
registerTemplateFunc(
  "templateConfigureGlobalPrompts",
  templateConfigureGlobalPrompts,
);

/**
 * Generate configure command prompts for user preferences (output-format).
 * These are always available regardless of security/globals.
 */
function templateConfigurePreferences(): string {
  const lines: string[] = [];

  // Output format prompt
  lines.push(
    `fmt.Fprintf(out, "Default output format (pretty, json, yaml, table, toon) [%s]: ", cfg.OutputFormat)`,
  );
  lines.push(`{`);
  lines.push(`    input, _ := reader.ReadString('\\n')`);
  lines.push(`    v := strings.TrimSpace(input)`);
  lines.push(`    if v == "-" {`);
  lines.push(`        cfg.OutputFormat = ""`);
  lines.push(
    `        fmt.Fprintln(out, "Cleared output format (will use pretty)")`,
  );
  lines.push(`    } else if v != "" {`);
  lines.push(`        switch v {`);
  lines.push(`        case "pretty", "json", "yaml", "table", "toon":`);
  lines.push(`            cfg.OutputFormat = v`);
  lines.push(`        default:`);
  lines.push(
    `            fmt.Fprintf(out, "Unknown format %q, keeping current value\\n", v)`,
  );
  lines.push(`        }`);
  lines.push(`    }`);
  lines.push(`}`);

  return lines.join("\n        ");
}
registerTemplateFunc(
  "templateConfigurePreferences",
  templateConfigurePreferences,
);

function templateConfigurePreferenceVarDeclarations(): string {
  return `var cfgOutputFormat string`;
}
registerTemplateFunc(
  "templateConfigurePreferenceVarDeclarations",
  templateConfigurePreferenceVarDeclarations,
);

function templateConfigurePreferenceFormFields(): string {
  const lines: string[] = [];
  lines.push(`forms.NewSelect(&cfgOutputFormat,`);
  lines.push(`    forms.NewOption("Keep current", ""),`);
  lines.push(
    `    forms.NewOption("Clear (use built-in default: pretty)", "__CLEAR__"),`,
  );
  lines.push(`    forms.NewOption("pretty", "pretty"),`);
  lines.push(`    forms.NewOption("json", "json"),`);
  lines.push(`    forms.NewOption("yaml", "yaml"),`);
  lines.push(`    forms.NewOption("table", "table"),`);
  lines.push(`    forms.NewOption("toon", "toon"),`);
  lines.push(`).`);
  lines.push(`    Title("Default output format").`);
  lines.push(
    `    Description("Choose the default response rendering format for this CLI"),`,
  );
  return lines.join("\n        ");
}
registerTemplateFunc(
  "templateConfigurePreferenceFormFields",
  templateConfigurePreferenceFormFields,
);

function templateConfigurePreferenceStore(): string {
  const lines: string[] = [];
  lines.push(`switch cfgOutputFormat {`);
  lines.push(`case "__CLEAR__":`);
  lines.push(`    cfg.OutputFormat = ""`);
  lines.push(`case "pretty", "json", "yaml", "table", "toon":`);
  lines.push(`    cfg.OutputFormat = cfgOutputFormat`);
  lines.push(`}`);
  return lines.join("\n    ");
}
registerTemplateFunc(
  "templateConfigurePreferenceStore",
  templateConfigurePreferenceStore,
);

/**
 * Generate the Short description for the whoami command.
 * Adapts based on whether security and/or globals are present.
 */
function templateWhoamiShort(): string {
  const hasSec = hasGlobalSecurity();
  const hasGlob = hasGlobals();
  if (hasSec && hasGlob) {
    return "Display current authentication and global parameter configuration";
  }
  if (hasGlob) {
    return "Display current global parameter configuration";
  }
  return "Display current authentication configuration";
}
registerTemplateFunc("templateWhoamiShort", templateWhoamiShort);

/**
 * Generate whoami display section for global parameters.
 */
function templateWhoamiGlobals(): string {
  if (!hasGlobals()) return "";

  const globals = context.Global.AST.MainSDK.Globals;
  const lines: string[] = [];

  lines.push(`fmt.Fprintln(out)`);
  lines.push(`fmt.Fprintln(out, "Global Parameters:")`);

  for (const field of globals.Fields) {
    const flagName = sanitizeFlagNameWithReserved(field.Name);
    lines.push(``);
    const desc =
      field.Comments?.Summary?.trim() ||
      field.Comments?.Description?.trim()?.split(/[.\n]/)[0]?.trim() ||
      `Global ${sanitizeFlagName(field.Name)} parameter`;
    const singleLineDescription = desc
      .replace(/\n/g, " ")
      .replace(/\s+/g, " ")
      .trim();
    lines.push(`// ${singleLineDescription}`);
    lines.push(`{`);
    lines.push(
      `    value, source := config.ResolveCredential(cmd, "${flagName}", ${
        field.Default?.Value != null
      })`,
    );
    lines.push(
      `    fmt.Fprintf(out, "  --%-25s [%-7s] %s\\n", "${flagName}", source, value)`,
    );
    lines.push(`}`);
  }

  return lines.join("\n    ");
}
registerTemplateFunc("templateWhoamiGlobals", templateWhoamiGlobals);

/**
 * Machine-mode companion to templateWhoamiGlobals: fills a parameters map
 * for the structured whoami object. Emits nothing without globals.
 */
function templateWhoamiGlobalsStructured(): string {
  if (!hasGlobals()) return "";

  const globals = context.Global.AST.MainSDK.Globals;
  const lines: string[] = [`parameters := map[string]any{}`];
  for (const field of globals.Fields) {
    const flagName = sanitizeFlagNameWithReserved(field.Name);
    lines.push(`{`);
    lines.push(
      `    value, source := config.ResolveCredential(cmd, "${flagName}", ${
        field.Default?.Value != null
      })`,
    );
    lines.push(
      `    parameters["${flagName}"] = map[string]any{"source": source, "value": value}`,
    );
    lines.push(`}`);
  }
  lines.push(`info["global_parameters"] = parameters`);
  return lines.join("\n        ");
}
registerTemplateFunc(
  "templateWhoamiGlobalsStructured",
  templateWhoamiGlobalsStructured,
);

// =============================================================================
// Auth Login / Logout Helpers (Interactive Auth)
// =============================================================================

/**
 * Generate Go variable declarations for the auth login form.
 * Each security field gets a string variable that the form binds to.
 */
function templateAuthLoginVarDeclarations(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) return "";

  const lines: string[] = [];
  for (const field of fields) {
    const varName = `auth${caser().ToPascal(field.field.Name)}`;
    lines.push(`var ${varName} string`);
  }
  return lines.join("\n    ");
}
registerTemplateFunc(
  "templateAuthLoginVarDeclarations",
  templateAuthLoginVarDeclarations,
);

/**
 * Generate form fields for every CLI security field (configure command).
 */
function templateAuthLoginFormFields(): string {
  return genSchemeFormFields(getCLISecurityFields()).join("\n            ");
}
registerTemplateFunc(
  "templateAuthLoginFormFields",
  templateAuthLoginFormFields,
);

/**
 * Generate credential storage code after the auth login form completes.
 * Secrets go to keyring (with config fallback), non-secrets go to config.
 */
function templateAuthLoginStore(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) return "";

  // Add strings import if any security field is an array (uses strings.Split)
  if (fields.some((f) => f.isArray)) {
    addImport("strings");
  }

  const acc = securityConfigAccessor();
  const lines: string[] = [];

  for (const field of fields) {
    const varName = `auth${caser().ToPascal(field.field.Name)}`;
    const fieldName = caser().ToPascal(field.field.Name);
    const cfgField = `${acc}${fieldName}`;

    if (field.isArray) {
      lines.push(`if ${varName} != "" {`);
      lines.push(
        `    ${cfgField} = strings.Split(strings.TrimSpace(${varName}), ",")`,
      );
      lines.push(`}`);
    } else if (field.isSecret) {
      lines.push(`if ${varName} != "" {`);
      lines.push(
        `    if config.StoreSecret("${field.flagName}", ${varName}, &${cfgField}) == nil {`,
      );
      lines.push(`        keychainStored = true`);
      lines.push(`    }`);
      lines.push(`}`);
    } else {
      lines.push(`if ${varName} != "" {`);
      lines.push(`    ${cfgField} = ${varName}`);
      lines.push(`}`);
    }
    lines.push(``);
  }

  return lines.join("\n    ");
}
registerTemplateFunc("templateAuthLoginStore", templateAuthLoginStore);

/**
 * Generate credential clearing code for the auth logout command.
 * Clears all security fields from both keyring and config.
 */
function templateAuthLogoutClear(): string {
  const fields = getCLISecurityFields();
  if (fields.length === 0) return "";

  const acc = securityConfigAccessor();
  const lines: string[] = [];

  for (const field of fields) {
    const fieldName = caser().ToPascal(field.field.Name);
    const cfgField = `${acc}${fieldName}`;

    if (field.isSecret) {
      lines.push(`if config.KeyringAvailable() {`);
      lines.push(`    _ = config.DeleteKeyringValue("${field.flagName}")`);
      lines.push(`}`);
    }
    if (field.isArray) {
      lines.push(`${cfgField} = nil`);
    } else {
      lines.push(`${cfgField} = ""`);
    }
  }

  return lines.join("\n    ");
}
registerTemplateFunc("templateAuthLogoutClear", templateAuthLogoutClear);

/**
 * Generate the Short description for the auth login command.
 */
function templateAuthLoginShort(): string {
  return "Interactively configure authentication credentials";
}
registerTemplateFunc("templateAuthLoginShort", templateAuthLoginShort);

/**
 * Generate the Long description for the auth login command.
 */
function templateAuthLoginLong(): string {
  const cliName = sanitizeCliName();
  let desc = `Interactively configure authentication credentials for ${cliName}.\n`;
  desc += `Secret credentials are stored in the OS keychain when available,\n`;
  desc += `with a config file fallback.\n`;
  desc += `\n`;
  desc += `All fields are optional — press Enter to skip any field you don't need.\n`;
  desc += `Use the configure command for both authentication and global parameters.`;
  return desc;
}
registerTemplateFunc("templateAuthLoginLong", templateAuthLoginLong);

/**
 * Generate the non-interactive flag-only store logic.
 * When guided forms are disabled, reads any explicitly-set flags and stores
 * them to config without prompting.
 * Returns the lines for the non-interactive guard block.
 */
function genNonInteractiveSecurityStore(): string[] {
  const fields = getCLISecurityFields();
  if (fields.length === 0) {
    return [nonInteractiveEmptyStoreError()];
  }

  const acc = securityConfigAccessor();
  const lines: string[] = [];

  lines.push(
    `// Non-interactive: store any explicitly-set flags without prompting`,
  );
  lines.push(`changed := false`);

  for (const field of fields) {
    const fieldName = caser().ToPascal(field.field.Name);
    const cfgField = `${acc}${fieldName}`;

    if (field.isArray) {
      addImport("strings");
      lines.push(
        `if f := cmd.Flags().Lookup("${field.flagName}"); f != nil && f.Changed {`,
      );
      lines.push(`    v, _ := cmd.Flags().GetStringSlice("${field.flagName}")`);
      lines.push(`    ${cfgField} = v`);
      lines.push(`    changed = true`);
      lines.push(`}`);
    } else if (field.isSecret) {
      lines.push(
        `if f := cmd.Flags().Lookup("${field.flagName}"); f != nil && f.Changed {`,
      );
      lines.push(`    v, _ := cmd.Flags().GetString("${field.flagName}")`);
      lines.push(
        `    if config.StoreSecret("${field.flagName}", v, &${cfgField}) == nil {`,
      );
      lines.push(`        keychainStored = true`);
      lines.push(`    }`);
      lines.push(`    changed = true`);
      lines.push(`}`);
    } else {
      lines.push(
        `if f := cmd.Flags().Lookup("${field.flagName}"); f != nil && f.Changed {`,
      );
      lines.push(`    v, _ := cmd.Flags().GetString("${field.flagName}")`);
      lines.push(`    ${cfgField} = v`);
      lines.push(`    changed = true`);
      lines.push(`}`);
    }
  }

  lines.push(``);
  lines.push(`if !changed {`);
  lines.push(
    `    return flagutil.WithCLIValidation(fmt.Errorf("no flags provided; use flags to store credentials in %s, or pass --interactive to open the form", config.GetConfigPath()))`,
  );
  lines.push(`}`);

  return lines;
}

function nonInteractiveEmptyStoreError(): string {
  if (isInteractiveAuthEnabled()) {
    return `return flagutil.WithCLIValidation(fmt.Errorf("no flag-configurable values for %s; pass --interactive to open the form", config.GetConfigPath()))`;
  }
  return `return flagutil.WithCLIValidation(fmt.Errorf("no flag-configurable values for %s", config.GetConfigPath()))`;
}

function templatePolicyTestSecurityFlag(): string {
  const fields = getCLISecurityFields();
  return (fields.find((field) => !field.isSecret) || fields[0])?.flagName || "";
}
registerTemplateFunc(
  "templatePolicyTestSecurityFlag",
  templatePolicyTestSecurityFlag,
);

/**
 * Generate the full auth login command body.
 * When multiple OR security alternatives exist, shows a scheme selector first,
 * then only prompts for the selected scheme's fields.
 * When there is only one scheme, shows all fields directly.
 */
// templateHasSecuritySchemeGroups gates the auth-login form machinery: a
// document with no global security has nothing for auth login to store.
function templateHasSecuritySchemeGroups(): boolean {
  return getSecuritySchemeGroups().length > 0;
}
registerTemplateFunc(
  "templateHasSecuritySchemeGroups",
  templateHasSecuritySchemeGroups,
);

function templateAuthLoginBody(): string {
  const groups = getSecuritySchemeGroups();
  if (groups.length === 0) return "";

  const lines: string[] = [];

  // Policy guard: store flags directly, skip all prompts
  lines.push(`if formMode == interactive.FormOff {`);
  lines.push(
    ...genNonInteractiveSecurityStore().map((l) => (l ? `    ${l}` : "")),
  );
  lines.push(`} else {`);
  lines.push(``);

  if (groups.length === 1) {
    // Single scheme — show all fields directly (no selector needed)
    const group = groups[0];
    lines.push(...genSchemeVarDecls(group.fields));
    lines.push(``);
    lines.push(`accessible := formMode == interactive.FormAccessible`);
    lines.push(``);
    lines.push(`// #region custom-auth-vars`);
    lines.push(`// #endregion custom-auth-vars`);
    lines.push(``);
    lines.push(`fields := []*forms.Field{`);
    lines.push(...genSchemeFormFields(group.fields));
    lines.push(`}`);
    lines.push(``);
    lines.push(`// #region custom-auth-groups`);
    lines.push(`// #endregion custom-auth-groups`);
    lines.push(``);
    lines.push(
      `form := forms.New(forms.NewPage("", fields...)).Accessible(accessible).IO(cmd.InOrStdin(), cmd.OutOrStdout())`,
    );
    lines.push(``);
    lines.push(`if err := form.Run(); err != nil {`);
    lines.push(`    return fmt.Errorf("auth login: %w", err)`);
    lines.push(`}`);
    lines.push(``);
    lines.push(...genSchemeStore(group.fields));
  } else {
    // Multiple OR alternatives — show scheme selector first. Both forms read
    // line prompts from one shared reader so piped answers carry over.
    lines.push(`accessible := formMode == interactive.FormAccessible`);
    lines.push(`promptInput := forms.NewReader(cmd.InOrStdin())`);
    lines.push(``);
    lines.push(`var selectedScheme string`);
    lines.push(`schemeSelect := forms.NewSelect(&selectedScheme,`);
    for (const group of groups) {
      lines.push(
        `    forms.NewOption("${escapeGoString(group.label)}", "${
          group.key
        }"),`,
      );
    }
    lines.push(`).`);
    lines.push(`    Title("Authentication Method").`);
    lines.push(`    Description("Choose which credentials to configure")`);
    lines.push(``);
    lines.push(
      `if err := forms.New(forms.NewPage("", schemeSelect)).Accessible(accessible).IO(promptInput, cmd.OutOrStdout()).Run(); err != nil {`,
    );
    lines.push(`    return fmt.Errorf("auth login: %w", err)`);
    lines.push(`}`);
    lines.push(``);
    lines.push(`// #region custom-auth-vars`);
    lines.push(`// #endregion custom-auth-vars`);
    lines.push(``);
    lines.push(`switch selectedScheme {`);

    for (const group of groups) {
      lines.push(`case "${group.key}":`);
      lines.push(...genSchemeVarDecls(group.fields).map((l) => `    ${l}`));
      lines.push(``);
      lines.push(`    fields := []*forms.Field{`);
      lines.push(
        ...genSchemeFormFields(group.fields).map((l) => `        ${l}`),
      );
      lines.push(`    }`);
      lines.push(``);
      lines.push(
        `    form := forms.New(forms.NewPage("", fields...)).Accessible(accessible).IO(promptInput, cmd.OutOrStdout())`,
      );
      lines.push(``);
      lines.push(`    if err := form.Run(); err != nil {`);
      lines.push(`        return fmt.Errorf("auth login: %w", err)`);
      lines.push(`    }`);
      lines.push(``);
      lines.push(...genSchemeStore(group.fields).map((l) => `    ${l}`));
    }

    lines.push(`}`);
    lines.push(``);
    lines.push(`// #region custom-auth-groups`);
    lines.push(`// #endregion custom-auth-groups`);
  }

  lines.push(`}`);
  lines.push(``);

  return lines.join("\n    ");
}

const DEFAULT_OUTPUT_FORMAT_FLAG = "default-output-format";

/**
 * Generate the non-interactive flag-only store logic for the configure command.
 * Handles both security fields AND global parameters.
 * When guided forms are disabled, reads any explicitly-set flags and stores
 * them to config without prompting.
 */
function templateConfigureNonInteractiveStore(): string {
  const secFields = getCLISecurityFields();
  const hasGlob = hasGlobals();

  const lines: string[] = [];
  lines.push(`changed := false`);
  lines.push(
    `if f := cmd.Flags().Lookup("${DEFAULT_OUTPUT_FORMAT_FLAG}"); f != nil && f.Changed {`,
  );
  lines.push(`    switch v := f.Value.String(); v {`);
  lines.push(`    case "":`);
  lines.push(`        cfg.OutputFormat = ""`);
  lines.push(`    case "pretty", "json", "yaml", "table", "toon":`);
  lines.push(`        cfg.OutputFormat = v`);
  lines.push(`    default:`);
  lines.push(
    `        return flagutil.WithCLIValidation(fmt.Errorf("invalid --${DEFAULT_OUTPUT_FORMAT_FLAG} %q; options: pretty, json, yaml, table, toon", v))`,
  );
  lines.push(`    }`);
  lines.push(`    changed = true`);
  lines.push(`}`);

  // Security fields
  if (secFields.length > 0) {
    const acc = securityConfigAccessor();
    for (const field of secFields) {
      const fieldName = caser().ToPascal(field.field.Name);
      const cfgField = `${acc}${fieldName}`;

      if (field.isArray) {
        lines.push(
          `if f := cmd.Flags().Lookup("${field.flagName}"); f != nil && f.Changed {`,
        );
        lines.push(
          `    v, _ := cmd.Flags().GetStringSlice("${field.flagName}")`,
        );
        lines.push(`    ${cfgField} = v`);
        lines.push(`    changed = true`);
        lines.push(`}`);
      } else if (field.isSecret) {
        lines.push(
          `if f := cmd.Flags().Lookup("${field.flagName}"); f != nil && f.Changed {`,
        );
        lines.push(`    v, _ := cmd.Flags().GetString("${field.flagName}")`);
        lines.push(
          `    if config.StoreSecret("${field.flagName}", v, &${cfgField}) == nil {`,
        );
        lines.push(`        keychainStored = true`);
        lines.push(`    }`);
        lines.push(`    changed = true`);
        lines.push(`}`);
      } else {
        lines.push(
          `if f := cmd.Flags().Lookup("${field.flagName}"); f != nil && f.Changed {`,
        );
        lines.push(`    v, _ := cmd.Flags().GetString("${field.flagName}")`);
        lines.push(`    ${cfgField} = v`);
        lines.push(`    changed = true`);
        lines.push(`}`);
      }
    }
  }

  // Global parameters
  if (hasGlob) {
    const globals = context.Global.AST.MainSDK.Globals;
    const acc = globalsConfigAccessor();
    for (const field of globals.Fields) {
      const flagName = sanitizeFlagNameWithReserved(field.Name);
      const fieldName = caser().ToPascal(field.Name);
      const cfgField = `${acc}${fieldName}`;

      lines.push(
        `if f := cmd.Flags().Lookup("${flagName}"); f != nil && f.Changed {`,
      );
      lines.push(`    ${cfgField} = f.Value.String()`);
      lines.push(`    changed = true`);
      lines.push(`}`);
    }
  }

  lines.push(...genConfigureClear());

  lines.push(``);
  lines.push(`if !changed {`);
  lines.push(
    `    return flagutil.WithCLIValidation(fmt.Errorf("no flags provided; use flags to store values in %s, or pass --interactive to open the form", config.GetConfigPath()))`,
  );
  lines.push(`}`);

  return lines.join("\n        ");
}
registerTemplateFunc(
  "templateConfigureNonInteractiveStore",
  templateConfigureNonInteractiveStore,
);

const CONFIGURE_CLEAR_FLAG = "clear";

function configureClearableNames(): string[] {
  return [...storedFlagNames("configure"), DEFAULT_OUTPUT_FORMAT_FLAG];
}

function genConfigureClear(): string[] {
  const options = configureClearableNames().join(", ");
  const lines: string[] = [];
  lines.push(
    `if f := cmd.Flags().Lookup("${CONFIGURE_CLEAR_FLAG}"); f != nil && f.Changed {`,
  );
  lines.push(
    `    names, _ := cmd.Flags().GetStringSlice("${CONFIGURE_CLEAR_FLAG}")`,
  );
  lines.push(`    for _, name := range names {`);
  lines.push(
    `        switch key := strings.TrimPrefix(strings.TrimSpace(name), "--"); key {`,
  );

  const secAcc = securityConfigAccessor();
  for (const field of getCLISecurityFields()) {
    const cfgField = `${secAcc}${caser().ToPascal(field.field.Name)}`;
    lines.push(`        case "${field.flagName}":`);
    if (field.isSecret) {
      lines.push(`            if config.KeyringAvailable() {`);
      lines.push(
        `                _ = config.DeleteKeyringValue("${field.flagName}")`,
      );
      lines.push(`            }`);
    }
    lines.push(`            ${cfgField} = ${field.isArray ? "nil" : `""`}`);
  }
  if (hasGlobals()) {
    const globAcc = globalsConfigAccessor();
    for (const field of context.Global.AST.MainSDK.Globals.Fields) {
      lines.push(`        case "${sanitizeFlagNameWithReserved(field.Name)}":`);
      lines.push(`            ${globAcc}${caser().ToPascal(field.Name)} = ""`);
    }
  }
  lines.push(`        case "${DEFAULT_OUTPUT_FORMAT_FLAG}":`);
  lines.push(`            cfg.OutputFormat = ""`);
  lines.push(`        default:`);
  lines.push(
    `            return flagutil.WithCLIValidation(fmt.Errorf("invalid --${CONFIGURE_CLEAR_FLAG} %q; options: ${escapeGoString(
      options,
    )}", name))`,
  );
  lines.push(`        }`);
  lines.push(
    `        cleared = append(cleared, strings.TrimPrefix(strings.TrimSpace(name), "--"))`,
  );
  lines.push(`    }`);
  lines.push(`    changed = true`);
  lines.push(`}`);
  return lines;
}

registerTemplateFunc("configureClearFlag", () => CONFIGURE_CLEAR_FLAG);
registerTemplateFunc("templateConfigureClearOptions", () =>
  escapeGoString(configureClearableNames().join(", ")),
);

registerTemplateFunc(
  "defaultOutputFormatFlag",
  () => DEFAULT_OUTPUT_FORMAT_FLAG,
);

function goFlagNameArgs(names: string[]): string {
  return names.map((n) => `"${n}"`).join(", ");
}

function storedFlagNames(command: string): string[] {
  const names = getCLISecurityFields().map((f) => f.flagName);
  if (command === "configure" && hasGlobals()) {
    for (const field of context.Global.AST.MainSDK.Globals.Fields) {
      names.push(sanitizeFlagNameWithReserved(field.Name));
    }
  }
  return names;
}

function templateAuthLoginFlagNames(): string {
  return goFlagNameArgs(storedFlagNames("login"));
}
registerTemplateFunc("templateAuthLoginFlagNames", templateAuthLoginFlagNames);

function templateConfigureFlagNames(): string {
  return goFlagNameArgs([
    ...storedFlagNames("configure"),
    DEFAULT_OUTPUT_FORMAT_FLAG,
    CONFIGURE_CLEAR_FLAG,
  ]);
}
registerTemplateFunc("templateConfigureFlagNames", templateConfigureFlagNames);

function templateSettingsHelpFlags(command: string): string {
  return escapeGoString(storedFlagNames(command).join(","));
}
registerTemplateFunc("templateSettingsHelpFlags", templateSettingsHelpFlags);

function primarySchemeFields(): CLISecurityFieldInfo[] {
  const security = context.Global.AST.MainSDK.Security;
  const fields = getCLISecurityFields();
  const primary = primaryCLISecurityField(fields);
  if (!security || !primary) return [];
  const scalars: FieldDef[] = [];
  for (const top of security.Type.Fields || []) {
    if (top.Const) continue;
    const kind = top.Type.Type.toString();
    if (kind === "class" || kind === "union") {
      const scheme = flattenCLISecurityObject({ Fields: [top] } as TypeDef);
      if (scheme.some((f) => f.flagName === primary.flagName)) return scheme;
    } else {
      scalars.push(top);
    }
  }
  const flat = flattenCLISecurityObject({ Fields: scalars } as TypeDef);
  const self = flat.find((f) => f.flagName === primary.flagName);
  if (!self) return [primary];
  if (self.composite) return flat.filter((f) => f.composite);
  if (!self.schemeKey) return [primary];
  return flat.filter((f) => f.schemeKey === self.schemeKey);
}

function templateSettingsHelpExample(commandPath: string): string {
  const cliName = sanitizeCliName();
  const lines = [`  ${cliName} ${commandPath}`];
  const scheme = primarySchemeFields();
  const prefix = context.Global.Config.EnvVarPrefix
    ? String(context.Global.Config.EnvVarPrefix).toUpperCase()
    : "";
  if (scheme.length > 0 && scheme.length <= 2) {
    const flags = scheme
      .map(
        (f) =>
          `--${f.flagName} "$${
            prefix ? `${prefix}_${f.envVarSuffix}` : f.envVarSuffix
          }"`,
      )
      .join(" ");
    lines.push(`  ${cliName} ${commandPath} ${flags}`);
  }
  return escapeGoString(lines.join("\n"));
}
registerTemplateFunc(
  "templateSettingsHelpExample",
  templateSettingsHelpExample,
);

function templateSettingsHelpSeeAlso(command: string): string {
  const paths: string[] = [];
  if (command !== "configure") paths.push("configure");
  if (hasConfigurableSettings()) paths.push("whoami");
  if (isInteractiveAuthEnabled() && hasGlobalSecurity()) {
    paths.push("auth logout");
  }
  return escapeGoString(paths.join(","));
}
registerTemplateFunc(
  "templateSettingsHelpSeeAlso",
  templateSettingsHelpSeeAlso,
);

/** Generate var declarations for a set of scheme fields */
function genSchemeVarDecls(fields: CLISecurityFieldInfo[]): string[] {
  return fields.map((f) => `var auth${caser().ToPascal(f.field.Name)} string`);
}

/**
 * Build the Description() string for a form field.
 * Includes the flag name and appends example from the spec if available.
 */
function buildFormDescription(
  flagName: string,
  field: FieldDef,
  suffix?: string,
): string {
  let desc = `--${flagName}`;
  if (suffix) {
    desc += ` ${suffix}`;
  }
  // @ts-ignore — Example is a dynamic Go proxy field not in TS type defs
  const example = field.Example?.Value;
  if (example !== undefined && example !== null && String(example) !== "") {
    desc += ` (e.g., ${escapeGoString(String(example))})`;
  }
  return desc;
}

/**
 * Build the Placeholder() Go expression for a form field.
 * Falls back to the field's default value when the stored config value is empty.
 */
function buildFormPlaceholder(configExpr: string, field: FieldDef): string {
  const defaultVal = field.Default?.Value;
  if (
    defaultVal !== undefined &&
    defaultVal !== null &&
    String(defaultVal) !== ""
  ) {
    addImport("cmp");
    return `cmp.Or(${configExpr}, "${escapeGoString(String(defaultVal))}")`;
  }
  return configExpr;
}

/**
 * Render a Go builder chain as one composite-literal element:
 * `head.\n    call1.\n    call2,`.
 */
function goChain(head: string, calls: string[]): string[] {
  return [
    `${head}.`,
    ...calls.map((c, i) => `    ${c}${i === calls.length - 1 ? "," : "."}`),
  ];
}

/** Generate form fields for a set of scheme fields */
function genSchemeFormFields(fields: CLISecurityFieldInfo[]): string[] {
  const acc = securityConfigAccessor();
  const lines: string[] = [];

  for (const field of fields) {
    const varName = `auth${caser().ToPascal(field.field.Name)}`;
    const cfgExpr = `${acc}${caser().ToPascal(field.field.Name)}`;
    const calls = [`Title("${escapeGoString(field.description)}")`];

    if (field.isArray) {
      calls.push(
        `Description("${buildFormDescription(
          field.flagName,
          field.field,
          "(comma-separated)",
        )}")`,
      );
    } else if (field.isSecret) {
      calls.push(
        `Description("${buildFormDescription(field.flagName, field.field)}")`,
        `Password()`,
        `Placeholder(maskSecret(${buildFormPlaceholder(
          `config.GetStoredSecret("${field.flagName}", ${cfgExpr})`,
          field.field,
        )}))`,
      );
    } else {
      calls.push(
        `Description("${buildFormDescription(field.flagName, field.field)}")`,
        `Placeholder(${buildFormPlaceholder(cfgExpr, field.field)})`,
      );
    }
    lines.push(...goChain(`forms.NewInput(&${varName})`, calls));
  }

  return lines;
}

/** Generate credential storage code for a set of scheme fields */
function genSchemeStore(fields: CLISecurityFieldInfo[]): string[] {
  // Add strings import if any field is an array
  if (fields.some((f) => f.isArray)) {
    addImport("strings");
  }

  const acc = securityConfigAccessor();
  const lines: string[] = [];

  for (const field of fields) {
    const varName = `auth${caser().ToPascal(field.field.Name)}`;
    const fieldName = caser().ToPascal(field.field.Name);
    const cfgField = `${acc}${fieldName}`;

    if (field.isArray) {
      lines.push(`if ${varName} != "" {`);
      lines.push(
        `    ${cfgField} = strings.Split(strings.TrimSpace(${varName}), ",")`,
      );
      lines.push(`}`);
    } else if (field.isSecret) {
      lines.push(`if ${varName} != "" {`);
      lines.push(
        `    if config.StoreSecret("${field.flagName}", ${varName}, &${cfgField}) == nil {`,
      );
      lines.push(`        keychainStored = true`);
      lines.push(`    }`);
      lines.push(`}`);
    } else {
      lines.push(`if ${varName} != "" {`);
      lines.push(`    ${cfgField} = ${varName}`);
      lines.push(`}`);
    }
    lines.push(``);
  }

  return lines;
}

registerTemplateFunc("templateAuthLoginBody", templateAuthLoginBody);

// ─── Configure Form Helpers ──────────────────────────────────────────────────
// These generate form fields for the configure command when
// interactiveAuth is enabled. They cover both security AND global parameters.

/**
 * Generate var declarations for global parameter form fields.
 */
function templateConfigureGlobalVarDeclarations(): string {
  if (!hasGlobals()) return "";

  const globals = context.Global.AST.MainSDK.Globals;
  const lines: string[] = [];

  for (const field of globals.Fields) {
    const varName = `cfgGlobal${caser().ToPascal(field.Name)}`;
    lines.push(`var ${varName} string`);
  }

  return lines.join("\n    ");
}
registerTemplateFunc(
  "templateConfigureGlobalVarDeclarations",
  templateConfigureGlobalVarDeclarations,
);

/**
 * Generate form fields for global parameters.
 */
function templateConfigureGlobalFormFields(): string {
  if (!hasGlobals()) return "";

  const globals = context.Global.AST.MainSDK.Globals;
  const acc = globalsConfigAccessor();
  const lines: string[] = [];

  for (const field of globals.Fields) {
    const fieldName = caser().ToPascal(field.Name);
    const varName = `cfgGlobal${fieldName}`;
    const cfgExpr = `${acc}${fieldName}`;
    const desc =
      field.Comments?.Summary?.trim() ||
      field.Comments?.Description?.trim()?.split(/[.\n]/)[0]?.trim() ||
      `Global ${sanitizeFlagName(field.Name)} parameter`;
    const flagName = sanitizeFlagName(field.Name);

    lines.push(
      ...goChain(`forms.NewInput(&${varName})`, [
        `Title("${escapeGoString(desc)}")`,
        `Description("${buildFormDescription(flagName, field)}")`,
        `Placeholder(${buildFormPlaceholder(cfgExpr, field)})`,
      ]),
    );
  }

  return lines.join("\n        ");
}
registerTemplateFunc(
  "templateConfigureGlobalFormFields",
  templateConfigureGlobalFormFields,
);

/**
 * Generate store code for global parameters after form submission.
 */
function templateConfigureGlobalStore(): string {
  if (!hasGlobals()) return "";

  const globals = context.Global.AST.MainSDK.Globals;
  const acc = globalsConfigAccessor();
  const lines: string[] = [];

  for (const field of globals.Fields) {
    const fieldName = caser().ToPascal(field.Name);
    const varName = `cfgGlobal${fieldName}`;
    const cfgField = `${acc}${fieldName}`;

    lines.push(`if ${varName} != "" {`);
    lines.push(`    ${cfgField} = ${varName}`);
    lines.push(`}`);
    lines.push(``);
  }

  return lines.join("\n    ");
}
registerTemplateFunc(
  "templateConfigureGlobalStore",
  templateConfigureGlobalStore,
);
