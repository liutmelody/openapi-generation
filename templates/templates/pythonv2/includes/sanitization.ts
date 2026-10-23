// @ts-ignore
const pythonReservedKeywords = [
  "def",
  "if",
  "raise",
  "del",
  "import",
  "return",
  "elif",
  "in",
  "try",
  "and",
  "else",
  "is",
  "while",
  "as",
  "except",
  "lambda",
  "with",
  "assert",
  "finally",
  "nonlocal",
  "yield",
  "break",
  "for",
  "not",
  "class",
  "from",
  "or",
  "continue",
  "global",
  "pass",
  "async",
  "self",
  "filter",
  "format",
  "isinstance",
  "await",
];

// @ts-ignore
// Only true Python reserved keywords that cause syntax errors (not built-ins)
const methodReservedKeywords = [
  "def",
  "if",
  "raise",
  "del",
  "import",
  "return",
  "elif",
  "in",
  "try",
  "and",
  "else",
  "is",
  "while",
  "as",
  "except",
  "lambda",
  "with",
  "assert",
  "finally",
  "nonlocal",
  "yield",
  "break",
  "for",
  "not",
  "class",
  "from",
  "or",
  "continue",
  "global",
  "pass",
  "async",
  "await",
];

// @ts-ignore
const paramReservedKeywords = [
  "type",
  "date",
  "datetime",
  "deprecated",
  "str",
  "any",
  "input",
  "bool",
  "int",
  "map",
  "bytes",
  "server_url",
  "timeout_ms",
  "http_headers",
  "retries",
  "self",
  "url_override",
  "accept_header_override",
];
if (context.Global.Config.MethodTimeoutArgument === "timeout") {
  paramReservedKeywords.push("timeout");
}
if (context.Global.Config.MethodArguments === "positional-path-with-extras") {
  paramReservedKeywords.push("extra_headers", "extra_query", "extra_body");
}
const filteredParamReservedKeywords = paramReservedKeywords.filter(
  (i) => !context?.Global?.Config?.AllowedRedefinedBuiltins.includes(i),
);

const classReservedKeywords = [
  "False",
  "None",
  "True",
  "Warning",
  "Exception",
  "EnvironmentError",
  "ImportError",
  "RuntimeError",
  "ValueError",
  "BaseException",
  "GeneratorExit",
  "KeyboardInterrupt",
  "SystemExit",
  "Exception",
  "StopIteration",
  "OSError",
  "ArithmeticError",
  "AssertionError",
  "AttributeError",
  "BufferError",
  "EOFError",
  "ImportError",
  "LookupError",
  "MemoryError",
  "NameError",
  "ReferenceError",
  "RuntimeError",
  "StopAsyncIteration",
  "SyntaxError",
  "SystemError",
  "TypeError",
  "ValueError",
  "FloatingPointError",
  "OverflowError",
  "ZeroDivisionError",
  "ModuleNotFoundError",
  "IndexError",
  "KeyError",
  "UnboundLocalError",
  "BlockingIOError",
  "ChildProcessError",
  "ConnectionError",
  "BrokenPipeError",
  "ConnectionAbortedError",
  "ConnectionRefusedError",
  "ConnectionResetError",
  "FileExistsError",
  "FileNotFoundError",
  "InterruptedError",
  "IsADirectoryError",
  "NotADirectoryError",
  "PermissionError",
  "ProcessLookupError",
  "TimeoutError",
  "NotImplementedError",
  "RecursionError",
  "IndentationError",
  "TabError",
  "UnicodeError",
  "UnicodeDecodeError",
  "UnicodeEncodeError",
  "UnicodeTranslateError",
  "Warning",
  "UserWarning",
  "DeprecationWarning",
  "SyntaxWarning",
  "RuntimeWarning",
  "FutureWarning",
  "PendingDeprecationWarning",
  "ImportWarning",
  "UnicodeWarning",
  "BytesWarning",
  "ResourceWarning",
  "BaseExceptionGroup",
  "PythonFinalizationError",
  "List",
  "Enum",
  "Any",
  "Undefined",
  "Field",
  // Pydantic names imported into generated model files — a schema with one of
  // these names would shadow the import and break the generated code.
  "ConfigDict",
  "Discriminator",
  "SkipValidation",
  // SDK `types` module names imported into generated model/enum files.
  // Keep in sync with types/__init__.py's __all__.
  "BaseModel",
  "OptionalNullable",
  "Nullable",
  "UNSET",
  "UNSET_SENTINEL",
  "Base64EncodedString",
  "Base64FileInput",
  "UnrecognizedInt",
  "UnrecognizedStr",
  // typing module names commonly imported in generated model files
  "Optional",
  "Union",
  "Dict",
  "Callable",
  "Mapping",
  "Tuple",
  "Literal",
];
const fieldReservedKeywords = [
  "json",
  "str",
  "int",
  "float",
  "bool",
  "dict",
  "date",
  "datetime",
  "bytes",
  // Pydantic base model fields
  "schema",
  "validate",
  // No longer required for Pydantic (moved to class method as of 2025-03), but
  // this reserved keyword is left for backwards compatibility to prevent code
  // churn.
  "model_fields",
  // Pydantic BaseModel class-level attributes — property names that collide
  // with these override the attribute with an incompatible type.
  "model_config",
  "model_computed_fields",
  // Pydantic BaseModel methods — property names that collide with these
  // override the method with an incompatible type signature.
  "copy",
  "parse_obj",
  "model_copy",
  "model_dump",
  "model_dump_json",
  "model_validate",
  "model_validate_json",
  "model_json_schema",
  "model_post_init",
];

// @ts-ignore
function sanitizeSDKAccess(sdk: SDK): string {
  const group = sdk.Group;

  if (!group) {
    return "";
  }

  let formattedName = "";
  if (group.includes(".") && ".".repeat(group.length) !== group) {
    const pieces = group.split(".");
    formattedName = sanitizeSubSDKFieldName(pieces[0]);
    for (const i of pieces.slice(1)) {
      formattedName += "." + sanitizeSubSDKFieldName(i);
    }
  } else {
    formattedName = sanitizeSubSDKFieldName(group);
  }

  if (formattedName != "") {
    formattedName = `.${formattedName}`;
  }

  return formattedName;
}

registerTemplateFunc("sanitizeSDKAccess", sanitizeSDKAccess);

// SDK-internal module and package names that must not be shadowed by
// operation-group files.  A tag whose sanitised name matches one of these
// entries will have a trailing underscore appended to the generated file name
// (e.g. "errors" → "errors_.py").
// @ts-ignore
const reservedTagNames = [
  "models",
  "utils",
  "types",
  "sdk",
  "sdkconfiguration",
  "basesdk",
  "httpclient",
  "errors",
];

// @ts-ignore
function sanitizeSDKFileName(
  name: string,
  forceLower: boolean = false,
): string {
  let fileName = sanitizeFileName(name);
  if (reservedTagNames.includes(fileName)) {
    fileName += "_";
  }

  if (forceLower) {
    fileName = fileName.toLowerCase();
  }

  return fileName;
}

registerTemplateFunc("sanitizeSDKFileName", sanitizeSDKFileName);

// @ts-ignore
function sanitizeFieldName(name: string): string {
  name = sanitizeAttributeName(name);

  if (
    context.Global.Config.ModelSerializationMethods &&
    ["to_dict", "to_json"].includes(name)
  ) {
    name += "_";
  }

  return name;
}
registerTemplateFunc("sanitizeFieldName", sanitizeFieldName);

function sanitizeAttributeName(name: string): string {
  name = sanitizeName(name);
  if (name.endsWith("_")) {
    name = name.substring(0, name.length - 1);
  }
  name = caser().ToSnake(name);

  if (
    fieldReservedKeywords.includes(name) ||
    pythonReservedKeywords.includes(name)
  ) {
    name += "_";
  }

  return name;
}

function useConflictResistantModelImports(): boolean {
  return !!context.Global.Config.FixFlags?.[
    "conflictResistantModelImportsFeb2026"
  ];
}

function doesNameConflictWithModelsPackage(name: string): boolean {
  const paths = [
    getModelsLocation(context.Global.Config.Imports.GetSharedPath()),
    getModelsLocation(context.Global.Config.Imports.GetOperationsPath()),
    getModelsLocation(context.Global.Config.Imports.GetErrorsPath()),
    getModelsLocation(context.Global.Config.Imports.GetWebhooksPath()),
    getModelsLocation(context.Global.Config.Imports.GetCallbacksPath()),
  ];

  return paths.some((path) => path == name);
}

// Checks if a module name (e.g., "identity") conflicts with a top-level
// output location. This detects cases where `from pkg.errors import identity`
// would shadow `from pkg import identity`.
function doesModuleNameConflictWithTopLevelLocation(
  moduleName: string,
): boolean {
  let topLevelLocations: string[] | undefined =
    context?.GlobalComputed?.TopLevelModelLocations;
  if (!topLevelLocations) {
    topLevelLocations = [];
    for (const [outputLocation] of sequencedMapEntries(
      context.Global.AST.BucketedTypes,
    )) {
      const loc = getModelsLocation(outputLocation);
      if (!loc.includes("/")) {
        topLevelLocations.push(loc);
      }
    }
    if (context.GlobalComputed) {
      context.GlobalComputed.TopLevelModelLocations = topLevelLocations;
    }
  }

  return topLevelLocations.includes(moduleName);
}

// Checks if a module leaf name (e.g., "oauth_helper") appears in multiple
// distinct bucket locations (e.g., "models/oauth_helper" and "errors/oauth_helper").
// When two different parent paths share the same leaf module name, importing
// both in the same file would shadow one another, so we need aliased imports.
function doesModuleNameConflictAcrossNamespaces(
  moduleName: string,
  currentOutputLocation: string,
): boolean {
  let conflictingLeafNames: string[] | undefined =
    context?.GlobalComputed?.ConflictingLeafNamespaces;
  if (!conflictingLeafNames) {
    const leafCounts: Record<string, string[]> = {};
    for (const [outputLocation] of sequencedMapEntries(
      context.Global.AST.BucketedTypes,
    )) {
      const loc = getModelsLocation(outputLocation);
      const locParts = loc.split("/");
      const leafName = locParts[locParts.length - 1];
      if (!leafCounts[leafName]) {
        leafCounts[leafName] = [];
      }
      if (!leafCounts[leafName].includes(loc)) {
        leafCounts[leafName].push(loc);
      }
    }
    conflictingLeafNames = [];
    for (const [leafName, locations] of Object.entries(leafCounts)) {
      if (locations.length > 1) {
        conflictingLeafNames.push(leafName);
      }
    }
    if (context.GlobalComputed) {
      context.GlobalComputed.ConflictingLeafNamespaces = conflictingLeafNames;
    }
  }

  return conflictingLeafNames.includes(moduleName);
}

