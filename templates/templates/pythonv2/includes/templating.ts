type PythonV2Type = {
  ModelName: string;
  Type: TypeDef;
  TypedDictName: string;
  // Parsed model validator file content attached to the class that owns
  // it within this modelfile (the class whose sanitized name matches the
  // modelfile name). Other types in the same file leave this null.
  ModelValidator?: ParsedModelValidator | null;
};

type PythonV2Model = {
  Name: string;
  OutputLocation: string;
  Types: PythonV2Type[];
  Servers?: Servers;
};

type PythonV2ModelToExport = {
  Name: string;
  Types: string[];
};

type PythonPublicExportImport = {
  Module: string;
  Specifiers: { Name: string; Alias: string }[];
};

type PythonPublicExportAlias = {
  Name: string;
};

type PythonPublicExportGroup = {
  Group: string;
  Parts: string[];
  FilePath: string;
  Imports: PythonPublicExportImport[];
  Assignments: { Name: string; Target: string }[];
  Aliases: PythonPublicExportAlias[];
  SubPackages: string[];
  Exports: string;
};

const PYTHON_V2_TYPES_SUPPORT_MODELS: PythonV2ModelToExport[] = [
  {
    Name: "basemodel",
    Types: [
      "BaseModel",
      "Nullable",
      "OptionalNullable",
      "UnrecognizedInt",
      "UnrecognizedStr",
      "UNSET",
      "UNSET_SENTINEL",
    ],
  },
  {
    Name: "base64fileinput",
    Types: ["Base64EncodedString", "Base64FileInput"],
  },
];

function willTemplateGeneratedTypesInit(): boolean {
  if (
    getModelsLocation(context.Global.Config.Imports.GetErrorsPath()) == "types"
  ) {
    return true;
  }

  for (const [outputLocation] of sequencedMapEntries(
    context.Global.AST.BucketedTypes,
  )) {
    if (getModelsLocation(outputLocation) == "types") {
      return true;
    }
  }

  return false;
}

function shouldSkipAuxiliaryFile(auxFilePath: string): boolean {
  // Normalize to the on-disk output path so the predicate doesn't care
  // whether the auxiliary source is a raw copy or a `.stmpl` template.
  const resolved = auxFilePath.replace(/\.stmpl$/, "");

  if (
    willTemplateGeneratedTypesInit() &&
    resolved == `${getSourcePath()}/types/__init__.py`
  ) {
    return true;
  }

  return (
    !pythonRawResponseHelpersEnabled() &&
    resolved == `${getSourcePath()}/utils/response_helpers.py`
  );
}

// @ts-ignore (Ignore override of function implementation)
function templateAuxiliaryFiles(data: any, sourcePath: string = "auxiliary") {
  let files = getDirectoryFiles(sourcePath);

  for (const file of files) {
    let outFile = file.replace(`${sourcePath}/`, "");

    outFile = templateStringInput("auxPath:" + outFile, outFile, data);
    if (outFile.trim().length == 0 || shouldSkipAuxiliaryFile(outFile)) {
      continue;
    }

    if (file.endsWith(".stmpl")) {
      outFile = outFile.replace(".stmpl", "");

      templateFile(file, outFile, context.Global);
    } else {
      copy(file, outFile);
    }
  }
}

function collectExportNames(
  types: PythonV2Type[],
  model: string,
  servers?: Servers,
): string[] {
  const typesToExport: string[] = [];

  if (
    types.find((t) => t.Type.Scope.toString() == "operations") ||
    types.length == 0
  ) {
    if (servers) {
      let modelName = sanitizeFieldName(model);
      typesToExport.push(`${modelName.toUpperCase() + "_SERVERS"}`);

      if (servers.ServerMap) {
        for (const server of servers.Servers) {
          typesToExport.push(
            `${templateConstName(server.ID, modelName + "Server")}`,
          );
        }
      }
    }
  }

  for (const type of types) {
    let typeName = sanitizeClassName(type.Type.Name);
    typesToExport.push(typeName);

    if (type.Type.IsUnionOpen && type.Type.Discriminator) {
      typesToExport.push(`Unknown${typeName}`);
    }

    if (type.Type.Scope.toString() == "errors") {
      if (type.Type.Type.toString() == "error") {
        typesToExport.push(`${typeName}Data`);
      } else if (isUnionOfErrors(type.Type)) {
        typesToExport.push(`${typeName}Union`);
      }
    }

    if (
      type.Type.Type.toString() == "class" ||
      (type.Type.Scope.toString() != "errors" &&
        type.Type.Type.toString() == "union")
    ) {
      typesToExport.push(type.TypedDictName);
    }
  }

  return typesToExport;
}

function sanitizePythonPublicExportGroupPart(part: string): string {
  return sanitizeFileName(String(part || "").trim());
}

function sanitizePythonPublicExportName(name: string): string {
  return sanitizeClassName(name);
}

function pythonPublicExportImportModule(
  typeDef: TypeDef,
  groupFilePath: string,
): string {
  const modelLocation = getModelsLocation(typeDef.OutputLocation)
    .split("/")
    .join(".");
  const subModule = `${modelLocation}.${resolveModelName(typeDef)}`;

  // Export groups are collected once and rendered into multiple files at
  // different package depths, so relative imports must be derived from the
  // group's own file path rather than the file currently being templated
  // (context.OutFile, which getModuleImport would use).
  if (useRelativeImports()) {
    const groupFilePkg = groupFilePath
      .slice(SRC_ROOT.length)
      .split("/")
      .slice(0, -1)
      .join(".");
    const root = getModuleDirectory().replaceAll("/", ".");
    return pythonRelativeImport(groupFilePkg, `${root}.${subModule}`);
  }

  return getAbsoluteModuleImport(subModule);
}

function pythonPublicExportImportName(
  aliasName: string,
  typeDef: TypeDef,
  wantInputCompanion: boolean,
): string {
  const typeName = sanitizeClassName(typeDef.Name);
  if (
    typeDef.Type.toString() == "class" ||
    (typeDef.Scope.toString() != "errors" && typeDef.Type.toString() == "union")
  ) {
    const typedDictName = resolveTypedDictName(typeDef, typeName);
    // Input exports alias the input companion type; targets without a
    // companion (enums, errors) fall through to the model class below.
    if (wantInputCompanion || aliasName == typedDictName) {
      return typedDictName;
    }
  }
  return typeName;
}

function uniqueSortedPythonPublicExportValues(values: string[]): string[] {
  return [...new Set(values)].sort();
}

function pythonPublicExports(items: string[]): string {
  const exports = uniqueSortedPythonPublicExportValues(items);
  return exports.length > 0
    ? `__all__ = [\n${exports.map((item) => `"${item}"`).join(",\n")}\n]`
    : "";
}

// The resources package location comes from imports.paths.resources in
// gen.yaml. Configuring it also opts the SDK into implicit model-namespace
// exports; explicit x-speakeasy-exports render at the default location
// whether or not it is configured.
function pythonConfiguredResourcesPath(): string {
  // Normalize away empty, ".", and ".." segments so values like
  // "./resources" cannot produce malformed package paths.
  return (context.Global.Config.Imports?.GetResourcesPath?.() ?? "")
    .split("/")
    .filter((segment) => segment !== "" && segment !== "." && segment !== "..")
    .join("/");
}

function pythonResourcesSegments(): string[] {
  const segments = (pythonConfiguredResourcesPath() || "resources")
    .split("/")
    .map((segment) => sanitizePythonPublicExportGroupPart(segment))
    .filter((segment) => segment.length > 0);
  // A configured value that sanitizes to nothing falls back to the default
  // rather than producing an invalid package root.
  return segments.length > 0 ? segments : ["resources"];
}

function pythonResourcesRootPackage(): string {
  return pythonResourcesSegments()[0];
}
registerTemplateFunc("pythonResourcesRootPackage", pythonResourcesRootPackage);

// A resources path overlapping a model-scope location (or a core SDK
// directory) would have the export surface and other jobs writing the same
// __init__.py nondeterministically; the surface is skipped instead.
function pythonResourcesPathCollides(): boolean {
  const path = pythonResourcesSegments().join("/");
  const imports = context.Global.Config.Imports;
  const reserved = [
    getModelsLocation(""),
    getModelsLocation(imports.GetSharedPath()),
    getModelsLocation(imports.GetErrorsPath()),
    getModelsLocation(imports.GetOperationsPath()),
    getModelsLocation(imports.GetWebhooksPath()),
    getModelsLocation(imports.GetCallbacksPath()),
    "_hooks",
    "_version",
    "basesdk",
    "errors",
    "hooks",
    "httpclient",
    "models",
    "sdk",
    "sdkconfiguration",
    "types",
    "utils",
  ].filter((p) => p);
  return reserved.some(
    (p) => path === p || path.startsWith(`${p}/`) || p.startsWith(`${path}/`),
  );
}

