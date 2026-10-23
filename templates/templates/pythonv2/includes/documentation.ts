// Strip "models." prefix from doc type names, but only when there's a
// sub-namespace (2+ dots), e.g. "models.components.Foo" → "components.Foo".
// Leaves "models.Widget" (1 dot) unchanged.
// When fieldDef is provided, wraps the result with Optional/Nullable.
function stripModelsPrefix(name: string, fieldDef?: FieldDef): string {
  const dots = name.split(".").length - 1;
  if (name.startsWith("models.") && dots > 1) {
    name = name.slice("models.".length);
  }
  if (fieldDef) {
    const opt = fieldDef.Optional;
    const nul = fieldDef.Nullable;
    if (opt && nul) {
      name = `OptionalNullable[${name}]`;
    } else if (opt) {
      name = `Optional[${name}]`;
    } else if (nul) {
      name = `Nullable[${name}]`;
    }
  }
  return name;
}

//@ts-ignore
function templateMethodResponseDocsSDKs(operation: Operation): SDKDocType[] {
  const link = `../../${getModelsLocation(
    operation.Response.Type.OutputLocation,
  )}/${sDKDocsFileFunc(sanitizeClassName(operation.Response.Type.Name))}.md`;

  let contents: SDKDocType[] = [
    {
      Name: "",
      Type: `[${stripModelsPrefix(
        templateSimpleType(operation.Response.Type),
      )}](${link})`,
      Optional: false,
    },
  ];

  return contents;
}

registerTemplateFunc(
  "templateMethodResponseDocsSDKs",
  templateMethodResponseDocsSDKs,
);

//@ts-ignore
function templateMethodParametersDocsSDKs(operation: Operation): SDKDocType[] {
  let contents: SDKDocType[] = [];

  if (operationParametersFlattened(operation)) {
    if (operation.Security) {
      addSecurityParamDocumentationForSDKDocs(contents, operation);
    }

    const parameters = sortMethodParams(operation.Request.Field.Type.Fields);

    for (const parameter of parameters) {
      const description = templateDocumentationComments(parameter.Comments);

      let example = "";

      if (parameter.Type.Examples.length > 0) {
        example = formatExamples(parameter.Type.Examples);
      }

      contents.push({
        Name: `${sanitizeParameterName(parameter.Name)}`,
        Type: templateTypeMarkdown({
          fieldDef: parameter,
          source: GeneratedDocSource.SDKDOCS,
        }),
        Optional: parameter.Optional,
        Description: description != "" ? description : undefined,
        Example: example != "" ? example : undefined,
      });
    }
  } else {
    if (operation.Request) {
      let reqType: string;
      if (operation.Request.Field.Type.Name == "") {
        reqType = templateTypeMarkdown({
          fieldDef: operation.Request.Field,
          source: GeneratedDocSource.SDKDOCS,
        });
      } else {
        const link = `../../${getModelsLocation(
          operation.Request.Field.Type.OutputLocation,
        )}/${sDKDocsFileFunc(
          sanitizeClassName(operation.Request.Field.Type.Name),
        )}.md`;
        reqType = `[${stripModelsPrefix(
          templateSimpleType(operation.Request.Field.Type),
        )}](${link})`;
      }

      contents.push({
        Name: "request",
        Type: reqType,
        Optional: false,
        Description: "The request object to use for the request.",
      });
    }

    if (operation.Security) {
      addSecurityParamDocumentationForSDKDocs(contents, operation);
    }
  }

  if (operation.Extensions.Retries) {
    contents.push({
      Name: "retries",
      Type: `[Optional[utils.RetryConfig]](../../${getModelsLocation(
        context.Global.Config.Imports.GetSharedPath(),
      )}/retryconfig.md)`,
      Optional: true,
      Description:
        "Configuration to override the default retry behavior of the client.",
    });
  }

  if (operation.Servers) {
    contents.push({
      Name: "server_url",
      Type: `*Optional[str]*`,
      Optional: true,
      Description: "An optional server URL to use.",
    });
  }

  return contents;
}

