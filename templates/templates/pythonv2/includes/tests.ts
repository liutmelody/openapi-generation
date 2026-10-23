// @ts-ignore
function getTestDirectory(): string {
  // Ensure config.ts is updated with any updates.
  return "tests";
}

// @ts-ignore
function getTestFileName(testGroupName: string): string {
  const directory = getTestDirectory();
  const fileName = `test_${sanitizeFileName(testGroupName)}.py`;
  return `${directory}/${fileName}`;
}

// @ts-ignore
function getTestHelpersFileName(): string {
  const directory = getTestDirectory();
  const fileName = `test_helpers.py`;
  return `${directory}/${fileName}`;
}

// @ts-ignore
function sanitizeTestName(name: string): string {
  return caser().ToSnake(sanitizeName(name));
}
registerTemplateFunc("sanitizeTestName", sanitizeTestName);

// @ts-ignore
function templateStatusCodeAssertion(
  usageContext: UsageContext,
  assertion: Assertion,
): string[] {
  const responseFormat = getResponseFormat();

  const statusCode = assertion.Value as string;
  const response = assertion.Target as ResponseDef;

  const lines = [];

  let statusCodeAccessor = "";

  const resVariableName = getResponseVariableName(usageContext.StepID);

  switch (responseFormat) {
    case "envelope":
      statusCodeAccessor = `${resVariableName}.status_code`;
      break;
    case "envelope-http":
      lines.push(`assert ${resVariableName}.http_meta is not None`);
      lines.push(`assert ${resVariableName}.http_meta.response is not None`);

      statusCodeAccessor = `${resVariableName}.http_meta.response.status_code`;
      break;
    case "flat":
      if (
        response.Responses.find((r) =>
          doesSubResponseContainStatusCode(r, statusCode),
        )?.Content.length > 0 ||
        response.Responses.find((r) => r.Headers) !== undefined
      ) {
        return [`assert ${resVariableName} is not None`];
      }
      return [];
    default:
      throw new Error(`Unknown response format: ${responseFormat}`);
  }

  switch (true) {
    case statusCode == "default":
      const statusCodesToNotMatch = [];

      for (const subResponse of response.Responses) {
        for (const code of subResponse.Code) {
          if (code !== "default") {
            statusCodesToNotMatch.push(...getStatusCodesForRange(code));
          }
        }
      }

      lines.push(
        `assert ${statusCodeAccessor} not in [${statusCodesToNotMatch.join(
          ", ",
        )}]`,
      );
      break;
    case statusCode.toLowerCase().endsWith("xx"):
      lines.push(
        `assert ${statusCodeAccessor} in [${getStatusCodesForRange(
          statusCode,
        ).join(", ")}]`,
      );
      break;
    default:
      lines.push(`assert ${statusCodeAccessor} == ${statusCode}`);
      break;
  }

  return lines;
}

// @ts-ignore
function addRequiredTestImports() {}

// @ts-ignore
function addNullableOptionalAssertions(
  lines: string[],
  path: string,
  field: FieldDef,
  example: any,
): string[] {
  switch (true) {
    case field.Optional && example === undefined:
    case field.Nullable && example === null:
      lines.push(`assert ${path} is None`);
      break;
  }
  return lines;
}

// @ts-ignore
function addDefinedAssertion(lines: string[], path: string): string[] {
  lines.push(`assert ${path} is not None`);

  return lines;
}

// @ts-ignore
function addNotNullAssertion(lines: string[], path: string): string[] {
  lines.push(`assert ${path} is not None`);

  return lines;
}

// @ts-ignore
function addNotEmptyAssertion(lines: string[], path: string): string[] {
  lines.push(`assert ${path} is not None and len(${path}) > 0`);

  return lines;
}