function buildPythonPublicExportGroup(
  sourcePath: string,
  group: ResolvedPublicExportGroup,
): PythonPublicExportGroup | undefined {
  const parts = (group.Parts || [])
    .map((part) => sanitizePythonPublicExportGroupPart(part))
    .filter((part) => part.length > 0);
  if (parts.length === 0) {
    return undefined;
  }

  const includeImplicit = pythonConfiguredResourcesPath() !== "";
  const filePath = [
    sourcePath,
    ...pythonResourcesSegments(),
    ...parts,
    "__init__.py",
  ].join("/");
  const importsByModule = new Map<string, PythonPublicExportImport>();
  const aliases: PythonPublicExportAlias[] = [];

  for (const publicExport of group.Exports || []) {
    const target = publicExport.Target;
    if (!target) {
      continue;
    }
    if (publicExport.Implicit && !includeImplicit) {
      continue;
    }

    const aliasName = sanitizePythonPublicExportName(publicExport.Name || "");
    if (!aliasName) {
      continue;
    }

    const module = pythonPublicExportImportModule(target, filePath);
    let importBucket = importsByModule.get(module);
    if (!importBucket) {
      importBucket = { Module: module, Specifiers: [] };
      importsByModule.set(module, importBucket);
    }
    importBucket.Specifiers.push({
      Name: pythonPublicExportImportName(
        aliasName,
        target,
        publicExport.Input === true,
      ),
      Alias: aliasName,
    });
    aliases.push({ Name: aliasName });
  }

  const subPackages = uniqueSortedPythonPublicExportValues(
    (group.Children || [])
      .map((child) => sanitizePythonPublicExportGroupPart(child.Name))
      .filter((name) => name.length > 0),
  );

  // Importing the same source name twice from one module trips pylint's
  // reimported check, so secondary bindings become plain assignments after
  // the imports instead. The kept import prefers the unaliased binding.
  const assignments: { Name: string; Target: string }[] = [];
  for (const importBucket of importsByModule.values()) {
    const keeperBySource = new Map<string, string>();
    for (const specifier of importBucket.Specifiers) {
      const keeper = keeperBySource.get(specifier.Name);
      if (keeper === undefined || specifier.Alias === specifier.Name) {
        keeperBySource.set(specifier.Name, specifier.Alias);
      }
    }
    importBucket.Specifiers = importBucket.Specifiers.filter((specifier) => {
      if (keeperBySource.get(specifier.Name) === specifier.Alias) {
        return true;
      }
      assignments.push({
        Name: specifier.Alias,
        Target: keeperBySource.get(specifier.Name) ?? specifier.Name,
      });
      return false;
    });
  }
  assignments.sort((a, b) => a.Name.localeCompare(b.Name));

  return {
    Group: parts.join("."),
    Parts: parts,
    FilePath: filePath,
    Imports: [...importsByModule.values()]
      .map((imp) => ({
        ...imp,
        Specifiers: imp.Specifiers.sort((a, b) =>
          a.Alias.localeCompare(b.Alias),
        ),
      }))
      .sort((a, b) => a.Module.localeCompare(b.Module)),
    Assignments: assignments,
    Aliases: aliases,
    SubPackages: subPackages,
    Exports: pythonPublicExports([
      ...aliases.map((alias) => alias.Name),
      ...subPackages,
    ]),
  };
}

function collectPythonPublicExportGroups(
  sourcePath: string,
): PythonPublicExportGroup[] {
  if (context.GlobalComputed?._pythonPublicExportGroups !== undefined) {
    return context.GlobalComputed._pythonPublicExportGroups;
  }
  context.GlobalComputed ??= {};

  if (pythonResourcesPathCollides()) {
    context.GlobalComputed._pythonPublicExportGroups = [];
    return [];
  }

  const sorted = (context.Global.AST.PublicExports?.Groups || [])
    .map((group) => buildPythonPublicExportGroup(sourcePath, group))
    .filter((group): group is PythonPublicExportGroup => Boolean(group))
    .sort((a, b) => a.FilePath.localeCompare(b.FilePath));
  context.GlobalComputed._pythonPublicExportGroups = sorted;
  return sorted;
}

function hasPublicExports(): boolean {
  return collectPythonPublicExportGroups(getSourcePath()).some(
    (group) => group.Aliases.length > 0,
  );
}
registerTemplateFunc("hasPublicExports", hasPublicExports);

function getPythonPublicExportJobs(sourcePath: string): TemplateFileJob[] {
  const groups = collectPythonPublicExportGroups(sourcePath);
  if (!groups.some((group) => group.Aliases.length > 0)) {
    return [];
  }

  const rootSubPackages = uniqueSortedPythonPublicExportValues(
    (context.Global.AST.PublicExports?.RootChildren || [])
      .map((child) => sanitizePythonPublicExportGroupPart(child.Name))
      .filter((name) => name.length > 0),
  );

  // One __init__.py per segment of the configured resources path; only the
  // leaf package re-exports the export groups.
  const segments = pythonResourcesSegments();
  const jobs: TemplateFileJob[] = segments.map((_, i) => {
    const subPackages =
      i === segments.length - 1 ? rootSubPackages : [segments[i + 1]];
    return createTemplateFileJob(
      "resources.py.stmpl",
      [sourcePath, ...segments.slice(0, i + 1), "__init__.py"].join("/"),
      {
        Imports: [],
        Aliases: [],
        SubPackages: subPackages,
        Exports: pythonPublicExports(subPackages),
      },
    );
  });

  for (const group of groups) {
    jobs.push(
      createTemplateFileJob("resources.py.stmpl", group.FilePath, group),
    );
  }

  return jobs;
}

