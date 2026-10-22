// @ts-ignore
const reservedGoKeywords = [
  "break",
  "default",
  "func",
  "interface",
  "select",
  "case",
  "defer",
  "go",
  "map",
  "struct",
  "chan",
  "else",
  "goto",
  "package",
  "switch",
  "const",
  "fallthrough",
  "if",
  "range",
  "type",
  "continue",
  "for",
  "import",
  "return",
  "var",
  "strings",
  "string",
  "bytes",
  "bool",
  "uint",
  "uint8",
  "uint32",
  "uint64",
  "int",
  "int8",
  "int32",
  "int64",
  "float",
  "float32",
  "float64",
  "complex64",
  "complex128",
  "decimal",
  "any",
  "context",
  "error",
];

// @ts-ignore
function getSDKReservedKeywords() {
  // NOTE: Shared class-name reservations should also be reflected in the
  // terraform template goSDKReservedKeywords. Go SDK-only entries (e.g.
  // baseURL) can be skipped when they don't apply to terraform.
  if (typeof getAccessNamespace === "undefined") {
    // getAccessNamespace may not be available during early pipeline stages
    // (e.g. model file naming). Fall back to hardcoded defaults.
    return [
      "shared",
      "operations",
      "utils",
      "url",
      "options",
      "timeout",
      "s",
      // paginationCtx is a Go SDK-only local captured by generated Next closures.
      "paginationCtx",
      // streamCancel is a Go SDK-only local holding the timeout cancel handed
      // to returned streams in streaming methods.
      "streamCancel",
      // baseURL is Go SDK-only; the terraform provider does not use an
      // internal baseURL variable so it does not need this reservation.
      "baseURL",
    ];
  }
  return [
    getAccessNamespace("usage", "shared"),
    getAccessNamespace("usage", "operations"),
    "utils",
    "url",
    "option",
    "options",
    "timeout",
    "s",
    // paginationCtx is a Go SDK-only local captured by generated Next closures.
    "paginationCtx",
    // streamCancel is a Go SDK-only local holding the timeout cancel handed
    // to returned streams in streaming methods.
    "streamCancel",
    // Go SDK-only: prevents collision with the internal baseURL variable
    // used for server URL resolution. Not needed in terraform.
    "baseURL",
  ];
}

// @ts-ignore
function sanitizeSDKAccess(sdk: SDK): string {
  const group = sdk.Group;

  if (!group) {
    return "";
  }
  let formattedName = "";
  if (group.includes(".") && ".".repeat(group.length) !== group) {
    const pieces = group.split(".");
    formattedName = caser().ToGoPascal(pieces[0]);
    for (const i of pieces.slice(1)) {
      formattedName += "." + caser().ToGoPascal(i);
    }
  } else {
    formattedName = sanitizeFieldName(group);
  }

  if (formattedName != "") {
    formattedName = `.${formattedName}`;
  }

  return formattedName;
}

registerTemplateFunc("sanitizeSDKAccess", sanitizeSDKAccess);

// @ts-ignore
function sanitizeFieldName(name: string): string {
  name = sanitizeName(name);

  return caser().ToGoPascal(name);
}
registerTemplateFunc("sanitizeFieldName", sanitizeFieldName);

// @ts-ignore
function sanitizeSDKFieldName(name: string): string {
  const regex = /^[a-zA-Z]+(\.[a-zA-Z]+)+$/;
  if (regex.test(name)) {
    const pieces = name.split(".");
    name = pieces[pieces.length - 1];
  }

  name = sanitizeName(name);

  return caser().ToGoPascal(name);
}
registerTemplateFunc("sanitizeSDKFieldName", sanitizeSDKFieldName);

// Go-specific collision overrides for reserved method names where a better
// replacement exists than the generic `<name>Value` rule, used when
// idiomaticMethodCollisionNames is enabled.
// @ts-ignore
const goMethodCollisionTable: Record<string, string> = {
  Error: "ErrorInfo",
};

// @ts-ignore
function goErrorStructReservedMethods(typeDef?: TypeDef): string[] {
  // The extended union set is gated so that flag-off output stays byte
  // identical to the legacy behavior, which only resolved Error.
  if (
    context.Global.Config.IdiomaticMethodCollisionNames &&
    typeDef &&
    typeDef.Type.toString() === "union"
  ) {
    return ["Error", "MarshalJSON", "UnmarshalJSON"];
  }
  return ["Error"];
}

// @ts-ignore
function goStructSiblingNames(typeDef?: TypeDef): string[] {
  if (!typeDef) {
    return [];
  }

  const names: string[] = [];
  for (const field of typeDef.Fields ?? []) {
    names.push(sanitizeFieldName(field.Name));
  }
  if (typeDef.Type.toString() === "union") {
    for (const assocType of typeDef.AssociatedTypes ?? []) {
      names.push(sanitizeFieldName(sanitizeUnionTypeName(assocType)));
    }
    for (const mapping of typeDef.Discriminator?.Mapping ?? []) {
      names.push(sanitizeFieldName(sanitizeUnionTypeName(mapping.Type)));
    }
  }

  return names;
}