// @ts-ignore
function sanitizeSubSDKFieldName(name: string): string {
  return sanitizeAttributeName(name);
}
registerTemplateFunc("sanitizeSubSDKFieldName", sanitizeSubSDKFieldName);

// @ts-ignore
function sanitizeParameterName(name: string): string {
  name = sanitizeName(name);
  if (name.endsWith("_")) {
    name = name.substring(0, name.length - 1);
  }
  name = caser().ToSnake(name);

  if (
    filteredParamReservedKeywords.includes(name) ||
    pythonReservedKeywords.includes(name)
  ) {
    name += "_";
  }

  return name;
}
registerTemplateFunc("sanitizeParameterName", sanitizeParameterName);

function sanitizeFlattenedParameterName(name: string): string {
  return sanitizeParameterName(name);
}
registerTemplateFunc(
  "sanitizeFlattenedParameterName",
  sanitizeFlattenedParameterName,
);

// @ts-ignore
function sanitizeMethodName(operation: Operation): string {
  let name = caser().ToSnake(sanitizeName(operation.GetID()));

  if (methodReservedKeywords.includes(name)) {
    name += "_";
  }

  // If this operation's method name collides with another operation's async
  // variant (e.g., processFileAsync -> process_file_async collides with
  // processFile's async variant process_file_async), append a trailing
  // underscore to disambiguate.
  const collisions = context.RecursiveComputed?.AsyncMethodCollisions as
    | string[]
    | undefined;
  if (collisions && collisions.includes(operation.GetID())) {
    name += "_";
  }

  return name;
}
registerTemplateFunc("sanitizeMethodName", sanitizeMethodName);

// @ts-ignore
function sanitizePrivateMethodName(name: string): string {
  let name_sanitized = caser().ToSnake(sanitizeName(name));

  if (methodReservedKeywords.includes(name_sanitized)) {
    name_sanitized += "_";
  }

  return name_sanitized;
}
registerTemplateFunc("sanitizePrivateMethodName", sanitizePrivateMethodName);

// @ts-ignore
function sanitizeClassName(name: string): string {
  name = sanitizeName(name);
  name = caser().ToGoPascal(name);

  if (classReservedKeywords.includes(name)) {
    name += "T";
  }

  return name;
}

registerTemplateFunc("sanitizeClassName", sanitizeClassName);

// @ts-ignore
function sanitizeClassNameForLazyLoading(name: string): string {
  return `"${sanitizeClassName(name)}"`;
}

registerTemplateFunc(
  "sanitizeClassNameForLazyLoading",
  sanitizeClassNameForLazyLoading,
);

// @ts-ignore
function sanitizeClassNameForLazyLoadingAsync(name: string): string {
  return `"Async${sanitizeClassName(name)}"`;
}

registerTemplateFunc(
  "sanitizeClassNameForLazyLoadingAsync",
  sanitizeClassNameForLazyLoadingAsync,
);

function truncateName(
  name: string,
  maxLength: number,
  separator: string = "_",
): string {
  if (name.length <= maxLength) return name;
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = ((hash << 5) - hash + name.charCodeAt(i)) | 0;
  }
  const hashStr = Math.abs(hash).toString(16).padStart(8, "0").substring(0, 8);
  const suffixLen = separator.length + 8;
  return name.substring(0, maxLength - suffixLen) + separator + hashStr;
}

// @ts-ignore
function sanitizeFileName(name: string): string {
  name = sanitizeFile(name, "_").toLowerCase();

  if (pythonReservedKeywords.includes(name)) {
    name += "_";
  }

  // Python __pycache__ adds .cpython-3XX.pyc (~20 chars) plus temp suffixes,
  // so we use a conservative limit to account for compiled filenames.
  name = truncateName(name, 220);

  return name;
}

registerTemplateFunc("sanitizeFileName", sanitizeFileName);

/**
 * Sanitizes an output location path into a valid Python module path by
 * converting each segment to snake_case and appending "_" to reserved words.
 *
 * Examples:
 *   "" → ""
 *   "models/components" → "models/components"
 *   "MyNamespace" → "my_namespace"
 *   "models/Match" → "models/match_"  (soft keyword)
 *   "Foo-Bar/Type" → "foo_bar/type_"  (reserved keyword)
 */
// @ts-ignore
function sanitizeOutputLocation(outputLocation: string): string {
  if (!outputLocation) return outputLocation;
  // Python 3.10+ soft keywords that cause issues as directory/module names
  // but are safe as field/param names (so they're not in pythonReservedKeywords).
  const outputLocationReservedKeywords = ["match", "case", "type"];
  return outputLocation
    .split("/")
    .map((segment) => {
      let name = caser().ToSnake(sanitizeName(segment));
      if (
        pythonReservedKeywords.includes(name) ||
        outputLocationReservedKeywords.includes(name)
      ) {
        name += "_";
      }
      return name;
    })
    .join("/");
}

// @ts-ignore
function useUpperCaseConsts(): boolean {
  return context?.Global?.Config?.ConstFieldCasing !== "normal";
}

registerTemplateFunc("useUpperCaseConsts", useUpperCaseConsts);

// @ts-ignore
function templateConstName(name: string, prefix: string): string {
  name = sanitizeName(name);
  return caser().ToSNAKE(`${prefix}_${name}`);
}

registerTemplateFunc("templateConstName", templateConstName);

const DEFAULT_TYPED_DICT_SUFFIX = "TypedDict";

function inputTypedDictSuffix(): string {
  return caser().ToPascal(context.Global.Config.InputTypedDictSuffix as string);
}

// Cache for TypedDict name overrides, lazily computed from BucketedTypes.
// Keyed by TypeDef.GetRegistrationID() so two TypeDefs that sanitize to
// the same class name across buckets resolve independently.
let _typedDictNameOverrides: Record<string, string> | null = null;

function getTypedDictNameOverrides(): Record<string, string> {
  if (_typedDictNameOverrides) {
    return _typedDictNameOverrides;
  }
  _typedDictNameOverrides = {};
  const bucketedTypes = context.Global.AST.BucketedTypes;
  const isInputTypedDictSuffixRenamed =
    inputTypedDictSuffix() !== DEFAULT_TYPED_DICT_SUFFIX;

  for (const [, models] of sequencedMapEntries(bucketedTypes)) {
    const allClassNames = new Set<string>();
    const claimedNames = new Set<string>();

    if (isInputTypedDictSuffixRenamed) {
      for (const [, tt] of sequencedMapEntries(models)) {
        for (const t of tt) {
          if (t.Scope.toString() !== "utils") {
            allClassNames.add(sanitizeClassName(t.Name));
          }
        }
      }
    }

    for (const [, tt] of sequencedMapEntries(models)) {
      if (!isInputTypedDictSuffixRenamed) {
        allClassNames.clear();
        claimedNames.clear();
        for (const t of tt) {
          if (t.Scope.toString() !== "utils") {
            allClassNames.add(sanitizeClassName(t.Name));
          }
        }
      }

      for (const t of tt) {
        if (t.Scope.toString() === "utils") continue;
        const cn = sanitizeClassName(t.Name);
        const inputSuffix = t.UsedInRequest
          ? inputTypedDictSuffix()
          : DEFAULT_TYPED_DICT_SUFFIX;
        const typedDictSuffixes = [
          inputSuffix,
          inputSuffix + "Model",
          inputSuffix + "Companion",
          inputSuffix + "CompanionModel",
        ];
        const defaultCompanion = cn + inputSuffix;

        if (
          !allClassNames.has(defaultCompanion) &&
          !claimedNames.has(defaultCompanion)
        ) {
          // No conflict — use the chosen suffix.
          claimedNames.add(defaultCompanion);
          if (inputSuffix !== DEFAULT_TYPED_DICT_SUFFIX) {
            _typedDictNameOverrides[t.GetRegistrationID()] = defaultCompanion;
          }
          continue;
        }

        // Try cascading suffixes.
        let resolved = false;
        for (const suffix of typedDictSuffixes) {
          const candidate = cn + suffix;
          if (!allClassNames.has(candidate) && !claimedNames.has(candidate)) {
            _typedDictNameOverrides[t.GetRegistrationID()] = candidate;
            claimedNames.add(candidate);
            resolved = true;
            break;
          }
        }

        // All fixed suffixes exhausted — fall back to numeric counter.
        if (!resolved) {
          let counter = 2;
          while (true) {
            const candidate = cn + `${inputSuffix}${counter}`;
            if (!allClassNames.has(candidate) && !claimedNames.has(candidate)) {
              _typedDictNameOverrides[t.GetRegistrationID()] = candidate;
              claimedNames.add(candidate);
              break;
            }
            counter++;
          }
        }
      }
    }
  }
  return _typedDictNameOverrides;
}

function resolveTypedDictName(typeDef: TypeDef, baseName: string): string {
  return (
    getTypedDictNameOverrides()[typeDef.GetRegistrationID()] ??
    baseName + DEFAULT_TYPED_DICT_SUFFIX
  );
}

// Cache: maps model name → set of sanitized class names in that model.
// Used to detect when a pydantic import (e.g. Tag) would shadow a user-defined
// class in the same file, so we can fall back to namespace import.
let _modelClassNameSets: Map<string, Set<string>> | null = null;