// @ts-ignore (ignore-override)
function getModelJobs(types: BucketedTypes, path: string): TemplateFileJob[] {
  const jobs: TemplateFileJob[] = [];

  const errorsPath = context.Global.Config.Imports.GetErrorsPath();

  const [__, ok] = types.Get(errorsPath);
  if (!ok) {
    createBucket(types, errorsPath);
  }

  const initPyLocs = {};

  // Pre-collect all leaf model paths so we can skip creating empty parent
  // __init__.py files for paths that will also be created as leaf files with
  // actual model exports. Without this, randomized job execution can cause
  // the empty parent to overwrite the leaf's exports (race condition).
  const leafModelPaths = new Set<string>();
  for (const [outputLocation] of sequencedMapEntries(types)) {
    leafModelPaths.add(getModelsLocation(outputLocation));
  }

  // Pre-collect subpackage names for each parent directory so parent __init__.py
  // files can expose them via lazy loading (e.g., models.test.Widget).
  const subPackagesByParent: Record<string, string[]> = {};
  for (const [outputLocation] of sequencedMapEntries(types)) {
    const modelPath = getModelsLocation(outputLocation);
    const pathParts = modelPath.split("/");
    if (pathParts.length > 1) {
      const parentPath = pathParts.slice(0, -1).join("/");
      const childName = pathParts[pathParts.length - 1];
      if (!subPackagesByParent[parentPath]) {
        subPackagesByParent[parentPath] = [];
      }
      if (!subPackagesByParent[parentPath].includes(childName)) {
        subPackagesByParent[parentPath].push(childName);
      }
    }
  }
  // Sort subpackage lists for deterministic output
  for (const parent of Object.keys(subPackagesByParent)) {
    subPackagesByParent[parent].sort();
  }

  for (const [outputLocation, models] of sequencedMapEntries(types)) {
    let modelPath = getModelsLocation(outputLocation);

    let modelsToExport: PythonV2ModelToExport[] = [];

    // Pre-add built-in error classes
    if (
      modelPath ==
      getModelsLocation(context.Global.Config.Imports.GetErrorsPath())
    ) {
      modelsToExport.push({
        Name: getDefaultErrorFileName(),
        Types: [getDefaultErrorClassName()],
      });
      modelsToExport.push({
        Name: getBaseErrorFileName(),
        Types: [getBaseErrorClassName()],
      });
      modelsToExport.push({
        Name: getNoResponseErrorFileName(),
        Types: [getNoResponseErrorClassName()],
      });
      modelsToExport.push({
        Name: "responsevalidationerror",
        Types: ["ResponseValidationError"],
      });
    }

    for (let [model, tt] of sequencedMapEntries(models)) {
      // Sort types to avoid usage before declaration
      tt = topologicalSortTypeDefs(tt, useConflictResistantModelImports());

      const filteredTypes = tt.filter((t) => {
        return t.Scope != "utils";
      });

      let modelName = sanitizeFileName(model);

      const modelValidatorFile = modelValidatorPath(modelPath, modelName);
      const modelValidator = readModelValidator(modelValidatorFile);
      const ownerClassName = sanitizeClassName(model);

      let types = filteredTypes.map((t) => {
        const className = sanitizeClassName(t.Name);
        // Use the same disambiguation logic as sanitizeClass to compute
        // the TypedDict companion name.
        const typedDictName = resolveTypedDictName(t, className);
        return {
          ModelName: model,
          Type: t,
          TypedDictName: typedDictName,
          ModelValidator:
            modelValidator && className === ownerClassName
              ? modelValidator
              : null,
        };
      });

      if (modelValidator && !types.some((t) => t.ModelValidator)) {
        logWarning(
          `${modelValidatorFile}: no class matches owner '${ownerClassName}'; ` +
            `validator file is unused. Rename file or owner class.`,
          0,
        );
      }

      // Resolve servers for operations models
      let servers: Servers | undefined;
      if (
        types.find((t) => t.Type.Scope.toString() == "operations") ||
        types.length == 0
      ) {
        servers = context.Global.AST.OperationServers.Get(model)[0];
      }

      if (types.length === 0 && !servers) {
        continue;
      }

      let ctx: PythonV2Model = {
        Name: model,
        Types: types,
        OutputLocation: outputLocation,
        Servers: servers,
      };

      const typesToExport = collectExportNames(types, model, servers);

      modelsToExport.push({
        Name: modelName,
        Types: typesToExport.sort(),
      });

      jobs.push(
        createTemplateFileJob(
          `modelfile.py.stmpl`,
          `${path}/${modelPath}/${sanitizeFileName(model)}.py`,
          ctx,
        ),
      );
    }

    if (modelPath == "types") {
      modelsToExport.push(...PYTHON_V2_TYPES_SUPPORT_MODELS);
    }

    modelsToExport.sort((a, b) => {
      return a.Name.localeCompare(b.Name);
    });

    let allExports = modelsToExport
      .flatMap((m) => m.Types.map((t) => `"${t}"`))
      .sort()
      .join(",\n");

    if (allExports) {
      allExports = `__all__ = [\n${allExports}\n]`;
    }

    // identify types with circular dependencies
    const allSdkModels: TypeDef[] = sequencedMapEntries(models)
      .map(([_, tt]) => tt)
      .flat()
      .filter((t) => t.Scope.toString() != "utils");

    // We've used forward references to work around circular dependencies in
    // Pydantic models.
    //
    // Pydantic schemas need to be explicitly rebuilt after all models are loaded,
    // making __init__.py the ideal place to do this.
    //
    // See: https://docs.pydantic.dev/latest/concepts/models/#rebuilding-model-schema
    const circularTypes = identifyCircularTypes(allSdkModels);
    const pydanticTypes = circularTypes.filter(
      (t) => t.Type.toString() == "class",
    );

    const circularReferencedModels = new Set(
      circularTypes.map((item) => sanitizeClassName(item.Name)),
    );

    // We need to eagerly import models which have circular references. This is needed because we
    // need to model_rebuild (https://docs.pydantic.dev/latest/concepts/models/#rebuilding-model-schema) such types.
    const modelsToExportEagerly = modelsToExport
      .filter((model) =>
        model.Types.some((typeName) => circularReferencedModels.has(typeName)),
      )
      .sort((a, b) => a.Name.localeCompare(b.Name));

    if (
      modelPath ==
      getModelsLocation(context.Global.Config.Imports.GetErrorsPath())
    ) {
      modelsToExportEagerly.push({
        Name: getBaseErrorFileName(),
        Types: [getBaseErrorClassName()],
      });

      modelsToExport = modelsToExport.filter(
        (model) => model.Name !== getBaseErrorFileName(),
      );
    }

    if (modelPath == "types") {
      modelsToExportEagerly.push(...PYTHON_V2_TYPES_SUPPORT_MODELS);
      modelsToExport = modelsToExport.filter(
        (model) =>
          !PYTHON_V2_TYPES_SUPPORT_MODELS.some((m) => m.Name === model.Name),
      );
    }

    // Lazily imported models to improve module loading performance
    let modelsToExportLazily = modelsToExport
      .filter((model) =>
        model.Types.every(
          (typeName) => !circularReferencedModels.has(typeName),
        ),
      )
      .sort((a, b) => a.Name.localeCompare(b.Name));

    modelsToExportLazily = modelsToExport
      .filter((model) => !modelsToExportEagerly.includes(model))
      .sort((a, b) => a.Name.localeCompare(b.Name));

    const pathParts = modelPath.split("/");
    let currentParts = [];

    for (const part of pathParts) {
      currentParts.push(part);

      let currentPath = currentParts.join("/");

      if (
        !(currentPath in initPyLocs) &&
        currentPath != modelPath &&
        !leafModelPaths.has(currentPath)
      ) {
        let initData: {
          EagerlyExportedModels: PythonV2ModelToExport[];
          LazilyExportedModels: PythonV2ModelToExport[];
          Exports: string;
          PartialPydanticTypes: TypeDef[];
          SubPackages: string[];
        } = {
          EagerlyExportedModels: [],
          LazilyExportedModels: [],
          Exports: "",
          PartialPydanticTypes: [],
          SubPackages: subPackagesByParent[currentPath] ?? [],
        };

        // When creating a parent init for the errors location, include
        // built-in error class exports so they are always accessible
        const errorsLocation = getModelsLocation(
          context.Global.Config.Imports.GetErrorsPath(),
        );
        if (currentPath == errorsLocation) {
          const builtInModels: PythonV2ModelToExport[] = [
            {
              Name: getDefaultErrorFileName(),
              Types: [getDefaultErrorClassName()],
            },
            {
              Name: getBaseErrorFileName(),
              Types: [getBaseErrorClassName()],
            },
            {
              Name: getNoResponseErrorFileName(),
              Types: [getNoResponseErrorClassName()],
            },
            {
              Name: "responsevalidationerror",
              Types: ["ResponseValidationError"],
            },
          ];
          initData.EagerlyExportedModels = [
            {
              Name: getBaseErrorFileName(),
              Types: [getBaseErrorClassName()],
            },
          ];
          initData.LazilyExportedModels = builtInModels.filter(
            (m) => m.Name !== getBaseErrorFileName(),
          );
          const builtInExports = builtInModels
            .flatMap((m) => m.Types.map((t) => `"${t}"`))
            .sort()
            .join(",\n");
          if (builtInExports) {
            initData.Exports = `__all__ = [\n${builtInExports}\n]`;
          }
        }

        if (currentPath == "types") {
          initData.EagerlyExportedModels.push(
            ...PYTHON_V2_TYPES_SUPPORT_MODELS,
          );
          const supportExports = PYTHON_V2_TYPES_SUPPORT_MODELS.flatMap(
            (m) => m.Types,
          ).map((t) => `"${t}"`);
          const existingExports = initData.Exports
            ? initData.Exports.match(/"[^"]+"/g) || []
            : [];
          initData.Exports = `__all__ = [\n${existingExports
            .concat(supportExports)
            .sort()
            .join(",\n")}\n]`;
        }

        jobs.push(
          createTemplateFileJob(
            `__init__.py.stmpl`,
            `${path}/${currentPath}/__init__.py`,
            initData,
          ),
        );

        initPyLocs[currentPath] = true;
      }
    }
    jobs.push(
      createTemplateFileJob(
        `__init__.py.stmpl`,
        `${path}/${modelPath}/__init__.py`,
        {
          Exports: allExports,
          PartialPydanticTypes: pydanticTypes,
          EagerlyExportedModels: modelsToExportEagerly,
          LazilyExportedModels: modelsToExportLazily,
          SubPackages: subPackagesByParent[modelPath] ?? [],
        },
      ),
    );
    initPyLocs[modelPath] = true;
  }
  return jobs;
}

/**
 * Templates a comment and a list of calls to `model_rebuild()` for each type
 * that has a forward reference.
 *
 * Generated code:
 *
 * ```python
 * # Pydantic models with forward references
 * <type1>.model_rebuild()
 * <type2>.model_rebuild()
 * ```
 */
function templatePartialPydanticTypes(types: TypeDef[]): string {
  let lines = ["# Pydantic models with forward references"];

  for (const type of types) {
    lines.push(`${sanitizeClassName(type.Name)}.model_rebuild()`);
  }

  return lines.join("\n");
}

