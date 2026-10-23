// =============================================================================
// Declarative intent commands (x-speakeasy-cli-commands)
//
// The OpenAPI document is the machine-readable manifest: declared commands
// render as generated Cobra commands. The decoder has already normalized the
// manifest into IR (source discrimination, singular-path binds lowered to
// RFC 6901 pointers, presets validated against the variant schema), so this
// layer only maps IR onto the generated command tree:
//   - source.type "operation": a preset over an existing operation, emitted
//     into that operation's command package so it reuses the generated flag
//     metadata and run function (auth, dry-run, streaming, output plumbing).
//   - source.type "planned": a visible placeholder that documents the intent
//     surface before its backing API lands (RunE returns the declared note).
//   - source.type "group": tags an existing command group into a help
//     category (cobra command group).
// Declared commands keep their manifest (document) order in help output via a
// deterministic reorder pass (cobra alphabetizes otherwise).
// =============================================================================

interface IntentBodyEntry {
  Key: string; // top-level body key (from a single-segment JSON Pointer)
  Path?: string[]; // [Key, field] for a flag bound beneath an object preset
  Name?: string; // flag name
  SatisfiedBy?: string[];
  Shorthand?: string;
  Summary?: string;
  Type?: string; // string | int | float | bool (string when absent)
  Variadic?: boolean; // positional consumes the remaining tokens
  Required?: boolean; // enforced when the body is built from inputs
  PresetCovered?: boolean; // a preset already satisfies the bound pointer
  DefaultValue?: any; // displayed default (preset, declared, or schema)
  HasDefault?: boolean;
  Suggestions?: string[]; // full suggestion list for machine-readable surfaces
  RouteIDs?: string[]; // dispatch routes that declare this body key
  RequiredRouteIDs?: string[]; // routes on which this input remains required
  Group?: string; // route-specific help group ("Model variant")
  GroupOrder?: string; // first route index represented by the help group
  Positional?: boolean;
}

interface IntentDispatchRoute {
  ID: string;
  Label: string;
  SelectorFlag: string;
  Default: boolean;
  PresetJSON: string;
  VariantLabel: string;
  ForeignSelectors: string[];
  PresetMergePoints: IntentPresetMergePoint[];
}

// Pointer at which a preset object fills a caller object (flagutil.PresetMergePoint).
interface IntentPresetMergePoint {
  Pointer: string;
  Key: string;
  ValuesJSON: string[];
}

interface IntentDispatchKey {
  BodyKey: string;
  RouteIDs: string[];
  UnroutedVariants: string[];
}

interface IntentAsyncParameter {
  In: string;
  Name: string;
  Expected: string;
}

interface IntentCmdCtx {
  ID: string;
  Use: string;
  FuncName: string; // Pascal, e.g. "Generate"
  PkgPath: string; // slash package path under internal/cli ("" = root)
  PkgName: string; // Go package identifier ("cli" for the root)
  ParentPath: string[]; // command path of the parent ([] = root)
  Category: string;
  Summary: string;
  Description: string;
  Example: string;
  HelpDefaults: string;
  HelpLearn: string;
  HelpEscalate: string;
  MetaVar: string;
  SecurityFlags: string; // per-operation security flag registration snippet
  HasMeta: boolean; // whether the operation registers flag metadata
  BodyFlag: string; // flag the run function reads the JSON body from ("" = no body)

  BodyFieldPath: string; // metadata field path identifying request-body fields
  HasBody: boolean; // the backing operation carries a request body
  RunFunc: string;
  ExecuteFunc: string;
  OpID: string;
  HasSchema: boolean;
  MinArgs: number;
  Args: IntentBodyEntry[];
  ArgsHelp: string; // "Arguments:" section lines for the help page
  Flags: IntentBodyEntry[];
  PromptFlags: IntentBodyEntry[]; // required declared flags a preset does not cover (interactive prompting)
  Implied: { Key: string; Values: string }[]; // flagutil.ImpliedBody rules from flag implies:
  PresetJSON: string; // JSON object merged into the body before user inputs
  // Partial-body preset merge (flagutil.PresetMerge): how the pinned request
  // variant is told apart from its union siblings, so a caller-supplied
  // --body keeps the presets instead of falling back to the union default.
  CommandDisplay: string; // "image" / "jobs cancel"
  VariantLabel: string; // pinned component schema name ("" when not a union)
  ForeignSelectors: string[]; // keys that select another variant
  DiscriminatorKey: string;
  DiscriminatorValueJSON: string; // JSON text of the pinned discriminator value ("" when unknown)
  DiscriminatorAliasesJSON: string[]; // JSON text of every value that selects the pinned variant
  PresetMergePoints: IntentPresetMergePoint[];
  EscapeCommand: string; // "<cli> agent run": full-control invocation for conflict errors
  HasPresetMerge: boolean; // a caller-supplied body needs merging/checking (presets or variant selectors exist)
  HintsJSON: string; // JSON object of reason → hint lines (agent envelope)
  JQ: string;
  ArtifactJSON: string; // artifact runtime config carried via cobra annotation
  ArtifactKind: string; // image | audio | video (help text)
  ArtifactDefaultPath: string; // default filename pattern (help text)
  ArtifactResponseCode: string; // linked 2xx status code (harness tests)
  AsyncJSON: string; // polling runtime config carried via cobra annotation
  PollOp: Operation | null; // typed poll call rendered into the intent package
  PollSDKAccessor: string;
  AsyncParameterIn: string;
  AsyncParameterName: string;
  AsyncParams: IntentAsyncParameter[];
  AsyncResume: string;
  AsyncCreateResponseCode: string;
  AsyncResponseCode: string;
  AsyncCreateTestJSON: string;
  AsyncPendingTestJSON: string;
  AsyncSuccessTestJSON: string;
  AsyncFailureTestJSON: string;
  AsyncHandoffTestJSON: string;
  AsyncUnknownTestJSON: string;
  AsyncMissingStateTestJSON: string;
  AsyncMissingHandleTestJSON: string;
  AsyncFailureHint: string;
  StreamMetadataJSON: string;
  StreamSelect: string; // RFC 6901 pointer of output.stream.select ("" = none)
  StreamToggle: { Name: string; Key: string } | null;
  StreamKind: string; // "sse" | "jsonl" | "" — how the backing operation streams
  StreamSentinel: string; // SSE end-of-stream sentinel declared on the response ("" = none)
  OpHasRequiredParams: boolean; // backing operation has required path/query/header parameters
  Dispatch: boolean;
  Override: boolean;
  DispatchRoutes: IntentDispatchRoute[];
  DispatchInputs: IntentBodyEntry[];
  DispatchKeys: IntentDispatchKey[];
  VariantHelp: string;
}

interface PlannedCmdCtx {
  Name: string;
  ParentPath: string[];
  Category: string;
  Summary: string;
  Tagline: string;
  Description: string;
  Note: string;
  Custom: boolean;
  HelpDefaults: string;
  HelpLearn: string;
  HelpEscalate: string;
}

interface GroupTagCtx {
  Name: string;
  Category: string;
  HelpDefaults: string;
  HelpLearn: string;
  HelpEscalate: string;
}

interface IntentOrderEntry {
  ParentPath: string[];
  Name: string;
}

interface IntentManifestCtx {
  Categories: string[];
  Bound: IntentCmdCtx[]; // operation-backed (rendered per group package)
  Planned: PlannedCmdCtx[];
  GroupTags: GroupTagCtx[];
  Order: IntentOrderEntry[]; // every declared command, in manifest order
}

interface CLIOperationFlagCtx {
  Name: string;
  Shorthand: string;
  Summary: string;
  Type: string;
  Kind: string;
  Key: string;
  DefaultLiteral: string;
  DefaultValue: any;
  HasDefault: boolean;
  DefaultResolves: boolean;
  Suggestions: string[];
}

interface CLIOperationCtx {
  OperationID: string;
  StreamMetadataJSON: string;
  StreamSelect: string;
  Flags: CLIOperationFlagCtx[];
  BodyFlags: string[];
  CanonicalBodyFlag: string;
  BodyRequired: boolean;
}

// Shell-safe, visibly synthetic value for generated fallback examples. Keep
// the angle-bracket convention while preventing command substitution or
// variable expansion inside the double-quoted token.
function intentExamplePlaceholder(name: string): string {
  return intentExampleQuoted(`<${name}>`);
}

function intentExampleQuoted(raw: string): string {
  const value = raw
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\$/g, "\\$")
    .replace(/`/g, "\\`");
  return `"${value}"`;
}

// Intent commands reuse their backing operation's parameter flags. Include
// every required parameter in a synthesized example even though those flags
// are not declared as intent body bindings in the manifest.
function intentRequiredParamFallbacks(op: Operation): string[] {
  if (!op.Request?.Params) return [];
  const allParams = [
    ...(op.Request.Params.PathParams || []),
    ...(op.Request.Params.QueryParams || []),
    ...(op.Request.Params.HeaderParams || []),
  ];
  return allParams
    .filter((param) => !param.Field.Const && !param.Field.Optional)
    .map((param) => {
      const flagName = sanitizeFlagNameWithReserved(param.Field.Name);
      let value: any;
      try {
        if (param.Examples?.length > 0) {
          const example = findExampleByName(param.Examples, "");
          if (example) value = getExampleValue(example);
        }
      } catch (_e) {
        // Fall through to the field-level/default/type-derived example.
      }
      if (!hasExampleValue(param.Field, value)) {
        // getCLIExampleValue returns a rendered value plus a marker for the
        // generic angle-bracket placeholder; name that placeholder after the
        // flag so the fallback stays visibly synthetic but specific.
        const example = getCLIExampleValue(param.Field);
        const placeholder = `<${flagName}>`;
        value = !example.SynthesizedAnglePlaceholder
          ? example.Value
          : isStringOrEnumItemArrayField(param.Field) &&
            !isRepeatableFlagField(param.Field)
          ? JSON.stringify([placeholder])
          : placeholder;
      }
      if (typeof value === "boolean") return `--${flagName}=${value}`;
      const repeatable = exampleRepeatableFlagValue(param.Field, value, false);
      if (repeatable) {
        return repeatable.Value.map(
          (element) => `--${flagName} ${intentExampleQuoted(element)}`,
        ).join(" ");
      }
      return `--${flagName} ${intentExampleQuoted(exampleScalarValue(value))}`;
    });
}

function intentFallbackFlag(entry: IntentBodyEntry): string {
  if (entry.Type === "bool") return `--${entry.Name}=true`;
  if (entry.Type === "int" || entry.Type === "float") {
    return `--${entry.Name} 1`;
  }
  return `--${entry.Name} ${intentExamplePlaceholder(entry.Name || "value")}`;
}