// @ts-ignore
function templateAssertion(
  lines: string[],
  assertionValue: string,
  field: FieldDef,
  example: any,
  usageContext: UsageContext,
): string[] {
  let operator = "==";

  if (field.Type.Type.toString() == "boolean") {
    operator = "is";
  }

  const expected = templateValue(field, example, false, {
    usageContext: usageContext,
    operation: usageContext?.Operation,
    test: usageContext.Test,
    isTest: true,
    typedDictDisabled: true,
    isResponse: true,
    targetingMockServer: usageContext.UsingMockServer,
  });

  if (
    context.Global.Config.PreserveUnknownFields &&
    ["class", "array", "map", "union"].includes(field.Type.Type.toString())
  ) {
    addImport(".unknown_fields", "drop_unexpected_fields");
    lines.push(`expected = ${expected}`);
    lines.push(
      `assert drop_unexpected_fields(${assertionValue}, expected) == expected`,
    );
    return lines;
  }

  lines.push(`assert ${assertionValue} ${operator} ${expected}`);
  return lines;
}

// @ts-ignore
function getTestResponseClassFieldPath(
  path: string,
  parentField: FieldDef,
  classField: FieldDef,
): string {
  return `${path}.${sanitizeFieldName(classField.Name)}`;
}

// @ts-ignore
function sanitizeValuesWithDirectives(
  extensions: Record<string, any>,
  value: string,
): string {
  if (`x-speakeasy-test-internal-directives` in (extensions ?? {})) {
    const directives = extensions[
      `x-speakeasy-test-internal-directives`
    ] as any[];

    const sortQueryParametersDirective = directives.find(
      (d) => "sortQueryParameters" in d,
    );

    if (sortQueryParametersDirective) {
      addImport(".helpers", "*");
      const json =
        sortQueryParametersDirective["sortQueryParameters"].jsonParams;
      const jsonArg = json ? `, json_params=${JSON.stringify(json)}` : "";
      value = `sort_query_parameters(${value}${jsonArg})`;
    }

    const sortSerializedMapsDirectives = directives.filter(
      (d) => "sortSerializedMaps" in d,
    );

    for (const directive of sortSerializedMapsDirectives) {
      const sortSerializedMaps = directive["sortSerializedMaps"];
      addImport(".helpers", "*");
      value = `sort_serialized_maps(${value}, r"${sortSerializedMaps.regex}", "${sortSerializedMaps.delim}")`;
    }

    const sortJSONObjectKeysDirective = directives.find(
      (d) => "sortJSONObjectKeys" in d,
    );

    if (sortJSONObjectKeysDirective) {
      addImport(".helpers", "*");
      const fields = sortJSONObjectKeysDirective["sortJSONObjectKeys"].fields;
      const fieldsArg = fields ? `, ${JSON.stringify(fields)}` : "";
      value = `sort_json_object_keys(${value}${fieldsArg})`;
    }

    const sanitizeTimestampDirective = directives.find(
      (d) => "sanitizeTimestamp" in d,
    );

    if (sanitizeTimestampDirective) {
      addImport(".helpers", "*");
      value = `sanitize_timestamp(${value})`;
    }
  }

  return value;
}

// @ts-ignore
function templateTestID(test: Test): string {
  const id = getTestInternalID(test);

  if (!id) {
    return "";
  }

  addImport(".common_helpers", "*");
  return `\n    record_test("${id}")\n`;
}
registerTemplateFunc("templateTestID", templateTestID);

// @ts-ignore
function templateTestEnvVars(test: Test): string {
  const id = getTestInternalID(test);

  if (!id) {
    return "";
  }

  const envVars = test.InternalEnvVars;
  if (!envVars) {
    return "";
  }

  if (envVars.length == 0) {
    return "";
  }

  addImport("os", "");

  return (
    indentLines(
      envVars.map(
        (envVar) => `os.environ["${envVar.Name}"] = "${envVar.Value}"`,
      ),
      1,
    ) + "\n"
  );
}
registerTemplateFunc("templateTestEnvVars", templateTestEnvVars);