registerTemplateFunc(
  "templatePartialPydanticTypes",
  templatePartialPydanticTypes,
);
// @ts-ignore
function templateGoodNames(sdk: SDK): string {
  const fixedNames = ["i", "j", "k", "ex", "Run", "_", "e"];
  const names = [];

  for (const global of sdk.Globals?.Fields ?? []) {
    let name = templateSDKInitFieldName(global.Name, "globals");
    if (name.length < 3) {
      if (!names.includes(name)) {
        names.push(name);
      }
    }
  }

  for (const serverVariable of sdk.Servers?.GetVariables() ?? []) {
    let name = templateSDKInitFieldName(serverVariable.Name, "server");
    if (name.length < 3) {
      if (!names.includes(name)) {
        names.push(name);
      }
    }
  }

  for (const subSDK of sdk.SubSDKs) {
    for (const sub of flattenSubSDKs(subSDK)) {
      let name = sanitizeFieldName(sub.Type.Name);
      if (name.length < 3) {
        if (!names.includes(name)) {
          names.push(name);
        }
      }

      for (const operation of sub.Operations) {
        let name = sanitizeMethodName(operation);
        if (name.length < 3) {
          if (!names.includes(name)) {
            names.push(name);
          }
        }

        if (operation.Request) {
          for (const field of operation.Request.Field.Type.Fields) {
            let name = sanitizeFieldName(field.Name);
            if (name.length < 3) {
              if (!names.includes(name)) {
                names.push(name);
              }
            }
          }
        }
      }
    }
  }

  for (const [, models] of sequencedMapEntries(
    context.Global.AST.BucketedTypes,
  )) {
    for (const [, types] of sequencedMapEntries(models)) {
      for (const type of types) {
        for (const field of type.Fields) {
          let name = sanitizeFieldName(field.Name);
          if (name.length < 3) {
            if (!names.includes(name)) {
              names.push(name);
            }
          }
        }
      }
    }
  }

  return fixedNames.concat(names.sort()).join(",\n           ");
}

registerTemplateFunc("templateGoodNames", templateGoodNames);

// @ts-ignore
function templateSDKInitFieldName(name: string, suffix: string): string {
  let reserved = ["security", "server_url", "url_params", "client"];

  if (
    context.Global.AST.MainSDK.Servers &&
    context.Global.AST.MainSDK.Servers.ServerMap
  ) {
    reserved.push("server");
  } else {
    reserved.push("server_idx");
  }

  let fieldName = sanitizeAttributeName(name);

  if (reserved.includes(name)) {
    fieldName += "_" + suffix;
  }

  return fieldName;
}

registerTemplateFunc("templateSDKInitFieldName", templateSDKInitFieldName);

// @ts-ignore
function getGlobalSecurity(
  options?: SecurityUsageContext | Example,
  additionalContext?: TemplateValueContext,
): string {
  if (!context.Global.AST.MainSDK.Security) {
    return "";
  }

  let index = undefined;
  let example = undefined;

  if (options) {
    if ("index" in options || "example" in options) {
      ({ index, example } = options);
    } else {
      example = getExampleValue(options as Example, additionalContext);
    }
  }

  if (
    index === undefined &&
    example === undefined &&
    context.Global.AST.MainSDK.Security.Optional &&
    context.Global.AST.MainSDK.SecurityConfig.OptionalityReason ==
      "optional-scheme"
  ) {
    return "";
  }

  let securityFieldName = "security";
  if (
    context.Global.AST.MainSDK.Security.Type.Fields.length == 1 &&
    context.Global.Config.FlattenGlobalSecurity == true
  ) {
    securityFieldName = caser().ToSnake(
      context.Global.AST.MainSDK.Security.Type.Fields[0].Name,
    );
  }

  return `${securityFieldName}=${templateSecurityUsage(
    context.Global.AST.MainSDK.Security.Type,
    1,
    true,
    context.Global.Config.FlattenGlobalSecurity,
    index,
    example,
    additionalContext,
  ).trim()},`;
}

// @ts-ignore
function joinSDKParams(params: string[]): string {
  if (params.length > 0) {
    params.unshift("");

    return params.join(`\n${"    ".repeat(1)}`) + "\n";
  }

  return "";
}

// @ts-ignore
function templateUsageSDKParams(local: UsageContext): string {
  const params = [];
  let hasSecurity = local.Operation.Security != undefined;
  let hasAbsoluteServerURL =
    context.Global.AST.MainSDK.Servers?.HasAbsoluteURL() || false;

  const ctx: TemplateValueContext = {
    usageContext: local, // TODO see if this can be decomposed into the required fields below
    operation: local.Operation,
    isTest: local.Test && true,
    test: local.Test,
  };

  const hasOperationServers = local.Operation?.Servers?.Servers?.length > 0;

  const hasAnyOperationServers =
    context.Global.AST.MainSDK.HasAnyOperationServers() || false;

  if (!hasAbsoluteServerURL && !hasOperationServers) {
    const url = getUsageServerUrl(local, undefined, ctx);
    if (hasAnyOperationServers) {
      // Constructor takes optional server_url — use keyword arg
      params.push(`server_url=${url},`);
    } else {
      // Constructor requires positional server_url — pass raw value
      params.push(`${url},`);
    }
  }

  if (local.Scopes != undefined) {
    local.Scopes.forEach((scope) => {
      if (scope.IsGlobal && scope.Feature != "") {
        const feature = scope.Feature.toString();

        switch (feature) {
          case "security":
            if (!hasSecurity) {
              const globalSecurity = getGlobalSecurity(scope.Value, ctx);
              if (globalSecurity) {
                params.push(globalSecurity);
                hasSecurity = true;
              }
            }
            break;
          case "server_url": {
            if (hasAbsoluteServerURL && !hasOperationServers) {
              params.push(
                `server_url=${getUsageServerUrl(local, scope, ctx)},`,
              );
            }
            break;
          }
          case "server_selection": {
            const server: UsageGlobalServer = getUsageGlobalServer(
              !!scope.Value,
            );

            if (server.ID) {
              params.push(`server="${server.ID}",`);
            } else if (server.Index !== undefined) {
              params.push(`server_idx=${server.Index},`);
            }

            if (server.Variables) {
              server.Variables.forEach((v: ServerVariable) => {
                const fieldName = templateSDKInitFieldName(v.Name, "global");
                params.push(
                  `${fieldName}="${getUsageServerVariableValue(v)}",`,
                );
              });
            }

            break;
          }
          case "retries": {
            params.push(
              `retry_config=${templateString("usage/retries.stmpl", {})},`,
            );
            break;
          }
          case "http_client": {
            params.push(`client=${templateHTTPClient(scope.Value)},`);
            break;
          }
          case "parameter": {
            const parameter = scope.Value as ParameterUsage;

            const value = templateValueAsRequired(
              parameter.field,
              getExampleValue(parameter.example),
              {
                ...ctx,
                shouldTemplateConstValue: shouldTemplateConstValue,
              },
            );

            if (value === "") {
              return;
            }

            params.push(
              `${templateSDKInitFieldName(
                parameter.field.Name,
                "global",
              )}=${value},`,
            );
            break;
          }
        }
      }
    });
  }

  if (!hasSecurity && opUsesGlobalSecurity(local.Operation)) {
    const globalSecurity = getGlobalUsageSecurity(local);
    if (globalSecurity) {
      params.push(globalSecurity);
    }
  }

  return joinSDKParams(params);
}

registerTemplateFunc("templateUsageSDKParams", templateUsageSDKParams);

function templateDeprecated(
  comments?: CommentDef,
  scope: "class" | "method" | "field" = "class",
): string {
  if (!comments) {
    return "";
  }

  if (!comments.Deprecated) {
    return "";
  }

  let deprecated = `warning: ** DEPRECATED ** - ${comments.DeprecationMessage}.`;

  if (comments.DeprecationReplacement) {
    const replacement = sanitizeDeprecationReplacement(
      comments.DeprecationReplacement,
      scope,
    );

    if (replacement) {
      deprecated += ` Use ${replacement} instead.`;
    }
  }

  deprecated = deprecated.includes("\n")
    ? `"""${deprecated}"""`
    : `"${deprecated}"`;

  if (scope === "field") {
    return deprecated;
  }

  addImport("typing_extensions", "deprecated");
  return `\n${scope == "method" ? "    " : ""}@deprecated(${deprecated})`;
}
registerTemplateFunc("templateDeprecated", templateDeprecated);

// @ts-ignore
function templateMethodReturnType(op: Operation): string {
  if (!op.Response?.Type) {
    return "";
  }

  const pagination = op.Extensions?.Pagination;
  const responseBodyType = op.Response.Type;

  // Only apply sseFlatResponse for event-stream responses
  if (responseBodyType.Type?.toString() === "event-stream") {
    const dataField = getFlattenedEventStreamField(responseBodyType.ItemType);
    if (dataField) {
      return ` -> ${sanitizePydanticType(typeDefToFieldDef(dataField.Type), {
        optional: isReturnTypeOptional(op.Response, pagination),
        nullable: dataField.Nullable,
        methodReturnType: true,
      })}`;
    }
  }

  return ` -> ${sanitizePydanticType(typeDefToFieldDef(op.Response.Type), {
    optional: isReturnTypeOptional(op.Response, pagination),
    nullable: false,
    methodReturnType: true,
  })}`;
}
registerTemplateFunc("templateMethodReturnType", templateMethodReturnType);