function intentImpliedRules(flags: any[]): { Key: string; Values: string }[] {
  const rules: { Key: string; Values: string }[] = [];
  for (const f of flags) {
    const implies: any[] = f.Implies || [];
    const key = intentPointerKey(f.Bind?.Pointer || "");
    if (!implies.length || !key) continue;
    const values: Record<string, any> = Object.create(null);
    for (const implied of implies) {
      values[intentPointerKey(implied.Bind?.Pointer || "")] = implied.Value;
    }
    rules.push({ Key: key, Values: JSON.stringify(values) });
  }
  return rules;
}

// Single-segment RFC 6901 pointer → top-level body key. Args, presets and
// dispatch keys bind single-segment pointers only, so a miss here is
// unreachable; flags may nest and use intentPointerPath.
function intentPointerKey(pointer: string): string {
  if (!pointer || !pointer.startsWith("/")) return "";
  const rest = pointer.slice(1);
  if (rest.includes("/")) return "";
  return rest.replace(/~1/g, "/").replace(/~0/g, "~");
}

function intentHasOwn(obj: object, key: string): boolean {
  return Object.prototype.hasOwnProperty.call(obj, key);
}

// RFC 6901 pointer → decoded body path segments.
function intentPointerPath(pointer: string): string[] {
  if (!pointer || !pointer.startsWith("/")) return [];
  return pointer
    .slice(1)
    .split("/")
    .map((token) => token.replace(/~1/g, "/").replace(/~0/g, "~"));
}

function mergeIntentTestValue(left: any, right: any): any {
  if (Array.isArray(left) && Array.isArray(right)) {
    const merged = [...left];
    for (let i = 0; i < right.length; i++) {
      if (right[i] === undefined || right[i] === null) continue;
      merged[i] =
        merged[i] === undefined || merged[i] === null
          ? right[i]
          : mergeIntentTestValue(merged[i], right[i]);
    }
    return merged;
  }
  if (
    left &&
    right &&
    typeof left === "object" &&
    typeof right === "object" &&
    !Array.isArray(left) &&
    !Array.isArray(right)
  ) {
    const merged = { ...left };
    for (const [key, value] of Object.entries(right)) {
      merged[key] =
        key in merged ? mergeIntentTestValue(merged[key], value) : value;
    }
    return merged;
  }
  return right;
}

function foldIntentTestPath(segments: any[], terminal: any): any {
  let node = terminal;
  for (const segment of [...(segments || [])].reverse()) {
    if (segment.IsIndex) {
      const array = new Array(Number(segment.Index) + 1).fill(null);
      array[Number(segment.Index)] = node;
      node = array;
    } else {
      node = { [segment.Field]: node };
    }
  }
  return node;
}

function foldIntentArtifactPath(segments: any[], terminal: any): any {
  let node = terminal;
  for (const segment of [...(segments || [])].reverse()) {
    node = segment.wild ? [node] : { [segment.field]: node };
  }
  return node;
}

interface IntentOpLocation {
  op: Operation;
  pkgPath: string; // slash-joined package path under internal/cli ("" = root)
  pkgName: string; // Go package identifier ("cli" for the root)
  sdkAccessor: string; // generated SDK field chain ("" = root SDK)
}

function findIntentOperation(opID: string): IntentOpLocation | null {
  const main = context.Global.AST.MainSDK;
  let fallback: IntentOpLocation | null = null;
  const match = (
    op: Operation,
    pkgPath: string,
    pkgName: string,
    sdkAccessor: string,
  ): IntentOpLocation | null => {
    const found = { op, pkgPath, pkgName, sdkAccessor };
    if (op.GetID() === opID) return found;
    if (fallback === null && op.OriginalID === opID) fallback = found;
    return null;
  };
  for (const op of main.Operations || []) {
    const found = match(op, "", "cli", "");
    if (found) return found;
  }
  const walk = (
    sdk: SDK,
    segments: string[],
    sdkAccessor: string,
  ): IntentOpLocation | null => {
    for (const op of sdk.Operations || []) {
      const found = match(
        op,
        segments.join("/"),
        segments[segments.length - 1],
        sdkAccessor,
      );
      if (found) return found;
    }
    for (const child of sdk.SubSDKs || []) {
      const found = walk(
        child,
        [...segments, sanitizeCLIPkgName(child.Type.Name)],
        `${sdkAccessor}.${sanitizeSDKFieldName(child.FieldName)}`,
      );
      if (found) return found;
    }
    return null;
  };
  for (const sub of main.SubSDKs || []) {
    const found = walk(
      sub,
      [sanitizeCLIPkgName(sub.Type.Name)],
      `.${sanitizeSDKFieldName(sub.FieldName)}`,
    );
    if (found) return found;
  }
  return fallback;
}

// Command paths of every generated sub-SDK group (space-joined, including
// nested groups with their de-stuttered names) that can host nested declared
// commands.
function generatedGroupPaths(): Set<string> {
  const paths = new Set<string>();
  const main = context.Global.AST.MainSDK;
  const walk = (sdk: SDK, parentName: string, prefix: string[]) => {
    const sdkGroupName = getSDKGroupName(sdk);
    if (!sdkGroupName) return;
    const name = getDeStutteredCommandName(parentName, sdkGroupName);
    const path = [...prefix, name];
    paths.add(path.join(" "));
    for (const child of sdk.SubSDKs || []) {
      walk(child, sdkGroupName, path);
    }
  };
  for (const sub of main.SubSDKs || []) {
    walk(sub, "", []);
  }
  return paths;
}

// Validates that a nested declared command's parent chain resolves: every
// proper prefix must be another declared command or a generated top-level
// group. A typo here must fail generation, not vanish from the CLI.
function validateIntentParents(manifest: any): void {
  const declared = new Set<string>();
  for (const cmd of manifest.Commands || []) {
    declared.add((cmd.Path || []).join(" "));
  }
  const generated = generatedGroupPaths();
  for (const cmd of manifest.Commands || []) {
    const path: string[] = cmd.Path || [];
    for (let depth = 1; depth < path.length; depth++) {
      const prefix = path
        .slice(0, depth)
        .map((seg: string) => sanitizeCLICommand(seg))
        .join(" ");
      if (declared.has(path.slice(0, depth).join(" "))) continue;
      if (generated.has(prefix)) continue;
      throw new Error(
        `x-speakeasy-cli-commands: command "${path.join(
          " ",
        )}" nests beneath "${prefix}", ` +
          `which is neither a declared command nor a generated command group ` +
          `(generated groups: ${[...generated].sort().join(", ") || "none"})`,
      );
    }
  }
}

// Validates that declared flag names do not collide with the generated flags
// of the backing operation (non-body parameters), which would panic
// at cobra registration time.
function validateIntentFlagNames(
  cmdKey: string,
  cmd: any,
  op: Operation,
  bodyFlag: string,
) {
  const generatedFlags = new Set<string>();
  try {
    const metadata = bodyFlag
      ? collectOperationNonBodyFlagMeta(op)
      : collectOperationFlagMeta(op);
    for (const flagMeta of metadata) {
      generatedFlags.add(flagMeta.flagName);
    }
  } catch {
    // Metadata differences do not hide collisions with the explicit body
    // surfaces computed by the same operation renderer below.
  }
  if (bodyFlag) generatedFlags.add(bodyFlag);

  for (const flag of cmd.Flags || []) {
    if (generatedFlags.has(flag.Name)) {
      throw new Error(
        `x-speakeasy-cli-commands: command "${cmdKey}" declares flag "--${flag.Name}", ` +
          `which collides with a generated flag of operation ${cmd.Source.Routes[0].OperationID}; ` +
          `rename the declared flag`,
      );
    }
  }
  // Auto-assigned shorthands mirror templateFlagMetadataVar's post-pass
  // exactly (operationAutoShorthandOwners shares its entry traversal). A
  // declared shorthand matching one on a registered flag panics pflag at
  // startup, so it must be a generation error (the README promises flag
  // collisions fail generation). Intent commands register only the
  // NonBodyMeta subset, so owners are restricted to it.
  const autoShorthandOwners = operationAutoShorthandOwners(
    op,
    Boolean(bodyFlag),
  );
  for (const flag of cmd.Flags || []) {
    if (!flag.Shorthand) continue;
    const owner = autoShorthandOwners.get(flag.Shorthand);
    if (owner) {
      throw new Error(
        `x-speakeasy-cli-commands: command "${cmdKey}" declares shorthand "-${flag.Shorthand}" on flag "--${flag.Name}", ` +
          `which collides with the auto-assigned shorthand of generated flag "--${owner}" on operation ${cmd.Source.Routes[0].OperationID}; ` +
          `choose a different shorthand`,
      );
    }
  }
  // The artifact runtime registers --out and --raw-response on the command;
  // a backing operation whose generated flags claim those names would panic
  // at cobra registration time.
  if (cmd.Output?.Artifact) {
    for (const owned of ["out", "raw-response"]) {
      if (generatedFlags.has(owned)) {
        throw new Error(
          `x-speakeasy-cli-commands: command "${cmdKey}" declares output.artifact, ` +
            `which registers the --${owned} flag, but operation ${cmd.Source.Routes[0].OperationID} ` +
            `already generates a flag with that name`,
        );
      }
    }
  }
  if (cmd.Async) {
    for (const owned of ["async", "resume", "poll-interval", "poll-timeout"]) {
      if (generatedFlags.has(owned)) {
        throw new Error(
          `x-speakeasy-cli-commands: command "${cmdKey}" declares async, ` +
            `which registers the --${owned} flag, but operation ${cmd.Source.Routes[0].OperationID} ` +
            `already generates or uses a flag with that name`,
        );
      }
    }
  }
}

// Value suffix for a declared input: one parenthetical holding the
// suggestions (a declared list in full, else the first four schema enum
// values) and, when withDefault is set, the default last. The default never
// repeats inside the suggestions, whether it is appended here or printed by
// Cobra from the registered flag value. These are suggestions, never
// validation: the server owns the value space.
function intentValueSuffix(input: any, withDefault: boolean): string {
  const hasDefault = input.Default !== undefined && input.Default !== null;
  const defaultText = hasDefault ? `${input.Default}` : null;
  const values = (input.Enum || [])
    .map((v: any) => `${v}`)
    .filter((v: string) => v !== defaultText);
  const declared = Boolean(input.SuggestionsDeclared);
  const shown = declared ? values : values.slice(0, 4);
  const more = !declared && values.length > 4 ? ", ..." : "";
  const parts: string[] = [];
  if (shown.length > 0) parts.push(`e.g. ${shown.join(", ")}${more}`);
  if (withDefault && defaultText !== null) {
    parts.push(`default: ${defaultText}`);
  }
  return parts.length > 0 ? ` (${parts.join(", ")})` : "";
}

function intentFlagHelp(input: any): string {
  return `${input.Summary || ""}${intentValueSuffix(input, true)}`.trim();
}