registerTemplateFunc(
  "templateMethodParametersDocsSDKs",
  templateMethodParametersDocsSDKs,
);

//@ts-ignore
function templateMethodParametersDocs(operation: Operation): string {
  let contents: string[][] = [
    ["Parameter", "Type", "Required", "Description", "Example"],
  ];

  let validExamplesAdded = false;

  if (operation.Arguments.Flattening !== "none") {
    for (const parameter of operation.Arguments.Sorted) {
      const comments = templateDocumentationComments(parameter.Comments);

      let description = comments || "N/A";

      let example = "";

      if (parameter.Type.Examples.length > 0) {
        example = formatExamples(parameter.Type.Examples);
        validExamplesAdded = true;
      }

      contents.push([
        `\`${sanitizeParameterName(parameter.Name)}\``,
        templateTypeMarkdown({
          fieldDef: parameter,
        }),
        parameter.Optional ? ":heavy_minus_sign:" : ":heavy_check_mark:",
        description,
        example,
      ]);
    }
  } else {
    if (operation.Request) {
      const link = `../../${getModelsLocation(
        operation.Request.Field.Type.OutputLocation,
      )}/${sanitizeFileName(
        sanitizeClassName(operation.Request.Field.Type.Name),
      )}.md`;

      contents.push([
        "`request`",
        `[${stripModelsPrefix(
          templateSimpleType(operation.Request.Field.Type),
        )}](${link})`,
        ":heavy_check_mark:",
        "The request object to use for the request.",
        "",
      ]);
    }

    if (operation.Security) {
      addSecurityParamDocumentation(contents, operation);
    }
  }

  if (operation.Extensions.Retries) {
    templateFile(
      "readme/retryconfig.stmpl",
      "docs/models/utils/retryconfig.md",
      null,
    );

    contents.push([
      "`retries`",
      "[Optional[utils.RetryConfig]](../../models/utils/retryconfig.md)",
      ":heavy_minus_sign:",
      "Configuration to override the default retry behavior of the client.",
      "",
    ]);
  }

  if (operation.Servers) {
    contents.push([
      "`server_url`",
      `*Optional[str]*`,
      ":heavy_minus_sign:",
      "An optional server URL to use.",
      "http://localhost:8080",
    ]);
  }

  if (!validExamplesAdded) {
    contents = contents.map((row) => {
      row.pop();
      return row;
    });
  }

  if (contents.length == 1) {
    return "";
  }

  return createMarkdownTable(contents);
}

registerTemplateFunc(
  "templateMethodParametersDocs",
  templateMethodParametersDocs,
);

//@ts-ignore
function templateMethodResponseDocs(operation: Operation): string {
  if (!operation.Response.Type) {
    return "";
  }

  const link = `../../${getModelsLocation(
    operation.Response.Type.OutputLocation,
  )}/${sanitizeFileName(sanitizeClassName(operation.Response.Type.Name))}.md`;

  let documentation = `**[${stripModelsPrefix(
    templateSimpleType(operation.Response.Type),
  )}](${link})**`;

  return documentation;
}

registerTemplateFunc("templateMethodResponseDocs", templateMethodResponseDocs);

// @ts-ignore
function templateTypeMarkdown({
  fieldDef,
  docsRoot = "../..",
  source = GeneratedDocSource.README,
}: TemplateTypeMarkdownParams): string {
  let itemMarkdown = "";
  if (fieldDef.Type.ItemType != undefined) {
    itemMarkdown = templateTypeMarkdown({
      fieldDef: typeDefToFieldDef(
        fieldDef.Type.ItemType,
        undefined,
        undefined,
        fieldDef.Type.ContainsNull,
      ),
      docsRoot: docsRoot,
      source: source,
    });
  }

  switch (fieldDef.Type.Type.toString()) {
    case "class":
    case "enum":
    case "union": {
      const filename = classDocsFileName(fieldDef.Type.Name, source);
      const dir = `${docsRoot}/${getModelsLocation(
        fieldDef.Type.OutputLocation,
      )}`.replace(/\/+$/, "");
      const link = `${dir}/${filename}`;

      return `[${stripModelsPrefix(
        sanitizeType(fieldDef, { optional: false, nullable: false }),
        fieldDef,
      )}](${link})`;
    }
    case "array":
      return `List[${itemMarkdown}]`;
    case "map":
      return `Dict[str, ${itemMarkdown}]`;
    case "date-time": {
      return `[date](https://docs.python.org/3/library/datetime.html#date-objects)`;
    }
    case "date": {
      return `[datetime](https://docs.python.org/3/library/datetime.html#datetime-objects)`;
    }
    case "request":
      return `[httpx.Request](https://www.python-httpx.org/api/#request)`;
    case "response":
      return `[httpx.Response](https://www.python-httpx.org/api/#response)`;
    default:
      return `*${sanitizeType(fieldDef)}*`;
  }
}