// @ts-ignore
function resolveMethodCollision(
  name: string,
  reservedMethods: string[],
  typeDef?: TypeDef,
): string {
  if (!reservedMethods.includes(name)) {
    return name;
  }

  const idiomatic = context.Global.Config.IdiomaticMethodCollisionNames
    ? goMethodCollisionTable[name] ?? `${name}Value`
    : undefined;

  if (idiomatic && !goStructSiblingNames(typeDef).includes(idiomatic)) {
    return idiomatic;
  }

  return `${name}_`;
}

// @ts-ignore
function sanitizeErrorFieldName(name: string, typeDef?: TypeDef): string {
  name = sanitizeName(name);

  name = caser().ToGoPascal(name);

  return resolveMethodCollision(
    name,
    goErrorStructReservedMethods(typeDef),
    typeDef,
  );
}

registerTemplateFunc("sanitizeErrorFieldName", sanitizeErrorFieldName);

// error-scope fields use sanitizeErrorFieldName; all others use sanitizeFieldName.
function sanitizeScopedFieldName(name: string, typeDef: TypeDef): string {
  return typeDef.Scope.toString() === "errors"
    ? sanitizeErrorFieldName(name, typeDef)
    : sanitizeFieldName(name);
}
registerTemplateFunc("sanitizeScopedFieldName", sanitizeScopedFieldName);

// True when any SDK operation emits an event-stream response, which causes the
// generator to import pkg/types/stream — a method parameter named "stream"
// would shadow that package at file scope.
let _isStreamPackageImported: boolean | undefined;

// @ts-ignore
function isStreamPackageImported(): boolean {
  if (_isStreamPackageImported !== undefined) {
    return _isStreamPackageImported;
  }
  const sdkEmitsEventStream = (sdk: SDK): boolean =>
    sdk.Operations.some(
      (op) =>
        op.Response?.Responses?.some(
          (sub) =>
            sub.Content?.some((c) => c.SerializationMethod === "eventstream"),
        ),
    ) || (sdk.SubSDKs ?? []).some(sdkEmitsEventStream);
  _isStreamPackageImported = sdkEmitsEventStream(context.Global.AST.MainSDK!);
  return _isStreamPackageImported;
}

// @ts-ignore
function sanitizePrivateFieldName(name: string): string {
  name = sanitizeName(name);

  name = caser().ToGoCamel(name);

  if (
    reservedGoKeywords.includes(name) ||
    getSDKReservedKeywords().includes(name) ||
    (name === "stream" && isStreamPackageImported())
  ) {
    name = `${name}_`;
  }

  return name;
}

registerTemplateFunc("sanitizePrivateFieldName", sanitizePrivateFieldName);

// @ts-ignore
function sanitizeVariableName(name: string): string {
  name = sanitizeName(name);

  name = caser().ToGoCamel(name);

  if (
    reservedGoKeywords.includes(name) ||
    getSDKReservedKeywords().includes(name)
  ) {
    name = `${name}Var`;
  }

  return name;
}

registerTemplateFunc("sanitizeVariableName", sanitizeVariableName);

// @ts-ignore
function sanitizeClassName(name: string): string {
  name = sanitizeName(name);

  if (getSDKReservedKeywords().includes(name.toLowerCase())) {
    name = `${name}Obj`;
  }

  return caser().ToGoPascal(name);
}

registerTemplateFunc("sanitizeClassName", sanitizeClassName);

// @ts-ignore
function sanitizePrivateClassName(name: string): string {
  name = sanitizeName(name);
  name = caser().ToGoCamel(name);

  if (
    reservedGoKeywords.includes(name) ||
    getSDKReservedKeywords().includes(name.toLowerCase())
  ) {
    name = `${name}T`;
  }

  return name;
}

registerTemplateFunc("sanitizePrivateClassName", sanitizePrivateClassName);

// @ts-ignore
function sanitizeMethodName(operation: Operation): string {
  return caser().ToGoPascal(sanitizeName(operation.GetID()));
}
registerTemplateFunc("sanitizeMethodName", sanitizeMethodName);

// @ts-ignore
registerTemplateFunc("sanitizeName", sanitizeName);

// @ts-ignore
function sanitizePrivateMethodName(name: string): string {
  return caser().ToGoCamel(sanitizeName(name));
}
registerTemplateFunc("sanitizePrivateMethodName", sanitizePrivateMethodName);

// @ts-ignore
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
  name = sanitizeFile(name, "");

  name = name.toLowerCase();

  if (getSDKReservedKeywords().includes(name)) {
    name = `${name}_`;
  }

  name = truncateName(name, 250);

  return name;
}

registerTemplateFunc("sanitizeFileName", sanitizeFileName);

// @ts-ignore
function sanitizeOutputLocation(outputLocation: string): string {
  if (!outputLocation) return outputLocation;
  return outputLocation
    .split("/")
    .map((segment) => sanitizeFile(segment, "").toLowerCase())
    .join("/");
}