function compactDefaultValue(value: any): string | null {
  if (Array.isArray(value)) {
    if (
      value.some(
        (item: any) =>
          item === undefined || item === null || typeof item === "object",
      )
    ) {
      return null;
    }
    if (value.length === 0) return "[]";
    return value
      .map((item: any) => {
        const rendered = String(item).replace(/\s+/g, " ").trim();
        return rendered === "" && typeof item === "string" ? `""` : rendered;
      })
      .join(",");
  }
  if (value === undefined || value === null || typeof value === "object") {
    return null;
  }
  const rendered = String(value).replace(/\s+/g, " ").trim();
  return rendered === "" && typeof value === "string" ? `""` : rendered;
}

// Explicit prose wins. Otherwise show values that describe the request when
// the caller omits an input: presets first in manifest order, then declared
// flags that explicitly asked to surface a scalar schema default. Authored
// `default:` values remain inline display hints and are intentionally absent.
function intentHelpDefaults(cmd: any): string {
  if (cmd.Help?.Defaults) return cmd.Help.Defaults;

  const entries: string[] = [];
  const presetPointers = new Set<string>();
  for (const preset of cmd.Presets || []) {
    const pointer = preset.Bind?.Pointer || "";
    const key = intentPointerKey(pointer);
    const value = compactDefaultValue(preset.Value);
    if (!key || value === null) continue;
    entries.push(`${key} ${value}`);
    presetPointers.add(pointer);
  }
  for (const flag of cmd.Flags || []) {
    const pointer = flag.Bind?.Pointer || "";
    if (flag.DefaultFrom !== "schema" || presetPointers.has(pointer)) {
      continue;
    }
    const value = compactDefaultValue(flag.Default);
    if (value === null) continue;
    entries.push(`${sanitizeFlagNameWithReserved(flag.Name)} ${value}`);
  }
  return entries.join(" · ");
}

function intentPresetJSON(presets: any[]): string {
  const obj: Record<string, any> = {};
  for (const preset of presets || []) {
    const key = intentPointerKey(preset.Bind?.Pointer || "");
    if (!key || preset.Value === undefined || preset.Value === null) continue;
    obj[key] = preset.Value;
  }
  return Object.keys(obj).length > 0 ? JSON.stringify(obj) : "";
}

function intentPresetMergePoints(
  discriminators: any[],
): IntentPresetMergePoint[] {
  return (discriminators || [])
    .map((d: any) => ({
      Pointer: d.Pointer || "",
      Key: d.Key || "",
      ValuesJSON: (d.Values || []).map((v: any) => JSON.stringify(v)),
    }))
    .filter((d) => d.Pointer !== "");
}

function intentRouteGroup(routeIDs: string[], routes: any[]): string {
  if (routeIDs.length === 0 || routeIDs.length === routes.length) return "";
  const labels = routes
    .filter((route: any) => routeIDs.includes(route.ID))
    .map((route: any) => route.Label);
  if (labels.length === 1) return `${labels[0]} variant`;
  return `${labels.join(" / ")} variants`;
}

function intentRouteGroupOrder(routeIDs: string[], routes: any[]): string {
  if (routeIDs.length === 0 || routeIDs.length === routes.length) return "";
  const indices = routeIDs
    .map((id) => routes.findIndex((route: any) => route.ID === id))
    .filter((index) => index >= 0);
  return indices.length > 0 ? String(Math.min(...indices)) : "";
}

function intentOperationIsPromoted(op: Operation): boolean {
  const chain = getOwningSDKChain(op);
  if (chain.length === 0 || hasNameOverride(op)) return false;
  const owning = chain[chain.length - 1];
  return getStutterKind(getSDKGroupName(owning), op.GetID()) === "exact";
}

// Validated canonical command paths whose generated operation registration
// is replaced by an exact-path dispatch intent. The operation file is still
// emitted so the intent can reuse its metadata and run function.
let overriddenOperationIDsManifest: any = null;
let overriddenOperationIDsCache: Set<string> | null = null;
function overriddenOperationIDs(): Set<string> {
  const manifest = context.Global.AST.CLICommands;
  if (
    manifest === overriddenOperationIDsManifest &&
    overriddenOperationIDsCache !== null
  ) {
    return overriddenOperationIDsCache;
  }
  const out = new Set<string>();
  for (const cmd of manifest?.Commands || []) {
    if (!(cmd as any).Override) continue;
    const path: string[] = (cmd.Path || []).map((segment: string) =>
      sanitizeCLICommand(segment),
    );
    const route = cmd.Source?.Routes?.[0];
    const found = route ? findIntentOperation(route.OperationID) : null;
    if (!found) continue; // collectIntentManifest reports the missing op.
    const command = (cmd.Path || []).join(" ");
    const generated = getCLICommandPath(found.op);
    if (intentOperationIsPromoted(found.op)) {
      throw new Error(
        `x-speakeasy-cli-commands: command "${command}" cannot override operation "${route.OperationID}" because it is promoted onto a generated command group; override supports non-promoted leaf operation commands only`,
      );
    }
    if (path.join(" ") !== generated.join(" ")) {
      throw new Error(
        `x-speakeasy-cli-commands: command "${command}" sets override: true, but operation "${
          route.OperationID
        }" is generated at "${generated.join(
          " ",
        )}"; override requires that exact canonical command path`,
      );
    }
    // Identity is the canonical generated command path: operation IDs are
    // not unique across split operations or same-named operations in
    // different groups, and override already requires this exact path.
    out.add(generated.join(" "));
  }
  overriddenOperationIDsManifest = manifest;
  overriddenOperationIDsCache = out;
  return out;
}

function isOperationOverridden(op: Operation): boolean {
  return overriddenOperationIDs().has(getCLICommandPath(op).join(" "));
}
registerTemplateFunc("isOperationOverridden", isOperationOverridden);