// @ts-ignore
function finalizeComments(
  lines: string[],
  indent: number,
  docString: boolean,
  breakLine: boolean,
): string {
  if (docString) {
    if (lines.length == 1) {
      return (
        (breakLine ? "\n" : "") + indentLines([`r"""${lines[0]}"""`], indent)
      );
    }

    lines[0] = `r"""${lines[0]}`;
    lines.push(`"""`);
  } else {
    lines = lines.map((line) => {
      return `# ${line}`;
    });
  }

  return (breakLine ? "\n" : "") + indentLines(lines, indent);
}

// @ts-ignore
function templateComments(
  comments: CommentDef | null,
  indent: number,
  docString: boolean = true,
): string {
  return templateCommentDef(comments, null, indent, docString);
}
registerTemplateFunc("templateComments", templateComments);

// @ts-ignore
function templateMethodComments(
  op: Operation,
  fieldsOverride?: FieldDef[],
  indent: number = 2,
): string {
  const fields =
    fieldsOverride && fieldsOverride.length > 0
      ? fieldsOverride
      : op.Arguments?.Sorted ?? [];
  return templateCommentDef(
    op.Comments,
    getMethodParamDescriptions(op, fields),
    indent,
    true,
    getHoistedSecurityRemark(op),
  );
}
registerTemplateFunc("templateMethodComments", templateMethodComments);

// @ts-ignore
function templateSDKCtorComments(sdk: SDK, isAsync: boolean = false): string {
  return templateCommentDef(
    {
      Summary:
        "Instantiates the SDK configuring it with the provided parameters.",
    } as CommentDef,
    getSDKParamDescriptions(sdk, isAsync),
    2,
    true,
    "",
  );
}
registerTemplateFunc("templateSDKCtorComments", templateSDKCtorComments);

// @ts-ignore
function templateHoistedSecurityRemark(
  fields: HoistedSecurityField[],
  required: boolean,
): string {
  const fieldList = fields.map((f) => `\`${sanitizeFieldName(f.Name)}\``);

  const remark = (items: string) =>
    required
      ? `This operation requires ${items} to be set on the \`security\` parameter when initializing the SDK.`
      : `If set, this operation will use ${items} from the global security.`;

  if (fieldList.length === 1) {
    return remark(fieldList[0]);
  }

  const last = fieldList.pop();
  if (fieldList.length === 1) {
    return remark(`either ${fieldList[0]} or ${last}`);
  }

  return remark(`one of ${fieldList.join(", ")}, or ${last}`);
}

function getSDKParamDescriptions(sdk: SDK, isAsync: boolean = false): string[] {
  let descriptions = [];

  if (sdk.Security) {
    if (
      sdk.Security.Type.Fields.length == 1 &&
      context.Global.Config.FlattenGlobalSecurity
    ) {
      const first = sdk.Security.Type.Fields[0];
      descriptions.push(
        `:param ${sanitizeFieldName(first.Name)}: The ${sanitizeFieldName(
          first.Name,
        )} required for authentication`,
      );
    } else {
      descriptions.push(
        `:param security: The security details required for authentication`,
      );
    }
  }

  if (sdk.Globals) {
    for (const global of sdk.Globals.Fields) {
      descriptions.push(
        `:param ${templateSDKInitFieldName(
          global.Name,
          "global",
        )}: Configures the ${sanitizeFieldName(
          global.Name,
        )} parameter for all supported operations`,
      );
    }
  }

  if (sdk.Servers) {
    for (const variable of sdk.Servers.GetVariables()) {
      descriptions.push(
        `:param ${templateSDKInitFieldName(
          variable.Name,
          "server",
        )}: Allows setting the ${variable.Name} variable for url substitution`,
      );
    }
    if (sdk.Servers.ServerMap) {
      descriptions.push(
        `:param server: The server by name to use for all methods`,
      );
    } else {
      descriptions.push(
        `:param server_idx: The index of the server to use for all methods`,
      );
    }
  }

  descriptions.push(`:param server_url: The server URL to use for all methods`);
  descriptions.push(
    `:param url_params: Parameters to optionally template the server URL with`,
  );
  // In split mode: sync SDK gets client, async SDK gets async_client
  // In both mode: SDK gets both client and async_client
  if (isSplitMode()) {
    if (isAsync) {
      descriptions.push(
        `:param async_client: The Async HTTP client to use for all asynchronous methods`,
      );
    } else {
      descriptions.push(
        `:param client: The HTTP client to use for all synchronous methods`,
      );
    }
  } else {
    // Both mode - include both parameters
    descriptions.push(
      `:param client: The HTTP client to use for all synchronous methods`,
    );
    if (includeAsyncProp()) {
      descriptions.push(
        `:param async_client: The Async HTTP client to use for all asynchronous methods`,
      );
    }
  }
  descriptions.push(
    `:param retry_config: The retry configuration to use for all supported methods`,
  );
  descriptions.push(
    `:param timeout_ms: Optional request timeout applied to each operation in milliseconds`,
  );

  return descriptions;
}
registerTemplateFunc("getSDKParamDescriptions", getSDKParamDescriptions);

function getMethodParamDescriptions(
  op: Operation,
  fields: FieldDef[],
): string[] {
  const descriptions = fields.map((field) => {
    let description = "";

    const fieldName = sanitizeParameterName(field.Name);
    if (fieldName === "request") {
      description = "The request object to send.";
    } else if (field.Comments && field.Comments.Description) {
      // Preserve multi-line formatting for parameter descriptions
      // with additional indent for subsequent lines

      const [firstLine, ...otherLines] = field.Comments.Description.split("\n");
      description = [
        sanitizeComments(firstLine.trim()),
        ...otherLines.map((l) => `    ${sanitizeComments(l.trim())}`),
      ].join("\n");
    }

    return `:param ${fieldName}: ${description}`;
  });

  if (op.Extensions?.Retries && !pythonMethodRequestExtrasEnabled()) {
    descriptions.push(
      `:param retries: Override the default retry configuration for this method`,
    );
  }

  if (pythonMethodRequestExtrasEnabled()) {
    descriptions.push(
      `:param extra_headers: Additional headers to set or replace on requests.`,
    );
    descriptions.push(
      `:param extra_query: Additional query parameters to append to requests.`,
    );
    if (pythonMethodExtraBodyEnabled(op)) {
      descriptions.push(
        `:param extra_body: Additional JSON object fields to merge into request bodies.`,
      );
    }
  } else {
    descriptions.push(
      `:param server_url: Override the default server URL for this method`,
    );
  }

  if (pythonMethodTimeoutArgumentName() === "timeout") {
    descriptions.push(
      `:param timeout: Override the default request timeout configuration for this method in ${pythonMethodTimeoutUnitsDescription()}`,
    );
  } else {
    descriptions.push(
      `:param timeout_ms: Override the default request timeout configuration for this method in ${pythonMethodTimeoutUnitsDescription()}`,
    );
  }

  if (op.GetAcceptTypes().length > 1 && !op.Extensions?.SSEOverload) {
    descriptions.push(
      `:param accept_header_override: Override the default accept header for this method`,
    );
  }

  if (!pythonMethodRequestExtrasEnabled()) {
    descriptions.push(
      `:param http_headers: Additional headers to set or replace on requests.`,
    );
  }

  return descriptions;
}
registerTemplateFunc("getMethodParamDescriptions", getMethodParamDescriptions);

// @ts-ignore
function templateCommentDef(
  comments: CommentDef | null,
  additionalLines: string[] | null,
  indent: number,
  docString: boolean,
  additionalNotes: string = "",
): string {
  if (!comments && !additionalLines && !additionalNotes) {
    return "";
  }

  let lines = [];

  if (comments?.Summary?.trim()) {
    lines.push(sanitizeComments(comments.Summary.trim()));
  }

  if (comments?.Description?.trim()) {
    if (lines.length > 0) {
      lines.push("");
    }

    lines.push(
      ...comments.Description.split("\n").map((l) =>
        sanitizeComments(l.trim()),
      ),
    );
  }

  if (comments?.ExternalDocs) {
    let externalDocs = comments.ExternalDocs.URL;
    let desc = comments?.ExternalDocs.Description?.trim();
    if (desc) {
      externalDocs += ` - ${sanitizeComments(desc)}`;
    }

    lines.push(...externalDocs.split("\n"));
  }

  if (additionalNotes) {
    if (lines.length > 0) {
      lines.push("");
    }

    lines.push(...additionalNotes.split("\n"));
  }

  if (additionalLines?.length > 0) {
    lines.push("");

    lines = lines.concat(additionalLines);
  }

  if (lines.length == 0) {
    return "";
  }

  return finalizeComments(lines, indent, docString, true);
}