// @ts-ignore
function getPointerTypes(): string[] {
  return [
    "bytes",
    "response-stream",
    "map",
    "any",
    "array",
    "set",
    "bigint",
    "decimal",
    "event-stream",
    "jsonl",
    "request",
    "response",
  ];
}

function handleAsPointer(fieldDef: FieldDef): boolean {
  return (
    (fieldDef.Optional || fieldDef.Nullable) && !isPointerType(fieldDef.Type)
  );
}

// @ts-ignore
function sanitizeFieldType(fieldDef: FieldDef, usageLocation: string): string {
  const nullableAndOptional =
    fieldDef.Nullable &&
    fieldDef.Optional &&
    context.Global.Config.NullableOptionalWrapper;

  const nullableOrOptional = fieldDef.Optional || fieldDef.Nullable;

  const fieldType = sanitizeType(
    fieldDef.Type,
    nullableAndOptional ? false : nullableOrOptional,
    usageLocation,
  );

  if (nullableAndOptional) {
    addImport("optionalnullable");
    return `optionalnullable.OptionalNullable[${fieldType}]`;
  }

  return fieldType;
}
registerTemplateFunc("sanitizeFieldType", sanitizeFieldType);

// @ts-ignore
function sanitizeType(
  typeDef: TypeDef,
  optional?: boolean,
  usageLocation: string = "",
  addImports = true,
): string {
  let type = (() => {
    switch (typeDef.Type.toString()) {
      case "union":
      case "enum":
      case "error":
      case "class":
        return sanitizeClass(typeDef, usageLocation, false, addImports);
      case "string":
        return "string";
      case "date":
        addImports && addImport("types");
        return "types.Date";
      case "date-time":
        addImports && addImport("time");
        return "time.Time";
      case "integer":
        return "int64";
      case "int32":
        return "int";
      case "bigint":
        addImports && addImport("math/big");
        return "*big.Int"; // Always has to be a pointer due to big.Int pointer receiver on the MarshalJSON method which does mean we lose requiredness for this type
      case "number":
        return "float64";
      case "float32":
        return "float32";
      case "decimal":
        addImports && addImport("github.com/ericlagergren/decimal");
        return "*decimal.Big"; // Always has to be a pointer due to decimal.Big pointer receiver on the MarshalText method which does mean we lose requiredness for this type
      case "boolean":
        return "bool";
      case "bytes":
        return "[]byte";
      case "request-stream":
        // These fields were always bytes ([]byte) previously. Prevent a
        // backwards incompatible change with io.Reader or io.ReadCloser by
        // instead implementing the empty interface.
        return "any";
      case "response-stream":
        addImports && addImport("io");
        return "io.ReadCloser";
      case "map":
        return sanitizeMap(typeDef, usageLocation, addImports);
      case "set":
      case "array":
        return (
          "[]" +
          sanitizeType(
            typeDef.ItemType,
            typeDef.ContainsNull,
            usageLocation,
            addImports,
          )
        );
      case "any":
        return "any";
      case "request":
        addImports && addImport("net/http");
        return "*http.Request"; // Always has to be a pointer due to http.Request being returned as a pointer
      case "response":
        addImports && addImport("net/http");
        return "*http.Response"; // Always has to be a pointer due to http.Response being returned as a pointer
      case "event-stream":
        addImports && addImport("stream");
        return `*stream.EventStream[${sanitizeType(
          typeDef.ItemType,
          false,
          usageLocation,
          addImports,
        )}]`;
      case "jsonl":
        addImports && addImport("jsonl");
        return `*jsonl.JsonLStream[${sanitizeType(
          typeDef.ItemType,
          false,
          usageLocation,
          addImports,
        )}]`;
      default:
        throw new Error(`Unknown type: ${typeDef.Type.toString()}`);
    }
  })();
  return templateOptional(typeDef, optional) + type;
}

registerTemplateFunc("sanitizeType", sanitizeType);

// @ts-ignore
function sanitizeClass(
  typeDef: TypeDef,
  usageLocation: string,
  definition: boolean,
  addImports = true,
) {
  let prefix = "";
  if (
    !definition &&
    typeDef.OutputLocation != usageLocation &&
    typeDef.OutputLocation != "tests" &&
    !typeDef.Truncated
  ) {
    if (addImports) {
      // @ts-ignore - addModelImport is only available in the Go SDK template context
      if (typeof addModelImport !== "undefined") {
        // @ts-ignore
        addModelImport(typeDef.OutputLocation);
      } else {
        addImport(typeDef.OutputLocation, true);
      }
    }
    // @ts-ignore - sanitizeModelPackageName signature varies across templates
    prefix = sanitizeModelPackageName(typeDef.OutputLocation, true) + ".";
  }

  return prefix + sanitizeClassName(typeDef.Name);
}

registerTemplateFunc("sanitizeClass", sanitizeClass);

// @ts-ignore
function templateStatusCode(statusCode: string): string {
  if (statusCode.toUpperCase().includes("X")) {
    let codeRange = parseInt(statusCode[0], 10);

    return `httpRes.StatusCode >= ${codeRange}00 && httpRes.StatusCode < ${
      codeRange + 1
    }00`;
  }

  return `httpRes.StatusCode == ${statusCode}`;
}