function getModelClassNames(modelName: string): Set<string> {
  if (!_modelClassNameSets) {
    _modelClassNameSets = new Map();
    const bucketedTypes = context.Global.AST.BucketedTypes;
    for (const [, models] of sequencedMapEntries(bucketedTypes)) {
      for (const [model, tt] of sequencedMapEntries(models)) {
        const names = _modelClassNameSets.get(model) ?? new Set<string>();
        for (const t of tt) {
          if (t.Scope.toString() !== "utils") {
            names.add(sanitizeClassName(t.Name));
          }
        }
        _modelClassNameSets.set(model, names);
      }
    }
  }
  return _modelClassNameSets.get(modelName) ?? new Set();
}

/**
 * ClassPrefix resolves the Python module prefix needed to reference a type
 * from a given usage location. Python doesn't have a single global namespace
 * like TypeScript — each .py file is its own module, so cross-module
 * references need explicit `import` + qualified access (e.g. `models.Widget`).
 *
 * The prefix depends on:
 *  - Whether we're inside the same model file (no prefix needed)
 *  - Whether the type is in a different module (needs module prefix)
 *  - Whether names collide with SDK methods, params, or sub-SDK classes
 *  - Whether circular imports require forward-quoted type strings
 *
 * Each function also side-effects the import list (via addImport/addTypeImport)
 * so the correct `from ... import ...` statement appears at the top of the file.
 */
namespace ClassPrefix {
  // ── Public API ──────────────────────────────────────────────────────

  export function resolve(
    typeDef: TypeDef & { ParentName?: string },
    usageLocation: string,
    owningModelInfo: PythonV2Type | undefined,
    addImports: boolean,
    suffixForTypes: string | undefined,
    quoteCustomTypes: boolean,
  ): { prefix: string; quoteCustomTypes: boolean } {
    // Case 1: Referenced from within a model definition (not tests).
    // owningModelInfo is set when rendering field type annotations inside a
    // Pydantic model file (model.py.stmpl passes .Local as owningModelInfo).
    // It is null for all other contexts: SDK methods, pagination, usage
    // examples, SDK configuration, and enum definitions.
    // Delegates to resolvePrefixFromWithinModel which uses the owning model
    // to detect circular imports and cross-model qualification needs.
    if (owningModelInfo && usageLocation != "tests") {
      return resolvePrefixFromWithinModel(
        typeDef,
        usageLocation,
        owningModelInfo,
        addImports,
        suffixForTypes,
        quoteCustomTypes,
      );
    }

    // Case 2: Usage-site reference (e.g. README snippets) with global imports
    // enabled. The type lives in a models module, so prefix with the global
    // module name (e.g. `mypackage.Widget`).
    if (
      usageLocation == "usage" &&
      usingGlobalImports() &&
      typeDef.OutputLocation != "usage" &&
      typeDef.OutputLocation != "tests"
    ) {
      const moduleImport = getModuleImport();
      addImport(moduleImport, "");
      return { prefix: `${moduleImport}.`, quoteCustomTypes };
    }

    // Case 3: Cross-location reference — the type's models location differs
    // from where it's being used. Resolves the appropriate namespace prefix
    // (flat namespace, sub-SDK collision alias, or plain module import).
    const modelsLocation = getModelsLocation(typeDef.OutputLocation);
    if (modelsLocation != usageLocation) {
      const prefix = resolveCrossLocationPrefix(
        typeDef,
        usageLocation,
        addImports,
        suffixForTypes,
      );
      return { prefix, quoteCustomTypes };
    }

    // Case 4: Same-location reference — type is defined in the same module,
    // so no prefix is needed.
    return { prefix: "", quoteCustomTypes };
  }

  /**
   * Inside method bodies, module-reference prefixes may collide with local
   * variables or parameters that shadow the module name. This appends "_" to
   * the prefix (e.g. `models` → `models_`) so the downstream rewriter
   * (`rewriteMethodLocals`) can detect and resolve the collision. Outside
   * method bodies, the prefix is returned unchanged.
   */
  export function applyMethodBodyRewriting(prefix: string): string {
    if (!isInMethodBody()) {
      return prefix;
    }
    const prefixName = prefix.slice(0, -1); // strip trailing "."
    const moduleRefs = getModuleRefNames();
    if (moduleRefs.includes(prefixName)) {
      return `${prefixName}_.`;
    }
    return prefix;
  }

  // ── Internal helpers ────────────────────────────────────────────────

  /**
   * Returns the sanitized field names of all sub-SDK classes (e.g. ["users",
   * "billing"]). These names can collide with module imports when a type's
   * namespace happens to match a sub-SDK name. Cached on GlobalComputed
   * since sub-SDK structure is constant for the entire generation run.
   */
  function getSubSDKClassMembers(): string[] {
    let classMembers = context?.GlobalComputed?.SubSDKClassMembers;
    if (classMembers) {
      return classMembers;
    }
    classMembers =
      context?.Global?.AST?.MainSDK?.SubSDKs?.map((s: SDK) =>
        sanitizeSubSDKFieldName(s.FieldName),
      ) || [];
    if (context.GlobalComputed) {
      context.GlobalComputed.SubSDKClassMembers = classMembers;
    }
    return classMembers;
  }

  /**
   * Resolves the prefix for a type referenced from within another model's
   * definition (i.e. owningModelInfo is set). This is the most complex case
   * because it must handle:
   *
   * 1. Utils-scoped types — always prefixed with `utils.` (note: currently
   *    no Python TypeDefs use utils scope, but the path exists for parity
   *    with other targets)
   * 2. Cross-model references in a different location — qualified with the
   *    type's import alias (e.g. `listtest1op.SomeType`)
   * 3. Circular references — if two models reference each other, the type
   *    annotation must be forward-quoted (`"SomeType"`) to avoid import cycles
   */
  function resolvePrefixFromWithinModel(
    typeDef: TypeDef & { ParentName?: string },
    usageLocation: string,
    owningModelInfo: PythonV2Type,
    addImports: boolean,
    suffixForTypes: string | undefined,
    quoteCustomTypes: boolean,
  ): { prefix: string; quoteCustomTypes: boolean } {
    let prefix = "";

    // Utils-scoped types live in the well-known `utils` module and always
    // use the fixed `utils.` prefix rather than a dynamic import alias.
    if (typeDef.Scope.toString() == "utils") {
      prefix = `utils.`;
      addImports && addUtilsImport();
    }

    const isCrossModel =
      !typeDef.Truncated &&
      !inSameModel(owningModelInfo.ModelName, usageLocation, typeDef);

    // Cross-model reference in a different models location and not already
    // prefixed (e.g. by utils). Qualify with the type's import alias so the
    // reference resolves across module boundaries.
    if (
      !prefix &&
      isCrossModel &&
      usageLocation != getModelsLocation(typeDef.OutputLocation)
    ) {
      prefix = `${templateTypeImportAlias(typeDef)}.`;
    }

    // For any cross-model reference, check for circular dependencies and
    // register the import. Circular refs get forward-quoted annotations.
    if (isCrossModel) {
      const isCircular = areCircular(owningModelInfo.Type, typeDef);
      quoteCustomTypes = isCircular;
      if (addImports) {
        addTypeImport(typeDef, usageLocation, suffixForTypes, isCircular);
      }
    }

    return { prefix, quoteCustomTypes };
  }

  /**
   * Resolves a prefix using the conflict-resistant import strategy. Instead of
   * importing each model module individually, this imports the root package
   * (e.g. `from pkg import models`) and uses dotted attribute access
   * (e.g. `models.Widget` or `models.foo.Widget` for nested namespaces).
   *
   * If the root name collides with an operation parameter or method name,
   * it's aliased with a trailing underscore (e.g. `models_`).
   */
  function resolveConflictResistantPrefix(
    parts: string[],
    addImports: boolean,
  ): string {
    const rootName = parts[0];
    // Check if the root module name (e.g. "models") collides with any
    // parameter or method name in the current scope
    const allColliders = [
      ...(context.RecursiveComputed?.AllParamNames ?? []),
      ...(context.RecursiveComputed?.AllMethodNames ?? []),
    ];
    const needsAlias = allColliders.includes(rootName);
    if (needsAlias) {
      addModuleAlias(rootName);
    }
    const rootPrefix = needsAlias ? `${rootName}_` : rootName;
    addImports && addImport("", rootName, true);
    // e.g. ["models"] → "models.", ["models", "foo"] → "models.foo."
    return `${[rootPrefix, ...parts.slice(1)].join(".")}.`;
  }

  /**
   * Resolves the prefix for a type whose models location differs from where
   * it's being referenced. This is the general cross-location handler, called
   * when no owningModelInfo is available (e.g. from SDK methods, __init__
   * files, or test files).
   *
   * Tries several strategies in order:
   * 1. Conflict-resistant import (if enabled) — `models.Widget`
   * 2. Sub-SDK collision alias — when the module name matches a sub-SDK class
   * 3. Cross-namespace collision alias — when the module name conflicts with
   *    another namespace or top-level location
   * 4. Plain module import — default fallback, e.g. `modulename.Widget`
   */
  function resolveCrossLocationPrefix(
    typeDef: TypeDef & { ParentName?: string },
    usageLocation: string,
    addImports: boolean,
    suffixForTypes: string | undefined,
  ): string {
    const modelsLocation = getModelsLocation(typeDef.OutputLocation);
    const parts = modelsLocation.split("/");
    let moduleName = parts[parts.length - 1];

    // Test helper types live in a "tests" directory but are imported as
    // "test_helpers" to avoid shadowing pytest's test discovery
    if (moduleName == "tests") {
      moduleName = "test_helpers";
    }

    // Strategy 1: Conflict-resistant import — use a single root import like
    // `from pkg import models` and reference via dotted access
    // (e.g., models.Widget or models.ns.Widget).
    if (
      useConflictResistantModelImports() &&
      parts[0] === "models" &&
      moduleName !== "tests"
    ) {
      return resolveConflictResistantPrefix(parts, addImports);
    }

    // Strategy 2: Input parameter type conflicting with a sub-SDK class
    // member. E.g. if there's a `users` sub-SDK and a `users` model module,
    // we can't `import users` — use an aliased import instead.
    if (
      context?.LocalComputed?.["isInputParameterType"] &&
      getSubSDKClassMembers().includes(moduleName)
    ) {
      const importAlias = templateTypeImportAlias(typeDef);
      addImports &&
        addTypeImport(typeDef, usageLocation, suffixForTypes, false, true);
      return `${importAlias}.`;
    }

    // Strategy 3: Multi-segment path where the sub-namespace name conflicts
    // with another namespace or top-level location — use aliased module
    // import to disambiguate (e.g. `models_foo` instead of `foo`).
    if (
      parts.length > 1 &&
      (doesModuleNameConflictWithTopLevelLocation(moduleName) ||
        doesModuleNameConflictAcrossNamespaces(
          moduleName,
          typeDef.OutputLocation,
        ))
    ) {
      const importAlias = templateTypeImportAlias(typeDef);
      addImports &&
        addTypeImport(typeDef, usageLocation, suffixForTypes, false, true);
      return `${importAlias}.`;
    }

    // Strategy 4: No conflicts — plain module import (e.g. `modulename.`)
    addImports && addTypeImport(typeDef, usageLocation, suffixForTypes);
    return `${moduleName}.`;
  }
}