function collectIntentManifest(): IntentManifestCtx {
  const out: IntentManifestCtx = {
    Categories: [],
    Bound: [],
    Planned: [],
    GroupTags: [],
    Order: [],
  };
  const manifest = context.Global.AST.CLICommands;
  if (!manifest) return out;

  validateCLIOperationDeclarations(manifest);
  if (!manifest.Commands) return out;

  validateIntentParents(manifest);
  // Validate override paths even for rendering surfaces that only consume the
  // manifest and never walk generated operation registrations.
  overriddenOperationIDs();

  const addCategory = (c: string) => {
    if (c && !out.Categories.includes(c)) out.Categories.push(c);
  };
  // The declared categories list fixes help-section order; categories that
  // only appear on commands follow in encounter order.
  for (const category of manifest.Categories || []) {
    addCategory(category);
  }

  for (const cmd of manifest.Commands) {
    const path: string[] = (cmd.Path || []).map((seg: string) =>
      sanitizeCLICommand(seg),
    );
    if (path.length === 0) continue;
    const name = path[path.length - 1];
    const parentPath = path.slice(0, -1);
    addCategory(cmd.Category);
    out.Order.push({ ParentPath: parentPath, Name: name });

    if (cmd.Source.Type.valueOf() === "group") {
      out.GroupTags.push({
        Name: name,
        Category: cmd.Category || "",
        HelpDefaults: cmd.Help?.Defaults || "",
        HelpLearn: cmd.Help?.Learn || "",
        HelpEscalate: cmd.Help?.Escalate || "",
      });
      continue;
    }

    if (cmd.Source.Type.valueOf() === "planned") {
      out.Planned.push({
        Name: name,
        ParentPath: parentPath,
        Category: cmd.Category || "",
        Summary: cmd.Summary || "",
        Tagline: cmd.Tagline || "",
        Description: cmd.Description || "",
        // Go error strings must not end with punctuation (staticcheck ST1005)
        Note: (
          cmd.Source.Note ||
          `"${name}" is declared but its backing API surface is not part of this build yet`
        )
          .trim()
          .replace(/[.\s]+$/, ""),
        Custom: false,
        HelpDefaults: cmd.Help?.Defaults || "",
        HelpLearn: cmd.Help?.Learn || "",
        HelpEscalate: cmd.Help?.Escalate || "",
      });
      continue;
    }

    if (cmd.Source.Type.valueOf() === "custom") {
      out.Planned.push({
        Name: name,
        ParentPath: parentPath,
        Category: cmd.Category || "",
        Summary: cmd.Summary || "",
        Tagline: cmd.Tagline || "",
        Description: cmd.Description || "",
        Note: "",
        Custom: true,
        HelpDefaults: cmd.Help?.Defaults || "",
        HelpLearn: cmd.Help?.Learn || "",
        HelpEscalate: cmd.Help?.Escalate || "",
      });
      continue;
    }

    if (cmd.Source.Type.valueOf() !== "operation") continue;
    const routes: any[] = cmd.Source.Routes || [];
    const route = routes[0];
    const dispatch = routes.length > 1;
    const found = route ? findIntentOperation(route.OperationID) : null;
    if (!found) {
      throw new Error(
        `x-speakeasy-cli-commands: command "${(cmd.Path || []).join(
          " ",
        )}" resolves operation ` +
          `${route?.OperationID}, which is not part of the generated CLI surface`,
      );
    }
    // Presets merge as one JSON object (supports scalars, arrays, objects).
    const presetJSON = intentPresetJSON(cmd.Presets || []);
    const presetObj: Record<string, any> = presetJSON
      ? JSON.parse(presetJSON)
      : {};

    // The hints map arrives from Go map iteration: sort it so the annotation
    // (and therefore the generated file) is deterministic.
    const sortedHints: Record<string, any> = {};
    for (const reason of Object.keys(cmd.Hints || {}).sort()) {
      sortedHints[reason] = cmd.Hints[reason];
    }
    const hintsJSON =
      Object.keys(sortedHints).length > 0 ? JSON.stringify(sortedHints) : "";

    const hasMeta = operationUsesMetadataFlags(found.op);
    const args: IntentBodyEntry[] = [];
    let minArgs = 0;
    for (const a of cmd.Args || []) {
      const key = intentPointerKey(a.Bind?.Pointer || "");
      if (!key) continue;
      args.push({
        Key: key,
        Name: a.Name || "input",

        Summary: a.Summary || "",
        Variadic: Boolean(a.Variadic),
        Required: Boolean(a.Required),
        PresetCovered: dispatch ? false : intentHasOwn(presetObj, key),
        Suggestions: (a.Enum || []).map((v: any) => `${v}`),
        RouteIDs: (a as any).RouteIDs || [],
        RequiredRouteIDs: (a as any).RequiredRouteIDs || [],
        Positional: true,
      });
      if (a.Required && (dispatch || !intentHasOwn(presetObj, key)))
        minArgs = 1;
      break; // v1: one (variadic) positional joined with spaces
    }

    const flags: IntentBodyEntry[] = [];
    for (const f of cmd.Flags || []) {
      const path = intentPointerPath(f.Bind?.Pointer || "");
      const key = path[0] || "";
      if (!key || !f.Name) continue;
      const declaredFlagName = sanitizeFlagNameWithReserved(f.Name);
      const presetParent = presetObj[key];
      const presetCovered =
        path.length > 1
          ? typeof presetParent === "object" &&
            presetParent !== null &&
            intentHasOwn(presetParent, path[1])
          : intentHasOwn(presetObj, key);

      flags.push({
        Key: key,
        Path: path,
        Name: declaredFlagName,
        Shorthand: f.Shorthand || "",
        Summary: intentFlagHelp(f),
        Type: f.Type || "string",
        Required: Boolean(f.Required),
        PresetCovered: dispatch ? false : presetCovered,
        DefaultValue: f.Default,
        HasDefault: f.Default !== undefined && f.Default !== null,
        Suggestions: (f.Enum || []).map((v: any) => `${v}`),
        RouteIDs: (f as any).RouteIDs || [],
        RequiredRouteIDs: (f as any).RequiredRouteIDs || [],
        Group: dispatch
          ? intentRouteGroup((f as any).RouteIDs || [], routes)
          : "",
        GroupOrder: dispatch
          ? intentRouteGroupOrder((f as any).RouteIDs || [], routes)
          : "",
      });
    }

    const declaredExamples = (cmd.Examples || [])
      .map((e: any) => `  ${e.Command}`)
      .slice(0, 3)
      .join("\n");
    const fallbackInputs = [
      ...args
        .filter((a) => a.Required && !a.PresetCovered)
        .map((a) => intentExamplePlaceholder(a.Name || "input")),
      ...flags
        .filter((f) => f.Required && !f.PresetCovered)
        .map(intentFallbackFlag),
      // Backing-operation parameter flags exist only on intents with flag
      // metadata; a metadata-less intent must not advertise flags it never
      // registers.
      ...(hasMeta ? intentRequiredParamFallbacks(found.op) : []),
    ];
    const fallbackSuffix =
      fallbackInputs.length > 0 ? ` ${fallbackInputs.join(" ")}` : "";
    // Authored examples win. Full help otherwise synthesizes an invocation
    // with shell-safe placeholders for the required inputs. Compact help's
    // "Just works:" shows only invocations that run as written, so it keeps
    // the synthesized fallback only when no placeholder would be needed.
    const example =
      declaredExamples ||
      (helpStyle() !== "compact" || fallbackInputs.length === 0
        ? `  ${sanitizeCliName()} ${path.join(" ")}${fallbackSuffix}`
        : "");

    // The generated operation command reads its JSON body from one of two
    // surfaces: a --body flag alongside flag metadata (expandable and mixed
    // param+body operations), or a single whole-body flag with no metadata
    // (complex JSON bodies such as top-level unions). The intent command
    // mirrors whichever surface its run function expects.
    let bodyFlag = "";
    if (found.op.Request) {
      if (dispatch) {
        if (!hasMeta || !templateHasBodyFlag(found.op)) {
          throw new Error(
            `x-speakeasy-cli-commands: command "${path.join(
              " ",
            )}": route dispatch requires the operation's generated command to expose --body alongside flag metadata`,
          );
        }
        bodyFlag = "body";
      } else if (hasMeta) {
        bodyFlag = templateHasBodyFlag(found.op) ? "body" : "";
      } else if (found.op.Request.IsRequestBody) {
        // Whole-body operations without metadata read a single body flag
        // named after the request body (the BuildRequestBody path).
        bodyFlag = getRequestBodyFlagName(found.op.Request.RequestBody);
      } else {
        bodyFlag = "body";
      }
    }
    if (
      bodyFlag === "" &&
      (args.length > 0 || flags.length > 0 || presetJSON !== "")
    ) {
      throw new Error(
        `x-speakeasy-cli-commands: command "${path.join(
          " ",
        )}" resolves operation ` +
          `${route?.OperationID} to a generated operation with no JSON body surface ` +
          `(multipart or body-less), so declared args, flags, or presets cannot be applied`,
      );
    }

    validateIntentFlagNames(
      (cmd.Path || []).join(" "),
      cmd,
      found.op,
      bodyFlag,
    );
    for (const arg of args) {
      arg.SatisfiedBy = Array.from(
        new Set([bodyFlag].filter((name) => name !== "")),
      );
    }
    // A supplied whole body satisfies declared input requirements;
    // interactive prompting reads it as a body source.
    for (const flag of flags) {
      flag.SatisfiedBy = Array.from(
        new Set([bodyFlag].filter((name) => name !== "")),
      );
    }

    // Artifact runtime config rides a cobra annotation: pointer segments in
    // walk order, the content kind, the default filename pattern, and any
    // block/identity field bindings that differ from the defaults. Default
    // bindings are omitted (the runtime re-applies them on read) so
    // declarations without a block:/identity: keep their annotation bytes.
    const artifact = cmd.Output?.Artifact;
    const artifactJSON = artifact
      ? JSON.stringify({
          pointer: (artifact.Segments || []).map((s: any) =>
            s.Wild ? { wild: true } : { field: s.Field },
          ),
          kind: artifact.Kind,
          defaultPath: artifact.DefaultPath,
          ...(artifact.TypeField && artifact.TypeField !== "type"
            ? { typeField: artifact.TypeField }
            : {}),
          ...(artifact.DataField && artifact.DataField !== "data"
            ? { dataField: artifact.DataField }
            : {}),
          ...(artifact.MimeTypeField && artifact.MimeTypeField !== "mime_type"
            ? { mimeTypeField: artifact.MimeTypeField }
            : {}),
          ...(artifact.URIField && artifact.URIField !== "uri"
            ? { uriField: artifact.URIField }
            : {}),
          ...(artifact.IDField && artifact.IDField !== "id"
            ? { idField: artifact.IDField }
            : {}),
          ...(artifact.StatusField && artifact.StatusField !== "status"
            ? { statusField: artifact.StatusField }
            : {}),
          ...(artifact.TerminalStatus && artifact.TerminalStatus !== "completed"
            ? { terminalStatus: artifact.TerminalStatus }
            : {}),
        })
      : "";

    let asyncJSON = "";
    let pollOp: Operation | null = null;
    let pollSDKAccessor = "";
    let asyncParams: IntentAsyncParameter[] = [];
    let asyncResume = "";
    let asyncCreateResponseCode = "200";
    let asyncResponseCode = "200";
    let asyncCreateTestJSON = "";
    let asyncPendingTestJSON = "";
    let asyncSuccessTestJSON = "";
    let asyncFailureTestJSON = "";
    let asyncHandoffTestJSON = "";
    let asyncUnknownTestJSON = "";
    let asyncMissingStateTestJSON = "";
    let asyncMissingHandleTestJSON = "";
    if (cmd.Async) {
      const pollFound = findIntentOperation(cmd.Async.OperationID);
      if (!pollFound) {
        throw new Error(
          `x-speakeasy-cli-commands: command "${path.join(
            " ",
          )}" poll operation ${
            cmd.Async.OperationID
          } is not part of the generated CLI surface`,
        );
      }
      if (!pollFound.op.Request?.Field?.Type || !pollFound.op.Response?.Type) {
        throw new Error(
          `x-speakeasy-cli-commands: command "${path.join(
            " ",
          )}" poll operation ${
            cmd.Async.OperationID
          } has no generated typed request/response surface`,
        );
      }
      const pollArgs = getArguments(pollFound.op);
      if ((pollArgs.security || []).length > 0) {
        throw new Error(
          `x-speakeasy-cli-commands: command "${path.join(
            " ",
          )}" poll operation ${
            cmd.Async.OperationID
          } requires method security arguments; async polling currently supports client-configured authentication only`,
        );
      }
      const params =
        cmd.Async.ID?.To?.In === "path"
          ? pollFound.op.Request.Params?.PathParams || []
          : pollFound.op.Request.Params?.QueryParams || [];
      const target = params.find(
        (p: any) =>
          (p.Field?.OriginalName || p.Field?.Name) === cmd.Async.ID.To.Name,
      );
      if (!target) {
        throw new Error(
          `x-speakeasy-cli-commands: command "${path.join(
            " ",
          )}" cannot map async.id.to ${cmd.Async.ID.To.In} parameter ${
            cmd.Async.ID.To.Name
          } to the generated poll request`,
        );
      }
      const resumePrefix = `${sanitizeCliName()} ${path.join(" ")} --resume`;
      asyncParams = (cmd.Async.ResolvedParams || []).map((param: any) => ({
        In: param.In,
        Name: param.Name,
        Expected: String(param.Value),
      }));
      // The states map arrives from Go map iteration: sort it so the
      // annotation (and therefore the generated file) is deterministic.
      const sortedStates: Record<string, string> = {};
      for (const state of Object.keys(cmd.Async.States || {}).sort()) {
        sortedStates[state] = cmd.Async.States[state];
      }
      // Failure-detail bindings equal to the v1 defaults are omitted (the
      // runtime re-applies them on read), keeping undeclared recipes'
      // annotation bytes stable.
      asyncJSON = JSON.stringify({
        idPointer: cmd.Async.ID.Pointer,
        statePointer: cmd.Async.StatePointer,
        states: sortedStates,
        ...(cmd.Async.ErrorField && cmd.Async.ErrorField !== "error"
          ? { errorField: cmd.Async.ErrorField }
          : {}),
        ...(cmd.Async.ErrorMessageField &&
        cmd.Async.ErrorMessageField !== "message"
          ? { errorMessageField: cmd.Async.ErrorMessageField }
          : {}),
        interval: cmd.Async.Interval,
        backoff: cmd.Async.Backoff,
        maxInterval: cmd.Async.MaxInterval,
        timeout: cmd.Async.Timeout,
        command: path.join(" "),
        resume: resumePrefix,
        parameterIn: cmd.Async.ID.To.In,
        parameterName: cmd.Async.ID.To.Name,
        params: (cmd.Async.ResolvedParams || []).map((param: any) => ({
          in: param.In,
          name: param.Name,
          value: param.Value,
        })),
      });
      pollOp = pollFound.op;
      pollSDKAccessor = pollFound.sdkAccessor;
      asyncResume = resumePrefix;
      asyncCreateResponseCode = /^2\d\d$/.test(cmd.Async.CreateResponseCode)
        ? cmd.Async.CreateResponseCode
        : "200";
      asyncResponseCode = /^2\d\d$/.test(cmd.Async.ResponseCode)
        ? cmd.Async.ResponseCode
        : "200";

      const handle = "async-test-handle";
      const classifiedStates = Object.keys(cmd.Async.States || {}).sort();
      const stateOf = (classification: string): string =>
        classifiedStates.find(
          (state) => cmd.Async.States[state] === classification,
        ) || "";
      const pendingState = stateOf("pending");
      const successState = stateOf("success");
      const failureState = stateOf("failure");
      const handoffState = stateOf("handoff");
      let unknownState = "unknown_async_state";
      while (classifiedStates.includes(unknownState)) unknownState += "_new";
      const withID = foldIntentTestPath(cmd.Async.ID.Segments || [], handle);
      const responseFor = (state: string, extra?: any): any => {
        let response = foldIntentTestPath(cmd.Async.StateSegments || [], state);
        if (extra !== undefined) {
          response = mergeIntentTestValue(response, extra);
        }
        return response;
      };
      asyncCreateTestJSON = JSON.stringify(withID);
      asyncPendingTestJSON = pendingState
        ? JSON.stringify(responseFor(pendingState))
        : "";
      let successExtra: any = undefined;
      if (artifactJSON) {
        const artifactSpec = JSON.parse(artifactJSON);
        successExtra = foldIntentArtifactPath(artifactSpec.pointer || [], {
          [artifactSpec.typeField || "type"]: artifactSpec.kind,
          [artifactSpec.dataField || "data"]: "YXN5bmMtYXJ0aWZhY3Q=",
          [artifactSpec.mimeTypeField || "mime_type"]: artifactTestKindMime(
            artifactSpec.kind,
          ),
        });
      }
      asyncSuccessTestJSON = JSON.stringify(
        responseFor(successState, successExtra),
      );
      // The failure mock carries its detail under the declared bindings so
      // the generated failure test exercises the shape the runtime reads.
      asyncFailureTestJSON = failureState
        ? JSON.stringify(
            responseFor(failureState, {
              [cmd.Async.ErrorField || "error"]: {
                [cmd.Async.ErrorMessageField || "message"]:
                  "async operation failed",
              },
            }),
          )
        : "";
      asyncHandoffTestJSON = handoffState
        ? JSON.stringify(
            responseFor(handoffState, {
              action: { type: "continue" },
            }),
          )
        : "";
      asyncUnknownTestJSON = JSON.stringify(responseFor(unknownState));
      asyncMissingStateTestJSON = "{}";
      asyncMissingHandleTestJSON = "{}";
    }

    const selectors = route?.Selectors;
    const discriminatorValue = selectors?.DiscriminatorValue;
    const discriminatorValueJSON =
      discriminatorValue === undefined || discriminatorValue === null
        ? ""
        : JSON.stringify(discriminatorValue);
    const dispatchRoutes: IntentDispatchRoute[] = dispatch
      ? routes.map((dispatchRoute: any) => ({
          ID: dispatchRoute.ID,
          Label: dispatchRoute.Label,
          SelectorFlag: sanitizeFlagNameWithReserved(dispatchRoute.Selector),
          Default: Boolean(dispatchRoute.Default),
          PresetJSON: intentPresetJSON(dispatchRoute.Presets || []),
          VariantLabel: (dispatchRoute.RequestVariant || "").replace(
            /^#\/components\/schemas\//,
            "",
          ),
          ForeignSelectors: dispatchRoute.Selectors?.Foreign || [],
          PresetMergePoints: intentPresetMergePoints(
            dispatchRoute.PresetMergePoints,
          ),
        }))
      : [];
    const dispatchKeys: IntentDispatchKey[] = dispatch
      ? ((cmd as any).DispatchKeys || []).map((key: any) => ({
          BodyKey: intentPointerKey(key.Pointer || ""),
          RouteIDs: key.RouteIDs || [],
          UnroutedVariants: key.UnroutedVariants || [],
        }))
      : [];
    const variantHelp = dispatch
      ? `Request variants: ${dispatchRoutes
          .map(
            (dispatchRoute) =>
              `${dispatchRoute.Label} (--${dispatchRoute.SelectorFlag}${
                dispatchRoute.Default ? "; default" : ""
              })`,
          )
          .join(", ")}.\nVariant-specific flags cannot be combined.`
      : "";
    const argToken = (cmd.Args || [])[0]?.Name || "input";
    const escapeCommand = `${sanitizeCliName()} ${getCLICommandPath(
      found.op,
    ).join(" ")}`;
    out.Bound.push({
      ID: cmd.ID,
      Use: args.length > 0 ? `${name} [${argToken}]` : name,
      FuncName: sanitizeClassName(cmd.ID),
      PkgPath: found.pkgPath,
      PkgName: found.pkgName,
      ParentPath: parentPath,
      Category: cmd.Category || "",
      Summary: (cmd.Summary || "") + (cmd.Tagline ? ` ${cmd.Tagline}` : ""),
      Description: cmd.Description || "",
      Example: example,
      HelpDefaults: intentHelpDefaults(cmd),
      HelpLearn: cmd.Help?.Learn || "",
      HelpEscalate:
        cmd.Help?.Escalate || `full request control via ${escapeCommand}`,
      MetaVar: templateMetaVarName(found.op),
      // Operation-level security flags must exist on the intent command too:
      // the reused run function reads them from the invoking command.
      SecurityFlags: operationHasSecurity(found.op)
        ? templateSecurityFlagRegistration(found.op)
        : "",
      HasMeta: hasMeta,
      BodyFlag: bodyFlag,

      BodyFieldPath: hasMeta ? templateBodyFieldPath(found.op) : "",
      HasBody: Boolean(
        found.op.Request &&
          (found.op.Request.IsRequestBody || getBodyFieldPath(found.op) !== ""),
      ),
      RunFunc: sanitizeRunFuncName(sanitizeCommandName(found.op.GetID())),
      ExecuteFunc: `execute${sanitizeCommandName(found.op.GetID())}`,
      OpID: found.op.OriginalID,
      HasSchema: hasBodySchemaForOp(found.op),
      MinArgs: minArgs,
      Args: args,
      ArgsHelp: (cmd.Args || [])
        .map((a: any) =>
          `  <${a.Name || "input"}>  ${a.Summary || ""}${intentValueSuffix(
            a,
            false,
          )}`.trimEnd(),
        )
        .join("\n"),
      Flags: flags,
      PromptFlags: flags.filter((f) => f.Required && !f.PresetCovered),
      Implied: intentImpliedRules(cmd.Flags || []),
      PresetJSON: presetJSON,
      CommandDisplay: path.join(" "),
      VariantLabel: (route?.RequestVariant || "").replace(
        /^#\/components\/schemas\//,
        "",
      ),
      ForeignSelectors: selectors?.Foreign || [],
      DiscriminatorKey: selectors?.DiscriminatorKey || "",
      DiscriminatorValueJSON: discriminatorValueJSON,
      DiscriminatorAliasesJSON: (selectors?.DiscriminatorAliases || []).map(
        (v: any) => JSON.stringify(v),
      ),
      PresetMergePoints: intentPresetMergePoints(route?.PresetMergePoints),
      EscapeCommand: escapeCommand,
      HasPresetMerge:
        !dispatch &&
        (presetJSON !== "" ||
          (selectors?.Foreign || []).length > 0 ||
          (Boolean(selectors?.DiscriminatorKey) &&
            discriminatorValueJSON !== "")),
      HintsJSON: hintsJSON,
      JQ: cmd.Output?.Projection?.JQ || "",
      ArtifactJSON: artifactJSON,
      ArtifactKind: artifact?.Kind || "",
      ArtifactDefaultPath: artifact?.DefaultPath || "",
      ArtifactResponseCode: artifact?.ResponseCode || "200",
      AsyncJSON: asyncJSON,
      PollOp: pollOp,
      PollSDKAccessor: pollSDKAccessor,
      AsyncParameterIn: cmd.Async?.ID?.To?.In || "",
      AsyncParameterName: cmd.Async?.ID?.To?.Name || "",
      AsyncParams: asyncParams,
      AsyncResume: asyncResume,
      AsyncCreateResponseCode: asyncCreateResponseCode,
      AsyncResponseCode: asyncResponseCode,
      AsyncCreateTestJSON: asyncCreateTestJSON,
      AsyncPendingTestJSON: asyncPendingTestJSON,
      AsyncSuccessTestJSON: asyncSuccessTestJSON,
      AsyncFailureTestJSON: asyncFailureTestJSON,
      AsyncHandoffTestJSON: asyncHandoffTestJSON,
      AsyncUnknownTestJSON: asyncUnknownTestJSON,
      AsyncMissingStateTestJSON: asyncMissingStateTestJSON,
      AsyncMissingHandleTestJSON: asyncMissingHandleTestJSON,
      AsyncFailureHint: cmd.Hints?.CLI_ASYNC_FAILED?.[0] || "",
      StreamSelect: cmd.Output?.Stream?.Pointer || "",
      StreamMetadataJSON: streamMetadataJSON(cmd.Output?.Stream?.Metadata),
      StreamToggle:
        !dispatch && intentStreamKind(found.op).kind
          ? intentDefaultOnStreamToggle(flags, presetJSON)
          : null,
      StreamKind: intentStreamKind(found.op).kind,
      StreamSentinel: intentStreamKind(found.op).sentinel,
      OpHasRequiredParams: intentOperationHasRequiredParams(found.op),
      Dispatch: dispatch,
      Override: Boolean((cmd as any).Override),
      DispatchRoutes: dispatchRoutes,
      DispatchInputs: [...args, ...flags],
      DispatchKeys: dispatchKeys,
      VariantHelp: variantHelp,
    });
  }

  validateIntentPackageIdentifiers(out.Bound);
  return out;
}