// @ts-ignore
function getOptionalUsageMethodParameters(
  local: UsageContext,
  indent: number,
): string[] {
  const params = [];

  if (local.Scopes != undefined) {
    local.Scopes.forEach((scope) => {
      if (!scope.IsGlobal && scope.Feature != "") {
        switch (scope.Feature.toString()) {
          case "server_url": {
            const url = getUsageServerUrl(local, scope, {
              isTest: local.Test && true,
            });
            params.push(`server_url=${url}`);
            break;
          }
          case "content_type": {
            const enumClass = sanitizeAcceptEnumClass(local.Operation);
            addImport(
              getModuleImport(sanitizeFileName(local.SDK.Type.Name)),
              enumClass,
            );
            const acceptHeaderOption = `${enumClass}.${sanitizeAcceptEnumKey(
              scope.OpFilter,
            )}`;
            params.push(`accept_header_override=${acceptHeaderOption}`);
            break;
          }
        }
      }
    });
  }

  return params;
}

// @ts-ignore
function templateUsageLocalDeclarations(usageContext: UsageContext): string {
  const declarations = [];
  const operation = usageContext.Operation;

  const ctx: TemplateValueContext = {
    usageContext: usageContext,
    operation: usageContext.Operation,
    isTest: usageContext.Test && true,
    test: usageContext.Test,
  };

  for (const field of operation.Arguments.Sorted) {
    if (isSecurityClassField(field)) {
      continue;
    }

    const fieldExample = getOperationMethodFieldExample(usageContext, field);

    const isReferenceWithReplacements =
      fieldExample &&
      isExampleReferenceValue(fieldExample) &&
      fieldExample?.replacements.length > 0;

    if (!isReferenceWithReplacements) {
      continue;
    }

    let additionalCode = "";

    const fieldName = sanitizeFieldName(field.Name);

    for (const replacement of fieldExample.replacements) {
      const path = templateJSONPointerPath(field, fieldName, replacement.Path);

      const val = templateValue(
        path.target,
        getExampleValue(replacement.Value, { usageContext: usageContext }),
        false,
        ctx,
      );

      additionalCode += `\n${path.path} = ${val}`;
    }

    const value = templateValue(field, fieldExample, false, ctx);

    if (value === "") {
      continue;
    }

    declarations.push(`${fieldName} = ${value}${additionalCode}`);
  }

  if (!declarations.length) {
    return "";
  }

  return "\n" + declarations.join("\n\n") + "\n";
}
registerTemplateFunc(
  "templateUsageLocalDeclarations",
  templateUsageLocalDeclarations,
);

// @ts-ignore
function templateUsageMethodParameters(
  local: UsageContext,
  indent: number,
): string {
  const operation = local.Operation;
  const methodParams = [];

  const ctx: TemplateValueContext = {
    usageContext: local,
    operation: local.Operation,
    isTest: local.Test && true,
    test: local.Test,
  };

  for (const field of operation.Arguments.Sorted) {
    const declaration = sanitizeParameterName(field.Name);

    if (isSecurityClassField(field)) {
      let securityScope = undefined;

      for (const scope of local.Scopes) {
        if (scope.Feature == "security") {
          securityScope = scope;
          break;
        }
      }

      let securityExample = undefined;
      if (securityScope?.Value) {
        if ("example" in securityScope.Value) {
          securityExample = securityScope.Value.example;
        } else {
          securityExample = getExampleValue(securityScope.Value, ctx);
        }
      }

      if (securityExample === undefined && field.Optional) {
        continue;
      }

      const value = templateSecurityUsage(
        local.Operation.Security.Type,
        indent,
        true,
        false,
        undefined,
        securityExample,
        ctx,
      ).trimStart();

      if (value === "") {
        continue;
      }

      methodParams.push(`${declaration}=${value}`);
    } else {
      const fieldExample = getOperationMethodFieldExample(local, field);

      const isReferenceWithReplacements =
        fieldExample &&
        isExampleReferenceValue(fieldExample) &&
        fieldExample?.replacements.length > 0;

      if (isReferenceWithReplacements) {
        methodParams.push(`${declaration}=${sanitizeFieldName(field.Name)}`);
      } else {
        const value = templateValue(field, fieldExample, false, {
          ...ctx,
          shouldTemplateConstValue: shouldTemplateConstValue,
        });

        if (value === "") {
          continue;
        }
        methodParams.push(`${declaration}=${value}`);
      }
    }
  }

  methodParams.push(...getOptionalUsageMethodParameters(local, indent));

  return methodParams.join(", ");
}

registerTemplateFunc(
  "templateUsageMethodParameters",
  templateUsageMethodParameters,
);

function templateSerializer(fieldDef: FieldDef): string {
  const requestAnnotation = fieldDef.Annotations.Get(
    "request",
  ) as RequestAnnotation;

  if (
    !requestAnnotation ||
    !/^(application|text)\/([^+]+\+)*json.*/.test(requestAnnotation.MediaType)
  ) {
    return "";
  }

  const serializer = getSerializer(fieldDef.Type);
  return serializer ? `, ${serializer}` : "";
}
registerTemplateFunc("templateSerializer", templateSerializer);

// @ts-ignore
function templateDefaultValue(
  fieldDef: FieldDef,
  usageLocation: string,
  owningModelInfo: any,
): string {
  let defaultValue = "";

  if (fieldDef.Const || fieldDef.Default) {
    defaultValue = ` = ${templateConstOrDefaultValue(fieldDef, {
      usageLocation: usageLocation,
      owningModelInfo: owningModelInfo,
    })}`;
  } else if (
    fieldDef.IsAdditionalProperties ||
    (fieldDef.Optional && !fieldDef.Nullable) ||
    isFieldIgnored(fieldDef)
  ) {
    defaultValue = " = None";
  } else if (fieldDef.Optional && fieldDef.Nullable) {
    addImport("types", "UNSET", true);
    defaultValue = " = UNSET";
  }

  return defaultValue;
}
registerTemplateFunc("templateDefaultValue", templateDefaultValue);

function isFieldIgnored(field: FieldDef): boolean {
  if (!field.Annotations.Has("json")) {
    return false;
  }

  const jsonAnnotation = field.Annotations.Get("json") as JSONAnnotation;
  return jsonAnnotation.Ignore;
}

// @ts-ignore
function templateOAuth2Scopes(op: Operation): string {
  const scopes = getRequiredOAuth2Scopes(op);
  if (scopes === null) {
    return "None";
  }

  return `[${scopes.map((s) => `"${s}"`).join(",")}]`;
}
registerTemplateFunc("templateOAuth2Scopes", templateOAuth2Scopes);

function escapeTemplateDelimiters(value: string): string {
  return value.replaceAll("{{", `{{"{{"}}`);
}

// @ts-ignore
function templateOperationTags(op: Operation): string {
  const tags = op.Tags;
  if (!tags || tags.length === 0) {
    return "None";
  }

  return escapeTemplateDelimiters(
    `[${tags.map((t) => JSON.stringify(t)).join(", ")}]`,
  );
}
registerTemplateFunc("templateOperationTags", templateOperationTags);

function toPythonLiteral(value: unknown): string {
  if (value === null || value === undefined) {
    return "None";
  }
  if (typeof value === "boolean") {
    return value ? "True" : "False";
  }
  if (typeof value === "number" || typeof value === "string") {
    return JSON.stringify(value);
  }
  if (Array.isArray(value)) {
    return `[${value.map((v) => toPythonLiteral(v)).join(", ")}]`;
  }
  if (typeof value === "object") {
    const entries = Object.keys(value as Record<string, unknown>)
      .sort()
      .map(
        (k) =>
          `${JSON.stringify(k)}: ${toPythonLiteral(
            (value as Record<string, unknown>)[k],
          )}`,
      );
    return `{${entries.join(", ")}}`;
  }
  return "None";
}

// @ts-ignore
function templateOperationExtensions(op: Operation): string {
  const all = op.Extensions?.All;
  if (!all) {
    return "None";
  }

  const keys = Object.keys(all)
    .filter((k) => !k.startsWith("x-speakeasy-"))
    .sort();
  if (keys.length === 0) {
    return "None";
  }

  const entries = keys.map(
    (k) => `${JSON.stringify(k)}: ${toPythonLiteral(all[k])}`,
  );
  return escapeTemplateDelimiters(`{${entries.join(", ")}}`);
}
registerTemplateFunc(
  "templateOperationExtensions",
  templateOperationExtensions,
);

// @ts-ignore
function templateGlobalParamAccess(param: FieldDef): string {
  return `self.sdk_configuration.globals.${sanitizeFieldName(param.Name)}`;
}
registerTemplateFunc("templateGlobalParamAccess", templateGlobalParamAccess);

