interface TemplatedAnnotation {
  Tag: string;
  Value: string;
}

// @ts-ignore
function templateAnnotations(
  fieldDef: FieldDef,
  outputEmptyString = false,
): string {
  let annotations: TemplatedAnnotation[] = [];

  var constsAndDefaultsAnno = templateConstsAndDefaultsAnnotation(fieldDef);
  if (constsAndDefaultsAnno) {
    annotations.push(constsAndDefaultsAnno);
  }

  if (fieldDef.IsAdditionalProperties) {
    annotations.push({
      Tag: "additionalProperties",
      Value: "true",
    });
  }

  var typeAnnotation = templateTypeAnnotation(fieldDef.Type);
  if (typeAnnotation) {
    annotations.push(typeAnnotation);
  }

  for (const annotation of fieldDef.Annotations) {
    switch (annotation.Type().toString()) {
      case "security":
        const secAnnotation = annotation as SecurityAnnotation;
        let env: string | undefined;
        if (
          context.Global.Config.EnvVarPrefix &&
          secAnnotation.FieldName != ""
        ) {
          env = templateSecurityEnvVars(fieldDef).toLowerCase();
        }

        annotations.push(templateSecurityAnnotation(secAnnotation, env));
        break;
      case "json":
        annotations.push(
          templateJSONAnnotation(fieldDef, annotation as JSONAnnotation),
        );
        break;
      case "param":
        annotations.push(
          templateParamAnnotation(annotation as ParamAnnotation),
        );
        break;
      case "request":
        annotations.push(
          templateRequestAnnotation(annotation as RequestAnnotation),
        );
        break;
      case "multipartForm":
        annotations.push(
          templateMultipartFormAnnotation(
            annotation as MultipartFormAnnotation,
          ),
        );
        break;
      case "form":
        annotations.push(templateFormAnnotation(annotation as FormAnnotation));
        break;
      case "encoding":
        // TODO determine if we need to do anything here
        break;
      case "response":
        return "";
      case "needsCasing":
        // This annotation is only used by TypeScript templates
        break;
      default:
        throw new Error(`unimplemented ${annotation.Type().toString()}`);
    }
  }

  if (annotations.length == 0) {
    if (outputEmptyString) {
      return '""';
    }

    return "";
  }

  // Check if any annotation contains backticks
  // In Go, struct tags with backticks cannot contain backtick characters
  // So we must use double-quote syntax instead
  const containsBackticks = annotations.some((anno) => {
    return `${anno.Value}`.includes("`");
  });

  let renderedAnnotations = "";

  for (const anno of annotations) {
    let delim = `"`;
    let value = anno.Value;
    if (containsBackticks) {
      delim = `\\"`;
      // When using double-quoted struct tags (because of backticks), we need to
      // double-escape for both Go source code compilation AND strconv.Unquote parsing.
      //
      // The value from quote().slice(1,-1) contains escape sequences like:
      // - \n for newlines (2 chars: backslash + n)
      // - \" for quotes (2 chars: backslash + quote)
      // - \\ for backslashes (2 chars: backslash + backslash)
      //
      // For Go double-quoted strings, we need:
      // - \n → \\n (so Go compiles to \n, then strconv.Unquote interprets as newline)
      // - \" → \\\" (so Go compiles to \", then strconv.Unquote interprets as quote)
      // - \\ → \\\\ (so Go compiles to \\, then strconv.Unquote interprets as backslash)
      //
      // IMPORTANT: Template escape sequences like {{ "{{" }} must be preserved as-is
      // because the template engine runs BEFORE Go compilation.
      //
      // We achieve this by:
      // 1. Temporarily replace template escapes with a placeholder
      // 2. Double all backslashes: \ → \\
      // 3. Escape all quotes: " → \" (the quote from \" needs its own escape after step 1)
      // 4. Restore template escapes (unmodified for the template engine)
      const templateEscapePlaceholder = "\x00TMPL_BRACE\x00";
      value = value
        .replaceAll('{{ "{{" }}', templateEscapePlaceholder)
        .replaceAll("\\", "\\\\")
        .replaceAll('"', '\\"')
        .replaceAll(templateEscapePlaceholder, '{{ "{{" }}');
    }

    renderedAnnotations += `${anno.Tag}:${delim}${value}${delim} `;
  }
  renderedAnnotations = renderedAnnotations.trimEnd();

  if (containsBackticks) {
    return `"${renderedAnnotations}"`;
  }

  return `\`${renderedAnnotations}\``;
}

registerTemplateFunc("templateAnnotations", templateAnnotations);

// @ts-ignore
function templateSecurityAnnotation(
  annotation: SecurityAnnotation,
  env?: string,
): TemplatedAnnotation {
  let name = escapeString(annotation.FieldName);

  let fields = [];

  if (annotation.Option) {
    fields.push("option");
  } else if (annotation.Scheme) {
    fields.push("scheme");

    fields.push(`type=${annotation.SecType}`);

    if (annotation.SubType) {
      fields.push(`subtype=${annotation.SubType}`);
    }

    if (annotation.Composite) {
      fields.push("composite");
    }
  }

  if (name) {
    fields.push(`name=${name}`);
  }

  if (env) {
    fields.push(`env=${env}`);
  }

  return {
    Tag: "security",
    Value: fields.join(","),
  };
}

// @ts-ignore
function templateJSONAnnotation(
  fieldDef: FieldDef,
  annotation: JSONAnnotation,
): TemplatedAnnotation {
  let name = escapeString(annotation.FieldName);

  let omitTag = "";

  if (fieldDef.Optional && !annotation.Ignore && !fieldDef.Default) {
    const includeEmptyObjects = context.Global.Config.IncludeEmptyObjects;
    if (includeEmptyObjects) {
      omitTag = ",omitzero"; // Only omit zero values
    } else {
      omitTag = ",omitempty"; // Omit zero values and empty objects
    }
  }

  return {
    Tag: "json",
    Value: name + omitTag,
  };
}