// intentStreamKind reports how an operation's success response streams (the
// generated command iterates it through output.StreamResult): "sse" for
// text/event-stream (with its declared end-of-stream sentinel, if any),
// "jsonl" for JSON Lines / JSON sequences, "" when it does not stream.
function intentStreamKind(op: Operation): { kind: string; sentinel: string } {
  for (const resp of op.Response?.Responses || []) {
    if (resp.Error) continue;
    for (const content of resp.Content || []) {
      const sm = content.SerializationMethod?.toString();
      if (sm === "eventstream") {
        return { kind: "sse", sentinel: content.SSESentinel || "" };
      }
      if (sm === "jsonl") return { kind: "jsonl", sentinel: "" };
    }
  }
  return { kind: "", sentinel: "" };
}

function streamMetadataJSON(
  fields: CLICommandStreamMetadata[] | null | undefined,
): string {
  if (!fields?.length) {
    return "";
  }
  return JSON.stringify(
    fields.map((field) => ({
      label: field.Label,
      pointers: field.Pointers,
      ...(field.Complete?.length ? { complete: field.Complete } : {}),
      ...(field.When ? { when: field.When } : {}),
    })),
  );
}

// intentOperationHasRequiredParams reports whether the operation declares
// required path/query/header parameters (inputs a declared intent does not
// bind in v1, so an invocation must supply them as the operation's own flags).
function intentOperationHasRequiredParams(op: Operation): boolean {
  if (!op.Request?.Params) return false;
  const allParams = [
    ...(op.Request.Params.PathParams || []),
    ...(op.Request.Params.QueryParams || []),
    ...(op.Request.Params.HeaderParams || []),
  ];
  return allParams.some((param) => !param.Field.Const && !param.Field.Optional);
}