/**
 * Returns the Python string to reference a type — either a bare name for
 * definitions or a module-qualified name for usages.
 *
 * When definition=true, returns the bare class name (e.g. "Widget").
 * When definition=false, resolves a module prefix via ClassPrefix.resolve():
 *
 *   - owningModelInfo provided: we're inside a Pydantic model file rendering
 *     field type annotations (model.py.stmpl passes .Local as owningModelInfo).
 *     Delegates to resolvePrefixFromWithinModel — handles circular-reference
 *     forward-quoting, cross-model import aliases, and utils scope.
 *
 *   - owningModelInfo is null for all other contexts: SDK method signatures,
 *     pagination return types, usage/README examples, SDK configuration, and
 *     enum class definitions. These fall through to the cases below.
 *
 *   - usageLocation="usage" + global imports: prefixes with the global module.
 *
 *   - cross-location reference (models location != usageLocation):
 *     resolveCrossLocationPrefix — flat namespace ("models.foo.Widget"),
 *     sub-SDK class member collisions, multi-segment namespace conflicts,
 *     or plain module prefix.
 *
 * Inside method bodies, applyMethodBodyRewriting appends "_" to module-ref
 * prefixes so the downstream rewriter can strip it when there's no collision.
 */
// @ts-ignore
function sanitizeClass(
  typeDef: TypeDef & { ParentName?: string },
  usageLocation: string,
  definition: boolean,
  owningModelInfo?: PythonV2Type,
  addImports = true,
  quoteCustomTypes = false,
  suffixForTypes?: string,
): string {
  const baseName = sanitizeClassName(typeDef.Name);
  let className: string;
  // When the suffix is exactly "TypedDict", check for disambiguation overrides.
  // If the default companion name "FooTypedDict" collides with another type's
  // class name in the same model, use "Foo_TypedDict" instead.
  if (suffixForTypes === DEFAULT_TYPED_DICT_SUFFIX) {
    className = resolveTypedDictName(typeDef, baseName);
  } else {
    className = baseName + (suffixForTypes || "");
  }

  if (definition) {
    return quoteCustomTypes ? `"${className}"` : className;
  }

  const resolved = ClassPrefix.resolve(
    typeDef,
    usageLocation,
    owningModelInfo,
    addImports,
    suffixForTypes,
    quoteCustomTypes,
  );

  if (resolved.prefix) {
    return ClassPrefix.applyMethodBodyRewriting(resolved.prefix) + className;
  }

  return resolved.quoteCustomTypes ? `"${className}"` : className;
}

registerTemplateFunc("sanitizeClass", sanitizeClass);

// @ts-ignore
function sanitizeAcceptEnumClass(operation: Operation): string {
  return `${caser().ToPascal(sanitizeName(operation.GetID()))}AcceptEnum`;
}
registerTemplateFunc("sanitizeAcceptEnumClass", sanitizeAcceptEnumClass);

// @ts-ignore
function sanitizeAcceptEnumMember(acceptType: string): string {
  let name = sanitizeAcceptEnumKey(acceptType);
  let value = `${acceptType.split(";")[0]}`;
  return `${name} = \"${value}\"`;
}
registerTemplateFunc("sanitizeAcceptEnumMember", sanitizeAcceptEnumMember);

// @ts-ignore
function sanitizeAcceptEnumKey(acceptType: string): string {
  return `${caser().ToSNAKE(sanitizeName(acceptType.split(";")[0]))}`;
}

// @ts-ignore
function templateStatusCode(statusCodes: string[]) {
  const checks: string[] = [];
  let reduce = statusCodes.length > 1;
  for (const statusCode of statusCodes) {
    if (statusCode.toUpperCase().includes("X")) {
      let codeRange = parseInt(statusCode[0], 10);

      checks.push(
        `http_res.status_code >= ${codeRange}00 and http_res.status_code < ${
          codeRange + 1
        }00`,
      );
      reduce = false;
    } else {
      checks.push(`http_res.status_code == ${statusCode}`);
    }
  }
  if (!reduce) {
    return checks.join(" or ");
  } else {
    return `http_res.status_code in [${statusCodes.join(", ")}]`;
  }
}

registerTemplateFunc("templateStatusCode", templateStatusCode);

function sanitizeTypeModifiers(
  fieldDef: FieldDef,
  type: string,
  flags: SanitizeTypeFlags = {},
  usageLocation: string = "",
  owningModelInfo: PythonV2Type = null,
  addImports = true,
): string {
  const optional = flags?.optional ?? fieldDef.Optional;
  const nullable = flags?.nullable ?? fieldDef.Nullable;

  if (nullable && !optional) {
    addImports && addImport("types", "Nullable", true);

    type = `Nullable[${type}]`;
  } else if (optional && !nullable) {
    addImports && addImport("typing", "Optional");

    type = `Optional[${type}]`;
  } else if (nullable && optional) {
    addImports && addImport("types", "OptionalNullable", true);

    type = `OptionalNullable[${type}]`;
  }

  return type;
}

function sanitizePydanticTypeModifiers(
  fieldDef: FieldDef,
  type: string,
  flags: SanitizeTypeFlags = {},
  usageLocation: string = "",
  owningModelInfo: PythonV2Type = null,
  addImports = true,
): string {
  type = sanitizeTypeModifiers(
    fieldDef,
    type,
    flags,
    usageLocation,
    owningModelInfo,
    addImports,
  );

  const annotations = templateTypeAnnotations(
    fieldDef,
    usageLocation,
    owningModelInfo,
  );

  if (!annotations) {
    return type;
  }

  addImports && addImport("typing_extensions", "Annotated");
  return `Annotated[${type}, ${annotations}]`;
}

function sanitizeTypedDictTypeModifiers(
  fieldDef: FieldDef,
  type: string,
  flags: SanitizeTypeFlags = {},
  usageLocation: string = "",
  owningModelInfo: PythonV2Type = null,
  addImports = true,
): string {
  const optional = flags?.optional ?? fieldDef.Optional;
  const nullable = flags?.nullable ?? fieldDef.Nullable;

  if (nullable) {
    addImports && addImport("types", "Nullable", true);

    type = `Nullable[${type}]`;
  }
  if (optional) {
    addImports && addImport("typing_extensions", "NotRequired");

    type = `NotRequired[${type}]`;
  }

  return type;
}

type SanitizeTypeFlags = {
  optional?: boolean;
  nullable?: boolean;
  /** Is this type being templated for a method return type? */
  methodReturnType?: boolean;
  /** Should array-shaped method inputs advertise iterable values? */
  iterableCollections?: boolean;
};

// @ts-ignore
function castSanitizedType(
  fieldDef: FieldDef,
  flags: SanitizeTypeFlags = {},
  usageLocation: string = "",
  owningModelInfo: PythonV2Type = null,
  addImports = true,
  quoteCustomTypes = false,
  suffixForTypes = "",
  modelSuffix = "",
  sanitizeTypeModifiersFunc = sanitizeTypeModifiers,
) {
  const type = sanitizeType(
    fieldDef,
    flags,
    usageLocation,
    owningModelInfo,
    addImports,
    quoteCustomTypes,
    suffixForTypes,
    modelSuffix,
    sanitizeTypeModifiersFunc,
  );

  if (
    fieldDef.Type.Type.toString() === "enum" &&
    fieldDef.Type.Enum?.Type.Type.toString() === "string"
  ) {
    return `utils.cast_partial(${type})`;
  } else if (fieldDef.Type.IsPrimitive()) {
    return type;
  } else {
    throw new Error(`Unhandled type: ${fieldDef.Type.Type.toString()}`);
  }
}
registerTemplateFunc("castSanitizedType", castSanitizedType);

