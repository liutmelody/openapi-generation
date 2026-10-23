/** Returns the structured dependency list for this template.
 * This is the single source of truth for all dependencies.
 * Used by both template rendering and security scanning. */
// @ts-ignore
function getTemplateDependencies() {
  return {
    // Runtime dependencies
    pydantic: {
      name: "pydantic",
      version: ">=2.11.2",
      cpe: "cpe:2.3:a:pydantic:pydantic:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "runtime",
    },
    httpx: {
      name: "httpx",
      version: ">=0.28.1",
      cpe: "cpe:2.3:a:encode:httpx:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "runtime",
      condition: "httpClientLibrary=httpx",
    },
    httpx2: {
      name: "httpx2",
      version: ">=2.12.0",
      cpe: "cpe:2.3:a:pydantic:httpx2:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "runtime",
      condition: "httpClientLibrary=httpx2",
    },
    httpcore: {
      name: "httpcore",
      version: ">=1.0.9",
      cpe: "cpe:2.3:a:encode:httpcore:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "runtime",
      condition: "httpClientLibrary=httpx",
    },
    "jsonpath-python": {
      name: "jsonpath-python",
      version: ">=1.0.6",
      cpe: "cpe:2.3:a:jsonpath-python:jsonpath-python:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "runtime",
      condition: "pagination",
    },
    // Dev dependencies
    pylint: {
      name: "pylint",
      version: "==3.2.3",
      cpe: "cpe:2.3:a:pylint:pylint:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "dev",
    },
    mypy: {
      name: "mypy",
      version: "==1.15.0",
      cpe: "cpe:2.3:a:mypy:mypy:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "dev",
    },
    pyright: {
      name: "pyright",
      version: "==1.1.398",
      cpe: "cpe:2.3:a:microsoft:pyright:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "dev",
    },
    pytest: {
      name: "pytest",
      version: "^9.0.3",
      cpe: "cpe:2.3:a:pytest:pytest:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "dev",
      condition: "tests",
    },
    "pytest-xdist": {
      name: "pytest-xdist",
      version: "^3.8.0",
      cpe: "cpe:2.3:a:pytest:pytest-xdist:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "dev",
      condition: "tests",
    },
    "pytest-asyncio": {
      name: "pytest-asyncio",
      version: ">=1.3.0",
      cpe: "cpe:2.3:a:pytest:pytest-asyncio:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "dev",
      condition: "tests",
    },
    "pytest-timeout": {
      name: "pytest-timeout",
      version: "^2.4.0",
      cpe: "cpe:2.3:a:pytest:pytest-timeout:*:*:*:*:*:python:*:*",
      ecosystem: "PyPI",
      category: "dev",
      condition: "pytestTimeout",
    },
  } as const satisfies Record<string, TemplateDependency>;
}

// @ts-ignore
function upgradeConfig(oldVersion, newVersion, cfg, defaults) {
  if (oldVersion == "") {
    let upgraded = defaults;

    if (cfg.version) {
      upgraded.version = cfg.version;
    }

    if (cfg.packagename) {
      upgraded.packageName = cfg.packagename;
    }

    if (cfg.moduleName) {
      upgraded.moduleName = cfg.moduleName;
    }

    if (cfg.author) {
      upgraded.author = cfg.author;
    }

    if (cfg.description) {
      upgraded.description = cfg.description;
    }

    return upgraded;
  }

  return cfg;
}

/** Returns the configuration fields available for customers. This data is
 * fetched early in the generation process so values can be passed back to
 * getGeneratorConfig. */