registerTemplateFunc("templateStatusCode", templateStatusCode);

// @ts-ignore
function templateConstName(name: string, prefix: string): string {
  name = sanitizeName(name);

  return caser().ToGoPascal(prefix + "_" + name);
}

registerTemplateFunc("templateConstName", templateConstName);

// @ts-ignore
function sanitizeUnionTypeName(typeDef: TypeDef): string {
  if (typeDef.Name) {
    return typeDef.Name;
  }

  switch (typeDef.Type.toString()) {
    case "string":
      return "str";
    case "map":
    case "array":
    case "set":
      return (
        typeDef.Type.toString() +
        "Of" +
        caser().ToGoPascal(sanitizeUnionTypeName(typeDef.ItemType))
      );
    default:
      return typeDef.Type.toString();
  }
}

registerTemplateFunc("sanitizeUnionTypeName", sanitizeUnionTypeName);

// @ts-ignore
function sanitizeUnionCast(typeDef: TypeDef, outputLocation: string): string {
  switch (typeDef.Type.toString()) {
    case "string":
      return "string";
    case "enum":
      return sanitizeClass(typeDef, outputLocation, false);
    default:
      throw new Error(`Unsupported type: ${typeDef.Type.toString()}`);
  }
}

registerTemplateFunc("sanitizeUnionCast", sanitizeUnionCast);

// @ts-ignore
function sanitizeMap(typeDef: TypeDef, outputLocation: string, addImports) {
  return `map[string]${sanitizeType(
    typeDef.ItemType,
    typeDef.ContainsNull,
    outputLocation,
    addImports,
  )}`;
}

// @ts-ignore
function templateOptional(typeDef: TypeDef, optional: boolean) {
  if (!optional || isPointerType(typeDef)) {
    return "";
  }

  return "*";
}
// @ts-ignore
function templateFieldDeclaration(
  field: FieldDef,
  parent?: TypeDef,
  additionalContext?: TemplateValueContext,
): string {
  return `${sanitizeFieldName(field.Name)}: `;
}

// @ts-ignore
function templateType(
  typeDef: TypeDef,
  additionalContext?: TemplateValueContext,
): string {
  return sanitizeType(typeDef, false, "usage");
}

// @ts-ignore
function templateBracket(
  typeDef: TypeDef,
  openingBracket: boolean,
  additionalContext?: TemplateValueContext,
): string {
  if (openingBracket) {
    return "{";
  }

  return "}";
}

// @ts-ignore
function templateOptionalSymbol(): string {
  return "&";
}

// @ts-ignore
function templateIndent(indent: number): string {
  return "    ".repeat(indent);
}

// @ts-ignore
function templateNullValue(fieldDef: FieldDef): string {
  return "nil";
}

// @ts-ignore
function templateFieldDelimiter(): string {
  return ",";
}

// @ts-ignore
function templateConstValue(
  fieldDef: FieldDef,
  usageLocation: string,
  skipPointerWrapping = false,
): string {
  if (fieldDef.Const.Value === null) {
    return "nil";
  }

  switch (fieldDef.Type.Type.toString()) {
    case "enum":
      return templateEnumValue(
        fieldDef,
        fieldDef.Type.Enum.Values.indexOf(fieldDef.Const?.Value.toString()),
        {
          skipPointerWrapping,
          outputLocation:
            fieldDef.Type.OutputLocation != usageLocation
              ? fieldDef.Type.OutputLocation
              : "inline",
        },
      );
    case "string":
      return templateStringValue(fieldDef.Const?.Value, fieldDef, {
        useTypesPackage: true,
        skipPointerWrapping,
      });
    case "date":
      return templateDateValue(fieldDef.Const?.Value, fieldDef, {
        skipPointerWrapping,
      });
    case "date-time":
      return templateDateTimeValue(fieldDef.Const?.Value, fieldDef, {
        skipPointerWrapping,
      });
    case "bigint":
    case "int32":
    case "integer":
      return templateIntValue(fieldDef.Const?.Value, fieldDef, {
        useTypesPackage: true,
        skipPointerWrapping,
      });
    case "number":
    case "float32":
    case "decimal":
      return templateFloatValue(fieldDef.Const?.Value, fieldDef, {
        useTypesPackage: true,
        skipPointerWrapping,
      });
    case "boolean":
      return templateBoolValue(fieldDef.Const?.Value, fieldDef, {
        useTypesPackage: true,
        skipPointerWrapping,
      });
    case "array":
    case "set":
    case "map":
      return templateConstContainerValue(fieldDef, usageLocation);
    case "bytes":
    default:
      throw new Error(
        `unsupported const type: ${fieldDef.Type.Type.toString()}`,
      );
  }
}

registerTemplateFunc("templateConstValue", templateConstValue);