// @ts-ignore
function sanitizeType(
  fieldDef: FieldDef,
  flags: SanitizeTypeFlags = {},
  usageLocation: string = "",
  owningModelInfo: PythonV2Type = null,
  addImports = true,
  quoteCustomTypes = false,
  suffixForTypes = "",
  modelSuffix = "",
  sanitizeTypeModifiersFunc = sanitizeTypeModifiers,
): string {
  let type = (() => {
    const typeStr = sanitizeConstFieldType(fieldDef);
    if (typeStr) {
      return typeStr;
    }

    switch (fieldDef.Type.Type.toString()) {
      case "enum":
      case "error":
      case "union":
      case "class":
        if (
          fieldDef.Type.Type.toString() != "class" &&
          fieldDef.Type.Type.toString() != "union"
        ) {
          modelSuffix = "";
        }

        const clss = sanitizeClass(
          fieldDef.Type,
          usageLocation,
          false,
          owningModelInfo,
          addImports,
          quoteCustomTypes,
          suffixForTypes + modelSuffix,
        );

        return clss;
      case "string":
        if (isBase64FileInputField(fieldDef)) {
          if (modelSuffix === DEFAULT_TYPED_DICT_SUFFIX) {
            addImports && addImport("types", "Base64FileInput", true);
            addImports && addImport("typing", "Union");
            return "Union[str, Base64FileInput]";
          }
          addImports && addImport("types", "Base64EncodedString", true);
          return "Base64EncodedString";
        }
        return "str";
      case "date":
        addImports && addImport("datetime", "date");
        return "date";
      case "date-time":
        addImports && addImport("datetime", "datetime");
        return "datetime";
      case "uuid":
        addImports && addImport("uuid", "UUID");
        return "UUID";
      case "duration":
        addImports && addImport("datetime", "timedelta");
        return "timedelta";
      case "bigint":
      case "integer":
      case "int32":
        return "int";
      case "decimal":
        addImports && addImport("decimal", "Decimal");
        return "Decimal";
      case "number":
      case "float32":
        return "float";
      case "boolean":
        return "bool";
      case "bytes":
        return "bytes";
      case "map": {
        const mapType = flags.iterableCollections ? "Mapping" : "Dict";
        addImports && addImport("typing", mapType);
        return `${mapType}[str, ${sanitizeType(
          typeDefToFieldDef(
            fieldDef.Type.ItemType,
            fieldDef,
            undefined,
            fieldDef.Type.ContainsNull,
          ),
          {
            optional: false,
            nullable: fieldDef.Type.ContainsNull,
            iterableCollections: flags.iterableCollections,
          },
          usageLocation,
          owningModelInfo,
          addImports,
          quoteCustomTypes,
          suffixForTypes,
          modelSuffix,
          sanitizeTypeModifiersFunc,
        )}]`;
      }
      case "array": {
        const collectionType = flags.iterableCollections ? "Iterable" : "List";
        addImports && addImport("typing", collectionType);
        return (
          `${collectionType}[` +
          sanitizeType(
            typeDefToFieldDef(
              fieldDef.Type.ItemType,
              fieldDef,
              undefined,
              fieldDef.Type.ContainsNull,
            ),
            {
              optional: false,
              nullable: fieldDef.Type.ContainsNull,
              iterableCollections: flags.iterableCollections,
            },
            usageLocation,
            owningModelInfo,
            addImports,
            quoteCustomTypes,
            suffixForTypes,
            modelSuffix,
            sanitizeTypeModifiersFunc,
          ) +
          "]"
        );
      }
      case "any":
        addImports && addImport("typing", "Any");
        return "Any";
      case "request":
        addImports && addImport("httpx", "");
        return "httpx.Request";
      case "response":
      case "response-stream":
        addImports && addImport("httpx", "");
        return "httpx.Response";
      case "request-stream":
        addImports && addImport("typing", "IO");
        addImports && addImport("typing", "Union");
        addImports && addImport("io", "");
        return "Union[bytes, IO[bytes], io.IOBase]";
      case "event-stream":
        addImports && addEventStreamingImport();

        let eventStreamType: string;

        // Check if we should use the flattened data field type
        const dataField = getFlattenedEventStreamField(fieldDef.Type.ItemType);
        if (dataField) {
          // Use the data field's type directly
          eventStreamType = sanitizeType(
            dataField,
            { optional: false, nullable: dataField.Nullable },
            usageLocation,
            owningModelInfo,
            addImports,
            quoteCustomTypes,
            suffixForTypes,
            modelSuffix,
            sanitizeTypeModifiersFunc,
          );
        }

        if (!eventStreamType) {
          eventStreamType = sanitizeType(
            typeDefToFieldDef(fieldDef.Type.ItemType, fieldDef),
            { optional: false, nullable: false },
            usageLocation,
            owningModelInfo,
            addImports,
            quoteCustomTypes,
            suffixForTypes,
            modelSuffix,
            sanitizeTypeModifiersFunc,
          );
        }

        if (flags.methodReturnType) {
          return `${pythonEventStreamMethodTypeRef()}[${eventStreamType}]`;
        }

        // We use a Union to avoid duplicating the models
        addImports && addImport("typing", "Union");
        return `Union[${pythonEventStreamSyncTypeRef()}[${eventStreamType}], ${pythonEventStreamAsyncTypeRef()}[${eventStreamType}]]`;
      case "jsonl":
        addImports && addJsonlImport();

        const jsonLType = sanitizeType(
          typeDefToFieldDef(fieldDef.Type.ItemType, fieldDef),
          { optional: false, nullable: false },
          usageLocation,
          owningModelInfo,
          addImports,
          quoteCustomTypes,
          suffixForTypes,
          modelSuffix,
          sanitizeTypeModifiersFunc,
        );

        if (flags.methodReturnType) {
          return `jsonl.JsonLStreamAsync[${jsonLType}]`;
        }

        // We use a Union to avoid duplicating the models
        addImports && addImport("typing", "Union");
        return `Union[jsonl.JsonLStream[${jsonLType}], jsonl.JsonLStreamAsync[${jsonLType}]]`;
      default:
        throw new Error(`Unknown type: ${fieldDef.Type.Type.toString()}`);
    }
  })();

  return sanitizeTypeModifiersFunc(
    fieldDef,
    type,
    flags,
    usageLocation,
    owningModelInfo,
    addImports,
  );
}
registerTemplateFunc("sanitizeType", sanitizeType);

function sanitizeTypedDictType(
  fieldDef: FieldDef,
  flags: SanitizeTypeFlags = {},
  usageLocation: string = "",
  owningModelInfo: PythonV2Type = null,
  addImports = true,
  quoteCustomTypes = false,
): string {
  return sanitizeType(
    fieldDef,
    flags,
    usageLocation,
    owningModelInfo,
    addImports,
    quoteCustomTypes,
    "",
    DEFAULT_TYPED_DICT_SUFFIX,
    sanitizeTypedDictTypeModifiers,
  );
}

function sanitizeConstFieldType(field: FieldDef): string {
  if (field.Const) {
    if (field.Const.Value === null) {
      addImport("typing", "Literal");
      return "Literal[None]";
    }

    switch (field.Type.Type.toString()) {
      case "string":
        addImport("typing", "Literal");
        // For open enums, include UnrecognizedStr to accept unknown values
        if (field.Type.Enum?.Open) {
          addImport("typing", "Union");
          addImport("types", "UnrecognizedStr", true);
          return `Union[Literal[${templateStringValue(
            field.Const.Value,
            field,
          )}], UnrecognizedStr]`;
        }
        return `Literal[${templateStringValue(field.Const.Value, field)}]`;
      case "bigint":
      case "integer":
      case "int32":
        addImport("typing", "Literal");
        // For open enums, include UnrecognizedInt to accept unknown values
        if (field.Type.Enum?.Open) {
          addImport("typing", "Union");
          addImport("types", "UnrecognizedInt", true);
          return `Union[Literal[${field.Const.Value}], UnrecognizedInt]`;
        }
        return `Literal[${field.Const.Value}]`;
      case "boolean":
        addImport("typing", "Literal");
        return `Literal[${field.Const.Value ? "True" : "False"}]`;
      default:
        break;
    }
  }
  return "";
}

function sanitizeTypedDictField(
  field: FieldDef,
  usageLocation: string,
  owningModelInfo: PythonV2Type = null,
): string {
  return sanitizeTypedDictType(
    field,
    { optional: field.Optional && !field.Const },
    usageLocation,
    owningModelInfo,
    true,
    false,
  );
}
registerTemplateFunc("sanitizeTypedDictField", sanitizeTypedDictField);

function sanitizePydanticType(
  fieldDef: FieldDef,
  flags: SanitizeTypeFlags = {},
  usageLocation: string = "",
  owningModelInfo: PythonV2Type = null,
  addImports = true,
  quoteCustomTypes = false,
  typeSuffix: string = "",
): string {
  return sanitizeType(
    fieldDef,
    flags,
    usageLocation,
    owningModelInfo,
    addImports,
    quoteCustomTypes,
    typeSuffix,
    "",
    sanitizePydanticTypeModifiers,
  );
}
registerTemplateFunc("sanitizePydanticType", sanitizePydanticType);

function sanitizeInputParameterType(
  fieldDef: FieldDef,
  flags: SanitizeTypeFlags = {},
  usageLocation: string = "",
  owningModelInfo: PythonV2Type = null,
): string {
  function impl() {
    const baseFlags = {
      optional: false,
      nullable: false,
      iterableCollections: flags.iterableCollections,
    };
    let type = sanitizeType(
      fieldDef,
      baseFlags,
      usageLocation,
      owningModelInfo,
    );
    const typedDictType = sanitizeType(
      fieldDef,
      baseFlags,
      usageLocation,
      owningModelInfo,
      true,
      false,
      "",
      DEFAULT_TYPED_DICT_SUFFIX,
    );

    if (type != typedDictType) {
      addImport("typing", "Union");
      type = `Union[${type}, ${typedDictType}]`;
    }
    return sanitizeTypeModifiers(
      fieldDef,
      type,
      flags,
      usageLocation,
      owningModelInfo,
      true,
    );
  }

  try {
    if (!context.LocalComputed) {
      context.LocalComputed = {};
    }
    context.LocalComputed["isInputParameterType"] = true;
    return impl();
  } finally {
    delete context.LocalComputed["isInputParameterType"];
  }
}
registerTemplateFunc("sanitizeInputParameterType", sanitizeInputParameterType);

function hasNullDefault(field: FieldDef): boolean {
  return field.Default != null && field.Default.Value === null;
}

function isOptionalNullableDiscriminatedUnion(field: FieldDef): boolean {
  const optional =
    field.Optional || isFieldIgnored(field) || hasNullDefault(field);
  const nullable = field.Nullable;

  if (!optional || !nullable) {
    return false;
  }

  const typeDef = field.Type;

  return (
    typeDef?.Type?.toString() === "union" &&
    typeDef?.Discriminator != null &&
    !typeDef?.IsUnionOpen
  );
}

function sanitizeFieldTypeAndAnnotations(
  field: FieldDef,
  usageLocation: string,
  owningModelInfo: PythonV2Type,
): string {
  let type = sanitizePydanticType(
    field,
    {
      optional:
        field.Optional || isFieldIgnored(field) || hasNullDefault(field),
    },
    usageLocation,
    owningModelInfo,
    true,
    false,
  );

  if (field.Type.Type.toString() == "event-stream") {
    addImport("pydantic", "SkipValidation");
    type = `SkipValidation[${type}]`;
  }

  if (isOptionalNullableDiscriminatedUnion(field)) {
    addImport("pydantic", "SerializeAsAny");
    type = `SerializeAsAny[${type}]`;
  }

  const annotations = templateFieldAnnotations(field);

  if (!annotations) {
    return type;
  }

  addImport("typing_extensions", "Annotated");
  return `Annotated[${type}, ${annotations}]`;
}
registerTemplateFunc(
  "sanitizeFieldTypeAndAnnotations",
  sanitizeFieldTypeAndAnnotations,
);