// @ts-ignore
function templateParamAnnotation(
  annotation: ParamAnnotation,
): TemplatedAnnotation {
  let name = escapeString(annotation.Name);

  let serialization = "";
  if (annotation.Serialization) {
    serialization = `serialization=${annotation.Serialization},`;
  } else if (annotation.Style) {
    serialization = `style=${annotation.Style},explode=${
      annotation.Explode ? "true" : "false"
    },`;
  }

  if (annotation.AllowReserved && !encodePathParams()) {
    warnAllowReservedIgnored(annotation);
  }

  const allowReserved =
    annotation.AllowReserved && encodePathParams() ? ",allowReserved=true" : "";

  return {
    Tag: annotation.ParamType,
    Value: `${serialization}name=${name}${allowReserved}`,
  };
}

// @ts-ignore
function templateRequestAnnotation(
  annotation: RequestAnnotation,
): TemplatedAnnotation {
  return {
    Tag: "request",
    Value: `mediaType=${annotation.MediaType}`,
  };
}

// @ts-ignore
function templateMultipartFormAnnotation(
  annotation: MultipartFormAnnotation,
): TemplatedAnnotation {
  if (annotation.File) {
    return {
      Tag: "multipartForm",
      Value: `file,name=${annotation.Name}`,
    };
  }
  if (annotation.Content) {
    return {
      Tag: "multipartForm",
      Value: "content",
    };
  }
  let json = "";
  if (annotation.JSON) {
    json = ",json";
  }
  return {
    Tag: "multipartForm",
    Value: `name=${annotation.Name}${json}`,
  };
}

// @ts-ignore
function templateFormAnnotation(
  annotation: FormAnnotation,
): TemplatedAnnotation {
  let name = escapeString(annotation.Name);

  let serialization = "";
  if (annotation.JSON) {
    serialization = `,json`;
  } else if (annotation.Style) {
    serialization = `,style=${annotation.Style},explode=${
      annotation.Explode ? "true" : "false"
    }`;
  }

  return {
    Tag: "form",
    Value: `name=${name}${serialization}`,
  };
}

// @ts-ignore
function templateConstsAndDefaultsAnnotation(
  fieldDef: FieldDef,
): TemplatedAnnotation | null {
  function sanitizeValue(value: any): string {
    if (value === null) {
      return "null";
    }

    if (typeof value === "string") {
      return quote(value).slice(1, -1).replaceAll(/{{/g, '{{ "{{" }}');
    }

    return value;
  }

  if (fieldDef.Type.IsContainer()) {
    if (fieldDef.Const) {
      return {
        Tag: "const",
        Value: sanitizeValue(JSON.stringify(fieldDef.Const.Value)),
      };
    }

    return null;
  }

  if (fieldDef.Const) {
    return {
      Tag: "const",
      Value: sanitizeValue(fieldDef.Const.Value),
    };
  }

  if (fieldDef.Default) {
    return {
      Tag: "default",
      Value: sanitizeValue(fieldDef.Default.Value),
    };
  }

  return null;
}

// @ts-ignore
function templateTypeAnnotation(typeDef: TypeDef): TemplatedAnnotation | null {
  switch (typeDef.Type.toString()) {
    case "map":
    case "array":
    case "set":
      return templateTypeAnnotation(typeDef.ItemType);
    case "integer":
      if (typeDef.Format == "string") {
        return {
          Tag: "integer",
          Value: "string",
        };
      }
      break;
    case "number":
      if (typeDef.Format == "string") {
        return {
          Tag: "number",
          Value: "string",
        };
      }
      break;
    case "bigint":
      if (typeDef.Format == "string") {
        return {
          Tag: "bigint",
          Value: "string",
        };
      }
      break;
    case "decimal":
      if (typeDef.Format !== "string") {
        return {
          Tag: "decimal",
          Value: "number",
        };
      }
      break;
  }

  return null;
}

// @ts-ignore
function templateTypeAnnotations(
  typeDef: TypeDef,
  outputEmptyString = false,
  outputBackTicks = false,
): string {
  const anno = templateTypeAnnotation(typeDef);
  if (!anno) {
    if (outputEmptyString) {
      return '""';
    }

    return "";
  }

  const annoString = `${anno.Tag}:"${anno.Value}"`;

  return outputBackTicks ? "`" + annoString + "`" : annoString;
}

registerTemplateFunc("templateTypeAnnotations", templateTypeAnnotations);

// @ts-ignore
function hasJSONEncodingAnnotation(fieldDef: FieldDef): boolean {
  for (const annotation of fieldDef.Annotations) {
    if (annotation.Type().toString() === "encoding") {
      const encodingAnnotation = annotation as EncodingAnnotation;
      return encodingAnnotation.MediaType === "application/json";
    }
  }

  return false;
}
registerTemplateFunc("hasJSONEncodingAnnotation", hasJSONEncodingAnnotation);

function encodePathParams(): boolean {
  return !!context.Global.Config.FixFlags?.["encodePathParams"];
}
registerTemplateFunc("encodePathParams", encodePathParams);

const allowReservedIgnoredWarnings = new Set<string>();

function warnAllowReservedIgnored(annotation: ParamAnnotation) {
  const key = `${annotation.ParamType}:${annotation.Name}`;
  if (allowReservedIgnoredWarnings.has(key)) {
    return;
  }
  allowReservedIgnoredWarnings.add(key);

  logWarning(
    `parameter "${annotation.Name}" declares allowReserved, which is ignored unless fixFlags.encodePathParams is enabled`,
    0,
  );
}