function templateConstContainerValue(
  fieldDef: FieldDef,
  usageLocation: string,
): string {
  const containerType = sanitizeType(fieldDef.Type, false, usageLocation);
  const renderItem = (value: any): string =>
    templateConstValue(
      {
        ...fieldDef,
        Type: fieldDef.Type.ItemType,
        Optional: false,
        Nullable: false,
        Const: { Value: value },
      } as FieldDef,
      usageLocation,
      true,
    );

  if (fieldDef.Type.Type.toString() === "map") {
    const entries = Object.entries(fieldDef.Const.Value ?? {}).map(
      ([key, value]) => `${JSON.stringify(key)}: ${renderItem(value)}`,
    );
    return `${containerType}{${entries.join(", ")}}`;
  }

  const items = ((fieldDef.Const.Value ?? []) as any[]).map(renderItem);
  return `${containerType}{${items.join(", ")}}`;
}

// @ts-ignore
function templateArrayValue(value: any): string {
  return value;
}

// @ts-ignore
function templateMapValue(key: string, value: any): string {
  return `"${key}": ${value}`;
}

/**
 * Converts an unquoted regex string into a backticked string with escaping.
 */
function templateRegexString(regex: string): string {
  const pattern = regex.replaceAll("`", '` + "`" + `');

  return `\`${pattern}\``;
}

// @ts-ignore
function templateStringValue(
  value: string,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  const needsPointer =
    handleAsPointer(fieldDef) && !additionalContext?.skipPointerWrapping;
  if (additionalContext?.useTypesPackage) {
    if (needsPointer) {
      // @ts-ignore - addImport signature varies across templates
      addImport(getTypesLocation(), true);
    }
  } else {
    if (needsPointer) {
      // @ts-ignore - addSDKPackageImport signature varies across templates
      addSDKPackageImport(true);
    }
  }
  if (value.includes("GetEnv(")) {
    return needsPointer
      ? `${
          additionalContext?.useTypesPackage
            ? "types"
            : sanitizeSDKPackageName(true)
        }.Pointer(${value})`
      : `${value}`;
  }

  value = quote(value).replaceAll("{{", `{{"{{"}}`);

  return needsPointer
    ? `${
        additionalContext?.useTypesPackage
          ? "types"
          : sanitizeSDKPackageName(true)
      }.Pointer(${value})`
    : `${value}`;
}

// @ts-ignore
function templateEnumValue(
  fieldDef: FieldDef,
  idx: number,
  additionalContext?: TemplateValueContext,
): string {
  let enumNames = getEnumNames(fieldDef.Type);

  let enumValue = enumNames[idx];

  if (additionalContext?.outputLocation != "inline") {
    // @ts-ignore - addModelImport is only available in the Go SDK template context
    if (typeof addModelImport !== "undefined") {
      // @ts-ignore
      addModelImport(fieldDef.Type.OutputLocation);
    } else {
      addImport(fieldDef.Type.OutputLocation, true);
    }
    enumValue = `${getAccessNamespace(
      "usage",
      fieldDef.Type.Scope.toString(),
      fieldDef.Type.OutputLocation,
    )}${enumValue}`;
  }

  // When using OptionalNullable wrapper, don't add .ToPointer() because
  // OptionalNullable.From() expects a pointer and templateValueWrapper() will handle it
  const isOptionalNullable =
    fieldDef.Nullable &&
    fieldDef.Optional &&
    context.Global.Config.NullableOptionalWrapper;

  if (handleAsPointer(fieldDef) && !isOptionalNullable) {
    enumValue += `.ToPointer()`;
  }

  return enumValue;
}

// @ts-ignore
function templateBoolValue(
  value: boolean,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  const strValue = value ? "true" : "false";
  const needsPointer =
    handleAsPointer(fieldDef) && !additionalContext?.skipPointerWrapping;

  if (needsPointer) {
    if (additionalContext?.useTypesPackage) {
      // @ts-ignore - addImport signature varies across templates
      addImport(getTypesLocation(), true);
    } else {
      // @ts-ignore - addSDKPackageImport signature varies across templates
      addSDKPackageImport(true);
    }
  }

  return needsPointer
    ? `${
        additionalContext?.useTypesPackage
          ? "types"
          : sanitizeSDKPackageName(true)
      }.Pointer(${strValue})`
    : strValue;
}

// @ts-ignore
function templateByteValue(
  value: string,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  return `[]byte("${value.replace(/"/g, '\\"')}")`;
}

// @ts-ignore
function templateDateValue(
  value: string,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  addImport(getTypesLocation(), true);
  return handleAsPointer(fieldDef) && !additionalContext?.skipPointerWrapping
    ? `types.MustNewDateFromString("${value}")`
    : `types.MustDateFromString("${value}")`;
}

// @ts-ignore
function templateDateTimeValue(
  value: string,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  addImport(getTypesLocation(), true);
  return handleAsPointer(fieldDef) && !additionalContext?.skipPointerWrapping
    ? `types.MustNewTimeFromString("${value}")`
    : `types.MustTimeFromString("${value}")`;
}