type UnionMember = {
  typeStr: string;
  pyrightIgnore?: string;
};

function sanitizeAdditionalPropertiesField(
  field: FieldDef,
  usageLocation: string,
  owningModelInfo: PythonV2Type,
): string {
  if (!field.IsAdditionalProperties) {
    throw new Error("Field is not additional properties");
  }
  if (field.Type.Type.toString() != "map") {
    throw new Error("Field is not a map");
  }

  addImport("typing", "Dict");
  addImport("pydantic", "");
  return `Dict[str, ${sanitizePydanticType(
    typeDefToFieldDef(
      field.Type.ItemType,
      field,
      undefined,
      field.Type.ContainsNull,
    ),
    { optional: false, nullable: field.Type.ContainsNull },
    usageLocation,
    owningModelInfo,
  )}] = pydantic.Field(init=False)`;
}
registerTemplateFunc(
  "sanitizeAdditionalPropertiesField",
  sanitizeAdditionalPropertiesField,
);

function collectUnionAssociatedTypes(
  typeDef: TypeDef,
  usageLocation: string,
  owningModelInfo: PythonV2Type = null,
  suffixForTypes?: string,
  typedDict: boolean = false,
): UnionMember[] {
  return sortTypeDefFieldCount(typeDef.AssociatedTypes).map(
    (associatedType: TypeDef & { ParentName?: string }): UnionMember => {
      const isForwardRef = areCircular(owningModelInfo.Type, associatedType);
      const fieldDef = typeDefToFieldDef(associatedType);
      const flags = { optional: false, nullable: false };
      const typeStr = typedDict
        ? sanitizeTypedDictType(
            fieldDef,
            flags,
            usageLocation,
            owningModelInfo,
            true,
            isForwardRef,
          )
        : sanitizePydanticType(
            fieldDef,
            flags,
            usageLocation,
            owningModelInfo,
            true,
            isForwardRef,
            suffixForTypes,
          );

      // Pyright flags quoted self-references inside TypeAliasType as unresolvable
      // expressions (reportInvalidTypeForm). Quoted references to *other* generated
      // aliases do not trigger this error, only true self-references do.
      const isSelfReferringArrayOrMap =
        isForwardRef &&
        (associatedType.Type.toString() === "array" ||
          associatedType.Type.toString() === "map") &&
        owningModelInfo.ModelName === associatedType.ItemType.Name;

      return {
        typeStr,
        pyrightIgnore: isSelfReferringArrayOrMap
          ? "reportInvalidTypeForm"
          : undefined,
      };
    },
  );
}

function renderTypeAliasType(
  unionName: string,
  members: UnionMember[],
): string {
  addImport("typing", "Union");
  addImport("typing_extensions", "TypeAliasType");

  if (members.some((t) => t.pyrightIgnore)) {
    const renderedMembers = members.map(({ typeStr, pyrightIgnore }) =>
      pyrightIgnore
        ? `${typeStr},  # pyright: ignore[${pyrightIgnore}]`
        : `${typeStr},`,
    );
    const unionType = `Union[\n${renderedMembers
      .map((member) => `        ${member}`)
      .join("\n")}\n    ]`;
    return `TypeAliasType("${unionName}", ${unionType})`;
  }

  return `TypeAliasType("${unionName}", Union[${members
    .map((m) => m.typeStr)
    .join(", ")}])`;
}

// @ts-ignore
// Helper function to get the Python field name for a discriminator field on a type
// When the discriminator is pre-applied as a const field, it's named based on constFieldCasing config
function getDiscriminatorFieldName(
  discriminatorPropertyName: string,
  discriminatorIsPreApplied: boolean,
): string {
  // If the discriminator is pre-applied, it's a const field and uses casing based on config
  if (discriminatorIsPreApplied) {
    const fieldName = sanitizeFieldName(discriminatorPropertyName);
    return useUpperCaseConsts() ? fieldName.toUpperCase() : fieldName;
  }
  // Otherwise, use snake_case field name
  return sanitizeFieldName(discriminatorPropertyName);
}

function sanitizeUnionType(
  typeDef: TypeDef,
  usageLocation: string,
  owningModelInfo: PythonV2Type = null,
  suffixForTypes?: string,
  suffixForUnionType?: string,
): string {
  if (typeDef.AssociatedTypes.length == 1) {
    return sanitizePydanticType(
      typeDefToFieldDef(typeDef.AssociatedTypes[0]),
      { optional: false, nullable: false },
      usageLocation,
      owningModelInfo,
      true,
      false, // do not quote if there is only one associate type, lest it get treated as a string literal.
      suffixForTypes,
    );
  }

  const unionName =
    sanitizeClassName(typeDef.Name) + (suffixForUnionType || "");

  if (typeDef.Discriminator) {
    // Open discriminated union: emit BeforeValidator(partial(parse_open_union, ...))
    if (typeDef.IsUnionOpen) {
      addImport("typing_extensions", "Annotated");
      addImport("typing", "Union");
      addImport("pydantic.functional_validators", "BeforeValidator");
      addImport("functools", "partial");
      addImport("utils.unions", "parse_open_union", true);

      const unknownClassName = `Unknown${sanitizeClassName(typeDef.Name)}`;
      const variantsDictName = `_${sanitizeFieldName(
        typeDef.Name,
      ).toUpperCase()}_VARIANTS`;
      const discKey = typeDef.Discriminator.TypePropertyName;
      const unionName = sanitizeClassName(typeDef.Name);

      // Build union members list (known variants + Unknown fallback)
      const unionMembers = typeDef.Discriminator.Mapping.map((mapping) =>
        sanitizePydanticType(
          typeDefToFieldDef(mapping.Type),
          { optional: false, nullable: false },
          usageLocation,
          owningModelInfo,
          true,
          false,
          suffixForTypes,
        ),
      );
      unionMembers.push(unknownClassName);

      const partialCall = `partial(parse_open_union, disc_key=${JSON.stringify(
        discKey,
      )}, variants=${variantsDictName}, unknown_cls=${unknownClassName}, union_name=${JSON.stringify(
        unionName,
      )}${responseSchemaValidationLenient() ? `, lenient=True` : ``})`;

      return `Annotated[Union[${unionMembers.join(
        ", ",
      )}], BeforeValidator(${partialCall})]`;
    }

    const unionMembers = [];
    let allMembersHavePreAppliedDiscriminator = true;
    let anyMemberHasOpenEnumDiscriminator = false;

    addImport("typing_extensions", "Annotated");
    addImport("typing", "Union");

    for (const mapping of typeDef.Discriminator.Mapping) {
      if (!mapping.Type.DiscriminatorPreApplied) {
        allMembersHavePreAppliedDiscriminator = false;
      }

      // Check if this member's discriminator field is an open enum
      const discriminatorField = mapping.Type.Fields?.find(
        (f) => f.Name === typeDef.Discriminator.TypePropertyName,
      );
      if (discriminatorField?.Type?.Enum?.Open === true) {
        anyMemberHasOpenEnumDiscriminator = true;
      }

      unionMembers.push(
        sanitizePydanticType(
          typeDefToFieldDef(mapping.Type),
          { optional: false, nullable: false },
          usageLocation,
          owningModelInfo,
          true,
          false,
          suffixForTypes,
        ),
      );
    }

    // For open enum discriminators, we can't use Field(discriminator='...')
    // because Pydantic requires Literal types. Use plain Union and rely on
    // smart union mode instead.
    if (
      allMembersHavePreAppliedDiscriminator &&
      anyMemberHasOpenEnumDiscriminator
    ) {
      return `Union[${unionMembers.join(", ")}]`;
    }

    if (allMembersHavePreAppliedDiscriminator) {
      // Use simpler Field(discriminator='...') syntax
      addImport("pydantic", "Field");

      // Get the Python field name for the discriminator
      const discriminatorFieldName = getDiscriminatorFieldName(
        typeDef.Discriminator.TypePropertyName,
        true, // discriminator is pre-applied
      );

      return `Annotated[Union[${unionMembers.join(
        ", ",
      )}], Field(discriminator="${discriminatorFieldName}")]`;
    }

    // Fall back to Discriminator + Tag syntax.
    // Check if "Tag" or "Discriminator" would shadow a class name in this
    // model file. When there is a conflict, use namespace import
    // (pydantic.Tag / pydantic.Discriminator) so the user's class is preserved.
    const modelClasses = owningModelInfo
      ? getModelClassNames(owningModelInfo.ModelName)
      : new Set<string>();
    const tagConflict = modelClasses.has("Tag");
    const discConflict = modelClasses.has("Discriminator");

    if (tagConflict || discConflict) {
      addImport("pydantic", "");
    }
    if (!discConflict) {
      addImport("pydantic", "Discriminator");
    }
    if (!tagConflict) {
      addImport("pydantic", "Tag");
    }
    addImport("utils", "get_discriminator", true);

    const tagRef = tagConflict ? "pydantic.Tag" : "Tag";
    const discRef = discConflict ? "pydantic.Discriminator" : "Discriminator";

    const taggedMembers = [];
    for (let i = 0; i < typeDef.Discriminator.Mapping.length; i++) {
      const mapping = typeDef.Discriminator.Mapping[i];
      taggedMembers.push(
        `Annotated[${unionMembers[i]}, ${tagRef}("${mapping.Name}")]`,
      );
    }

    const fieldName = sanitizeFieldName(typeDef.Discriminator.TypePropertyName);
    const lambda = `lambda m: get_discriminator(m, "${fieldName}", "${typeDef.Discriminator.TypePropertyName}")`;

    return `Annotated[Union[${taggedMembers.join(
      ", ",
    )}], ${discRef}(${lambda})]`;
  }

  const members = collectUnionAssociatedTypes(
    typeDef,
    usageLocation,
    owningModelInfo,
    suffixForTypes,
  );
  return renderTypeAliasType(unionName, members);
}
registerTemplateFunc("sanitizeUnionType", sanitizeUnionType);