// collectOperationFlagMeta lists the flag names an operation's generated
// command registers (parameters and expanded body fields), for collision
// validation against declared intent flags.
function collectOperationFlagMeta(op: Operation): { flagName: string }[] {
  const flags: { flagName: string }[] = [];
  if (op.Request?.Params) {
    const allParams = [
      ...(op.Request.Params.PathParams || []),
      ...(op.Request.Params.QueryParams || []),
      ...(op.Request.Params.HeaderParams || []),
    ];
    for (const param of allParams) {
      if (param.Field.Const) continue;
      flags.push({ flagName: sanitizeFlagNameWithReserved(param.Field.Name) });
    }
  }
  if (op.Request?.IsRequestBody && isRequestBodyExpandable(op)) {
    const bodyField = op.Request.RequestBody;
    if (bodyField?.Type?.Fields) {
      for (const field of bodyField.Type.Fields) {
        if (field.Const) continue;
        flags.push({ flagName: sanitizeFlagNameWithReserved(field.Name) });
      }
    }
  }
  if (op.Security) {
    walkSecurityLeafFields(op.Security.Type.Fields || [], (leaf) => {
      flags.push({ flagName: sanitizeFlagNameWithReserved(leaf.name) });
    });
  }
  return flags;
}

function cliOperationDeclaration(op: Operation): any | null {
  const operations = (context.Global.AST.CLICommands as any)?.Operations || [];
  for (const declared of operations) {
    if (
      declared.OperationID === op.OriginalID ||
      declared.OperationID === op.GetID()
    ) {
      return declared;
    }
  }
  return null;
}

function operationBodyFlagSurfaces(op: Operation): string[] {
  if (!op.Request) return [];
  const names: string[] = [];
  if (operationUsesMetadataFlags(op)) {
    if (templateHasBodyFlag(op)) names.push("body");
    const whole = wholeBodyFlagName(op);
    if (whole) names.push(whole);
  } else if (op.Request.IsRequestBody) {
    names.push(getRequestBodyFlagName(op.Request.RequestBody));
  } else {
    names.push("body");
  }
  return Array.from(new Set(names.filter(Boolean)));
}

// operationFlagDefaultLiteral renders the Go literal spliced into the flag
// registration. default: is optional in the manifest, so every branch must
// produce a valid literal for an absent value: zero for numbers, false for
// booleans, and the empty string otherwise.
function operationFlagDefaultLiteral(input: any): string {
  const value = input.Default;
  const absent = value === undefined || value === null;
  switch (input.Type) {
    case "bool":
      return value ? "true" : "false";
    case "int":
      return absent ? "0" : `${value}`;
    case "float":
      return absent ? "0" : `${value}`;
    default:
      return goStringLiteral(absent ? "" : `${value}`);
  }
}

function operationDeclaredFlagHelp(input: any): string {
  return `${input.Summary || ""}${intentValueSuffix(input, false)}`.trim();
}

function operationGeneratedFlagNames(op: Operation): Set<string> {
  const names = new Set<string>();
  for (const name of getGlobalFlagNames()) names.add(name);
  for (const item of collectOperationFlagMeta(op)) names.add(item.flagName);
  if (templateHasBodyFlag(op)) names.add("body");
  const whole = wholeBodyFlagName(op);
  if (whole) names.add(whole);
  if (op.Request?.IsRequestBody && !operationUsesMetadataFlags(op)) {
    names.add(getRequestBodyFlagName(op.Request.RequestBody));
  }
  if (hasBodySchemaForOp(op)) names.add("schema");
  if (hasPagination(op)) {
    names.add("all");
    names.add("max-pages");
  }
  if (hasBinaryResponse(op)) {
    names.add("output-file");
    names.add("output-b64");
  }
  return names;
}

function validateCLIOperationFlagNames(op: Operation, declared: any): void {
  const generated = operationGeneratedFlagNames(op);
  for (const flag of declared.Flags || []) {
    if (generated.has(flag.Name)) {
      throw new Error(
        `x-speakeasy-cli-commands: operation "${declared.OperationID}" declares flag "--${flag.Name}", ` +
          `which collides with a generated parameter, body, security, pagination, binary-output, or schema flag on that operation`,
      );
    }
  }
  // Declared shorthands must also avoid the auto-assigned single-letter
  // shorthands templateFlagMetadataVar gives the operation's generated
  // flags — a match panics pflag at startup, so it fails generation instead.
  // operationAutoShorthandOwners shares the metadata var's entry traversal.
  const autoShorthandOwners = operationAutoShorthandOwners(op, false);
  for (const flag of declared.Flags || []) {
    if (!flag.Shorthand) continue;
    const owner = autoShorthandOwners.get(flag.Shorthand);
    if (owner) {
      throw new Error(
        `x-speakeasy-cli-commands: operation "${declared.OperationID}" declares shorthand "-${flag.Shorthand}" on flag "--${flag.Name}", ` +
          `which collides with the auto-assigned shorthand of generated flag "--${owner}"; choose a different shorthand`,
      );
    }
  }
}

function validateCLIOperationDeclarations(manifest: any): void {
  for (const declared of manifest.Operations || []) {
    const found = findIntentOperation(declared.OperationID);
    if (!found) {
      throw new Error(
        `x-speakeasy-cli-commands: operation "${declared.OperationID}" is not part of the generated CLI surface`,
      );
    }
    validateCLIOperationFlagNames(found.op, declared);
  }
}

function getCLIOperationCtx(op: Operation): CLIOperationCtx | null {
  const declared = cliOperationDeclaration(op);
  if (!declared) return null;
  validateCLIOperationFlagNames(op, declared);
  const bodyFlags = operationBodyFlagSurfaces(op);
  if ((declared.Flags || []).length > 0 && bodyFlags.length === 0) {
    throw new Error(
      `x-speakeasy-cli-commands: operation "${declared.OperationID}" declares body flags but the generated command has no whole JSON body surface`,
    );
  }
  const flags: CLIOperationFlagCtx[] = (declared.Flags || []).map(
    (input: any) => ({
      Name: input.Name,
      Shorthand: input.Shorthand || "",
      Summary: operationDeclaredFlagHelp(input),
      Type: input.Type || "string",
      Kind:
        input.Type === "bool"
          ? "bool"
          : input.Type === "int"
          ? "int64"
          : input.Type === "float"
          ? "float64"
          : "string",
      Key: intentPointerKey(input.Bind?.Pointer || ""),
      DefaultLiteral: operationFlagDefaultLiteral(input),
      DefaultValue: input.Default,
      HasDefault: input.Default !== undefined && input.Default !== null,
      DefaultResolves: input.DefaultFrom === "schema",
      Suggestions: (input.Enum || []).map((v: any) => `${v}`),
    }),
  );
  return {
    OperationID: declared.OperationID,
    StreamSelect: declared.Output?.Stream?.Pointer || "",
    StreamMetadataJSON: streamMetadataJSON(declared.Output?.Stream?.Metadata),
    Flags: flags,
    BodyFlags: bodyFlags,
    CanonicalBodyFlag: bodyFlags[bodyFlags.length - 1] || "",
    BodyRequired: Boolean(declared.BodyRequired),
  };
}
registerTemplateFunc("getCLIOperationCtx", getCLIOperationCtx);

function intentDefaultOnStreamToggle(
  flags: IntentBodyEntry[],
  presetJSON: string,
): { Name: string; Key: string } | null {
  const toggle = intentStreamToggle(
    flags,
    presetJSON ? JSON.parse(presetJSON) : {},
  );
  return toggle ? { Name: toggle.Name || toggle.Key, Key: toggle.Key } : null;
}

function cliOperationStreamToggle(
  op: Operation,
  declared: any,
): { Name: string; Key: string; DefaultOn: boolean; InBody: boolean } | null {
  // Name the toggle after the declared stream toggle (see
  // operationStreamToggle in usage.ts), falling back to a schema-derived
  // operation flag named "stream" on the generated surface. The decoder
  // resolves defaultFrom: schema into Default, so the toggle's effective
  // default picks the wording: a default-on toggle documents the opt-out
  // (--<flag>=false), a default-off toggle documents the opt-in (--<flag>),
  // because one complete JSON response is already its default.
  const toggle = operationStreamToggle(declared);
  if (toggle) {
    const key = intentPointerKey(toggle.Bind?.Pointer || "");
    return {
      Name: toggle.Name || key,
      Key: key,
      DefaultOn: toggle.Default === true || String(toggle.Default) === "true",
      InBody: true,
    };
  }
  if (!collectOperationFlagMeta(op).some((f) => f.flagName === "stream")) {
    return null;
  }
  if (operationBodySchemaDefaultTrue(op, "stream")) {
    return { Name: "stream", Key: "stream", DefaultOn: true, InBody: true };
  }
  return {
    Name: "stream",
    Key: "stream",
    DefaultOn: operationQueryParamDefaultTrue(op, "stream"),
    InBody: false,
  };
}

function operationQueryParamDefaultTrue(op: Operation, key: string): boolean {
  return (op.Request?.Params?.QueryParams || []).some((param: any) => {
    if (param.Field?.OriginalName !== key && param.Field?.Name !== key) {
      return false;
    }
    return booleanFieldDefaultTrue(param.Field);
  });
}

function cliOperationDefaultOnStreamToggle(
  op: Operation,
): { Name: string; Key: string; InBody: boolean } | null {
  const declared = cliOperationDeclaration(op);
  if (!declared?.Output?.Stream) return null;
  const toggle = cliOperationStreamToggle(op, declared);
  return toggle?.DefaultOn && toggle.Key ? toggle : null;
}
registerTemplateFunc(
  "cliOperationDefaultOnStreamToggle",
  cliOperationDefaultOnStreamToggle,
);

function cliOperationStreamHelp(op: Operation): string {
  const declared = cliOperationDeclaration(op);
  if (!declared?.Output?.Stream) return "";
  const toggle = cliOperationStreamToggle(op, declared);
  const toggleName = toggle?.Name || "";
  const defaultOn = toggle
    ? toggle.DefaultOn
    : operationBodySchemaDefaultTrue(op, "stream");
  const select = declared.Output.Stream.Select;
  if (toggleName && !defaultOn) {
    return (
      `Streamed responses write the string selected by ${select} raw as it arrives. ` +
      `Pass --${toggleName} to request a streamed response; use -o json to keep each full streamed event.`
    );
  }
  const optOut = toggleName
    ? ` Use --${toggleName}=false to request one complete JSON response; use -o json to keep each full streamed event.`
    : " Use -o json to keep each full streamed event.";
  return `By default, streamed responses write the string selected by ${select} raw as it arrives.${optOut}`;
}