// @ts-ignore
function templateIntValue(
  value: any,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  const needsPointer =
    handleAsPointer(fieldDef) && !additionalContext?.skipPointerWrapping;
  let t = fieldDef.Type.Type.toString();
  if (t === "any") {
    // Reference: internal issue reference
    if (additionalContext?.isTest) {
      return `float64(${value})`;
    }

    t = "integer";
  }

  switch (t) {
    case "int32":
      if (needsPointer) {
        if (additionalContext?.useTypesPackage) {
          // @ts-ignore - addImport signature varies across templates
          addImport(getTypesLocation(), true);
        } else {
          // @ts-ignore - addSDKPackageImport signature varies across templates
          addSDKPackageImport(true);
        }
      }
      return needsPointer
        ? `${
            additionalContext?.useTypesPackage
              ? "types"
              : sanitizeSDKPackageName(true)
          }.Pointer[int](${value})`
        : `${value}`;
    case "integer":
      if (needsPointer) {
        if (additionalContext?.useTypesPackage) {
          // @ts-ignore - addImport signature varies across templates
          addImport(getTypesLocation(), true);
        } else {
          // @ts-ignore - addSDKPackageImport signature varies across templates
          addSDKPackageImport(true);
        }
      }
      return needsPointer
        ? `${
            additionalContext?.useTypesPackage
              ? "types"
              : sanitizeSDKPackageName(true)
          }.Pointer[int64](${value})`
        : `${value}`;
    case "bigint":
      const isNumber = typeof value === "number";
      if (isNumber) {
        value = value.toString();
      }

      if (!isNumber || value.length > `999999999999999999`.length) {
        addImport(getTypesLocation(), true);
        return `types.MustNewBigIntFromString("${value}")`;
      } else {
        addImport("math/big");
        return `big.NewInt(${value})`;
      }
  }
}

function normalizeGoFloatLiteral(value: any): string {
  const num = typeof value === "number" ? value : parseFloat(String(value));
  if (isNaN(num)) {
    return "0.0";
  }
  let str = String(num);
  if (!str.includes(".") && !str.includes("e") && !str.includes("E")) {
    str += ".0";
  }
  if (str.startsWith(".")) {
    str = "0" + str;
  }
  if (str.startsWith("-.")) {
    str = "-0" + str.substring(1);
  }
  return str;
}

// @ts-ignore
function templateFloatValue(
  value: number,
  fieldDef: FieldDef,
  additionalContext?: TemplateValueContext,
): string {
  const needsPointer =
    handleAsPointer(fieldDef) && !additionalContext?.skipPointerWrapping;
  let t = fieldDef.Type.Type.toString();
  if (t === "any") {
    // Reference: internal issue reference
    if (additionalContext?.isTest) {
      return `float64(${normalizeGoFloatLiteral(value)})`;
    }

    t = "number";
  }

  const normalized = normalizeGoFloatLiteral(value);

  switch (t) {
    case "float32":
      if (needsPointer) {
        if (additionalContext?.useTypesPackage) {
          // @ts-ignore - addImport signature varies across templates
          addImport(getTypesLocation(), true);
        } else {
          // @ts-ignore - addSDKPackageImport signature varies across templates
          addSDKPackageImport(true);
        }
      }
      return needsPointer
        ? `${
            additionalContext?.useTypesPackage
              ? "types"
              : sanitizeSDKPackageName(true)
          }.Pointer[float32](${normalized})`
        : `${normalized}`;
    case "number":
      if (needsPointer) {
        if (additionalContext?.useTypesPackage) {
          // @ts-ignore - addImport signature varies across templates
          addImport(getTypesLocation(), true);
        } else {
          // @ts-ignore - addSDKPackageImport signature varies across templates
          addSDKPackageImport(true);
        }
      }
      return needsPointer
        ? `${
            additionalContext?.useTypesPackage
              ? "types"
              : sanitizeSDKPackageName(true)
          }.Pointer[float64](${normalized})`
        : `${normalized}`;
    case "decimal":
      addImport(getTypesLocation(), true);
      return `types.MustNewDecimalFromString("${value}")`;
  }
}

// @ts-ignore
function templateSecurityValue(
  value: any,
  field: FieldDef,
  optional: boolean,
  additionalContext?: TemplateValueContext,
) {
  if (isExampleReferenceValue(value)) {
    return templateExampleReferenceValue(
      value as ExampleReferenceValue,
      field,
      additionalContext,
    );
  }

  const secVal = parseSecurityExampleDirectives(value, additionalContext);
  if (optional) {
    return `${sanitizeSDKPackageName(true)}.Pointer(${secVal})`;
  } else {
    return secVal;
  }
}