function sanitizeTypedDictUnionType(
  typeDef: TypeDef,
  usageLocation: string,
  owningModelInfo: PythonV2Type = null,
): string {
  if (typeDef.AssociatedTypes.length == 1) {
    return sanitizeTypedDictType(
      typeDefToFieldDef(typeDef.AssociatedTypes[0]),
      { optional: false, nullable: false },
      usageLocation,
      owningModelInfo,
      true,
      false, // do not quote if there is only one associate type, lest it get treated as a string literal.
    );
  }

  const unionName =
    owningModelInfo?.TypedDictName ??
    resolveTypedDictName(typeDef, sanitizeClassName(typeDef.Name));
  const associatedTypes = collectUnionAssociatedTypes(
    typeDef,
    usageLocation,
    owningModelInfo,
    undefined,
    true,
  );
  return renderTypeAliasType(unionName, associatedTypes);
}
registerTemplateFunc("sanitizeTypedDictUnionType", sanitizeTypedDictUnionType);

// @ts-ignore
function sanitizeComments(comment: string): string {
  return comment
    .trim()
    .replaceAll(`"`, '\\"')
    .replaceAll(/(\r)([^\n])/g, "\n$2")
    .replaceAll("\r\n", "\n")
    .replaceAll(/\u200B/g, "\\u200B")
    .replaceAll(/"$/g, `" `)
    .replaceAll(/\\$/g, `\\ `)
    .replaceAll("{{", `{{"{{"}}`); // Need to sanitize these so they don't conflict with the templates
}

// @ts-ignore
function templateFieldDeclaration(
  fieldDef: FieldDef,
  parent?: TypeDef,
  additionalContext?: TemplateValueContext,
): string {
  const sanitize = (fieldDef: FieldDef): string => {
    const name = sanitizeField(fieldDef, parent);

    if (fieldDef.IsAdditionalProperties) {
      return name;
    }

    return fieldDef.Const &&
      additionalContext?.typedDictDisabled &&
      useUpperCaseConsts()
      ? name.toUpperCase()
      : name;
  };

  if (
    !parent ||
    hasAdditionalPropertiesFieldRecursive(parent) ||
    additionalContext?.typedDictDisabled
  ) {
    if (fieldDef.IsAdditionalProperties) {
      return "**";
    }

    return `${sanitize(fieldDef)}=`;
  } else {
    return `"${sanitize(fieldDef)}": `;
  }
}

// @ts-ignore
function sanitizeField(field: FieldDef, _parent?: TypeDef): string {
  if (field.IsAdditionalProperties) {
    return `__pydantic_extra__`;
  }

  return sanitizeFieldName(field.Name);
}

// @ts-ignore
function templateNullValue(fieldDef: FieldDef): string {
  return "None";
}

// @ts-ignore
function templateFieldDelimiter() {
  return ",";
}

// @ts-ignore
function templateIndent(indent) {
  return "    ".repeat(indent);
}

// @ts-ignore
function templateType(
  typeDef: TypeDef,
  additionalContext?: TemplateValueContext,
): string {
  if (typeDef.Type.toString() != "class") {
    return "";
  }

  if (
    hasAdditionalPropertiesFieldRecursive(typeDef) ||
    additionalContext?.security ||
    additionalContext?.typedDictDisabled
  ) {
    // For tests, use "tests" as the usage location to ensure proper namespace imports
    const usageLocation = additionalContext?.isTest
      ? "tests"
      : additionalContext?.outputLocation || "usage";
    return sanitizeType(
      typeDefToFieldDef(typeDef),
      { optional: false, nullable: false },
      usageLocation,
    );
  }

  return "";
}

// @ts-ignore
function templateBracket(
  typeDef: TypeDef,
  opening: boolean,
  additionalContext?: TemplateValueContext,
): string {
  switch (typeDef.Type.toString()) {
    case "class":
      if (
        hasAdditionalPropertiesFieldRecursive(typeDef) ||
        additionalContext?.security ||
        additionalContext?.typedDictDisabled
      ) {
        return opening ? "(" : ")";
      } else {
        return opening ? "{" : "}";
      }
    case "array":
    case "union":
      return opening ? "[" : "]";
    case "map":
      return opening ? "{" : "}";
    default:
      throw new Error(`Unknown type: ${typeDef.Type.toString()}`);
  }
}

// @ts-ignore
function templateStringValue(
  value: string,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  if (value.includes(".getenv(")) {
    return value;
  }

  return quote(value).replaceAll("{{", `{{"{{"}}`);
}

registerTemplateFunc("templateStringValue", templateStringValue);

// @ts-ignore
function templateEnumValue(
  fieldDef: FieldDef,
  idx: number,
  additionalContext?: TemplateValueContext,
) {
  if (getEnumFormat(fieldDef.Type) == "union") {
    return sanitizeEnumValue(
      fieldDef.Type.Enum.Values[idx],
      fieldDef.Type.Enum.Type.Type,
    );
  } else {
    let enumNames = getEnumNames(fieldDef.Type);

    // For tests, use "tests" as the usage location to ensure proper namespace imports
    const usageLocation = additionalContext?.isTest
      ? "tests"
      : additionalContext?.usageLocation ?? "usage";
    return `${sanitizeClass(
      fieldDef.Type,
      usageLocation,
      false,
      additionalContext?.owningModelInfo ?? null,
    )}.${enumNames[idx]}`;
  }
}

// @ts-ignore
function templateBoolValue(
  value: boolean,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  return value ? "True" : "False";
}

registerTemplateFunc("templateBoolValue", templateBoolValue);

// @ts-ignore
function templateByteValue(
  value,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
) {
  return `"${value.replace(/"/g, '\\"')}".encode()`;
}

// @ts-ignore
function templateDateValue(
  val: string,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  addImport("datetime", "date");
  return `date.fromisoformat("${val}")`;
}

// @ts-ignore
function templateDateTimeValue(
  val: string,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  addImport("utils", "parse_datetime", true);
  return `parse_datetime("${val}")`;
}

// @ts-ignore
function templateUUIDValue(
  val: string,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  addImport("uuid", "UUID");
  return `UUID("${val}")`;
}

// @ts-ignore
function templateDurationValue(
  val: string,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  addImport("utils", "parse_duration", true);
  return `parse_duration("${val}")`;
}

// @ts-ignore
function templateIntValue(
  val: number,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  return `${val}`;
}

// @ts-ignore
function templateFloatValue(
  val: number,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  if (fieldDef.Type.Type.toString() == "decimal") {
    addImport("decimal", "Decimal");
    return `Decimal("${val}")`;
  }

  const s = `${val}`;
  return /^-?\d+$/.test(s) ? `${s}.0` : s;
}

// @ts-ignore
function templateArrayValue(value) {
  return value;
}

// @ts-ignore
function templateMapValue(key, val) {
  return `"${key}": ${val}`;
}

// @ts-ignore
function templateOptionalSymbol() {
  return "";
}

// @ts-ignore
function getEnumNamesFromValues(values: string[]): string[] {
  return disambiguateEnumNames(values, getEnumName);
}

function disambiguateEnumNames(
  values: string[],
  deriveName: (value: string) => string,
): string[] {
  const baseNames = values.map(deriveName);
  const counts = new Map<string, number>();
  for (const name of baseNames) {
    counts.set(name, (counts.get(name) || 0) + 1);
  }

  const used = new Set(baseNames.filter((name) => counts.get(name) === 1));
  return values.map((value, i) => {
    let name = baseNames[i];
    if (counts.get(name) > 1) {
      const candidate = `${name}_${getCasing(value).toUpperCase()}`;
      name = candidate;
      for (let suffix = 1; used.has(name); suffix++) {
        name = `${candidate}_${suffix}`;
      }
      used.add(name);
    }
    return name;
  });
}

// @ts-ignore
function getEnumName(value: string): string {
  let name = value.trim();

  if (name === "") {
    name = "unknown";
  }

  name = sanitizeName(name);

  name = caser().ToSNAKE(name);
  if (/^[0-9]/.test(name)) {
    name = caser().ToSNAKE(sanitizeName(name));
  }
  return name;
}

registerTemplateFunc("getEnumName", getEnumName);

// @ts-ignore
function sanitizeJsonPath(jsonpath: string): string {
  const path = jsonpath
    // Note (alexa): couldn't find a library that correctly parses this expression
    // This is not a long-term solution
    .replaceAll(/\(\@\.length ?- ?1\)/gi, "-1:");
  return path;
}

registerTemplateFunc("sanitizeJsonPath", sanitizeJsonPath);

// @ts-ignore
function sanitizeSecurityFieldName(fieldName: string): string {
  return sanitizeFieldName(fieldName);
}

// @ts-ignore
function sanitizeRetryConnectionErrors(retryConnectionErrors: object): string {
  return retryConnectionErrors.toString() == "true" ? "True" : "False";
}
registerTemplateFunc(
  "sanitizeRetryConnectionErrors",
  sanitizeRetryConnectionErrors,
);

let _inMethodBody = false;

function enterMethodBody(): string {
  _inMethodBody = true;
  return "";
}
registerTemplateFunc("enterMethodBody", enterMethodBody);

function exitMethodBody(): string {
  _inMethodBody = false;
  return "";
}
registerTemplateFunc("exitMethodBody", exitMethodBody);

function isInMethodBody(): boolean {
  return _inMethodBody;
}

const FUNC_LOCALS = new Set([
  "request",
  "req",
  "base_url",
  "url_variables",
  "retry_config",
  "http_res",
  "response_data",
  "http_res_text",
]);