function collectOperationNonBodyFlagMeta(
  op: Operation,
): { flagName: string }[] {
  const flags: { flagName: string }[] = [];
  if (op.Request?.Params) {
    const params = [
      ...(op.Request.Params.PathParams || []),
      ...(op.Request.Params.QueryParams || []),
      ...(op.Request.Params.HeaderParams || []),
    ];
    for (const param of params) {
      if (param.Field.Const) continue;
      flags.push({
        flagName: sanitizeFlagNameWithReserved(param.Field.Name),
      });
    }
  }
  if (op.Security) {
    walkSecurityLeafFields(op.Security.Type.Fields || [], (leaf) => {
      flags.push({ flagName: sanitizeFlagNameWithReserved(leaf.name) });
    });
  }
  return flags;
}

// Two bound commands emitted into different packages that share a Go
// identifier cannot both be imported from intents.go; fail generation with
// the fix named instead of emitting uncompilable code.
function validateIntentPackageIdentifiers(bound: IntentCmdCtx[]): void {
  const byName = new Map<string, string>();
  for (const cmd of bound) {
    if (!cmd.PkgPath) continue;
    const existing = byName.get(cmd.PkgName);
    if (existing !== undefined && existing !== cmd.PkgPath) {
      throw new Error(
        `x-speakeasy-cli-commands: intent commands resolve into packages "${existing}" and "${cmd.PkgPath}", ` +
          `which share the Go package identifier "${cmd.PkgName}"; bind the commands to operations in distinct packages`,
      );
    }
    byName.set(cmd.PkgName, cmd.PkgPath);
  }
}

function hasCliIntents(): boolean {
  const manifest = context.Global.AST.CLICommands;
  if (manifest) validateCLIOperationDeclarations(manifest as any);
  return Boolean(manifest?.Commands && manifest.Commands.length > 0);
}
registerTemplateFunc("hasCliIntents", hasCliIntents);

function getDryRunJQIntent(): IntentCmdCtx | null {
  const manifest = collectIntentManifest();
  for (const cmd of manifest.Bound) {
    if (!cmd.JQ) continue;
    if (cmd.PromptFlags.some((f) => f.Required && !f.PresetCovered)) continue;
    // The generated invocation supplies only declared inputs; a backing
    // operation with required path/query/header parameters is not runnable.
    if (cmd.OpHasRequiredParams) continue;
    return cmd;
  }
  return null;
}

function hasCliAsyncIntents(): boolean {
  const manifest = context.Global.AST.CLICommands;
  return Boolean(manifest?.Commands?.some((c: any) => c.Async));
}
registerTemplateFunc("hasCliAsyncIntents", hasCliAsyncIntents);

function isAsyncCreateOperation(op: Operation): boolean {
  const manifest = context.Global.AST.CLICommands;
  if (!manifest?.Commands) return false;
  return manifest.Commands.some((cmd) => {
    if (!cmd.Async || cmd.Source.Type.valueOf() !== "operation") return false;
    const routeID = cmd.Source.Routes?.[0]?.OperationID;
    return routeID === op.GetID() || routeID === op.OriginalID;
  });
}
registerTemplateFunc("isAsyncCreateOperation", isAsyncCreateOperation);

function getAsyncTestCommand(): IntentCmdCtx | null {
  for (const cmd of collectIntentManifest().Bound) {
    if (!cmd.AsyncJSON) continue;
    if (cmd.OpHasRequiredParams) continue;
    if (cmd.Flags.some((flag) => flag.Required && !flag.PresetCovered)) {
      continue;
    }
    return cmd;
  }
  return null;
}

function templateDryRunJQIntentTestEnabled(): boolean {
  return getDryRunJQIntent() !== null;
}
registerTemplateFunc(
  "templateDryRunJQIntentTestEnabled",
  templateDryRunJQIntentTestEnabled,
);

function templateDryRunJQIntentTestArgs(): string {
  const cmd = getDryRunJQIntent();
  if (!cmd) return "";
  const parts = [...cmd.ParentPath, firstUseWord(cmd.Use)];
  for (const arg of cmd.Args) {
    parts.push(
      arg.Type === "bool"
        ? "true"
        : arg.Type === "int" || arg.Type === "float"
        ? "1"
        : "test",
    );
    if (arg.Variadic) break;
  }
  return parts.map((part) => goStringLiteral(part)).join(", ");
}
registerTemplateFunc(
  "templateDryRunJQIntentTestArgs",
  templateDryRunJQIntentTestArgs,
);

function templateAsyncTestCommand(): IntentCmdCtx | null {
  return getAsyncTestCommand();
}
registerTemplateFunc("templateAsyncTestCommand", templateAsyncTestCommand);

function templateAsyncJQTestCommand(): IntentCmdCtx | null {
  const primary = getAsyncTestCommand();
  if (!primary) return null;
  for (const cmd of collectIntentManifest().Bound) {
    if (!cmd.AsyncJSON || !cmd.JQ || cmd.ArtifactJSON) continue;
    if (cmd.OpHasRequiredParams) continue;
    if (
      JSON.stringify(cmd.AsyncParams) !== JSON.stringify(primary.AsyncParams)
    ) {
      continue;
    }
    if (cmd.Flags.some((flag) => flag.Required && !flag.PresetCovered)) {
      continue;
    }
    return cmd;
  }
  return null;
}
registerTemplateFunc("templateAsyncJQTestCommand", templateAsyncJQTestCommand);

function templateAsyncTestEnabled(): boolean {
  return getAsyncTestCommand() !== null;
}
registerTemplateFunc("templateAsyncTestEnabled", templateAsyncTestEnabled);

function templateAsyncTestArgs(): string {
  const cmd = getAsyncTestCommand();
  if (!cmd) return "";
  const parts = [...cmd.ParentPath, firstUseWord(cmd.Use)];
  if (cmd.Args.length > 0) parts.push("a test prompt");
  return parts.map((part) => goStringLiteral(part)).join(", ");
}
registerTemplateFunc("templateAsyncTestArgs", templateAsyncTestArgs);

function templateAsyncTestPathArgs(): string {
  const cmd = getAsyncTestCommand();
  if (!cmd) return "";
  return [...cmd.ParentPath, firstUseWord(cmd.Use)]
    .map((p) => goStringLiteral(p))
    .join(", ");
}
registerTemplateFunc("templateAsyncTestPathArgs", templateAsyncTestPathArgs);
function templateIntentPathArgs(cmd: IntentCmdCtx): string {
  return [...cmd.ParentPath, firstUseWord(cmd.Use)]
    .map((p) => goStringLiteral(p))
    .join(", ");
}
registerTemplateFunc("templateIntentPathArgs", templateIntentPathArgs);
function templateAsyncPollTestArgs(): string {
  const cmd = getAsyncTestCommand();
  return cmd?.PollOp
    ? getCLICommandPath(cmd.PollOp)
        .map((p) => goStringLiteral(p))
        .join(", ")
    : "";
}
registerTemplateFunc("templateAsyncPollTestArgs", templateAsyncPollTestArgs);
function templateReviewAsyncPollTestEnabled(): boolean {
  const cmd = getAsyncTestCommand();
  return (
    (context.Global.AST as any).OpenAPIDocument?.Info?.Title === "SDK Review" &&
    cmd?.PollOp?.OriginalID === "getAsset" &&
    cmd.AsyncParameterName === "id" &&
    getStreamingFieldName(cmd.PollOp) === "AssetStatusStream"
  );
}
registerTemplateFunc(
  "templateReviewAsyncPollTestEnabled",
  templateReviewAsyncPollTestEnabled,
);

function templateIntentAsyncTestArgs(cmd: IntentCmdCtx | null): string {
  if (!cmd) return "";
  const parts = [...cmd.ParentPath, firstUseWord(cmd.Use)];
  if (cmd.Args.length > 0) parts.push("a test prompt");
  return parts.map((part) => goStringLiteral(part)).join(", ");
}
registerTemplateFunc(
  "templateIntentAsyncTestArgs",
  templateIntentAsyncTestArgs,
);

// =============================================================================
// Artifact output (output.artifact declarations)
// =============================================================================

function hasCliArtifacts(): boolean {
  const manifest = context.Global.AST.CLICommands;
  return Boolean(manifest?.Commands?.some((c: any) => c.Output?.Artifact));
}
registerTemplateFunc("hasCliArtifacts", hasCliArtifacts);

// The first artifact-declaring bound command drives the generated artifact
// harness tests. Commands with required flags a preset does not cover are
// skipped (the generic test cannot invent values for them).
function getArtifactTestCommand(): IntentCmdCtx | null {
  const manifest = collectIntentManifest();
  for (const cmd of manifest.Bound) {
    if (!cmd.ArtifactJSON) continue;
    if (cmd.Flags.some((f) => f.Required && !f.PresetCovered)) continue;
    return cmd;
  }
  return null;
}

function templateArtifactTestEnabled(): boolean {
  return getArtifactTestCommand() !== null;
}
registerTemplateFunc(
  "templateArtifactTestEnabled",
  templateArtifactTestEnabled,
);

// Go argument list invoking the artifact test command, e.g. `"render", "a prompt"`.
function templateArtifactTestArgs(): string {
  const cmd = getArtifactTestCommand();
  if (!cmd) return "";
  const parts = [...cmd.ParentPath, firstUseWord(cmd.Use)];
  if (cmd.Args.length > 0) parts.push("a test prompt");
  return parts.map((p) => goStringLiteral(p)).join(", ");
}
registerTemplateFunc("templateArtifactTestArgs", templateArtifactTestArgs);

function templateArtifactTestPathArgs(): string {
  const cmd = getArtifactTestCommand();
  if (!cmd) return "";
  return [...cmd.ParentPath, firstUseWord(cmd.Use)]
    .map((p) => goStringLiteral(p))
    .join(", ");
}
registerTemplateFunc(
  "templateArtifactTestPathArgs",
  templateArtifactTestPathArgs,
);

function templateArtifactTestPromptArg(): string {
  return goStringLiteral(getArtifactTestCommand()?.Args[0]?.Name || "input");
}
registerTemplateFunc(
  "templateArtifactTestPromptArg",
  templateArtifactTestPromptArg,
);

function templateArtifactInteractiveTestEnabled(): boolean {
  return Boolean(getArtifactTestCommand()?.Args.length);
}
registerTemplateFunc(
  "templateArtifactInteractiveTestEnabled",
  templateArtifactInteractiveTestEnabled,
);

function templateArtifactTestDefaultPath(): string {
  return getArtifactTestCommand()?.ArtifactDefaultPath || "";
}
registerTemplateFunc(
  "templateArtifactTestDefaultPath",
  templateArtifactTestDefaultPath,
);

function templateArtifactTestKind(): string {
  return getArtifactTestCommand()?.ArtifactKind || "";
}
registerTemplateFunc("templateArtifactTestKind", templateArtifactTestKind);