// @ts-ignore
function mutateAssertionValue(
  responseType: TypeDef,
  value: string,
  fieldDef: FieldDef,
  context?: ResponseAssertionContext,
  example?: any,
): string {
  switch (fieldDef.Type.Type.toString()) {
    case "response-stream":
      return `bytes().join(${value}.iter_bytes())`;
    default:
      return value;
  }
}

// @ts-ignore
function templateTestFieldAccessor(parent: FieldDef, field: FieldDef): string {
  return `.${sanitizeFieldName(field.Name)}`;
}

// @ts-ignore
function templateTestIndexAccessor(parent: FieldDef, idx: string): string {
  return `[${idx}]`;
}

// @ts-ignore
function templateTestKeyAccessor(parent: FieldDef, key: string): string {
  return `["${key}"]`;
}

// @ts-ignore
function templateTestAdditionalPropertiesAccessor(
  parent: FieldDef,
  additionalPropertiesField: FieldDef,
  fieldName: string,
): string {
  return `.${sanitizeFieldName(
    additionalPropertiesField.Name,
  )}["${fieldName}"]`;
}

// @ts-ignore
function templateTestFieldAccess(
  parent: FieldDef,
  currentPath: string,
): string[] {
  let lines = [];
  if ((parent.Optional || parent.Nullable) && !parent.IsAdditionalProperties) {
    lines = addDefinedAssertion(lines, currentPath);
  }
  return lines;
}

// @ts-ignore
function getResponseVariableName(stepID: string): string {
  return sanitizeFieldName(
    `${stepID && stepID !== "test" ? stepID + "_" : ""}res`,
  );
}
registerTemplateFunc("getResponseVariableName", getResponseVariableName);

// @ts-ignore
function getOutputVariableName(stepID: string): string {
  return sanitizeFieldName(
    `${stepID && stepID !== "test" ? stepID + "_" : ""}outputs`,
  );
}
registerTemplateFunc("getOutputVariableName", getOutputVariableName);

// @ts-ignore
function templateWorkflow(
  parent: ArazzoWorkflow,
  workflowStep: ArazzoWorkflowStep,
): string {
  const workflow = resolveWorkflowStepTarget(workflowStep);
  if (!workflow) {
    return "";
  }

  let assertions = "";
  let outputVariable = "";
  let inputs = "";

  if (workflow.Inputs) {
    const inputExample = {};

    const context: TemplateValueContext = {
      test: parent,
      isTest: true,
      scope: "usage",
      typedDictDisabled: true,
    };

    const exampleName = `${parent.Name}[${workflowStep.StepIdx}]`;

    for (const field of workflow.Inputs.Type.Fields) {
      const example = findExampleByName(field.Type.Examples ?? [], exampleName);

      inputExample[field.OriginalName] = getExampleValue(example, context);
    }

    inputs = templateValue(workflow.Inputs, inputExample, false, context);

    // Import the workflow inputs type if it's defined in test_helpers
    if (workflow.Inputs.Type.OutputLocation === "tests") {
      addImport(
        "tests.test_helpers",
        sanitizeClassName(workflow.Inputs.Type.Name),
      );
    }
  }

  if (workflow.Outputs) {
    const outputVariableName = getOutputVariableName(workflowStep.StepID);
    outputVariable = `${outputVariableName} = `;
    assertions = `\nassert ${outputVariableName} is not None`;
  }

  const methodName = sanitizePrivateMethodName(workflow.Name);
  addImport("tests.test_helpers", methodName);
  return `\n${outputVariable}${methodName}(${inputs})${assertions}`;
}
registerTemplateFunc("templateWorkflow", templateWorkflow);

// @ts-ignore
function templateHelperOutputs(workflow: ArazzoWorkflow): string {
  const value = templateWorkflowOutputsValue(workflow, {
    typedDictDisabled: true,
    outputLocation: "tests",
  });
  if (!value) {
    return "";
  }

  return `\nreturn ${value}`;
}
registerTemplateFunc("templateHelperOutputs", templateHelperOutputs);