// @ts-ignore
function templateEnumDocs(type: TypeDef): string {
  const enumFormat = getEnumFormat(type);
  const isOpen = type.Enum?.Open;
  const isString = type.Enum?.Type.Type.toString() === "string";

  if (enumFormat === "union") {
    const lines = type.Enum?.Values.map((v) => {
      return isString ? `- \`"${v}"\`` : `- \`${v}\``;
    });

    let output = "";
    if (isOpen) {
      output +=
        "This is an open enum. Unrecognized values will not fail type checks.\n\n";
    }
    output += lines.join("\n") + "\n";

    return output;
  }

  const contents: string[][] = [["Name", "Value"]];

  let enumNames = getEnumNames(type);

  type.Enum.Values.forEach((value, index) => {
    contents.push(["`" + enumNames[index] + "`", value]);
  });

  return createMarkdownTable(contents);
}

registerTemplateFunc("templateEnumDocs", templateEnumDocs);

// @ts-ignore
function templateModelSnippet(typeDef: TypeDef): string {
  if (typeDef.Type.toString() !== "enum") {
    return "";
  }

  const className = sanitizeClassName(typeDef.Name);
  const modelsLocation = getModelsLocation(typeDef.OutputLocation).replaceAll(
    "/",
    ".",
  );
  const importLine = `from ${getModuleImport()}.${modelsLocation} import ${className}`;

  const isOpen = typeDef.Enum?.Open;
  const isString = typeDef.Enum?.Type.Type.toString() === "string";
  const fallbackType = isString ? "UnrecognizedStr" : "UnrecognizedInt";

  const enumFormat = getEnumFormat(typeDef);
  if (enumFormat === "union") {
    if (!typeDef.Enum?.Values.length) {
      return "";
    }
    const firstValue = typeDef.Enum.Values[0];
    const exampleValue = isString ? `"${firstValue}"` : firstValue;
    let code = importLine;
    if (isOpen) {
      code += `\n\n# Open enum: unrecognized values are captured as ${fallbackType}`;
    }
    code += `\nvalue: ${className} = ${exampleValue}`;
    return code;
  }

  const enumNames = getEnumNames(typeDef);
  if (enumNames.length === 0) {
    return "";
  }

  let code = `${importLine}\n\nvalue = ${className}.${enumNames[0]}`;
  if (isOpen) {
    code += `\n\n# Open enum: unrecognized values are captured as ${fallbackType}`;
  }
  return code;
}

registerTemplateFunc("templateModelSnippet", templateModelSnippet);

// @ts-ignore
function templateUnionDocs(type: TypeDef): string {
  let unionDocs = "";

  let types: TypeDef[] = type.AssociatedTypes;
  if (type.Discriminator) {
    types = type.Discriminator.Mapping.map((m) => m.Type);
  }

  for (const subType of types) {
    const typename = stripModelsPrefix(templateSimpleType(subType));

    unionDocs += `### \`${typename}\`\n\n`;
    unionDocs += "```python\n";
    unionDocs += `value: ${typename} = /* values here */\n`;
    unionDocs += "```\n\n";
  }

  return unionDocs;
}

registerTemplateFunc("templateUnionDocs", templateUnionDocs);