// HTTP status the stub server answers with: the 2xx code whose schema the
// artifact pointer was linked against (an operation declaring only 201 must
// be served 201).
function templateArtifactTestStatus(): string {
  const code = getArtifactTestCommand()?.ArtifactResponseCode || "200";
  return /^2\d\d$/.test(code) ? code : "200";
}
registerTemplateFunc("templateArtifactTestStatus", templateArtifactTestStatus);

function artifactTestKindMime(kind: string): string {
  switch (kind) {
    case "audio":
      return "audio/wav";
    case "video":
      return "video/mp4";
    default:
      return "image/png";
  }
}

function templateArtifactTestMime(): string {
  return artifactTestKindMime(getArtifactTestCommand()?.ArtifactKind || "");
}
registerTemplateFunc("templateArtifactTestMime", templateArtifactTestMime);

function templateArtifactTestExt(): string {
  switch (getArtifactTestCommand()?.ArtifactKind) {
    case "audio":
      return ".wav";
    case "video":
      return ".mp4";
    default:
      return ".png";
  }
}
registerTemplateFunc("templateArtifactTestExt", templateArtifactTestExt);

function templateArtifactTestWrongExt(): string {
  return templateArtifactTestExt() === ".jpg" ? ".png" : ".jpg";
}
registerTemplateFunc(
  "templateArtifactTestWrongExt",
  templateArtifactTestWrongExt,
);

// Builds a mock response JSON matching the artifact command's content pointer
// by folding the pointer segments outside-in around a terminal content item.
// variant: "inline" (base64 data), "uri" (URI-only delivery), "empty" (no
// content items), "notready" (empty with a non-terminal status), "wrongkind"
// (a text block only), "multiple" (an intermediate block before the terminal),
// "multiple-missing-data" / "multiple-empty-data" (an intermediate inline block
// before a terminal of the declared kind that carries no usable data).
function templateArtifactTestMockJSON(variant: string, mime: string): string {
  const cmd = getArtifactTestCommand();
  if (!cmd) return "{}";
  return artifactTestMockJSON(JSON.parse(cmd.ArtifactJSON), variant, mime);
}
registerTemplateFunc(
  "templateArtifactTestMockJSON",
  templateArtifactTestMockJSON,
);

// Whether the artifact test command's pointer contains a [*]; only then can
// a response carry more than one candidate block.
function templateArtifactTestPointerHasWild(): boolean {
  const cmd = getArtifactTestCommand();
  if (!cmd) return false;
  const spec = JSON.parse(cmd.ArtifactJSON);
  return (spec.pointer || []).some((seg: any) => Boolean(seg.wild));
}
registerTemplateFunc(
  "templateArtifactTestPointerHasWild",
  templateArtifactTestPointerHasWild,
);

// artifactTestMockJSON builds the mock body from whatever field bindings the
// spec carries (absent keys are the defaults, matching the runtime), so the
// harness tracks the declared content-block shape instead of assuming one.
function artifactTestMockJSON(
  spec: any,
  variant: string,
  mime: string,
): string {
  const kind = spec.kind;
  const typeField = spec.typeField || "type";
  const dataField = spec.dataField || "data";
  const mimeField = spec.mimeTypeField || "mime_type";
  const uriField = spec.uriField || "uri";
  let terminal: any;
  switch (variant) {
    case "uri":
      terminal = {
        [typeField]: kind,
        [uriField]: "https://media.example.test/asset",
      };
      break;
    case "wrongkind":
      terminal = { [typeField]: "text", text: "no media here" };
      break;
    case "multiple-missing-data":
      terminal = { [typeField]: kind, [mimeField]: mime };
      break;
    case "multiple-empty-data":
      terminal = { [typeField]: kind, [dataField]: "", [mimeField]: mime };
      break;
    default:
      // "artifact-bytes" base64-encoded without relying on runtime helpers.
      terminal = {
        [typeField]: kind,
        [dataField]: "YXJ0aWZhY3QtYnl0ZXM=",
        [mimeField]: mime,
      };
  }
  // "multiple" puts an intermediate block before the terminal in the
  // innermost [*], wrapped in the same trailing field segments so both are
  // candidates. Without a [*] it degenerates to "inline".
  let leading: any = variant.startsWith("multiple")
    ? {
        [typeField]: kind,
        [dataField]: "aW50ZXJtZWRpYXRl",
        [mimeField]: mime,
      }
    : undefined;
  // "empty" and "notready" start the fold with no terminal: the innermost
  // [*] becomes an empty array and every outer segment wraps normally.
  let node: any =
    variant === "empty" || variant === "notready" ? undefined : terminal;
  const segments = [...(spec.pointer || [])].reverse();
  // Field segments folded before the innermost [*] also wrap the leading block.
  let beforeFirstWild = true;
  for (const seg of segments) {
    if (seg.wild) {
      if (node === undefined) {
        node = [];
      } else if (beforeFirstWild && leading !== undefined) {
        node = [leading, node];
      } else {
        node = [node];
      }
      beforeFirstWild = false;
    } else {
      node = node === undefined ? {} : { [seg.field]: node };
      if (beforeFirstWild && leading !== undefined) {
        leading = { [seg.field]: leading };
      }
    }
  }
  // Object roots gain response identity fields ("notready" carries a
  // non-terminal status); array roots ($[*]...) are preserved as-is.
  const status =
    variant === "notready"
      ? "artifact_test_pending"
      : spec.terminalStatus || "completed";
  const root =
    typeof node === "object" && node !== null && !Array.isArray(node)
      ? {
          [spec.idField || "id"]: "task-123",
          [spec.statusField || "status"]: status,
          ...node,
        }
      : node;
  return JSON.stringify(root ?? {});
}

// The custom-shape runtime test renames every binding by deriving from the
// command's real (declaration-or-default) shape: each member name and the
// terminal status gain a fixed prefix, so the synthetic shape is guaranteed
// distinct from the declared one — a body in the synthetic shape can never
// satisfy the real annotation, whatever names the document declared (even
// the example names the docs advertise).
function artifactCustomShape(spec: any): Record<string, string> {
  return {
    typeField: "custom_" + (spec.typeField || "type"),
    dataField: "custom_" + (spec.dataField || "data"),
    mimeTypeField: "custom_" + (spec.mimeTypeField || "mime_type"),
    uriField: "custom_" + (spec.uriField || "uri"),
    idField: "custom_" + (spec.idField || "id"),
    statusField: "custom_" + (spec.statusField || "status"),
    terminalStatus: "custom_" + (spec.terminalStatus || "completed"),
  };
}

// The real command's annotation with every binding renamed: injected over the
// generated annotation by the custom-shape harness test.
function templateArtifactTestCustomAnnotation(): string {
  const cmd = getArtifactTestCommand();
  if (!cmd) return "";
  const spec = JSON.parse(cmd.ArtifactJSON);
  return JSON.stringify({
    ...spec,
    ...artifactCustomShape(spec),
  });
}
registerTemplateFunc(
  "templateArtifactTestCustomAnnotation",
  templateArtifactTestCustomAnnotation,
);

function templateArtifactTestCustomMockJSON(
  variant: string,
  mime: string,
): string {
  const cmd = getArtifactTestCommand();
  if (!cmd) return "{}";
  const spec = JSON.parse(cmd.ArtifactJSON);
  return artifactTestMockJSON(
    { ...spec, ...artifactCustomShape(spec) },
    variant,
    mime,
  );
}
registerTemplateFunc(
  "templateArtifactTestCustomMockJSON",
  templateArtifactTestCustomMockJSON,
);

// The derived custom terminal status, echoed into the generated test's
// envelope assertion (the mock's status member carries it on success).
function templateArtifactTestCustomTerminalStatus(): string {
  const cmd = getArtifactTestCommand();
  if (!cmd) return "";
  return artifactCustomShape(JSON.parse(cmd.ArtifactJSON)).terminalStatus;
}
registerTemplateFunc(
  "templateArtifactTestCustomTerminalStatus",
  templateArtifactTestCustomTerminalStatus,
);

// Identity assertions only make sense when the mock root is an object (an
// array-rooted pointer cannot carry id/status members).
function templateArtifactTestRootIsObject(): boolean {
  const cmd = getArtifactTestCommand();
  if (!cmd) return false;
  const spec = JSON.parse(cmd.ArtifactJSON);
  return !(spec.pointer || [])[0]?.wild;
}
registerTemplateFunc(
  "templateArtifactTestRootIsObject",
  templateArtifactTestRootIsObject,
);

// =============================================================================
// Request-body JSON Schemas (--schema surface)
// =============================================================================

interface BodySchemaEntry {
  OpID: string;
  JSON: string; // exact bundled JSON Schema, Go-string-escaped by the template
}

function collectBodySchemaEntries(): BodySchemaEntry[] {
  const schemas = context.Global.AST.CLIBodySchemas;
  if (!schemas) return [];
  const entries: BodySchemaEntry[] = [];
  for (const opID of Object.keys(schemas)) {
    entries.push({ OpID: opID, JSON: schemas[opID] });
  }
  entries.sort((a, b) => (a.OpID < b.OpID ? -1 : 1));
  return entries;
}

function hasBodySchemas(): boolean {
  return collectBodySchemaEntries().length > 0;
}
registerTemplateFunc("hasBodySchemas", hasBodySchemas);

function hasBodySchemaForOp(op: Operation): boolean {
  const schemas = context.Global.AST.CLIBodySchemas;
  return Boolean(schemas && schemas[op.OriginalID]);
}
registerTemplateFunc("hasBodySchemaForOp", hasBodySchemaForOp);

// Stable cobra Group ID from a help category title (e.g. "Create" → "create").
// Mirrored by cliCategoryID in internal/extensions/cli_commands.go, which
// rejects titles that would collide (or normalize to nothing) in this space
// at decode time. Keep the two normalizations identical.
function intentCategoryID(category: string): string {
  return category
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}
registerTemplateFunc("intentCategoryID", intentCategoryID);

// First word of a cobra Use string ("generate [prompt]" → "generate").
function firstUseWord(use: string): string {
  return use.split(" ")[0];
}
registerTemplateFunc("firstUseWord", firstUseWord);

// flagutil.FlagKind identifier for a declared intent flag type. Declared
// flags are limited to scalars (string | int | float | bool) by the decoder.
function intentFlagKind(type: string): string {
  switch (type) {
    case "bool":
      return "FlagKindBool";
    case "int":
      return "FlagKindInt64";
    case "float":
      return "FlagKindFloat64";
    default:
      return "FlagKindString";
  }
}
registerTemplateFunc("intentFlagKind", intentFlagKind);

// Go []string literal for a command path, e.g. `[]string{"agent", "run"}`.
function intentPathLiteral(path: string[]): string {
  return `[]string{${path
    .map((seg) => `"${escapeGoString(seg)}"`)
    .join(", ")}}`;
}
registerTemplateFunc("intentPathLiteral", intentPathLiteral);