// @ts-ignore
function getEnumNamesFromValues(values: string[]): string[] {
  let enumNames = [];

  let names = {};
  for (const value of values) {
    let name = getEnumName(value);
    if (!names[name]) {
      names[name] = 0;
    }

    names[name] += 1;
  }

  for (const value of values) {
    let name = getEnumName(value);
    if (names[name] > 1) {
      name = `${name}${caser().ToGoPascal(getCasing(value))}`;
    }

    enumNames.push(sanitizeFieldName(name));
  }

  // If casing-based dedup still left duplicates (e.g. values that differ only
  // in punctuation like {account_id} vs {account-id} vs {account.id}), append
  // numeric suffixes to disambiguate.
  const dupCounts: Record<string, number> = {};
  for (const n of enumNames) {
    dupCounts[n] = (dupCounts[n] || 0) + 1;
  }
  const dupCounters: Record<string, number> = {};
  for (let i = 0; i < enumNames.length; i++) {
    const n = enumNames[i];
    if (dupCounts[n] > 1) {
      dupCounters[n] = (dupCounters[n] || 0) + 1;
      enumNames[i] = `${n}${dupCounters[n]}`;
    }
  }

  return enumNames;
}

// @ts-ignore
function getEnumName(value: string): string {
  let name = value.trim();

  if (name === "") {
    name = "Unknown";
  }

  name = sanitizeName(name);

  return caser().ToGoPascalIgnoreAcronyms(name);
}

registerTemplateFunc("getEnumName", getEnumName);

/**
 * Returns the module path from the modulePath configuration or legacy
 * packageName configuration, optionally with a version suffix if SDK major
 * version is above 1. For example, if the module path is "example.com/openapi"
 * and SDK version is 2.0.0, the result will be
 * "example.com/example/openapi/v2".
 */
function getRootModulePath(): string {
  const versionParts = context.Global.Config.SDKVersion.split(".");
  const majorVersion = parseInt(versionParts[0], 10);
  const modulePath =
    context.Global.Config.ModulePath || context.Global.Config.PackageName;

  if (majorVersion > 1 && !context.Global.Config.WrapperName) {
    return `${modulePath}/v${majorVersion}`;
  } else {
    return `${modulePath}`;
  }
}

registerTemplateFunc("getRootModulePath", getRootModulePath);