function templateFieldList(
  typeDef: TypeDef,
  predicate: (f: FieldDef) => boolean,
): string {
  return typeDef.Fields.filter((f) => predicate(f) && !f.IsAdditionalProperties)
    .map((f) => `"${f.Name.replaceAll(`"`, `\\"`)}"`)
    .join(", ");
}

function templateOptionalFieldList(typeDef: TypeDef): string {
  return templateFieldList(typeDef, (f) => f.Optional);
}
registerTemplateFunc("templateOptionalFieldList", templateOptionalFieldList);

function templateConstFieldAttributeList(typeDef: TypeDef): string {
  return typeDef.Fields.filter(
    (f) =>
      f.Const &&
      !(f.Optional && f.Const.Value === null) &&
      !f.IsAdditionalProperties,
  )
    .map((f) => {
      const fieldName = sanitizeFieldName(f.Name);
      return `"${useUpperCaseConsts() ? fieldName.toUpperCase() : fieldName}"`;
    })
    .join(", ");
}
registerTemplateFunc(
  "templateConstFieldAttributeList",
  templateConstFieldAttributeList,
);

function templateNullableFieldList(typeDef: TypeDef): string {
  return templateFieldList(typeDef, (f) => f.Nullable);
}
registerTemplateFunc("templateNullableFieldList", templateNullableFieldList);

function templateNullDefaultFieldList(typeDef: TypeDef): string {
  return templateFieldList(
    typeDef,
    (f) => f.Nullable && f.Default?.Value === null,
  );
}
registerTemplateFunc(
  "templateNullDefaultFieldList",
  templateNullDefaultFieldList,
);

function templateRequestBodyVariable(op: Operation): string {
  if (!op.Request || !op.Request.RequestBody) {
    return "";
  }

  let output = "request_body = ";

  if (op.Request.IsRequestBody) {
    return "";
  }

  output += `request.${sanitizeFieldName(op.Request.RequestBody.Name)}`;

  return output + "\n";
}
registerTemplateFunc(
  "templateRequestBodyVariable",
  templateRequestBodyVariable,
);

// @ts-ignore
function templateStream(
  fieldDef: FieldDef,
  filePath: string,
  example?: any,
  additionalContext?: TemplateValueContext,
): string {
  let stream = "";

  if (filePath) {
    if (additionalContext?.isTest) {
      filePath = `.speakeasy/testfiles/${filePath}`;
    }

    stream = `open("${filePath}", "rb")`;
  } else {
    addImport("io", "");
    if (typeof example !== "string") {
      example = JSON.stringify(example);
    }
    stream = `io.BytesIO(${templateByteValue(
      example,
      fieldDef,
      additionalContext,
    )})`;
  }

  return `${stream}${additionalContext?.isResponse ? ".read()" : ""}`;
}

// @ts-ignore
function templateFileToByteArrayValue(
  fieldDef: FieldDef,
  filePath: string,
  example: any,
  additionalContext?: TemplateValueContext,
): string {
  if (additionalContext?.isTest) {
    filePath = `.speakeasy/testfiles/${filePath}`;
  }

  return `open("${filePath}", "rb").read()`;
}

// @ts-ignore
function templateFileToStringValue(
  fieldDef: FieldDef,
  filePath: string,
  example: any,
  additionalContext?: TemplateValueContext,
): string {
  if (additionalContext?.isTest) {
    filePath = `.speakeasy/testfiles/${filePath}`;
  }

  return `open("${filePath}", "rb").read().decode("utf-8")`;
}

function templateSerializeRequestBody(op: Operation): string {
  addUtilsImport();

  let requestBodyVarName = "request_";
  if (!op.Request.IsRequestBody) {
    const bodyFieldName = sanitizeFieldName(op.Request.RequestBody.Name);
    if (!op.Request.IsRequestBodyRequired) {
      requestBodyVarName = `request_.${bodyFieldName} if request_ is not None else None`;
    } else {
      requestBodyVarName = `request_.${bodyFieldName}`;
    }
  }

  const extraBodyArg = pythonMethodRequestExtrasEnabled()
    ? ", extra_body=extra_body"
    : "";

  return `lambda: utils_.serialize_request_body(${requestBodyVarName}, ${templateBoolValue(
    op.Request.RequestBody.Nullable,
    op.Request.RequestBody,
  )}, ${templateBoolValue(
    op.Request.RequestBody.Optional,
    op.Request.RequestBody,
  )}, "${op.SerializationMethod}", ${sanitizePydanticType(
    op.Request.RequestBody,
  )}${extraBodyArg}`;
}
registerTemplateFunc(
  "templateSerializeRequestBody",
  templateSerializeRequestBody,
);

function templateInputParamDefault(op: Operation, field: FieldDef): string {
  if (field.Default) {
    return ` = ${templateConstOrDefaultValue(field, { usageLocation: "" })}`;
  } else if (
    field.Annotations?.Has("request") &&
    op.Request?.IsRequestBodyRequired &&
    field.Optional
  ) {
    // If the field represents the request body and it's marked as optional but
    // the request body is required in the OpenAPI spec then it is because
    // `methodArguments: infer-optional-args` was set and we _inferred_ that the
    // request body argument should be optional. The conditions to be met for
    // it to be an optional FieldDef are: 1) its AST type is "class" and 2) all
    // of its child fields were optional. When this happens we need to ensure
    // the default value is a blank instance of the request class.
    return ` = ${sanitizeType(field, { optional: false, nullable: false })}()`;
  } else if (field.Annotations?.Has("requestWrapper") && field.Optional) {
    // Similar to the above condition but now instead we're dealing with the
    // request wrapper type (the class that holds parameters and possibly a
    // request body).
    return ` = ${sanitizeType(field, { optional: false, nullable: false })}()`;
  } else if (field.Optional && field.Nullable) {
    addImport("types", "UNSET", true);
    return ` = UNSET`;
  } else if (field.Optional) {
    return ` = None`;
  }

  return "";
}
registerTemplateFunc("templateInputParamDefault", templateInputParamDefault);

function isArgumentOptional(op: Operation, field: FieldDef): boolean {
  if (field.Annotations?.Has("requestWrapper")) {
    return false; // because we'll set a nice default value instead
  } else if (
    field.Annotations?.Has("request") &&
    op.Request?.IsRequestBodyRequired &&
    field.Optional
  ) {
    return false; // again, because we'll set a nice default value instead
  } else {
    return field.Optional;
  }
}
registerTemplateFunc("isArgumentOptional", isArgumentOptional);

function positionalPathParamsMethodArgumentsEnabled(): boolean {
  return (
    context.Global.Config.MethodArguments === "positional-path-params" ||
    context.Global.Config.MethodArguments === "positional-path-with-extras"
  );
}

function pythonMethodRequestExtrasEnabled(): boolean {
  return (
    context.Global.Config.MethodArguments === "positional-path-with-extras"
  );
}
registerTemplateFunc(
  "pythonMethodRequestExtrasEnabled",
  pythonMethodRequestExtrasEnabled,
);

function pythonRawResponseHelpersEnabled(): boolean {
  return context.Global.Config.RawResponseHelpers === true;
}
registerTemplateFunc(
  "pythonRawResponseHelpersEnabled",
  pythonRawResponseHelpersEnabled,
);

function pythonMethodExtraBodyEnabled(op: Operation): boolean {
  return (
    pythonMethodRequestExtrasEnabled() &&
    !!op.Request?.RequestBody &&
    !!op.SerializationMethod
  );
}
registerTemplateFunc(
  "pythonMethodExtraBodyEnabled",
  pythonMethodExtraBodyEnabled,
);

function pythonMethodTimeoutAliasEnabled(): boolean {
  return context.Global.Config.MethodTimeoutArgument === "timeout";
}
registerTemplateFunc(
  "pythonMethodTimeoutAliasEnabled",
  pythonMethodTimeoutAliasEnabled,
);

function pythonMethodTimeoutSecondsEnabled(): boolean {
  return context.Global.Config.MethodTimeoutUnits === "seconds";
}
registerTemplateFunc(
  "pythonMethodTimeoutSecondsEnabled",
  pythonMethodTimeoutSecondsEnabled,
);

function pythonMethodTimeoutArgumentName(): "timeout_ms" | "timeout" {
  return context.Global.Config.MethodTimeoutArgument === "timeout"
    ? "timeout"
    : "timeout_ms";
}
registerTemplateFunc(
  "pythonMethodTimeoutArgumentName",
  pythonMethodTimeoutArgumentName,
);

function pythonMethodTimeoutUnitsDescription(): "seconds" | "milliseconds" {
  return pythonMethodTimeoutSecondsEnabled() ? "seconds" : "milliseconds";
}
registerTemplateFunc(
  "pythonMethodTimeoutUnitsDescription",
  pythonMethodTimeoutUnitsDescription,
);