function rewriteMethodLocals(op: Operation, methodSource: string): string {
  // 1. Extract docstrings: replace r"""...""" blocks with placeholders
  const docstrings: string[] = [];
  let result = methodSource.replace(/r"""[\s\S]*?"""/g, (match) => {
    const idx = docstrings.length;
    docstrings.push(match);
    return `__DOCSTRING_${idx}__`;
  });

  // 2. Build externalNames set from op.Arguments.Sorted
  const externalNames = new Set(
    op.Arguments.Sorted.map((a) => sanitizeParameterName(a.Name)),
  );

  // In non-flattened operations, "request" the user param IS the internal
  // request object — they're the same variable, so no rename is needed.
  // Only treat "request" as a collision in flattened operations where
  // request_ is a NEW wrapper object constructed from individual params.
  if (op.Arguments.Flattening === "none") {
    externalNames.delete("request");
  }

  // 3. FUNC_LOCALS pass: strip _ suffix when no collision
  const rewrites: string[] = [];
  for (const local of FUNC_LOCALS) {
    if (!externalNames.has(local)) {
      rewrites.push(local);
    }
  }

  for (const local of rewrites) {
    result = result.replace(
      new RegExp(`(__ASYNC_ONLY__|__SYNC_ONLY__)${local}_\\b`, "g"),
      `$1${local}`,
    );
    result = result.replace(new RegExp(`\\b${local}_\\b`, "g"), local);
  }

  // 4. Self-assignment cleanup for request
  if (rewrites.includes("request")) {
    result = result.replace(/^[ \t]*request = request[ \t]*\n/gm, "");
  }

  // 5. Module refs pass (FLIPPED direction): templates emit suffixed names
  //    (e.g. utils_.foo()), rewriter strips _ when safe.
  const moduleRefNames = getModuleRefNames();
  const allParamNames: string[] =
    context.RecursiveComputed?.AllParamNames ?? [];
  // Method names (operation IDs) can shadow module-level imports at the class
  // level — e.g. `def utils(self, ...)` makes `utils` resolve to the method
  // in type annotations of subsequent methods in the same class body.
  const allMethodNames: string[] =
    context.RecursiveComputed?.AllMethodNames ?? [];
  const allParamNamesSet = new Set([...allParamNames, ...allMethodNames]);

  for (const name of moduleRefNames) {
    const suffixedPattern = new RegExp(`\\b${name}_\\.`, "g");
    if (!allParamNamesSet.has(name)) {
      // No collision — strip the suffix: name_. → name.
      result = result.replace(suffixedPattern, `${name}.`);
    } else {
      // Collision exists — keep the suffixed refs and ensure the import alias exists.
      // We only ever go suffixed→clean, never clean→suffixed, because clean→suffixed
      // could convert things by mistake. If clean refs appear here, fix the source
      // template to emit suffixed refs instead.
      if (suffixedPattern.test(result)) {
        addModuleAlias(name);
      }
    }
  }

  // 6. Restore docstrings from placeholders
  for (let i = 0; i < docstrings.length; i++) {
    result = result.replace(`__DOCSTRING_${i}__`, docstrings[i]);
  }

  return result;
}
registerTemplateFunc("rewriteMethodLocals", rewriteMethodLocals);

function sanitizeSynchronousMethod(method: string): string {
  return method
    .replaceAll(/__ASYNC_ONLY__(.*?)__ASYNC_ONLY__/g, "")
    .replaceAll(/__SYNC_ONLY__(.*?)__SYNC_ONLY__/g, "$1")
    .replaceAll(
      pythonEventStreamMethodTypeRef(),
      pythonEventStreamSyncTypeRef(),
    )
    .replaceAll("jsonl.JsonLStreamAsync", "jsonl.JsonLStream");
}
registerTemplateFunc("sanitizeSynchronousMethod", sanitizeSynchronousMethod);

function sanitizeAsynchronousMethod(method: string): string {
  return method
    .replaceAll(/__ASYNC_ONLY__(.*?)__ASYNC_ONLY__/g, "$1")
    .replaceAll(/__SYNC_ONLY__(.*?)__SYNC_ONLY__/g, "")
    .replaceAll(
      pythonEventStreamMethodTypeRef(),
      pythonEventStreamAsyncTypeRef(),
    );
}
registerTemplateFunc("sanitizeAsynchronousMethod", sanitizeAsynchronousMethod);

function sanitizeFlattenedRequestParamAccessor(field: FieldDef): string {
  let paramName = sanitizeParameterName(field.Name);

  if (field.Type && typeIncludesPydanticModel(field.Type)) {
    addUtilsImport();

    paramName = `utils_.get_pydantic_model(${paramName}, ${sanitizePydanticType(
      field,
    )})`;
  } else if (
    field.Type &&
    (typeIncludesArray(field.Type) || typeIncludesMap(field.Type))
  ) {
    addUtilsImport();

    paramName = `utils_.unmarshal(${paramName}, ${sanitizePydanticType(
      field,
    )})`;
  }

  return paramName;
}
registerTemplateFunc(
  "sanitizeFlattenedRequestParamAccessor",
  sanitizeFlattenedRequestParamAccessor,
);

function sanitizeFlattenedAdditionalPropertiesRequestParam(
  field: FieldDef,
): string {
  const flattened = sanitizeFlattenedRequestParamAccessor(field);

  if (field.Optional) {
    return `(${flattened} or {})`;
  }

  return flattened;
}
registerTemplateFunc(
  "sanitizeFlattenedAdditionalPropertiesRequestParam",
  sanitizeFlattenedAdditionalPropertiesRequestParam,
);

function paginationInputNeedsNarrowing(
  op: Operation,
  input: PaginationInputs,
): boolean {
  if (input.Optional) return true;
  // a nullable request or body arrives as None, so the read always needs narrowing
  if (op.Request?.Field?.Nullable || op.Request?.RequestBody?.Nullable)
    return true;
  if (!op.Request?.Field?.Type) return false;
  const field = findPaginationFieldDeep(op.Request.Field.Type, input.Name);
  return field ? field.Nullable : false;
}
registerTemplateFunc(
  "paginationInputNeedsNarrowing",
  paginationInputNeedsNarrowing,
);

// a nullable non-flattened body arrives as None with a union type; the
// pagination closure narrows it once into a `request_body_` local
// @ts-ignore
function paginationRequestIsNullableBody(op: Operation): boolean {
  return Boolean(op.Request?.IsRequestBody && op.Request?.Field?.Nullable);
}
registerTemplateFunc(
  "paginationRequestIsNullableBody",
  paginationRequestIsNullableBody,
);

function sanitizePaginationAccess(
  op: Operation,
  input: PaginationInputs,
): string {
  if (!op.Request) {
    throw new Error(
      `${op.ID}: ${input.Name}: attempted to access pagination field but operation has no request`,
    );
  }

  const field = sanitizeFieldName(input.Name);
  switch (input.In.toString()) {
    case "parameters":
      return `request_.${field}`;
    case "requestBody":
      if (!op.Request.RequestBody) {
        throw new Error(
          `${op.ID}: ${input.Name}: attempted to access pagination field in request body which is not defined`,
        );
      }
      if (op.Request.IsRequestBody) {
        if (paginationRequestIsNullableBody(op)) {
          return `request_body_.${field}`;
        }
        return `request_.${field}`;
      }

      return `${paginationRequestVar(op)}.${sanitizeFieldName(
        op.Request.RequestBody.Name,
      )}.${field}`;
    default:
      throw new Error(
        `${op.ID}: ${
          input.Name
        }: unknown pagination input location "${input.In.toString()}"]`,
      );
  }
}
registerTemplateFunc("sanitizePaginationAccess", sanitizePaginationAccess);

// a non-flattened wrapper request is read through an in-closure cast alias:
// mypy does not narrow the method's Model/TypedDict union parameter inside
// the nested pagination closure, so the closure re-narrows it once
// @ts-ignore
function paginationRequestNeedsWrapperAlias(op: Operation): boolean {
  if (!op.Request?.RequestBody || op.Request.IsRequestBody) return false;
  const flattening = op.Arguments?.Flattening?.toString() ?? "";
  return !["all", "body", "params"].includes(flattening);
}
registerTemplateFunc(
  "paginationRequestNeedsWrapperAlias",
  paginationRequestNeedsWrapperAlias,
);

function paginationRequestVar(op: Operation): string {
  return paginationRequestNeedsWrapperAlias(op)
    ? "request_wrapper_"
    : "request_";
}

// copy a non-pagination field into the next-page body; None when the request was null
// @ts-ignore
function paginationRequestFieldCopy(op: Operation, name: string): string {
  const field = sanitizeFieldName(name);
  if (paginationRequestIsNullableBody(op)) {
    return `request_body_.${field} if request_body_ is not None else None`;
  }
  return `${paginationRequestVar(op)}.${field}`;
}
registerTemplateFunc("paginationRequestFieldCopy", paginationRequestFieldCopy);

// copy a non-pagination field from a wrapped body into the next-page body; None when the body was null
// @ts-ignore
function paginationWrappedBodyFieldCopy(
  op: Operation,
  bodyName: string,
  name: string,
): string {
  const body = `${paginationRequestVar(op)}.${sanitizeFieldName(bodyName)}`;
  const access = `${body}.${sanitizeFieldName(name)}`;
  if (op.Request?.RequestBody?.Nullable || op.Request?.RequestBody?.Optional) {
    return `${access} if ${body} is not None else None`;
  }
  return access;
}
registerTemplateFunc(
  "paginationWrappedBodyFieldCopy",
  paginationWrappedBodyFieldCopy,
);

// @ts-ignore
function paginationRequestNoneGuard(
  op: Operation,
  input: PaginationInputs,
): string {
  // guard every holder that can be None, outermost first so it short-circuits
  const parts = sanitizePaginationAccess(op, input).split(".");
  const guards: string[] = [];
  if (op.Request?.Field?.Nullable) {
    guards.push(`${parts[0]} is not None`);
  }
  if (
    (op.Request?.RequestBody?.Nullable || op.Request?.RequestBody?.Optional) &&
    parts.length > 2
  ) {
    guards.push(`${parts.slice(0, -1).join(".")} is not None`);
  }
  if (guards.length === 0) return "";
  return `${guards.join(" and ")} and `;
}
registerTemplateFunc("paginationRequestNoneGuard", paginationRequestNoneGuard);

// @ts-ignore
function languageSpecificEnvVarWrapping(
  envVar: EnvVar,
  additionalContext?: TemplateValueContext,
): string {
  addImport("os", "");
  return `os.getenv("${envVar.name}", "${envVar.defaultValue}")`;
}