// @ts-ignore
function sanitizeComments(comment: string): string {
  if (typeof comment !== "string") {
    return String(comment ?? "");
  }
  return comment
    .replaceAll("\uFEFF", "")
    .replaceAll("\r\n", "\n")
    .replaceAll(/{{/g, '{{ "{{" }}'); // Need to sanitize these so they don't conflict with the templates
}

registerTemplateFunc("sanitizeComments", sanitizeComments);

// @ts-ignore
function sanitizeAcceptEnumKey(acceptType: string): string {
  return `AcceptHeaderEnum${caser().ToPascal(
    sanitizeName(acceptType.split(";")[0]),
  )}`;
}

registerTemplateFunc("sanitizeAcceptEnumKey", sanitizeAcceptEnumKey);

// @ts-ignore
function sanitizeAcceptEnumValue(acceptType: string): string {
  return `${acceptType.split(";")[0]}`;
}

registerTemplateFunc("sanitizeAcceptEnumValue", sanitizeAcceptEnumValue);

/**
 * Returns the root module package name from the sdkPackageName configuration or
 * by sanitizing the final segment of getRootModulePath.
 */
// @ts-ignore
function sanitizeSDKPackageName(useAlias?: boolean): string {
  const sdkPackageAlias = context.Global.Config.SdkPackageAlias;

  if (sdkPackageAlias && useAlias) {
    return sdkPackageAlias;
  }

  let packageName: string = context.Global.Config.SdkPackageName;

  if (!packageName) {
    const modulePath = getRootModulePath();

    if (modulePath.includes("/")) {
      const modulePathParts = modulePath.split("/");
      packageName = modulePathParts[modulePathParts.length - 1];
    } else {
      packageName = modulePath;
    }
  }
  const sanitizedPackageName = sanitizeName(packageName);

  return caser().ToGoPascal(sanitizedPackageName).toLowerCase();
}

registerTemplateFunc("sanitizeSDKPackageName", sanitizeSDKPackageName);

// Returns the first `${className}${suffix}` (trying suffixes in order) that does
// not collide with an existing type in the same output location. Used to pick
// generated helper-type names that are not registered with the namer. Throws if
// every candidate collides.
function findAvailableTypeName(
  className: string,
  outputLocation: string,
  suffixes: string[],
): string {
  const allTypes = getAllTypes();
  for (const suffix of suffixes) {
    const name = `${className}${suffix}`;
    const foundConflict = allTypes.find(
      (type) =>
        type.OutputLocation == outputLocation &&
        sanitizeClassName(type.Name) == name,
    );
    if (!foundConflict) {
      return name;
    }
  }
  throw new Error(
    `Unable to find a non-colliding type name for ${className}. Contact Speakeasy for support.`,
  );
}
registerTemplateFunc("findAvailableTypeName", findAvailableTypeName);

// @ts-ignore
function getUnionTypeName(className: string, outputLocation: string): string {
  // Candidate suffixes preserve the historical escalation order
  // (XType -> XUnionType -> XOneOfUnion).
  return findAvailableTypeName(className, outputLocation, [
    "Type",
    "UnionType",
    "OneOfUnion",
  ]);
}

registerTemplateFunc("getUnionTypeName", getUnionTypeName);

// @ts-ignore
function sanitizeSecurityFieldName(fieldName: string): string {
  return sanitizeFieldName(fieldName);
}

// @ts-ignore
function getEnumNames(type: TypeDef): string[] {
  const baseNames =
    type.Enum?.Names.length > 0
      ? type.Enum.Names.map(
          (n) => `${sanitizeFieldName(type.Name)}${getEnumName(n)}`,
        )
      : getEnumNamesFromValues(type.Enum.Values).map(
          (n) => `${sanitizeFieldName(type.Name)}${n}`,
        );

  return resolveEnumNameConflicts(baseNames, type.OutputLocation);
}

// @ts-ignore
function resolveEnumNameConflicts(
  baseNames: string[],
  outputLocation: string,
): string[] {
  if (typeof getAccessNamespace === "undefined") {
    // Check if required dependencies are available if not fallback to default behavior
    return baseNames;
  }

  // Get all existing type names in the same scope for conflict detection
  const allTypes = getAllTypes();
  const existingTypeNames = new Set<string>();

  // Collect all type names in the same output location (scope)
  allTypes.forEach((t) => {
    if (t.OutputLocation === outputLocation) {
      existingTypeNames.add(sanitizeClassName(t.Name));
    }
  });

  const usedNames = new Set<string>(existingTypeNames);

  return baseNames.map((baseName) => {
    const candidate = findUniqueName(baseName, usedNames);
    usedNames.add(candidate);
    return candidate;
  });
}

function findUniqueName(baseName: string, usedNames: Set<string>): string {
  if (!usedNames.has(baseName)) {
    return baseName;
  }

  const withValue = `${baseName}Value`;
  if (!usedNames.has(withValue)) {
    return withValue;
  }

  const withEnumValue = `${baseName}EnumValue`;
  if (!usedNames.has(withEnumValue)) {
    return withEnumValue;
  }

  let counter = 2;
  while (counter < 100) {
    const candidate = `${baseName}${counter}`;
    if (!usedNames.has(candidate)) {
      return candidate;
    }
    counter++;
  }

  throw new Error(
    `Unable to resolve enum name conflict for "${baseName}". ` +
      `All variants conflict with existing names.`,
  );
}

// @ts-ignore
function getUnionEnumNamesFromDiscriminator(
  unionType: string,
  discriminatorMapping: any[],
  outputLocation: string,
): string[] {
  const baseNames = discriminatorMapping.map(
    (mapping) =>
      `${unionType}${sanitizeClassName(getDiscriminatorDisplayName(mapping))}`,
  );
  return resolveEnumNameConflicts(baseNames, outputLocation);
}

registerTemplateFunc(
  "getUnionEnumNamesFromDiscriminator",
  getUnionEnumNamesFromDiscriminator,
);

// @ts-ignore
function getUnionEnumNamesFromAssociatedTypes(
  unionType: string,
  associatedTypes: any[],
  outputLocation: string,
): string[] {
  const baseNames = associatedTypes.map((type) => {
    const name = sanitizeUnionTypeName(type);
    return `${unionType}${sanitizeClassName(name)}`;
  });
  return resolveEnumNameConflicts(baseNames, outputLocation);
}

registerTemplateFunc(
  "getUnionEnumNamesFromAssociatedTypes",
  getUnionEnumNamesFromAssociatedTypes,
);

// @ts-ignore
function sanitizeURLPath(path: string): string {
  const fragmentIndex = path.indexOf("#");
  if (fragmentIndex !== -1) {
    return path.slice(0, fragmentIndex);
  }
  return path;
}
registerTemplateFunc("sanitizeURLPath", sanitizeURLPath);

//@ts-ignore
function convertFloatToType(value: string, type: TypeDef): string {
  switch (type.Type.toString()) {
    case "integer":
      return `int64(${value})`;
    case "string":
      addImport("strconv");
      return `strconv.FormatFloat(${value}, 'f', 0, 64)`;
  }
  return "";
}
registerTemplateFunc("convertFloatToType", convertFloatToType);

//@ts-ignore
function sanitizeResponseType(responseType: TypeDef): string {
  const respTypeStr = responseType.Type.toString();
  switch (respTypeStr) {
    case "union":
    case "class":
      return `*${sanitizeClass(responseType, "", false)}`;
  }

  const ptrSymbol = getPointerTypes().includes(respTypeStr) ? "" : "*";
  return `${ptrSymbol}${sanitizeType(responseType, false, "")}`;
}

registerTemplateFunc("sanitizeResponseType", sanitizeResponseType);

// @ts-ignore
function languageSpecificEnvVarWrapping(
  envVar: EnvVar,
  additionalContext?: TemplateValueContext,
): string {
  if (additionalContext?.isTest) {
    addImport("utils");
    return `utils.GetEnv("${envVar.name}", "${envVar.defaultValue}")`;
  } else {
    addImport("os");
    return `os.Getenv("${envVar.name}")`;
  }
}

// sanitizeDiscriminatorGetters is in common/go/discriminators.ts (loaded via go/includes.ts)