function pythonHttpClientLibrary(): "httpx" | "httpx2" {
  // gen.yaml keys are Go-Pascal-cased, so httpClientLibrary arrives as
  // HTTPClientLibrary.
  return context.Global.Config.HTTPClientLibrary === "httpx2"
    ? "httpx2"
    : "httpx";
}
registerTemplateFunc("pythonHttpClientLibrary", pythonHttpClientLibrary);

// Aliasing the fork keeps every usage site and public annotation spelled
// httpx.* regardless of which library the SDK depends on.
function pythonHttpxImport(): string {
  return pythonHttpClientLibrary() === "httpx2"
    ? "import httpx2 as httpx"
    : "import httpx";
}
registerTemplateFunc("pythonHttpxImport", pythonHttpxImport);

function templatePaginationTimeoutArgumentValue(): string {
  if (pythonMethodTimeoutArgumentName() === "timeout") {
    return "timeout";
  }

  if (pythonMethodTimeoutSecondsEnabled()) {
    return "timeout_ms / 1000 if timeout_ms is not None else None";
  }

  return "timeout_ms";
}
registerTemplateFunc(
  "templatePaginationTimeoutArgumentValue",
  templatePaginationTimeoutArgumentValue,
);

function isPythonPositionalOperationArgument(field: FieldDef): boolean {
  if (!positionalPathParamsMethodArgumentsEnabled()) {
    return false;
  }

  const param = field.Annotations?.Get("param") as ParamAnnotation | null;
  if (!param) {
    return false;
  }

  return (
    param.ParamType?.toString() === "pathParam" &&
    param.RequiredForOperation === true &&
    field.Optional !== true &&
    param.Hidden !== true &&
    param.IsGlobal !== true &&
    param.HasGlobal !== true
  );
}

function getPythonMethodPositionalArgs(op: Operation): FieldDef[] {
  return (op.Arguments?.Sorted ?? []).filter((field) =>
    isPythonPositionalOperationArgument(field),
  );
}
registerTemplateFunc(
  "getPythonMethodPositionalArgs",
  getPythonMethodPositionalArgs,
);

function getPythonMethodKeywordOnlyArgs(op: Operation): FieldDef[] {
  return (op.Arguments?.Sorted ?? []).filter(
    (field) => !isPythonPositionalOperationArgument(field),
  );
}
registerTemplateFunc(
  "getPythonMethodKeywordOnlyArgs",
  getPythonMethodKeywordOnlyArgs,
);

function templateCommonMethodParams(
  op: Operation,
  pagination: any,
  canOverrideAcceptHeader: boolean,
): string {
  const lines: string[] = [];
  const requestExtrasEnabled = pythonMethodRequestExtrasEnabled();
  const timeoutArgumentName = pythonMethodTimeoutArgumentName();

  // Retries parameter
  if (op.Extensions?.Retries && !requestExtrasEnabled) {
    addImport("types", "OptionalNullable", true);
    addImport("types", "UNSET", true);
    lines.push("retries: OptionalNullable[utils_.RetryConfig] = UNSET");
  }

  // Standard parameters
  if (!requestExtrasEnabled) {
    lines.push("server_url: Optional[str] = None");
  }

  // URL override for pagination
  if (pagination && hasPaginationURL(op)) {
    lines.push("url_override: Optional[str] = None");
  }

  // Accept header override
  if (canOverrideAcceptHeader) {
    lines.push(
      `accept_header_override: Optional[${sanitizeAcceptEnumClass(op)}] = None`,
    );
  }

  const addTimeoutParam = () => {
    if (pythonMethodTimeoutSecondsEnabled()) {
      addImport("typing", "Union");
      addImport("httpx", "");
      lines.push(
        `${timeoutArgumentName}: Optional[Union[float, httpx.Timeout]] = None`,
      );
    } else {
      lines.push(`${timeoutArgumentName}: Optional[int] = None`);
    }
  };

  // HTTP headers
  addImport("typing", "Mapping");
  if (requestExtrasEnabled) {
    addImport("typing", "Any");
    lines.push("extra_headers: Optional[Mapping[str, str]] = None");
    lines.push("extra_query: Optional[Mapping[str, Any]] = None");
    if (pythonMethodExtraBodyEnabled(op)) {
      lines.push("extra_body: Optional[Mapping[str, Any]] = None");
    }
    addTimeoutParam();
  } else {
    addTimeoutParam();
    lines.push("http_headers: Optional[Mapping[str, str]] = None");
  }

  return lines.map((line) => `        ${line}`).join(",\n");
}
registerTemplateFunc("templateCommonMethodParams", templateCommonMethodParams);

// @ts-ignore
function templateGlobalType(field: FieldDef): string {
  return sanitizeType(field, { optional: false, nullable: false });
}
unregisterTemplateFunc("templateGlobalType");
registerTemplateFunc("templateGlobalType", templateGlobalType);

function templateSimpleType(
  typeDef: TypeDef,
  usageLocation: string = "",
): string {
  return sanitizeType(
    typeDefToFieldDef(typeDef),
    { optional: false, nullable: false },
    usageLocation,
  );
}
registerTemplateFunc("templateSimpleType", templateSimpleType);

function templatePydanticClassType(fieldDef: FieldDef): string {
  return sanitizePydanticType(fieldDef, {
    optional: false,
    nullable: false,
  });
}
registerTemplateFunc("templatePydanticClassType", templatePydanticClassType);

// @ts-ignore
function templateHTTPClient(value: HTTPClientUsage): string {
  switch (value.type) {
    case "test":
      return `test_http_client`;
    default:
      throw new Error(`unsupported http client type: ${value.type}`);
  }
}

function templateSDKVarName(): string {
  const varName = sanitizeFieldName(context.Global.AST.MainSDK.Type.Name);

  return varName !== getModuleImport()
    ? varName
    : varName
        .split("_")
        .map((word) => word[0])
        .join("") + "_client";
}
registerTemplateFunc("templateSDKVarName", templateSDKVarName);

function asyncOnly(strings: TemplateStringsArray): string {
  assert(strings.length === 1, "asyncOnly must be used with a single string");
  return _asyncOnly(strings[0]);
}

function _asyncOnly(x: string): string {
  return "__ASYNC_ONLY__" + x + "__ASYNC_ONLY__";
}
registerTemplateFunc("asyncOnly", _asyncOnly);

function asyncOnlyPagination(str: string): string {
  if (!isAsyncPaginationSep2025Fix()) {
    return "";
  }

  return _asyncOnly(str);
}
registerTemplateFunc("asyncOnlyPagination", asyncOnlyPagination);

function asyncOnlyInBothMode(str: string): string {
  if (context.Global.Config.AsyncMode !== "both") {
    return "";
  }
  return _asyncOnly(str);
}
registerTemplateFunc("asyncOnlyInBothMode", asyncOnlyInBothMode);

function asyncOnlyInBothModePagination(str: string): string {
  if (!isAsyncPaginationSep2025Fix()) {
    return "";
  }
  return asyncOnlyInBothMode(str);
}
registerTemplateFunc(
  "asyncOnlyInBothModePagination",
  asyncOnlyInBothModePagination,
);

function syncOnly(str: string): string {
  return "__SYNC_ONLY__" + str + "__SYNC_ONLY__";
}
registerTemplateFunc("syncOnly", syncOnly);

function asyncSelect(async: string, sync: string): string {
  return `${_asyncOnly(async)}${syncOnly(sync)}`;
}
registerTemplateFunc("asyncSelect", asyncSelect);

function templateRaiseParseError(): string {
  const call = asyncSelect(
    "await response_helpers.raise_parse_error_async(self.sdk_configuration.__dict__.get('_async_hooks'), self.sdk_configuration.__dict__.get('_hooks'),",
    "response_helpers.raise_parse_error(self.sdk_configuration.__dict__['_hooks'],",
  );
  return `
${call}
    AfterParseErrorContext(_speakeasy_hook_ctx),
    http_res_,
    parse_exc_,
)`;
}
registerTemplateFunc("templateRaiseParseError", templateRaiseParseError);

function asyncSelectPagination(async: string, sync: string): string {
  if (!isAsyncPaginationSep2025Fix()) {
    return sync;
  }

  return asyncSelect(async, sync);
}
registerTemplateFunc("asyncSelectPagination", asyncSelectPagination);

function templateSecurityEnvVarValue(fieldDef: FieldDef): string {
  const envVar = `os.getenv("${templateSecurityEnvVars(fieldDef)}")`;

  if (fieldDef.Type.Type.toString() === "array") {
    return `(${envVar} or "").split(",")`;
  }

  return envVar;
}
registerTemplateFunc(
  "templateSecurityEnvVarValue",
  templateSecurityEnvVarValue,
);

// @ts-ignore
function useAsyncHooks(): boolean {
  return context.Global.Config.UseAsyncHooks === true;
}
registerTemplateFunc("useAsyncHooks", useAsyncHooks);