// @ts-ignore
function getConfigFields(
  commonFields: SDKGenConfigFields,
  newSDK: boolean,
): SDKGenConfigFields {
  return {
    ...commonFields,
    version: {
      Name: "version",
      Required: true,
      DefaultValue: "0.0.1",
      Description: "The current version of the SDK",
      ValidationRegex: /^[\w\d.\-_]+$/.source,
      ValidationMessage: "Letters, numbers, or .-_ only",
    },
    packageName: {
      Name: "packageName",
      Required: true,
      DefaultValue: "openapi",
      Description:
        "The distribution name of the PyPI Package. https://packaging.python.org/en/latest/specifications/name-normalization/",
      ValidationRegex: /^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9._-]*[a-zA-Z0-9])$/
        .source,
      ValidationMessage:
        "Only ASCII letters and numbers, period, underscore, and hyphen characters allowed. It must start and end with a letter or number.",
    },
    moduleName: {
      Name: "moduleName",
      Required: false,
      DefaultValue: "",
      Description:
        "By default, the module name is the same as the package name used for installation ('packageName' configuration)." +
        "This configuration enables adjusting where the SDK is generated and how it is imported in consuming code. " +
        "For example, 'pip install {packageName}' and in consuming code 'from {moduleName} import SDK'. " +
        "PEP 420 implicit namespace packages are supported with period (.) characters, " +
        "where parent directories are created as implicit namespace packages and the SDK code is generated " +
        "in the final child directory. For example, setting this configuration to 'example_cloud.api_client' " +
        "will generate the SDK in the example_cloud/api_client directory and ensure the example_cloud directory " +
        "does not contain an __init.py__ file.",
      ValidationRegex: /^[a-zA-Z0-9_.]*$/.source,
      ValidationMessage:
        "Only letters, numbers, underscore (_), or period (.) are allowed",
    },
    authors: {
      Name: "authors",
      Required: true,
      DefaultValue: ["Speakeasy"],
      Description:
        "Authors of the published package. https://python-poetry.org/docs/pyproject/#authors",
    },
    description: {
      Name: "description",
      Required: false,
      DefaultValue: "Python Client SDK Generated by Speakeasy.",
      Description:
        "A short description of the project. https://python-poetry.org/docs/pyproject/#description",
    },
    homepage: {
      Name: "homepage",
      Required: false,
      Description:
        "The URL for the homepage of the project. https://python-poetry.org/docs/pyproject/#homepage",
    },
    documentationUrl: {
      Name: "documentationUrl",
      Required: false,
      Description:
        "The URL for the documentation of the project. https://python-poetry.org/docs/pyproject/#documentation",
    },
    license: {
      Name: "license",
      Required: false,
      DefaultValue: newSDK ? "Apache-2.0" : "",
      Description:
        "The SPDX license identifier or license text for the project. https://python-poetry.org/docs/pyproject/#license",
    },
    maxMethodParams: {
      Name: "maxMethodParams",
      Required: false,
      DefaultValue: newSDK ? 999 : 4,
      Description:
        "The maximum number of parameters a method can have before the resulting SDK endpoint is no longer 'flattened' and an input object is created instead. 0 will use input objects always. https://www.speakeasy.com/docs/customize-sdks/methods",
      ValidationRegex: /^\d+$/.source,
      ValidationMessage: "Numbers only",
    },
    flatteningOrder: {
      Name: "flatteningOrder",
      Required: false,
      DefaultValue: "parameters-first",
      Description:
        "When flattening parameters and body fields, determines the ordering of generated method arguments.",
      ValidationRegex: /^(parameters-first|body-first)$/.source,
      ValidationMessage: "parameters-first or body-first only",
    },
    clientServerStatusCodesAsErrors: {
      Name: "clientServerStatusCodesAsErrors",
      Required: false,
      DefaultValue: true,
      Description: "Whether to treat 4xx and 5xx status codes as errors.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    flattenGlobalSecurity: {
      Name: "flattenGlobalSecurity",
      Required: false,
      DefaultValue: true,
      Description:
        "Flatten the global security configuration if there is only a single option in the spec",
    },
    imports: {
      Name: "imports",
      Required: false,
      DefaultValue: {
        option: "openapi",
        paths: {
          shared: "",
          operations: "",
          errors: "errors",
          callbacks: "",
          webhooks: "",
        },
      },
      Description: "Configuration for model import structure",
    },
    additionalDependencies: {
      Name: "additionalDependencies",
      Required: false,
      DefaultValue: {
        main: {},
        dev: {},
      },
      Description:
        "Specify additional dependencies to include in the generated pyproject.toml file.",
    },
    optionalDependencies: {
      Name: "optionalDependencies",
      Required: false,
      DefaultValue: {},
      Description:
        "Specify optional dependency groups (extras) for the generated pyproject.toml file. " +
        "Each key is an extras name and the value is a map of package names to version specifiers. " +
        "Users install extras with: pip install package[extras_name]",
    },
    responseFormat: {
      Name: "responseFormat",
      Required: false,
      DefaultValue: "flat",
      Description:
        "Determines the shape of the response envelope that is returned from SDK methods",
      ValidationRegex: /^(envelope|envelope-http|flat)$/.source,
      ValidationMessage: '"envelope-http", "envelope" or "flat" only',
    },
    eventStreamClassNames: {
      Name: "eventStreamClassNames",
      Required: false,
      DefaultValue: {
        sync: "EventStream",
        async: "EventStreamAsync",
      },
      Description:
        "Controls the generated Python server-sent event stream wrapper class names.",
    },
    inputTypedDictSuffix: {
      Name: "inputTypedDictSuffix",
      Required: false,
      DefaultValue: "TypedDict",
      Description:
        "Suffix applied to TypedDict companion classes for schemas reachable from any operation input (request body, path/query/header parameters, and their transitive references). Defaults to 'TypedDict'.",
      ValidationRegex: /^[A-Za-z][A-Za-z0-9_]*$/.source,
      ValidationMessage:
        "Must start with a letter; remaining characters may be letters, digits, or underscores. PascalCased automatically (e.g. 'param' -> 'Param').",
    },
    rawResponseHelpers: {
      Name: "rawResponseHelpers",
      Required: false,
      DefaultValue: false,
      Description:
        "When true, generated resources expose `with_raw_response` and `with_streaming_response` helper views.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    errorSchemaValidation: {
      Name: "errorSchemaValidation",
      Required: false,
      DefaultValue: true,
      Description:
        "Whether to validate error response bodies against typed error models. When disabled, schema mismatches fall back to the SDK default error while preserving the raw body.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    preserveUnknownFields: {
      Name: "preserveUnknownFields",
      Required: false,
      DefaultValue: false,
      Description:
        "Preserve undeclared fields in JSON request and response models, including nested models and schemas with additionalProperties: false. Explicitly typed additional properties retain their validation. Flattened operation arguments accept only declared fields, so undeclared top-level request fields require flattenRequests: false; nested models passed as arguments keep theirs. Disabled by default.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    responseSchemaValidation: {
      Name: "responseSchemaValidation",
      Required: false,
      DefaultValue: true,
      Description:
        "Controls validation of success response bodies against typed response models. 'true' (default) enforces strict validation, raising ResponseValidationError on a schema mismatch. 'lenient' deserializes without raising: missing required fields are defaulted (None, or an empty typed instance for required models) and discriminated-union variants whose payload fails strict validation are still constructed as their typed variant instead of degrading to the Unknown fallback. 'false' disables validation and returns the raw decoded JSON.",
      ValidationRegex: /^(true|false|lenient)$/.source,
      ValidationMessage: "true, false or lenient only",
    },
    methodArguments: {
      Name: "methodArguments",
      Required: false,
      DefaultValue: "infer-optional-args",
      Description:
        "If infer-optional-args, if all the parameters and the request body is optional, the argument to the function will be optional",
      ValidationRegex:
        /^(infer-optional-args|require-security-and-request|positional-path-params|positional-path-with-extras)$/
          .source,
      ValidationMessage:
        '"infer-optional-args", "require-security-and-request", "positional-path-params", or "positional-path-with-extras" only',
    },
    methodTimeoutArgument: {
      Name: "methodTimeoutArgument",
      Required: false,
      DefaultValue: "timeout-ms",
      Description: "Controls the spelling of per-method timeout overrides.",
      ValidationRegex: /^(timeout-ms|timeout)$/.source,
      ValidationMessage: '"timeout-ms" or "timeout" only',
    },
    methodTimeoutUnits: {
      Name: "methodTimeoutUnits",
      Required: false,
      DefaultValue: "milliseconds",
      Description:
        "Controls the public units of per-method timeout overrides. The default milliseconds preserves the existing timeout_ms behavior. Seconds accepts float or httpx.Timeout and converts to the internal millisecond transport value.",
      ValidationRegex: /^(milliseconds|seconds)$/.source,
      ValidationMessage: '"milliseconds" or "seconds" only',
    },
    httpClientLibrary: {
      Name: "httpClientLibrary",
      Required: false,
      DefaultValue: "httpx",
      Description:
        "The HTTP client library the generated SDK depends on. httpx2 (Pydantic's API-compatible httpx fork) replaces the httpx dependency; the SDK's public API is unchanged and clients of either library can still be injected.",
      ValidationRegex: /^(httpx|httpx2)$/.source,
      ValidationMessage: '"httpx" or "httpx2" only',
    },
    templateVersion: {
      Name: "templateVersion",
      Required: false,
      DefaultValue: "v2",
      Description: "The template version to use",
      ValidationRegex: /^v\d+$/.source,
      ValidationMessage:
        "Template version must be in the format v1, v2, and so on",
    },
    flattenRequests: {
      Name: "flattenRequests",
      Required: false,
      DefaultValue: newSDK,
      Description:
        "Turn request parameters and body fields into a flat list of method arguments. This takes precedence over maxMethodParams. If there is no request body then maxMethodParams will be respected.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    bodyVariantOverloads: {
      Name: "bodyVariantOverloads",
      Required: false,
      DefaultValue: false,
      Description:
        "When true and `flattenRequests` is also true, top-level oneOf-of-classes request bodies are flattened into per-variant `@overload` method signatures.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    envVarPrefix: {
      Name: "envVarPrefix",
      Required: false,
      Description:
        "The environment variable prefix for security and global env variable overrides. If empty these overrides will not be possible",
    },
    uuidFormat: {
      Name: "uuidFormat",
      Required: false,
      DefaultValue: false,
      Description:
        "When true, string schemas with format: uuid are generated as uuid.UUID instead of str. This is opt-in because it changes the generated field types and is a breaking change for existing SDKs.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    durationFormat: {
      Name: "durationFormat",
      Required: false,
      DefaultValue: newSDK,
      Description:
        "When true, string schemas with format: duration are generated as datetime.timedelta instead of str.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    enumFormat: {
      Name: "enumFormat",
      Required: false,
      DefaultValue: newSDK ? "union" : "enum",
      Description: "Determines the format to express enums in Python",
      ValidationRegex: /^(union|enum)$/.source,
      ValidationMessage: '"union" or "enum" only',
    },
    forwardCompatibleEnumsByDefault: {
      Name: "forwardCompatibleEnumsByDefault",
      Required: false,
      DefaultValue: newSDK,
      Description:
        "When true, any enum which is used on a response will be automatically open/forward compatible - i.e. unknown values will be tolerated. Single value enums won't be automatically opened. Individual enums can be controlled with x-speakeasy-unknown-values: allow/disallow.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    forwardCompatibleUnionsByDefault: {
      Name: "forwardCompatibleUnionsByDefault",
      Required: false,
      DefaultValue: newSDK ? "tagged-only" : "false",
      Description:
        "Controls forward compatibility for discriminated unions in responses. 'tagged-only' makes discriminated unions open to unknown discriminator values. 'false' disables forward compatibility. Individual unions can be controlled with x-speakeasy-unknown-values: allow/disallow.",
      ValidationRegex: /^(tagged-only|false)$/.source,
      ValidationMessage: "tagged-only or false only",
    },
    fixes: {
      Name: "fixFlags",
      Required: false,
      DefaultValue: {
        responseRequiredSep2024: newSDK,
        asyncPaginationSep2025: newSDK,
        conflictResistantModelImportsFeb2026: newSDK,
      },
      Description:
        "Fixes to apply to the generated SDK, generally should be set to true but may be false for backwards compatibility",
    },
    defaultErrorName: {
      Name: "defaultErrorName",
      Required: false,
      // The default is handled in `fixBuiltInErrorNameConflicts()`
      DefaultValue: newSDK ? "" : "SDKError",
      Description:
        "The name of the default exception that is raised when an API error occurs.",
      ValidationRegex: /^[a-zA-Z0-9]*$/.source,
      ValidationMessage: "Must only contain letters and numbers",
    },
    baseErrorName: {
      Name: "baseErrorName",
      Required: false,
      // The default is handled in `fixBuiltInErrorNameConflicts()`
      DefaultValue: "",
      Description:
        "The name of the base error class used for HTTP error responses",
      ValidationRegex: /^[a-zA-Z0-9]*$/.source,
      ValidationMessage:
        "Must start with a capital letter and contain only letters and numbers",
    },
    enableCustomCodeRegions: {
      Name: "enableCustomCodeRegions",
      Required: false,
      DefaultValue: false,
      Description: "Allow custom code to be inserted into the generated SDK.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    pytestFilterWarnings: {
      Name: "pytestFilterWarnings",
      Required: false,
      DefaultValue: [],
      Description:
        "When array contains any strings, sets the global pytest filterwarnings configuration value, which are filters to control Python warnings, such as ignoring warnings or raising warnings as errors. Reference: https://docs.python.org/3/library/warnings.html#warning-filter",
    },
    pytestTimeout: {
      Name: "pytestTimeout",
      Required: false,
      DefaultValue: 0,
      Description:
        "When value is greater than 0, installs pytest-timeout and sets the global pytest-timeout configuration value, which is the number of seconds before individual tests are timed out.",
      ValidationRegex: /^\d+$/.source,
      ValidationMessage: "Integer number of seconds",
    },
    packageManager: {
      Name: "packageManager",
      Required: false,
      DefaultValue: newSDK ? "uv" : "poetry",
      Description:
        "The package manager to use for Python SDK generation. Choose between 'uv' (recommended) or 'poetry'.",
      ValidationRegex: /^(uv|poetry)$/.source,
      ValidationMessage: '"uv" or "poetry" only',
    },
    allowedRedefinedBuiltins: {
      Name: "allowedRedefinedBuiltins",
      Required: false,
      DefaultValue: newSDK
        ? ["id", "object", "input", "dir"]
        : ["id", "object", "dir"],
      Description:
        "List of names allowed to shadow builtins via SDK method parameters",
    },
    sseFlatResponse: {
      Name: "sseFlatResponse",
      Required: true,
      DefaultValue: false,
      Description:
        "Whether to flatten SSE (Server-Sent Events) responses by extracting the 'data' field from wrapper models, providing direct access to the event data instead of the wrapper object",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    asyncMode: {
      Name: "asyncMode",
      Required: false,
      DefaultValue: "both",
      Description:
        "Whether to generate sync and/or async methods. Valid values are 'both', and 'split'. 'both' will generate 1 constructor with _async suffix methods. 'split' will generate FooSDK and AsyncFooSDK but no suffixes on the methods.",
      ValidationRegex: /^(both|split)$/.source,
      ValidationMessage: "'both', or 'split' only",
    },
    multipartArrayFormat: {
      Name: "multipartArrayFormat",
      Required: false,
      DefaultValue: newSDK ? "standard" : "legacy",
      Description:
        "Format for array field names in multipart form data. 'legacy' appends '[]' to array field names (e.g., 'files[]'), which was the previous behavior. 'standard' uses the field name as-is without any suffix. Set to 'standard' for correct multipart/form-data handling.",
      ValidationRegex: /^(legacy|standard)$/.source,
      ValidationMessage: "legacy or standard only",
    },
    compileCommands: {
      Name: "compileCommands",
      Required: false,
      Description:
        "A list of commands to use for compiling the SDK, replacing the default compile pipeline. " +
        "This must be an array of arrays, where each inner array has the command as the first element and the rest are arguments.",
    },
    inferUnionDiscriminators: {
      Name: "inferUnionDiscriminators",
      Required: false,
      DefaultValue: true,
      Description:
        "Infer union discriminators for oneOfs missing explicit OpenAPI discriminator mapping",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    preApplyUnionDiscriminators: {
      Name: "preApplyUnionDiscriminators",
      Required: false,
      DefaultValue: newSDK,
      Description:
        "When true, discriminator values will be applied as const fields onto union member types when those types are consistently used with the same discriminator value. This simplifies the type structure by pre-applying discriminators where possible and enables the use of simpler `Annotated[Union[...], Field(discriminator='...')]` syntax.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
    constFieldCasing: {
      Name: "constFieldCasing",
      Required: false,
      DefaultValue: newSDK ? "normal" : "upper",
      Description:
        "Determines the casing convention for const fields in generated models. 'upper' uses SCREAMING_SNAKE_CASE (e.g., MY_CONST), 'normal' uses regular snake_case (e.g., my_const).",
      ValidationRegex: /^(upper|normal)$/.source,
      ValidationMessage: '"upper" or "normal" only',
    },
    useAsyncHooks: {
      Name: "useAsyncHooks",
      Required: false,
      DefaultValue: false,
      Description:
        "Enable async hooks infrastructure for async SDK methods. When enabled, SDK maintainers must explicitly register async hooks in the asyncregistration.py file. Adapters are provided to wrap sync hooks, but native async implementations are recommended for optimal performance.",
      ValidationRegex: /^(true|false)$/.source,
      ValidationMessage: "true or false only",
    },
  };
}

/** Represents the initial target implementation configuration presented to
 * the generator from the target. Customer configuration keys and values from
 * getConfigFields are fetched prior so those values are available. */
// @ts-ignore
function getGeneratorConfig(
  genConfig: GeneratorInitialConfiguration,
): TargetInitialConfiguration {
  const result: TargetInitialConfiguration = {
    compile: getCompileConfiguration(genConfig),
    testing: getTestingConfiguration(genConfig),
  };

  return result;
}

// @ts-ignore
function getPythonCommandDependencies(
  packageManager: string = "uv",
): RunnerCommandDependencies {
  const baseDependencies = [
    {
      command: "python3",
      version: {
        args: ["--version"],
        regex: `(?m)Python.*?(\\d+\\.\\d+\\.\\d+).*?`,
        minVersion: "3.10.0",
      },
      installDocumentation: `Install Python by following the instructions at https://www.python.org/downloads/.`,
    },
  ];

  if (packageManager === "poetry") {
    baseDependencies.push({
      command: "poetry",
      version: {
        args: ["--version"],
        regex: `(?m)Poetry.*?(\\d+\\.\\d+\\.\\d+).*?`,
        minVersion: "2.0.0",
      },
      installDocumentation: `Install Poetry by following the instructions at https://python-poetry.org/docs/#installing-with-pipx.`,
    });
  } else {
    baseDependencies.push({
      command: "uv",
      version: {
        args: ["--version"],
        regex: `(?m)uv.*?(\\d+\\.\\d+\\.\\d+).*?`,
        minVersion: "0.1.0",
      },
      installDocumentation: `Install uv by following the instructions at https://docs.astral.sh/uv/getting-started/installation/.`,
    });
  }

  return baseDependencies;
}

/** Parses the compileCommands config into RunnerCommands.
 *  Returns null when the option is not set. */
function parseCompileCommands(langConfig: any): RunnerCommands | null {
  if (langConfig.compileCommands == null) {
    return null;
  }

  if (!Array.isArray(langConfig.compileCommands)) {
    throw new Error(
      "compileCommands must be an array of arrays of the form [[COMMAND, ...ARGS], ...]",
    );
  }

  return langConfig.compileCommands.map(
    (entry: unknown, i: number): RunnerCommand => {
      if (!Array.isArray(entry)) {
        throw new Error(
          `compileCommands[${i}] must be an array of the form [COMMAND, ...ARGS]`,
        );
      }

      const [command, ...args] = entry;
      if (typeof command !== "string") {
        throw new Error(`compileCommands[${i}][0] must be a string`);
      }

      return { command, args };
    },
  );
}

// @ts-ignore
function getCompileConfiguration(
  genConfig: GeneratorInitialConfiguration,
): CompileConfiguration {
  const langConfig = genConfig.LangCfg;
  const packageManager = langConfig.packageManager || "uv";

  // Custom compile commands handle everything — no additional steps needed.
  // compileCommands (plural) takes precedence over compileCommand (singular).
  const customCommands = parseCompileCommands(langConfig);
  if (customCommands != null) {
    return {
      runner: {
        commands: customCommands,
        dependencies: {
          commands: getPythonCommandDependencies(packageManager),
        },
      },
    };
  }

  // Use full source directory path to prevent tooling from running on
  // potentially orphaned files after a generation configuration update, such as
  // packageName changes or v1 to v2 migration.
  // @ts-ignore sanitizeFile is externally provided in runtime.
  const moduleDirectory: string = langConfig.moduleName
    ? langConfig.moduleName
        .split(".")
        .map((part) => sanitizeFile(part, "_").toLowerCase())
        .join("/")
    : sanitizeFile(langConfig.packageName, "_").toLowerCase();

  const sourceDirectory = `src/${moduleDirectory}`;

  let runnerCommands: RunnerCommands = [];

  if (packageManager === "poetry") {
    runnerCommands = [
      {
        command: "poetry",
        args: ["version", "{{ .PackageVersion }}"],
      },
      {
        command: "poetry",
        args: ["lock"],
      },
    ];

    // README-PYPI.md must be in place before running poetry install, however
    // this file is only necessary if RepoURL is set. Otherwise, poetry
    // install will return an error such as:
    //   The current project could not be installed: Readme path `.../README-PYPI.md` does not exist.
    if (genConfig.RepoURL) {
      runnerCommands.push({
        command: "poetry",
        args: ["run", "python", "scripts/prepare_readme.py"],
      });
    }

    runnerCommands.push(
      {
        command: "poetry",
        args: ["install", "--with=dev"],
      },
      {
        command: "poetry",
        args: ["run", "python", "-m", "compileall", "-q", "."],
      },
      {
        command: "poetry",
        args: ["run", "python", "-m", "pylint", "-j=0", sourceDirectory],
        parallel: true,
      },
      {
        command: "poetry",
        args: ["run", "python", "-m", "mypy", sourceDirectory],
        parallel: true,
      },
      {
        command: "poetry",
        args: ["run", "python", "-m", "pyright", sourceDirectory],
        parallel: true,
      },
    );
  } else {
    // uv commands
    runnerCommands = [
      {
        command: "uv",
        args: ["lock"],
      },
    ];

    // README-PYPI.md must be in place before running uv sync, however
    // this file is only necessary if RepoURL is set.
    if (genConfig.RepoURL) {
      runnerCommands.push({
        command: "uv",
        args: ["run", "python", "scripts/prepare_readme.py"],
      });
    }

    runnerCommands.push(
      {
        command: "uv",
        args: ["sync", "--dev"],
      },
      {
        command: "uv",
        args: ["run", "python", "-m", "compileall", "-q", "."],
      },
      {
        command: "uv",
        args: ["run", "python", "-m", "pylint", "-j=0", sourceDirectory],
        parallel: true,
      },
      {
        command: "uv",
        args: ["run", "python", "-m", "mypy", sourceDirectory],
        parallel: true,
      },
      {
        command: "uv",
        args: ["run", "python", "-m", "pyright", sourceDirectory],
        parallel: true,
      },
    );
  }

  return {
    runner: {
      commands: runnerCommands,
      dependencies: {
        argVariables: {
          PackageVersion: ".+",
        },
        commands: getPythonCommandDependencies(packageManager),
      },
    },
  };
}

// @ts-ignore
function getTestingConfiguration(
  genConfig: GeneratorInitialConfiguration,
): TestingConfiguration {
  // Ideally, this would just use the getTestDirectory() function from tests.ts,
  // however the pared down config.ts environment does not include common.d.ts
  // and importing that file has its own issues. So for now, hardcode around
  // these issues.
  // const testDirectory = getTestDirectory();
  const testDirectory = "tests";
  const packageManager = genConfig.LangCfg.packageManager || "uv";

  let commands: RunnerCommands = [];

  if (packageManager === "poetry") {
    commands = [
      {
        command: "poetry",
        args: ["lock"],
      },
      // Reference: internal issue reference
      {
        command: "poetry",
        args: ["install", "--with=dev"],
      },
      {
        command: "poetry",
        args: [
          "run",
          "python",
          "-m",
          "pytest",
          `./${testDirectory}`,
          "-n",
          "auto",
          "-vv",
          "--junit-xml=./.speakeasy/reports/tests.xml", // TODO: the output folder for the tests report should probably be templated based on whether .speakeasy or .gen is used
        ],
      },
    ];
  } else {
    commands = [
      {
        command: "uv",
        args: ["lock"],
      },
      // Reference: internal issue reference
      {
        command: "uv",
        args: ["sync", "--dev"],
      },
      {
        command: "uv",
        args: [
          "run",
          "python",
          "-m",
          "pytest",
          `./${testDirectory}`,
          "-n",
          "auto",
          "-vv",
          "--junit-xml=./.speakeasy/reports/tests.xml", // TODO: the output folder for the tests report should probably be templated based on whether .speakeasy or .gen is used
        ],
      },
    ];
  }

  return {
    runner: {
      commands,
      dependencies: {
        commands: getPythonCommandDependencies(packageManager),
      },
    },
  };
}

function validateDependencyMap(
  cfg: Record<string, any>,
  field: string,
): string {
  if (!(field in cfg)) {
    return "";
  }

  const deps = cfg[field];

  if (typeof deps !== "object" || deps === null) {
    return `${field} must be an object`;
  }

  for (const [groupName, groupDeps] of Object.entries(deps)) {
    if (typeof groupDeps !== "object" || groupDeps === null) {
      return `${field}.${groupName} must be an object`;
    }

    for (const [name, version] of Object.entries(
      groupDeps as Record<string, any>,
    )) {
      if (typeof version !== "string") {
        return `${field}.${groupName}.${name} must be a string`;
      }
    }
  }

  return "";
}

// @ts-ignore
function validateConfig(cfg: Record<string, any>): string {
  let err = validateDependencyMap(cfg, "additionalDependencies");
  if (err) return err;

  err = validateDependencyMap(cfg, "optionalDependencies");
  if (err) return err;

  if ("optionalDependencies" in cfg) {
    for (const extrasName of Object.keys(cfg.optionalDependencies)) {
      if (!/^[a-zA-Z0-9]([a-zA-Z0-9._-]*[a-zA-Z0-9])?$/.test(extrasName)) {
        return `optionalDependencies.${extrasName} is not a valid extras name (letters, numbers, hyphens, dots, underscores only)`;
      }
    }
  }

  if (cfg.bodyVariantOverloads === true && cfg.flattenRequests !== true) {
    return "bodyVariantOverloads requires flattenRequests to also be true";
  }

  return "";
}

/**
 * Hardcoded configuration overlay for the target. This is used to configure
 * required settings for the target that are intentionally not exposed.
 */
// @ts-ignore
function getConfigOverlay(): Record<string, any> {
  return {
    reservedModelFileNames: [
      "__init__",
      "responsevalidationerror",
      "no_response_error",
    ],
  };
}
