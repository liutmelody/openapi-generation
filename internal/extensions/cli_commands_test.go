package extensions

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/speakeasy-api/openapi-generation/v2/internal/document"
	"github.com/speakeasy-api/openapi/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cliCommandsTestSpec is a neutral document that reproduces the structural
// shapes the extension has to handle: an operation whose request body is a
// oneOf of component variants, a variant composed via allOf with a $ref'd
// enum-with-default property, an open-enum, a closed-enum array, a union
// input property, readOnly fields, an explicitly-open inline body, a
// direct-$ref body, and parameterized operations.
const cliCommandsTestSpec = `openapi: 3.1.0
info:
  title: Task Service
  version: 1.0.0
paths:
  /tasks:
    post:
      operationId: CreateTask
      requestBody:
        required: true
        content:
          application/json:
            schema:
              oneOf:
                - $ref: '#/components/schemas/CreateRenderTaskParams'
                - $ref: '#/components/schemas/CreateWorkflowTaskParams'
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TaskResult'
    get:
      operationId: ListTasks
      parameters:
        - name: page-size
          in: query
          schema:
            type: integer
        - name: filter
          in: query
          schema:
            type: string
      responses:
        "200":
          description: ok
  /tasks/{taskId}:
    get:
      operationId: GetTask
      parameters:
        - name: taskId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/TaskResult'
  /notes:
    post:
      operationId: CreateNote
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                text:
                  type: string
                anything:
                  type: array
                  items: true
              required: [text]
              additionalProperties: true
      responses:
        "200":
          description: ok
  /documents:
    post:
      operationId: CreateDocument
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateDocumentParams'
      responses:
        "200":
          description: ok
  /uploads:
    post:
      operationId: CreateUpload
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUploadParams'
      responses:
        "200":
          description: ok
  /tasks/stream:
    post:
      operationId: StreamTask
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                prompt:
                  type: string
                stream:
                  type: boolean
                  default: true
              required: [prompt]
      responses:
        "200":
          description: A stream of task events
          content:
            application/json:
              schema:
                type: object
                properties:
                  output_text:
                    type: string
            text/event-stream:
              schema:
                title: TaskStream
                type: object
                required: [data]
                properties:
                  data:
                    $ref: '#/components/schemas/TaskEvent'
  /tasks/lines:
    post:
      operationId: StreamTaskLines
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                prompt:
                  type: string
      responses:
        "200":
          description: JSONL task events
          content:
            application/jsonl:
              schema:
                type: object
                additionalProperties: true
components:
  schemas:
    TaskEvent:
      # A property declared beside the union applies to every arm.
      properties:
        event_id:
          type: string
      oneOf:
        - $ref: '#/components/schemas/StepDeltaEvent'
        - $ref: '#/components/schemas/StepStopEvent'
        - $ref: '#/components/schemas/TaskErrorEvent'
    StepDeltaEvent:
      type: object
      properties:
        event_type:
          const: step.delta
        index:
          type: integer
        note:
          oneOf:
            - type: string
            - type: "null"
        label:
          allOf:
            - type: string
            - maxLength: 5
        delta:
          oneOf:
            - $ref: '#/components/schemas/TextDelta'
            - $ref: '#/components/schemas/ImageDelta'
      required: [event_type, delta]
    StepStopEvent:
      type: object
      properties:
        event_type:
          const: step.stop
        index:
          type: integer
        version:
          const: 2
        reason:
          $ref: '#/components/schemas/StopReason'
      required: [event_type]
    StopReason:
      type: string
      enum: [done, cancelled]
    TaskErrorEvent:
      type: object
      properties:
        event_type:
          const: error
        error:
          # A union lifted through allOf.
          allOf:
            - $ref: '#/components/schemas/ErrorBody'
      required: [event_type, error]
    ErrorBody:
      oneOf:
        - type: object
          properties:
            message:
              type: string
            code:
              type: integer
          additionalProperties: false
        - type: object
          properties:
            detail:
              type: string
          additionalProperties: false
    TextDelta:
      type: object
      properties:
        type:
          const: text
        text:
          type: ["string", "null"]
        parts:
          type: array
          items:
            type: string
      required: [type, text]
    ImageDelta:
      type: object
      properties:
        type:
          const: image
        data:
          type: string
        width:
          type: integer
      required: [type, data]
    TaskBase:
      type: object
      properties:
        input:
          description: Content to process. Accepts a plain string or structured parts.
          oneOf:
            - type: string
            - type: array
              items:
                $ref: '#/components/schemas/ContentPart'
      required: [input]
    CreateRenderTaskParams:
      allOf:
        - $ref: '#/components/schemas/TaskBase'
        - type: object
          properties:
            engine:
              $ref: '#/components/schemas/EngineOption'
            output_modalities:
              type: array
              items:
                $ref: '#/components/schemas/Modality'
            background:
              type: boolean
              description: Run the task in the background. Poll for the result.
            priority:
              type: integer
            sample_rate:
              type: integer
              enum: [8000, 16000, 44100]
            seeds:
              type: array
              items:
                type: integer
            created_at:
              type: string
              readOnly: true
            stream:
              type: boolean
              default: true
          required: [engine]
    CreateWorkflowTaskParams:
      type: object
      properties:
        workflow:
          type: string
        input:
          type: string
        mode:
          $ref: '#/components/schemas/WorkflowMode'
        stream:
          type: boolean
          default: true
      required: [workflow]
    WorkflowMode:
      type: string
      enum: [batch, streaming]
      x-speakeasy-unknown-values: disallow
    ContentPart:
      type: object
      properties:
        text:
          type: string
    EngineOption:
      type: string
      description: Engine used for the task. The service validates availability.
      enum: [render-standard-2, render-pro-2, render-mini-1]
      default: render-standard-2
      x-speakeasy-unknown-values: allow
    Modality:
      type: string
      enum: [text, image, audio]
    CreateUploadParams:
      type: object
      properties:
        name:
          type: string
        tags:
          type: array
          items:
            type: string
      required: [name]
    CreateDocumentParams:
      type: object
      properties:
        body/text:
          type: string
      required: [body/text]
    TaskResult:
      type: object
      properties:
        id:
          type: string
        status:
          type: string
          enum: [in_progress, completed, failed, requires_action]
        steps:
          type: array
          items:
            oneOf:
              - $ref: '#/components/schemas/NoteStep'
              - $ref: '#/components/schemas/OutputStep'
    NoteStep:
      type: object
      properties:
        type:
          const: note
        text:
          type: string
    OutputStep:
      type: object
      properties:
        type:
          const: output
        content:
          type: array
          items:
            oneOf:
              - $ref: '#/components/schemas/TextBlock'
              - $ref: '#/components/schemas/ImageBlock'
    TextBlock:
      type: object
      properties:
        type:
          const: text
        text:
          type: string
    ImageBlock:
      type: object
      properties:
        type:
          const: image
        data:
          type: string
          format: byte
        mime_type:
          type: string
        uri:
          type: string
`

// decodeCLITest loads the neutral spec with the given manifest body attached
// as x-speakeasy-cli-commands and runs the full decode + link.
func decodeCLITest(t *testing.T, manifestYAML string) (*CLICommandManifest, []string, error) {
	t.Helper()
	return decodeCLIWithSpec(t, cliCommandsTestSpec, manifestYAML)
}

func decodeCLIWithSpec(t *testing.T, spec, manifestYAML string) (*CLICommandManifest, []string, error) {
	t.Helper()

	var indented strings.Builder
	indented.WriteString("x-speakeasy-cli-commands:\n")
	for _, line := range strings.Split(strings.TrimRight(manifestYAML, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			indented.WriteString("\n")
			continue
		}
		indented.WriteString("  " + line + "\n")
	}
	fullYAML := spec + indented.String()

	ctx := context.Background()
	doc, _, err := openapi.Unmarshal(ctx, bytes.NewReader([]byte(fullYAML)))
	require.NoError(t, err)
	docInfo := &document.DocumentInfo{Doc: doc, Schema: []byte(fullYAML), SchemaPath: "test.yaml"}

	exts := doc.GetExtensions()
	for name, node := range exts.All() {
		if name == "x-speakeasy-cli-commands" {
			return DecodeCLICommandsManifest(ctx, docInfo, node)
		}
	}
	t.Fatalf("extension node not found in test document")
	return nil, nil, nil
}

const validAsyncManifest = `
version: 1
commands:
  produce:
    op: CreateTask#CreateRenderTaskParams
    async:
      op: GetTask
      id:
        from: $.id
        to: {in: path, name: taskId}
      params:
        stream: false
      statePointer: $.status
      states:
        in_progress: pending
        completed: success
        failed: failure
        requires_action: handoff
`

func TestCLICommands_AsyncDecodeAndLink(t *testing.T) {
	spec := strings.Replace(cliCommandsTestSpec, "        - name: taskId\n          in: path", "        - name: stream\n          in: query\n          schema: {type: boolean, default: true}\n        - name: taskId\n          in: path", 1)
	manifest, _, err := decodeCLIWithSpec(t, spec, strings.TrimSuffix(validAsyncManifest, "\n")+`
      interval: 20ms
      backoff: 1.25
      maxInterval: 60ms
      timeout: 5s
    output:
      artifact:
        contentPointer: $.steps[*].content[*]
        kind: image
        defaultPath: asset-{timestamp}.{ext}
`)
	require.NoError(t, err)
	require.Len(t, manifest.Commands, 1)
	recipe := manifest.Commands[0].Async
	require.NotNil(t, recipe)
	assert.Equal(t, "GetTask", recipe.OperationID)
	assert.Equal(t, "$.id", recipe.ID.From)
	assert.Equal(t, "/id", recipe.ID.Pointer)
	assert.Equal(t, CLICommandAsyncParameter{In: "path", Name: "taskId"}, recipe.ID.To)
	assert.Equal(t, map[string]any{"stream": false}, recipe.Params)
	assert.Equal(t, []CLICommandAsyncResolvedParameter{{In: "query", Name: "stream", Value: false}}, recipe.ResolvedParams)
	assert.Equal(t, "$.status", recipe.StateFrom)
	assert.Equal(t, "/status", recipe.StatePointer)
	assert.Equal(t, "20ms", recipe.Interval)
	assert.InDelta(t, 1.25, recipe.Backoff, 0)
	assert.Equal(t, "60ms", recipe.MaxInterval)
	assert.Equal(t, "5s", recipe.Timeout)
	assert.Equal(t, "200", recipe.CreateResponseCode)
	assert.Equal(t, "200", recipe.ResponseCode)
	assert.Equal(t, "200", manifest.Commands[0].Output.Artifact.ResponseCode)
}

func TestCLICommands_AsyncDefaults(t *testing.T) {
	manifestYAML := strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1)
	manifest, _, err := decodeCLITest(t, manifestYAML)
	require.NoError(t, err)
	recipe := manifest.Commands[0].Async
	assert.Equal(t, "2s", recipe.Interval)
	assert.InDelta(t, 1.5, recipe.Backoff, 0)
	assert.Equal(t, "30s", recipe.MaxInterval)
	assert.Equal(t, "10m", recipe.Timeout)
	assert.Equal(t, "error", recipe.ErrorField)
	assert.Equal(t, "message", recipe.ErrorMessageField)
	assert.False(t, recipe.errorExplicit)
}

func TestCLICommands_AsyncDecodeErrors(t *testing.T) {
	base := strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1)
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{"unknown key", strings.Replace(base, "      op: GetTask", "      operation: GetTask", 1), `async has unknown key "operation"`},
		{"states must be map", strings.Replace(base, "      states:\n        in_progress: pending\n        completed: success\n        failed: failure\n        requires_action: handoff", "      states: [completed]", 1), "async.states must be a mapping"},
		{"success required", strings.Replace(base, "completed: success", "completed: pending", 1), "must classify at least one state as success"},
		{"classification closed", strings.Replace(base, "failed: failure", "failed: error", 1), `unsupported classification "error"`},
		{"duration strings", strings.TrimSuffix(base, "\n") + "\n      interval: 2\n", "async.interval must be a string"},
		{"backoff bound", strings.TrimSuffix(base, "\n") + "\n      backoff: 0.9\n", "greater than or equal to 1"},
		{"interval ordering", strings.TrimSuffix(base, "\n") + "\n      interval: 2s\n      maxInterval: 1s\n", "maxInterval (1s) must be greater"},
		{"timeout ordering", strings.TrimSuffix(base, "\n") + "\n      interval: 2s\n      timeout: 1s\n", "timeout (1s) must be greater"},
		{"stream conflict", strings.TrimSuffix(base, "\n") + "\n    output:\n      stream:\n        select: $.status\n", "declares both async and output.stream"},
		{"async operation only", "version: 1\ncommands:\n  later:\n    planned: later\n    async:\n      op: GetTask\n      id:\n        from: $.id\n        to: {in: path, name: taskId}\n      statePointer: $.status\n      states: {completed: success}\n", "planned command and cannot declare async"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireDecodeError(t, tt.yaml, tt.want)
		})
	}
}

func TestCLICommands_AsyncParamsDecodeRejectsNonScalars(t *testing.T) {
	for name, value := range map[string]string{
		"object": "{enabled: true}",
		"array":  "[true]",
		"null":   "null",
	} {
		t.Run(name, func(t *testing.T) {
			manifest := strings.Replace(validAsyncManifest, "stream: false", "stream: "+value, 1)
			requireDecodeError(t, manifest, "async.params.stream must be a string, integer, number, or boolean")
		})
	}
}

func TestCLICommands_AsyncLinkErrors(t *testing.T) {
	base := strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1)
	tests := []struct {
		name string
		spec string
		yaml string
		want string
	}{
		{"missing operation", cliCommandsTestSpec, strings.Replace(base, "op: GetTask", "op: GetTusk", 1), `operation "GetTusk" does not exist`},
		{"GET required", strings.Replace(cliCommandsTestSpec, "  /tasks/{taskId}:\n    get:", "  /tasks/{taskId}:\n    post:", 1), base, "polling operations must use GET"},
		{"bodyless required", strings.Replace(cliCommandsTestSpec, "      operationId: GetTask\n      parameters:", "      operationId: GetTask\n      requestBody:\n        content:\n          application/json:\n            schema: {type: object}\n      parameters:", 1), base, "must be bodyless GET operations"},
		{"target required", strings.Replace(cliCommandsTestSpec, "        - name: taskId\n          in: path\n          required: true", "        - name: taskId\n          in: query", 1), strings.Replace(base, "in: path", "in: query", 1), "handle target must be required"},
		{"target string", strings.Replace(cliCommandsTestSpec, "        - name: taskId\n          in: path\n          required: true\n          schema:\n            type: string", "        - name: taskId\n          in: path\n          required: true\n          schema:\n            type: integer", 1), base, "must have a string schema"},
		{"extra required", strings.Replace(cliCommandsTestSpec, "        - name: taskId\n          in: path", "        - name: revision\n          in: query\n          required: true\n          schema: {type: string}\n        - name: taskId\n          in: path", 1), base, `required non-global query parameter "revision"`},
		{"state enum required", strings.Replace(cliCommandsTestSpec, "          enum: [in_progress, completed, failed, requires_action]", "          description: Current state", 1), base, "must declare a non-empty enum"},
		{"all enum members classified", cliCommandsTestSpec, strings.Replace(base, "        failed: failure\n", "", 1), `does not classify enum member "failed"`},
		{"listed state must exist", cliCommandsTestSpec, strings.Replace(base, "        failed: failure", "        rejected: failure", 1), `classifies "rejected", which is not an enum member`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := decodeCLIWithSpec(t, tt.spec, tt.yaml)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestCLICommands_AsyncPointersMustResolveInEveryResponseVariant(t *testing.T) {
	manifestYAML := strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1)
	unionSpec := strings.ReplaceAll(cliCommandsTestSpec, "$ref: '#/components/schemas/TaskResult'", "$ref: '#/components/schemas/AsyncTaskResult'") + `    AsyncTaskResult:
      oneOf:
        - $ref: '#/components/schemas/TaskResult'
        - type: object
          properties:
            id: {type: string}
`
	_, _, err := decodeCLIWithSpec(t, unionSpec, manifestYAML)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "async.statePointer $.status")
	assert.Contains(t, err.Error(), "does not resolve in every response variant")

	missingIDSpec := strings.Replace(unionSpec, "            id: {type: string}", "            status:\n              type: string\n              enum: [in_progress, completed, failed, requires_action]", 1)
	_, _, err = decodeCLIWithSpec(t, missingIDSpec, manifestYAML)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "async.id.from $.id")
	assert.Contains(t, err.Error(), "does not resolve in every response variant")
}

func TestCLICommands_AsyncAllowsRequiredGlobalPollParameters(t *testing.T) {
	spec := strings.Replace(cliCommandsTestSpec, "paths:\n", `x-speakeasy-globals:
  parameters:
    - name: revision
      in: query
      required: true
      schema:
        type: string
        default: stable
paths:
`, 1)
	spec = strings.Replace(spec, "        - name: taskId\n          in: path", "        - name: revision\n          in: query\n          required: true\n          schema: {type: string}\n        - name: taskId\n          in: path", 1)
	manifestYAML := strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1)
	manifest, _, err := decodeCLIWithSpec(t, spec, manifestYAML)
	require.NoError(t, err)
	require.NotNil(t, manifest.Commands[0].Async)
}

// Globals and operations commonly declare shared parameters through $ref
// (#/components/parameters); the linker must resolve those references before
// deciding whether a required poll parameter is a global.
func TestCLICommands_AsyncResolvesReferencedGlobalPollParameters(t *testing.T) {
	spec := strings.Replace(cliCommandsTestSpec, "paths:\n", `x-speakeasy-globals:
  parameters:
    - $ref: '#/components/parameters/revision'
paths:
`, 1)
	spec = strings.Replace(spec, "components:\n", `components:
  parameters:
    revision:
      name: revision
      in: query
      required: true
      schema:
        type: string
        default: stable
`, 1)
	spec = strings.Replace(spec, "        - name: taskId\n          in: path", "        - $ref: '#/components/parameters/revision'\n        - name: taskId\n          in: path", 1)
	manifestYAML := strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1)
	manifest, _, err := decodeCLIWithSpec(t, spec, manifestYAML)
	require.NoError(t, err)
	require.NotNil(t, manifest.Commands[0].Async)
}

func TestCLICommands_AsyncParamsLinking(t *testing.T) {
	withQuery := strings.Replace(cliCommandsTestSpec, "        - name: taskId\n          in: path", "        - name: stream\n          in: query\n          required: true\n          schema:\n            type: boolean\n            enum: [false]\n        - name: taskId\n          in: path", 1)
	manifest, _, err := decodeCLIWithSpec(t, withQuery, validAsyncManifest)
	require.NoError(t, err)
	assert.Equal(t, []CLICommandAsyncResolvedParameter{{In: "query", Name: "stream", Value: false}}, manifest.Commands[0].Async.ResolvedParams)

	withHeader := strings.Replace(cliCommandsTestSpec, "        - name: taskId\n          in: path", "        - name: X-Mode\n          in: header\n          schema: {type: string}\n        - name: taskId\n          in: path", 1)
	headerManifest := strings.Replace(validAsyncManifest, "stream: false", "x-mode: discrete", 1)
	manifest, _, err = decodeCLIWithSpec(t, withHeader, headerManifest)
	require.NoError(t, err)
	assert.Equal(t, []CLICommandAsyncResolvedParameter{{In: "header", Name: "X-Mode", Value: "discrete"}}, manifest.Commands[0].Async.ResolvedParams)

	tests := []struct {
		name string
		spec string
		yaml string
		want string
	}{
		{"unknown", withQuery, strings.Replace(validAsyncManifest, "stream: false", "stram: false", 1), `async.params.stram`},
		{"wrong scalar kind", withQuery, strings.Replace(validAsyncManifest, "stream: false", `stream: "false"`, 1), "is not a boolean"},
		{"closed enum", strings.Replace(withQuery, "enum: [false]", "enum: [true]", 1), validAsyncManifest, "is not in the closed enum"},
		{"handle duplicate", cliCommandsTestSpec, strings.Replace(validAsyncManifest, "stream: false", "taskId: value", 1), "duplicates the handle parameter"},
		{"path rejected", strings.Replace(cliCommandsTestSpec, "        - name: taskId\n          in: path", "        - name: stream\n          in: path\n          required: true\n          schema: {type: boolean}\n        - name: taskId\n          in: path", 1), validAsyncManifest, "path parameters must be bound by async.id.to"},
		{"ambiguous", strings.Replace(withQuery, "        - name: taskId\n          in: path", "        - name: stream\n          in: header\n          schema: {type: boolean}\n        - name: taskId\n          in: path", 1), validAsyncManifest, "is ambiguous"},
		{"global rejected", strings.Replace(withQuery, "paths:\n", "x-speakeasy-globals:\n  parameters:\n    - name: stream\n      in: query\n      required: true\n      schema: {type: boolean}\npaths:\n", 1), validAsyncManifest, "names global query parameter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := decodeCLIWithSpec(t, tt.spec, tt.yaml)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestCLICommands_AsyncParamsFormats(t *testing.T) {
	withParam := func(schema string) string {
		return strings.Replace(cliCommandsTestSpec, "        - name: taskId\n          in: path", "        - name: since\n          in: query\n          schema: "+schema+"\n        - name: taskId\n          in: path", 1)
	}
	withValue := func(value string) string {
		return strings.Replace(validAsyncManifest, "stream: false", "since: "+value, 1)
	}

	manifest, _, err := decodeCLIWithSpec(t, withParam("{type: string, format: date-time}"), withValue(`"2024-01-15T10:30:00Z"`))
	require.NoError(t, err)
	assert.Equal(t, []CLICommandAsyncResolvedParameter{{In: "query", Name: "since", Value: "2024-01-15T10:30:00Z"}}, manifest.Commands[0].Async.ResolvedParams)

	manifest, _, err = decodeCLIWithSpec(t, withParam("{type: string, format: date}"), withValue(`"2024-01-15"`))
	require.NoError(t, err)
	assert.Equal(t, []CLICommandAsyncResolvedParameter{{In: "query", Name: "since", Value: "2024-01-15"}}, manifest.Commands[0].Async.ResolvedParams)

	referenced := strings.Replace(withParam("{$ref: '#/components/schemas/Since'}"), "components:\n  schemas:\n", "components:\n  schemas:\n    Since:\n      type: string\n      format: date-time\n", 1)
	manifest, _, err = decodeCLIWithSpec(t, referenced, withValue(`"2024-01-15T10:30:00.250Z"`))
	require.NoError(t, err)
	assert.Equal(t, "2024-01-15T10:30:00.250Z", manifest.Commands[0].Async.ResolvedParams[0].Value)

	tests := []struct {
		name string
		spec string
		yaml string
		want string
	}{
		{"date-time text", withParam("{type: string, format: date-time}"), withValue("yesterday"), "is not an RFC 3339 date-time"},
		{"date-time from date", withParam("{type: string, format: date-time}"), withValue(`"2024-01-15"`), "is not an RFC 3339 date-time"},
		{"unquoted timestamp", withParam("{type: string, format: date-time}"), withValue("2024-01-15T10:30:00Z"), "unsupported YAML tag !!timestamp"},
		{"date from date-time", withParam("{type: string, format: date}"), withValue(`"2024-01-15T10:30:00Z"`), "is not a YYYY-MM-DD date"},
		{"bigint", withParam("{type: integer, format: bigint}"), withValue("12"), "format bigint is not supported for async params"},
		{"decimal", withParam("{type: number, format: decimal}"), withValue("1.5"), "format decimal is not supported for async params"},
		{"handle target", strings.Replace(cliCommandsTestSpec, "          schema:\n            type: string\n      responses:\n        \"200\":\n          description: ok\n          content:\n            application/json:\n              schema:\n                $ref: '#/components/schemas/TaskResult'\n  /notes:", "          schema:\n            type: string\n            format: date-time\n      responses:\n        \"200\":\n          description: ok\n          content:\n            application/json:\n              schema:\n                $ref: '#/components/schemas/TaskResult'\n  /notes:", 1), strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1), "has format date-time; the handle target must be a plain string"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := decodeCLIWithSpec(t, tt.spec, tt.yaml)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestCLICommands_AsyncValidatesEveryJSONSuccessResponse(t *testing.T) {
	manifestYAML := strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1)
	createMissingID := strings.Replace(cliCommandsTestSpec, "                $ref: '#/components/schemas/TaskResult'\n    get:\n      operationId: ListTasks", "                $ref: '#/components/schemas/TaskResult'\n        \"201\":\n          description: accepted\n          content:\n            application/json:\n              schema:\n                type: object\n                properties:\n                  status: {type: string}\n    get:\n      operationId: ListTasks", 1)
	_, _, err := decodeCLIWithSpec(t, createMissingID, manifestYAML)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create operation \"CreateTask\" response 201")
	assert.Contains(t, err.Error(), "async.id.from $.id")

	pollAddsState := strings.Replace(cliCommandsTestSpec, "                $ref: '#/components/schemas/TaskResult'\n  /notes:", "                $ref: '#/components/schemas/TaskResult'\n        \"201\":\n          description: accepted\n          content:\n            application/json:\n              schema:\n                type: object\n                properties:\n                  id: {type: string}\n                  status:\n                    type: string\n                    enum: [completed, paused]\n  /notes:", 1)
	_, _, err = decodeCLIWithSpec(t, pollAddsState, manifestYAML)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `does not classify enum member "paused"`)
}

func TestCLICommands_AsyncHintReasons(t *testing.T) {
	manifestYAML := strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1)
	manifest, _, err := decodeCLITest(t, strings.TrimSuffix(manifestYAML, "\n")+`
    hints:
      CLI_ASYNC_FAILED: [Inspect the terminal response.]
      CLI_ASYNC_TIMEOUT: [Resume polling with the handle.]
      CLI_ASYNC_UNKNOWN_STATE: [Update the CLI.]
`)
	require.NoError(t, err)
	assert.Len(t, manifest.Commands[0].Hints, 3)
}

func TestCLICommands_AsyncErrorBindings(t *testing.T) {
	base := strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1)
	withBindings := func(bindings string) string {
		return strings.TrimSuffix(base, "\n") + "\n" + bindings
	}
	// The poll response gains a custom-named failure-detail member; the
	// default "error"/"message" names resolve nowhere in it.
	withFailure := strings.Replace(cliCommandsTestSpec, `    TaskResult:
      type: object
      properties:
        id:
          type: string
`, `    TaskResult:
      type: object
      properties:
        id:
          type: string
        failure:
          type: object
          properties:
            detail:
              type: string
            code:
              type: integer
`, 1)
	require.NotEqual(t, cliCommandsTestSpec, withFailure, "fixture replacement must apply")

	// Undeclared bindings keep the v1 defaults and are deliberately not
	// linked: the base spec has no "error" root member yet still links.
	manifest, _, err := decodeCLIWithSpec(t, cliCommandsTestSpec, base)
	require.NoError(t, err)
	recipe := manifest.Commands[0].Async
	assert.Equal(t, "error", recipe.ErrorField)
	assert.Equal(t, "message", recipe.ErrorMessageField)
	assert.False(t, recipe.errorExplicit)

	// Declared bindings link against the declared member names.
	manifest, _, err = decodeCLIWithSpec(t, withFailure, withBindings("      errorField: failure\n      errorMessageField: detail\n"))
	require.NoError(t, err)
	recipe = manifest.Commands[0].Async
	assert.Equal(t, "failure", recipe.ErrorField)
	assert.Equal(t, "detail", recipe.ErrorMessageField)
	assert.True(t, recipe.errorExplicit)

	// Declaring either member opts the pair into strict linking: the
	// defaulted errorField "error" does not resolve in this response.
	_, _, err = decodeCLIWithSpec(t, withFailure, withBindings("      errorMessageField: detail\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `async.errorField: property "error" does not resolve at the root of the 200 response of poll operation "GetTask"`)

	// A misspelled errorField names the member and suggests the fix.
	_, _, err = decodeCLIWithSpec(t, withFailure, withBindings("      errorField: failur\n      errorMessageField: detail\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `async.errorField: property "failur" does not resolve at the root of the 200 response of poll operation "GetTask"`)
	assert.Contains(t, err.Error(), `did you mean "failure"?`)

	// The bound member must be an object able to carry the message.
	_, _, err = decodeCLIWithSpec(t, withFailure, withBindings("      errorField: id\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `async.errorField: property "id" in the 200 response of poll operation "GetTask" is not an object`)

	// Declaring only errorField still validates the defaulted message member.
	_, _, err = decodeCLIWithSpec(t, withFailure, withBindings("      errorField: failure\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `async.errorMessageField: property "message" does not resolve inside the "failure" member of the 200 response of poll operation "GetTask"`)

	// A misspelled message member is named with a did-you-mean.
	_, _, err = decodeCLIWithSpec(t, withFailure, withBindings("      errorField: failure\n      errorMessageField: detial\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `async.errorMessageField: property "detial" does not resolve inside the "failure" member`)
	assert.Contains(t, err.Error(), `did you mean "detail"?`)

	// The message member must be a string.
	_, _, err = decodeCLIWithSpec(t, withFailure, withBindings("      errorField: failure\n      errorMessageField: code\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `async.errorMessageField: property "code" inside the "failure" member of the 200 response of poll operation "GetTask" must be a string`)

	// Bindings are member names, not paths, and unknown keys get suggestions.
	_, _, err = decodeCLIWithSpec(t, withFailure, withBindings("      errorField: $.failure\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `async.errorField "$.failure" must be the exact JSON member name of one property`)
	_, _, err = decodeCLIWithSpec(t, withFailure, withBindings("      errorFeld: failure\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `async has unknown key "errorFeld" (did you mean "errorField"?)`)
}

func requireDecodeError(t *testing.T, manifestYAML, wantSubstring string) {
	t.Helper()
	_, _, err := decodeCLITest(t, manifestYAML)
	require.Error(t, err)
	assert.Contains(t, err.Error(), wantSubstring)
}

func requireCLIErrorExact(t *testing.T, spec, manifestYAML, want string) {
	t.Helper()
	_, _, err := decodeCLITestSpecWarn(t, spec, manifestYAML)
	require.EqualError(t, err, want)
}

// --- Full manifest fixture -------------------------------------------------

func TestCLICommands_FullManifest(t *testing.T) {
	manifest, warnings, err := decodeCLITest(t, `
version: 1
categories: [Create, Understand, Manage]
commands:
  render:
    category: Create
    summary: Render content
    tagline: (render-standard-2)
    description: |-
      Send content to a render engine and print the result.
    op: CreateTask#CreateRenderTaskParams
    args:
      prompt:
        to: $.input
        type: string
        required: true
        variadic: true
        summary: Content to render
    flags:
      engine:
        to: $.engine
        shorthand: e
        defaultFrom: schema
        summary: Engine to use
    hints:
      RESOURCE_EXHAUSTED: Quota hit; retry later or pick a lighter engine
      CLI_VALIDATION:
        - Print the exact request schema with --schema
    help:
      defaults: engine render-standard-2
      learn: task-cli docs render
      escalate: full request control via task-cli tasks create
    examples:
      simple:
        summary: Render with the default engine
        command: task-cli render "a lighthouse at sunset"
      alternate: task-cli render "a haiku" --engine render-pro-2
  image:
    category: Create
    summary: Generate images
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.engine: render-image-9
      $.output_modalities: image
      $.background: true
    args:
      prompt: {to: $.input, type: string, variadic: true, summary: Image prompt}
    jq: .steps
  speech:
    category: Create
    summary: Text to speech
    planned: >-
      "speech" needs an audio-capable engine surface that is not part of this
      build. Meanwhile use "task-cli render --body".
  workspaces: {category: Manage, group: true}
  workspaces info:
    category: Manage
    summary: Show workspace details
    op: GetTask
`)
	require.NoError(t, err)
	require.NotNil(t, manifest)

	assert.Equal(t, 1, manifest.Version)
	assert.Equal(t, []string{"Create", "Understand", "Manage"}, manifest.Categories)
	require.Len(t, manifest.Commands, 5)

	render := manifest.Commands[0]
	assert.Equal(t, "Render content", render.Summary)
	assert.Equal(t, "(render-standard-2)", render.Tagline)
	assert.Equal(t, "Create", render.Category)
	assert.Equal(t, "Send content to a render engine and print the result.", render.Description)
	assert.Equal(t, "render", render.ID)
	assert.Equal(t, []string{"render"}, render.Path)
	assert.Equal(t, CLICommandSourceOperation, render.Source.Type)
	require.Len(t, render.Source.Routes, 1)
	assert.Equal(t, "CreateTask", render.Source.Routes[0].OperationID)
	assert.Equal(t, "#/components/schemas/CreateRenderTaskParams", render.Source.Routes[0].RequestVariant)

	require.Len(t, render.Args, 1)
	arg := render.Args[0]
	assert.Equal(t, "prompt", arg.Name)
	assert.Equal(t, "string", arg.Type)
	assert.True(t, arg.Required)
	assert.True(t, arg.Variadic)
	require.NotNil(t, arg.Bind)
	assert.Equal(t, CLICommandBind{In: "body", Pointer: "/input", Mode: "set"}, *arg.Bind)

	require.Len(t, render.Flags, 1)
	flag := render.Flags[0]
	assert.Equal(t, "engine", flag.Name)
	assert.Equal(t, "e", flag.Shorthand)
	assert.Equal(t, "string", flag.Type)
	assert.Equal(t, "schema", flag.DefaultFrom)
	assert.Equal(t, "render-standard-2", flag.Default)
	assert.Equal(t, []any{"render-standard-2", "render-pro-2", "render-mini-1"}, flag.Enum)
	assert.Equal(t, CLICommandBind{In: "body", Pointer: "/engine", Mode: "set"}, *flag.Bind)

	require.Len(t, render.Examples, 2)
	assert.Equal(t, CLICommandExample{Summary: "Render with the default engine", Command: `task-cli render "a lighthouse at sunset"`}, render.Examples[0])
	assert.Equal(t, CLICommandExample{Command: `task-cli render "a haiku" --engine render-pro-2`}, render.Examples[1])

	require.NotNil(t, render.Hints)
	assert.Equal(t, []string{"Quota hit; retry later or pick a lighter engine"}, render.Hints["RESOURCE_EXHAUSTED"])
	assert.Equal(t, []string{"Print the exact request schema with --schema"}, render.Hints["CLI_VALIDATION"])
	require.NotNil(t, render.Help)
	assert.Equal(t, "engine render-standard-2", render.Help.Defaults)
	assert.Equal(t, "task-cli docs render", render.Help.Learn)
	assert.Equal(t, "full request control via task-cli tasks create", render.Help.Escalate)

	image := manifest.Commands[1]
	require.Len(t, image.Presets, 3)
	assert.Equal(t, CLICommandPreset{Bind: CLICommandBind{In: "body", Pointer: "/engine", Mode: "set"}, Value: "render-image-9"}, image.Presets[0])
	// Scalar promoted to a one-element array against the array<string> target.
	assert.Equal(t, CLICommandPreset{Bind: CLICommandBind{In: "body", Pointer: "/output_modalities", Mode: "set"}, Value: []any{"image"}}, image.Presets[1])
	assert.Equal(t, CLICommandPreset{Bind: CLICommandBind{In: "body", Pointer: "/background", Mode: "set"}, Value: true}, image.Presets[2])
	require.NotNil(t, image.Output)
	assert.Equal(t, ".steps", image.Output.Projection.JQ)
	// Image prompt arg: required inferred from the composed required[] since
	// no preset satisfies /input and the union arm has no default.
	assert.True(t, image.Args[0].Required)

	planned := manifest.Commands[2]
	assert.Equal(t, CLICommandSourcePlanned, planned.Source.Type)
	assert.Contains(t, planned.Source.Note, "audio-capable engine")

	group := manifest.Commands[3]
	assert.Equal(t, CLICommandSourceGroup, group.Source.Type)
	assert.Equal(t, []string{"workspaces"}, group.Path)

	nested := manifest.Commands[4]
	assert.Equal(t, "workspaces-info", nested.ID)
	assert.Equal(t, []string{"workspaces", "info"}, nested.Path)

	// Provenance warnings: promotion is logged, external hint reason noted.
	assert.Contains(t, strings.Join(warnings, "\n"), "promoted to a one-element array")
	assert.Contains(t, strings.Join(warnings, "\n"), "RESOURCE_EXHAUSTED")
}

// Idempotence law: decoding the same document twice yields the same IR.
func TestCLICommands_DecodeIsDeterministic(t *testing.T) {
	body := `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.output_modalities: image
`
	first, _, err := decodeCLITest(t, body)
	require.NoError(t, err)
	second, _, err := decodeCLITest(t, body)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

// --- Top-level gates -------------------------------------------------------

// yaml.v3 keeps the authored spelling of True/TRUE/False under the !!bool
// tag; the strict decoder must resolve those spellings rather than compare
// against the lowercase literal and silently invert the value.

// Under JSON Schema 2020-12, keywords beside $ref apply conjunctively: a
// property declared as {$ref: ..., readOnly: true} is read-only, so a preset
// targeting it must fail generation exactly like an inline readOnly one.
func TestCLICommands_PresetRefSiblingReadOnlyRejected(t *testing.T) {
	spec := strings.Replace(cliCommandsTestSpec, "            created_at:\n              type: string\n              readOnly: true\n", "            created_at:\n              type: string\n              readOnly: true\n            frozen:\n              $ref: '#/components/schemas/FrozenScalar'\n              readOnly: true\n", 1)
	spec = strings.Replace(spec, "components:\n  schemas:\n", "components:\n  schemas:\n    FrozenScalar:\n      type: string\n", 1)
	_, _, err := decodeCLIWithSpec(t, spec, `
version: 1
commands:
  image:
    summary: Generate images
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.engine: render-image-9
      $.output_modalities: image
      $.frozen: locked
    args:
      prompt: {to: $.input, type: string, variadic: true, summary: Image prompt}
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "readOnly")
	assert.Contains(t, err.Error(), "frozen")
}

// allOf intersects constraints, so a property redeclared across branches is
// a refinement (kept and intersected), not a conflict; only a genuinely
// unsatisfiable combination is an error.
func TestCLICommands_AllOfPropertyRefinementAccepted(t *testing.T) {
	refineManifest := `
version: 1
commands:
  image:
    summary: Generate images
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.engine: render-image-9
      $.output_modalities: image
      $.background: true
    args:
      prompt: {to: $.input, type: string, variadic: true, summary: Image prompt}
`
	refined := strings.Replace(cliCommandsTestSpec, "      allOf:\n        - $ref: '#/components/schemas/TaskBase'\n",
		"      allOf:\n        - type: object\n          properties:\n            background:\n              type: boolean\n              default: false\n        - $ref: '#/components/schemas/TaskBase'\n", 1)
	manifest, _, err := decodeCLIWithSpec(t, refined, refineManifest)
	require.NoError(t, err)
	require.NotNil(t, manifest)

	contradictory := strings.Replace(cliCommandsTestSpec, "      allOf:\n        - $ref: '#/components/schemas/TaskBase'\n",
		"      allOf:\n        - type: object\n          properties:\n            background:\n              type: string\n        - $ref: '#/components/schemas/TaskBase'\n", 1)
	_, _, err = decodeCLIWithSpec(t, contradictory, refineManifest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot both hold")
}

func TestCLICommands_MixedCaseBooleansDecode(t *testing.T) {
	manifest, _, err := decodeCLIWithSpec(t, cliCommandsTestSpec, `
version: 1
commands:
  image:
    summary: Generate images
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.engine: render-image-9
      $.output_modalities: image
      $.background: TRUE
    args:
      prompt: {to: $.input, type: string, required: True, variadic: True, summary: Image prompt}
`)
	require.NoError(t, err)
	require.Len(t, manifest.Commands, 1)
	cmd := manifest.Commands[0]
	require.Len(t, cmd.Args, 1)
	assert.True(t, cmd.Args[0].Required)
	assert.True(t, cmd.Args[0].Variadic)
	background := false
	for _, preset := range cmd.Presets {
		if preset.Bind.Pointer == "/background" {
			assert.Equal(t, true, preset.Value)
			background = true
		}
	}
	assert.True(t, background, "expected a /background preset")
}

func TestCLICommands_TopLevelGates(t *testing.T) {
	requireDecodeError(t, `
commands:
  render: {op: GetTask}
`, "version is required")

	requireDecodeError(t, `
version: 2
commands:
  render: {op: GetTask}
`, "unsupported version 2")

	requireDecodeError(t, `
version: "1"
commands:
  render: {op: GetTask}
`, "version must be the integer 1")

	requireDecodeError(t, `
version: 1
`, "at least one of commands or operations is required")

	requireDecodeError(t, `
version: 1
commands:
  - id: render
`, "sequence shape only existed in a pre-release draft")

	requireDecodeError(t, `
version: 1
commands: {}
`, "at least one command")

	requireDecodeError(t, `
version: 1
comands:
  render: {op: GetTask}
`, `did you mean "commands"?`)

	requireDecodeError(t, `
version: 1
categories: [Create, Create]
commands:
  render: {op: GetTask}
`, "duplicate category")

	// Distinct titles that normalize to the same cobra group ID would render
	// one duplicated help section; the decoder rejects them in the renderer's
	// normalization space.
	requireDecodeError(t, `
version: 1
categories: [Create, create]
commands:
  render: {op: GetTask}
`, `categories "Create" and "create" normalize to the same CLI group ID "create"`)

	requireDecodeError(t, `
version: 1
categories: ["Get / Fetch", "get--fetch"]
commands:
  render: {op: GetTask}
`, `normalize to the same CLI group ID "get-fetch"`)

	// Per-command categories used without a declared list get the same
	// normalization checks: they become cobra group IDs all the same.
	requireDecodeError(t, `
version: 1
commands:
  render: {op: GetTask, category: "!!!", summary: Render}
`, "normalizes to an empty CLI group ID")

	requireDecodeError(t, `
version: 1
commands:
  render: {op: GetTask, category: Create, summary: Render}
  image: {op: CreateTask#CreateRenderTaskParams, category: create, summary: Image, preset: {$.engine: render-image-9, $.output_modalities: image}, args: {prompt: {to: $.input, type: string, variadic: true, summary: P}}}
`, `normalize to the same CLI group ID "create"`)

	// A punctuation-only title normalizes to an empty group ID, under which
	// cobra's help template would swallow every ungrouped command.
	requireDecodeError(t, `
version: 1
categories: ["!!!"]
commands:
  render: {op: GetTask}
`, "normalizes to an empty CLI group ID")

	requireDecodeError(t, `
version: 1
categories: [Create]
commands:
  render: {category: Manage, op: GetTask}
`, `category "Manage", which is not in the categories list`)
}

// --- Command keys, ids, and nesting ---------------------------------------

func TestCLICommands_CommandKeys(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  Render: {op: GetTask}
`, "command key \"Render\" is invalid")

	requireDecodeError(t, `
version: 1
commands:
  "render  fast": {op: GetTask}
`, "is invalid")

	// "agent run" and "agent-run" derive the same id.
	requireDecodeError(t, `
version: 1
commands:
  agent run: {op: GetTask}
  agent-run: {op: GetTask}
`, "collides")

	// Nesting beneath an operation command is a path-prefix conflict.
	requireDecodeError(t, `
version: 1
commands:
  render: {op: GetTask}
  render fast: {op: GetTask}
`, "only group commands")

	// Nesting beneath a declared group is fine; an undeclared parent is
	// resolved by the renderer against the generated tree.
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  workspaces: {group: true}
  workspaces info: {op: GetTask}
  external info: {op: GetTask}
`)
	require.NoError(t, err)
	assert.Len(t, manifest.Commands, 3)
}

// --- Source discriminator --------------------------------------------------

func TestCLICommands_SourceDiscriminator(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  render: {summary: No source}
	`, "exactly one of op, routes, planned, custom, group, or source")

	requireDecodeError(t, `
version: 1
commands:
  render: {op: GetTask, group: true}
	`, "more than one of op, routes, planned, custom, group, source")

	requireDecodeError(t, `
version: 1
commands:
  render: {group: false}
`, "group must be the literal true")

	requireDecodeError(t, `
version: 1
commands:
  render: {planned: "   "}
`, "non-empty note")

	requireDecodeError(t, `
version: 1
commands:
  render: {custom: false}
`, "custom must be the literal true")

	requireDecodeError(t, `
version: 1
commands:
  render: {op: GetTask, custom: true}
	`, "more than one of op, routes, planned, custom, group, source")

	requireDecodeError(t, `
version: 1
commands:
  render: {op: "#CreateRenderTaskParams"}
`, "missing the operation id")

	requireDecodeError(t, `
version: 1
commands:
  render: {op: "CreateTask#"}
`, "empty variant after #")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: []
`, "must not be empty")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op:
      - CreateTask#CreateRenderTaskParams
      - CreateTask#CreateWorkflowTaskParams
	`, "use a routes: map")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op:
      - GetTask
      - GetTask
	`, "use a routes: map")
}

func TestCLICommands_CustomSource(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
categories: [Create]
commands:
  tts:
    category: Create
    summary: Text to speech
    tagline: (model-a)
    custom: true
`)
	require.NoError(t, err)
	require.Len(t, manifest.Commands, 1)
	tts := manifest.Commands[0]
	assert.Equal(t, CLICommandSourceCustom, tts.Source.Type)
	assert.Empty(t, tts.Source.Note)
	assert.Equal(t, "Text to speech", tts.Summary)
	assert.Equal(t, "Create", tts.Category)

	manifest, _, err = decodeCLITest(t, `
version: 1
commands:
  tts:
    summary: Text to speech
    source: {type: custom}
`)
	require.NoError(t, err)
	assert.Equal(t, CLICommandSourceCustom, manifest.Commands[0].Source.Type)

	requireDecodeError(t, `
version: 1
commands:
  tts:
    source: {type: custom, note: soon}
`, "custom sources take no routes or note")

	requireDecodeError(t, `
version: 1
commands:
  tts:
    custom: true
    flags:
      voice: {to: $.voice, type: string}
`, "custom command and cannot declare")
}

func TestCLICommands_OpSugarExpansion(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
  info:
    op: GetTask
  absolute:
    op: "CreateTask#/components/schemas/CreateWorkflowTaskParams"
`)
	require.NoError(t, err)
	assert.Equal(t, "#/components/schemas/CreateRenderTaskParams", manifest.Commands[0].Source.Routes[0].RequestVariant)
	assert.Empty(t, manifest.Commands[1].Source.Routes[0].RequestVariant)
	assert.Equal(t, "#/components/schemas/CreateWorkflowTaskParams", manifest.Commands[2].Source.Routes[0].RequestVariant)
}

func TestCLICommands_LongFormSource(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  render:
    source:
      type: operation
      routes:
        - operationId: CreateTask
          requestVariant: "#/components/schemas/CreateRenderTaskParams"
`)
	require.NoError(t, err)
	assert.Equal(t, CLICommandSourceOperation, manifest.Commands[0].Source.Type)
	assert.Equal(t, "#/components/schemas/CreateRenderTaskParams", manifest.Commands[0].Source.Routes[0].RequestVariant)

	requireDecodeError(t, `
version: 1
commands:
  render:
    source:
      type: operation
      routes:
        - operationId: CreateTask
      note: does not belong here
`, "note is only valid on planned sources")

	requireDecodeError(t, `
version: 1
commands:
  render:
    source:
      type: teleport
`, "unsupported source type")

	requireDecodeError(t, `
version: 1
commands:
  render:
    source:
      type: operation
      routes:
        - operationId: CreateTask
          requestVariant: CreateRenderTaskParams
`, "must be a document fragment reference")

	requireDecodeError(t, `
version: 1
commands:
  render:
    source:
      type: operation
      routes:
        - operationId: CreateTask
        - operationId: GetTask
`, "use the top-level routes: map")

	// Long-form source.routes deliberately retains its original list shape;
	// dispatch maps live only at the command's top-level routes: key.
	requireDecodeError(t, `
version: 1
commands:
  render:
    source:
      type: operation
      routes:
        engine: {operationId: CreateTask, requestVariant: "#/components/schemas/CreateRenderTaskParams"}
        pipeline: {operationId: CreateTask, requestVariant: "#/components/schemas/CreateWorkflowTaskParams"}
`, "source type operation requires a non-empty routes sequence")
}

func TestCLICommands_DispatchRoutesDecode(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  run:
    routes:
      engine: {op: CreateTask#CreateRenderTaskParams}
`, "routes must declare at least two route IDs")

	requireDecodeError(t, `
version: 1
commands:
  run:
    routes:
      Engine: {op: CreateTask#CreateRenderTaskParams}
      pipeline: {op: CreateTask#CreateWorkflowTaskParams}
`, `route ID "Engine" is invalid`)

	// Hyphen placement: a route ID is lowercase words joined by single
	// hyphens, so leading, trailing, and consecutive hyphens are rejected at
	// decode time rather than reaching the label humanizer.
	for _, id := range []string{"a-", "-a", "a--b", "a-b-"} {
		requireDecodeError(t, `
version: 1
commands:
  run:
    routes:
      `+id+`: {op: CreateTask#CreateRenderTaskParams}
      pipeline: {op: CreateTask#CreateWorkflowTaskParams}
`, `route ID "`+id+`" is invalid`)
	}
	requireDecodeError(t, `
version: 1
commands:
  run:
    routes:
      engine: {op: CreateTask#CreateRenderTaskParams, selector: engine--mode}
      pipeline: {op: CreateTask#CreateWorkflowTaskParams}
`, `selector "engine--mode" is invalid`)
	requireDecodeError(t, `
version: 1
commands:
  run:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine-: {to: $.engine}
`, `flag name "engine-" is invalid`)

	requireDecodeError(t, `
version: 1
commands:
  run:
    routes:
      engine: {op: CreateTask#CreateRenderTaskParams, selecter: engine}
      pipeline: {op: CreateTask#CreateWorkflowTaskParams}
`, `unknown key "selecter" (did you mean "selector"?)`)

	requireDecodeError(t, `
version: 1
commands:
  run:
    routes:
      engine: {op: CreateTask}
      pipeline: {op: CreateTask#CreateWorkflowTaskParams}
`, `route "engine" op must pin a request variant`)

	requireCLIErrorExact(t, cliCommandsTestSpec, `
version: 1
commands:
  run:
    routes:
      engine: {op: CreateTask#CreateRenderTaskParams, default: true}
      pipeline: {op: CreateTask#CreateWorkflowTaskParams, default: true}
`, `command "run": routes "engine" and "pipeline" both declare default: true; at most one default route is allowed`)

	requireCLIErrorExact(t, cliCommandsTestSpec, `
version: 1
commands:
  run:
    routes:
      engine: {op: CreateTask#CreateRenderTaskParams}
      pipeline: {op: GetTask#CreateWorkflowTaskParams}
`, `command "run": routes must all use one operation; route "engine" uses "CreateTask" and route "pipeline" uses "GetTask" (multi-operation dispatch is not part of v1)`)

	requireCLIErrorExact(t, cliCommandsTestSpec, `
version: 1
commands:
  run:
    routes:
      engine: {op: CreateTask#CreateRenderTaskParams}
      second-engine: {op: CreateTask#CreateRenderTaskParams}
`, `command "run": routes "engine" and "second-engine" both pin request variant "CreateRenderTaskParams"; each dispatch route must pin a distinct variant`)

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    override: true
`, `override is only valid with a routes: dispatch map`)
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    override: false
`, `override is only valid with a routes: dispatch map`)
}

// --- Reserved words and unknown keys ---------------------------------------

func TestCLICommands_ReservedAndUnknownKeys(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    payload:
      content: {}
`, "request-plan.payload-holes capability")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    format: json
`, "reserved for a future output-format capability")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    via: something
`, "reserved for a future routing capability")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    summry: typo
`, `did you mean "summary"?`)

	requireDecodeError(t, `
version: 1
commands:
  speech:
    planned: not yet
    args:
      prompt: {to: $.input}
`, "cannot declare args")

	requireDecodeError(t, `
version: 1
commands:
  workspaces:
    group: true
    jq: .x
`, "cannot declare")
}

func TestCLICommands_HelpValidation(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  speech:
    planned: not available in this build
    help:
      learn: task-cli docs speech
  workspaces:
    group: true
    help:
      defaults: region local
`)
	require.NoError(t, err)
	require.Equal(t, "task-cli docs speech", manifest.Commands[0].Help.Learn)
	require.Equal(t, "region local", manifest.Commands[1].Help.Defaults)

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    help: {}
`, "help must declare defaults, learn, or escalate")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    help:
      lern: task-cli docs render
`, `did you mean "learn"?`)

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    help:
      defaults: ""
`, "help defaults must be a non-empty string")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    help:
      defaults: 42
`, "help defaults must be a string")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    help:
      escalate: |-
        first line
        second line
`, "help escalate must be a single line")
}

// --- Input decode ----------------------------------------------------------

func TestCLICommands_InputRules(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    args:
      prompt: {to: $.input, type: string, variadic: true}
      extra: {to: $.engine, variadic: true}
`, "permits exactly one")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    args:
      prompt: {to: $.input, type: string}
`, "must declare variadic: true")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine, variadic: true}
`, "variadic is only valid on the positional argument")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    args:
      prompt: {to: $.input, type: string, variadic: true, shorthand: p}
`, "shorthand is only valid on flags")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      body: {to: $.engine}
`, "reserved by the generated runtime")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine, shorthand: ee}
`, "single alphanumeric character")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine, shorthand: e}
      extra: {to: $.background, type: bool, shorthand: e}
`, "both use shorthand -e")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine}
      motor: {to: $.engine}
`, "both bind /engine")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine, typ: string}
`, `did you mean "type"?`)

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine, type: text}
`, "unsupported type")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine, default: render-pro-2, defaultFrom: schema}
`, "both default and defaultFrom")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine, defaultFrom: spec}
`, `defaultFrom only supports "schema"`)

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {summary: Engine}
`, "must declare to:")
}

func TestCLICommands_ReservedPersistentFlags(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      debug: {to: $.background}
`, `flag name "debug" is reserved`)

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine, shorthand: d}
`, `persistent flag "debug"`)
}

// --- Path grammar ----------------------------------------------------------

func TestCLICommands_PathGrammar(t *testing.T) {
	// Bracket form binds properties that are not identifiers.
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  note:
    op: CreateNote
    flags:
      text: {to: $.text}
      odd:
        to: "$['weird name']"
        type: string
`)
	require.NoError(t, err)
	assert.Equal(t, "/weird name", manifest.Commands[0].Flags[1].Bind.Pointer)

	pathCases := []struct {
		path string
		want string
	}{
		{`/engine`, "JSON Pointer syntax"},
		{`engine`, "must start with $"},
		{`$`, "addresses the document root"},
		{`$.`, "dangling dot"},
		{`$.*`, "wildcard"},
		{`$..engine`, "recursive descent"},
		{`$[*]`, "wildcard"},
		{`$[?(@.x)]`, "filter expression"},
		{`$[0:2]`, "slice"},
		{`$['a','b']`, "union selector"},
		{`$[-1]`, "negative index"},
		{`$['unterminated]`, "unterminated"},
		{`$.content.parts.text`, "multi-segment"},
		{`$.content[0]`, "addresses an array index; a nested bind"},
		{`$[0]`, "array index at the body root"},
	}
	for _, testCase := range pathCases {
		requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      x:
        type: string
        to: >-
          `+testCase.path+`
`, testCase.want)
	}
}

// --- Non-body binds --------------------------------------------------------

func TestCLICommands_NonBodyBinds(t *testing.T) {
	// A valid parameter reference still gates on the renderer capability.
	requireDecodeError(t, `
version: 1
commands:
  list:
    op: ListTasks
    flags:
      filter: {to: {in: query, name: filter}, type: string}
`, "parameter-bind renderer capability")

	// An invalid parameter reference errors before the capability gate.
	requireDecodeError(t, `
version: 1
commands:
  list:
    op: ListTasks
    flags:
      filter: {to: {in: query, name: filtr}, type: string}
`, `did you mean "filter"?`)

	requireDecodeError(t, `
version: 1
commands:
  list:
    op: ListTasks
    flags:
      filter: {to: {in: body, name: filter}}
`, "body binds use the scalar form")

	requireDecodeError(t, `
version: 1
commands:
  list:
    op: ListTasks
    flags:
      filter: {to: {in: cookie, name: filter}}
`, "unsupported bind location")

	requireDecodeError(t, `
version: 1
commands:
  list:
    op: ListTasks
    flags:
      filter: {to: {in: query}}
`, "requires name:")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: {in: query, name: filter, mode: merge}}
`, "reserved for a future capability")
}

// --- Presets ---------------------------------------------------------------

func TestCLICommands_Presets(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.config.engine: render-image-9
`, "root-level in v1")

	requireDecodeError(t, `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.engine: null
`, "null presets are not supported")

	requireDecodeError(t, `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.created_at: "2024-01-01"
`, "readOnly")

	requireDecodeError(t, `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.background: definitely
`, "not a boolean")

	requireDecodeError(t, `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.priority: high
`, "not an integer")

	// Closed enum: an unknown value is rejected at decode.
	requireDecodeError(t, `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.output_modalities: video
`, "not in the closed enum")

	// Open enum: an unknown value passes (the server validates).
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.engine: render-experimental-1
`)
	require.NoError(t, err)
	assert.Equal(t, "render-experimental-1", manifest.Commands[0].Presets[0].Value)

	// Array value passes through unpromoted; each element enum-checked.
	manifest, _, err = decodeCLITest(t, `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.output_modalities: [image, audio]
`)
	require.NoError(t, err)
	assert.Equal(t, []any{"image", "audio"}, manifest.Commands[0].Presets[0].Value)

	// Promotion applies only to unambiguous array<string> targets: a scalar
	// against a string field stays scalar.
	manifest, warnings, err := decodeCLITest(t, `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.output_modalities: image
`)
	require.NoError(t, err)
	assert.Equal(t, []any{"image"}, manifest.Commands[0].Presets[0].Value)
	assert.Contains(t, strings.Join(warnings, "\n"), "promoted")
}

func TestCLICommands_NumericEnumPresets(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.sample_rate: 16000
`)
	require.NoError(t, err)
	assert.Equal(t, int64(16000), manifest.Commands[0].Presets[0].Value)

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.sample_rate: 12345
`, "not in the closed enum")
}

func TestCLIValuesEqual_ExactNumbers(t *testing.T) {
	tests := []struct {
		name  string
		a     any
		b     any
		equal bool
	}{
		{name: "int and int64", a: int(16000), b: int64(16000), equal: true},
		{name: "signed and unsigned", a: int64(16000), b: uint64(16000), equal: true},
		{name: "negative and unsigned", a: int64(-1), b: uint64(1), equal: false},
		{name: "integral float", a: int64(16000), b: float64(16000), equal: true},
		{name: "fractional float", a: int64(16000), b: float64(16000.5), equal: false},
		{name: "precision loss above 2^53", a: int64(9007199254740993), b: float64(9007199254740992), equal: false},
		{name: "exact large float", a: int64(1 << 60), b: float64(1 << 60), equal: true},
		{name: "signed float out of range", a: int64(9223372036854775807), b: float64(1 << 63), equal: false},
		{name: "unsigned float out of range", a: uint64(18446744073709551615), b: float64(1 << 64), equal: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, cliValuesEqual(tt.a, tt.b))
			assert.Equal(t, tt.equal, cliValuesEqual(tt.b, tt.a))
		})
	}
}

func TestCLICommands_ArrayPresetItems(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.seeds: [3, 5]
`)
	require.NoError(t, err)
	assert.Equal(t, []any{int64(3), int64(5)}, manifest.Commands[0].Presets[0].Value)

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.seeds: [three]
`, "not an integer")
}

func TestCLICommands_ArrayPresetBooleanItemsPassThrough(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  note:
    op: CreateNote
    preset:
      $.anything: [three, 3, true]
`)
	require.NoError(t, err)
	assert.Equal(t, []any{"three", int64(3), true}, manifest.Commands[0].Presets[0].Value)
}

func TestCLICommands_NormalizedPresetDuplicates(t *testing.T) {
	_, _, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.engine: render-standard-2
      $['engine']: render-pro-2
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
	assert.Contains(t, err.Error(), `"$.engine"`)
	assert.Contains(t, err.Error(), `"$['engine']"`)
	assert.Contains(t, err.Error(), `"/engine"`)
}

// --- Hints -----------------------------------------------------------------

func TestCLICommands_TypedPositionalGate(t *testing.T) {
	// The variadic positional joins argv into one string; a non-string
	// binding would lie about what the renderer sends.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    args:
      run-in-background: {to: $.background, variadic: true}
`, "only string bindings are part of v1")
}

func TestCLICommands_NestedGroupTagGate(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  workspaces: {group: true}
  workspaces inner: {group: true}
`, "nested group tags are not part of v1")

	// The long-form source object is subject to the same restriction.
	requireDecodeError(t, `
version: 1
commands:
  workspaces: {group: true}
  workspaces inner:
    source: {type: group}
`, "nested group tags are not part of v1")
}

func TestCLICommands_TypedPositionalGateOnOpenSchema(t *testing.T) {
	// An explicitly-open schema disables property lookup, but the positional
	// contract still holds: an explicitly non-string positional is rejected.
	requireDecodeError(t, `
version: 1
commands:
  note:
    op: CreateNote
    args:
      count: {to: $.count, type: int, variadic: true}
`, "only string bindings are part of v1")
}

func TestCLICommands_DisallowEnumIsClosed(t *testing.T) {
	// x-speakeasy-unknown-values: disallow keeps the enum closed — only the
	// explicit "allow" value opens it.
	requireDecodeError(t, `
version: 1
commands:
  workflow:
    op: CreateTask#CreateWorkflowTaskParams
    preset:
      $.mode: turbo
`, "not in the closed enum")

	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  workflow:
    op: CreateTask#CreateWorkflowTaskParams
    preset:
      $.mode: streaming
`)
	require.NoError(t, err)
	assert.Equal(t, "streaming", manifest.Commands[0].Presets[0].Value)
}

func TestCLICommands_Hints(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    hints:
      CLI_VALIDATION: validation
      CLI_CONNECTION: connection
      CLI_PROTOCOL: protocol
      CLI_RUNTIME: runtime
      CLI_UNAVAILABLE: unavailable
      CLI_AUTHENTICATION: authentication
      CLI_ASYNC_FAILED: async failed
      CLI_ASYNC_TIMEOUT: async timeout
      CLI_ASYNC_UNKNOWN_STATE: async unknown state
      CLI_OUTCOME_FAILED: outcome failed
      CLI_CONTINUATION_LIMIT: continuation limit
`)
	require.NoError(t, err)
	require.Len(t, manifest.Commands, 1)
	for _, reason := range CLIRuntimeHintReasons {
		assert.Contains(t, manifest.Commands[0].Hints, reason)
	}

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    hints:
      CLI_TURBO: not a runtime reason
`, "not a reason the generated runtime emits")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    hints:
      lower_case: bad reason grammar
`, "UPPER_SNAKE_CASE")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    hints:
      CLI_VALIDATION: ["ok", "  "]
`, "empty line")

	_, warnings, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    hints:
      SOME_SERVER_REASON: passes with a warning
`)
	require.NoError(t, err)
	assert.Contains(t, strings.Join(warnings, "\n"), "SOME_SERVER_REASON")
}

// --- Examples --------------------------------------------------------------

func TestCLICommands_Examples(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    examples:
      Bad Slug: task-cli render x
`, "example slug")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    examples:
      simple:
        summary: Missing the command
`, "non-empty command")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    examples:
      simple:
        comand: task-cli render x
`, `did you mean "command"?`)

	_, warnings, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    examples:
      one: task-cli render a
      two: task-cli render b
      three: task-cli render c
      four: task-cli render d
`)
	require.NoError(t, err)
	assert.Contains(t, strings.Join(warnings, "\n"), "at most 3")
}

// --- Schema link -----------------------------------------------------------

func TestCLICommands_SchemaLink(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  render: {op: CreateTsk}
`, `did you mean "CreateTask"?`)

	// A union request body with no pinned variant must name its members.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask
    args:
      prompt: {to: $.input, type: string, variadic: true}
`, "pin a variant")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#ContentPart
    args:
      prompt: {to: $.text, variadic: true}
`, "not a member of the operation's request body union")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParam
`, `did you mean "CreateRenderTaskParams"?`)

	// Pinning a variant on an inline (non-union) body is rejected.
	requireDecodeError(t, `
version: 1
commands:
  note:
    op: CreateNote#CreateRenderTaskParams
`, "inline schema")

	// Pinning a variant that is not the direct-$ref body is rejected.
	requireDecodeError(t, `
version: 1
commands:
  upload:
    op: CreateUpload#CreateRenderTaskParams
`, "not a union; the variant must match it")

	// Pinning the direct-$ref body itself is allowed.
	_, _, err := decodeCLITest(t, `
version: 1
commands:
  upload:
    op: CreateUpload#CreateUploadParams
    flags:
      name: {to: $.name}
`)
	require.NoError(t, err)

	// A pointer typo lands a did-you-mean from the composed properties.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engin}
`, `did you mean "engine"?`)

	// Binding into an operation without a JSON request body is an error.
	requireDecodeError(t, `
version: 1
commands:
  info:
    op: GetTask
    flags:
      engine: {to: $.engine, type: string}
`, "no application/json request body")

	// A union-typed property demands an explicit arm selection.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    args:
      prompt: {to: $.input, variadic: true}
`, "declare an explicit type")

	// The declared arm must exist (checked via a flag; positionals are
	// gated to string bindings before arm selection).
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      input-value: {to: $.input, type: bool}
`, "selects no union arm")

	// An explicit scalar type must match the schema.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine, type: int}
`, "schema property is string")

	// Binding a scalar input to an array property is a capability error.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      modes: {to: $.output_modalities, type: string}
`, "bind scalar fields only")

	// defaultFrom: schema demands an actual schema default.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      background: {to: $.background, defaultFrom: schema}
`, "has no default")
}

func TestCLICommands_Inference(t *testing.T) {
	manifest, warnings, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    args:
      prompt: {to: $.input, type: string, variadic: true}
    flags:
      background: {to: $.background}
      engine: {to: $.engine, defaultFrom: schema}
`)
	require.NoError(t, err)

	render := manifest.Commands[0]
	// Scalar type and summary inferred from the schema.
	background := render.Flags[0]
	assert.Equal(t, "bool", background.Type)
	assert.Equal(t, "Run the task in the background.", background.Summary)
	// Required inferred for the arg from the composed (allOf) required[].
	assert.True(t, render.Args[0].Required)
	// The arg's summary comes from the union property's description.
	assert.Equal(t, "Content to process.", render.Args[0].Summary)
	// engine is required in the schema but carries a schema default, so the
	// flag stays optional and the default is surfaced for display.
	engine := render.Flags[1]
	assert.False(t, engine.Required)
	assert.Equal(t, "render-standard-2", engine.Default)

	// No satisfiability warning: input is bound, engine has a schema default.
	assert.NotContains(t, strings.Join(warnings, "\n"), "required field")
}

func TestCLICommands_ExplicitRequiredIsPreserved(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    args:
      prompt: {to: $.input, type: string, variadic: true, required: false}
`)
	require.NoError(t, err)
	assert.False(t, manifest.Commands[0].Args[0].Required)
}

func TestCLICommands_EncodedPointerRequiredness(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  document:
    op: CreateDocument
    args:
      text: {to: "$['body/text']", variadic: true}
`)
	require.NoError(t, err)
	require.Len(t, manifest.Commands[0].Args, 1)
	assert.True(t, manifest.Commands[0].Args[0].Required)
}

func TestCLICommands_PresetSatisfiesRequiredInference(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  fixed:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.input: fixed input
    args:
      prompt: {to: $.input, type: string, variadic: true}
`)
	// Preset and bind on the same pointer are legal (preset seeds, input
	// wins), and the preset suppresses required inference for the arg.
	require.NoError(t, err)
	assert.False(t, manifest.Commands[0].Args[0].Required)
}

func TestCLICommands_SatisfiabilityDiagnostic(t *testing.T) {
	_, warnings, err := decodeCLITest(t, `
version: 1
commands:
  workflow:
    op: CreateTask#CreateWorkflowTaskParams
    flags:
      input: {to: $.input}
`)
	require.NoError(t, err)
	// workflow is required, has no default, and no input or preset covers it.
	assert.Contains(t, strings.Join(warnings, "\n"), `required field "workflow"`)
}

func TestCLICommands_OpenSchemaMiss(t *testing.T) {
	// CreateNote's body is explicitly open (additionalProperties: true): an
	// unknown pointer warns and requires an explicit type instead of erroring.
	requireDecodeError(t, `
version: 1
commands:
  note:
    op: CreateNote
    flags:
      extra: {to: $.extra}
`, "declare an explicit type")

	manifest, warnings, err := decodeCLITest(t, `
version: 1
commands:
  note:
    op: CreateNote
    flags:
      extra: {to: $.extra, type: string}
`)
	require.NoError(t, err)
	assert.Equal(t, "string", manifest.Commands[0].Flags[0].Type)
	assert.Contains(t, strings.Join(warnings, "\n"), "explicitly accepts unknown keys")
}

// --- Alias, merge keys, duplicate keys, tags -------------------------------

func TestCLICommands_YAMLStrictness(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    summary: one
    summary: two
`, "duplicate key")

	// Aliased maps are expanded, then duplicate keys are re-checked.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: GetTask
    <<: &shared
      summary: from merge
    summary: explicit
`, "duplicate key")

	// Alias expansion works for legitimate reuse.
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    preset: &shared-preset
      $.engine: render-image-9
  video:
    op: CreateTask#CreateRenderTaskParams
    preset: *shared-preset
`)
	require.NoError(t, err)
	assert.Equal(t, manifest.Commands[0].Presets, manifest.Commands[1].Presets)

	// Non-JSON tags are rejected.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.engine: 2024-01-01T00:00:00Z
`, "unsupported YAML tag")
}

func TestCLICommands_QuotedMergeKeyIsLiteral(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  note:
    op: CreateNote
    preset:
      $.meta:
        "<<": literal
`)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"<<": "literal"}, manifest.Commands[0].Presets[0].Value)
}

func TestCLICommands_PresetObjectKeysMustBeStrings(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  note:
    op: CreateNote
    preset:
      $.meta:
        1: one
`, "object keys must be strings; quote key")

	requireDecodeError(t, `
version: 1
commands:
  note:
    op: CreateNote
    preset:
      $.meta:
        ? [one, two]
        : pair
`, "object keys must be scalar strings")
}

// --- Absent extension ------------------------------------------------------

func TestCLICommands_AbsentExtension(t *testing.T) {
	ctx := context.Background()
	doc, _, err := openapi.Unmarshal(ctx, bytes.NewReader([]byte(cliCommandsTestSpec)))
	require.NoError(t, err)
	docInfo := &document.DocumentInfo{Doc: doc, Schema: []byte(cliCommandsTestSpec), SchemaPath: "test.yaml"}

	e := &Extensions{}
	manifest, err := e.HandleCLICommandsExtension(ctx, docInfo)
	require.NoError(t, err)
	assert.Nil(t, manifest)
}

// --- Artifact output -------------------------------------------------------

const cliArtifactBaseCommand = `
version: 1
commands:
  render:
    summary: Render an image to a file
    op: CreateTask#CreateRenderTaskParams
    args:
      prompt: {to: $.input, type: string, variadic: true, summary: Prompt}
    output:
      artifact:
        contentPointer: $.steps[*].content[*]
        kind: image
        defaultPath: "render-{timestamp}-{rand}.{ext}"
`

func TestCLICommands_ArtifactDecode(t *testing.T) {
	manifest, warnings, err := decodeCLITest(t, cliArtifactBaseCommand)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, manifest.Commands, 1)

	artifact := manifest.Commands[0].Output.Artifact
	require.NotNil(t, artifact)
	assert.Equal(t, "$.steps[*].content[*]", artifact.ContentPointer)
	assert.Equal(t, "image", artifact.Kind)
	assert.Equal(t, "render-{timestamp}-{rand}.{ext}", artifact.DefaultPath)
	assert.Equal(t, []CLICommandArtifactSegment{
		{Field: "steps"},
		{Wild: true},
		{Field: "content"},
		{Wild: true},
	}, artifact.Segments)
	assert.Nil(t, manifest.Commands[0].Output.Projection)

	// An undeclared block/identity falls back to the v1 shape defaults.
	assert.Equal(t, "type", artifact.TypeField)
	assert.Equal(t, "data", artifact.DataField)
	assert.Equal(t, "mime_type", artifact.MimeTypeField)
	assert.Equal(t, "uri", artifact.URIField)
	assert.Equal(t, "id", artifact.IDField)
	assert.Equal(t, "status", artifact.StatusField)
	assert.Equal(t, "completed", artifact.TerminalStatus)
	assert.False(t, artifact.blockExplicit)
	assert.False(t, artifact.identityExplicit)
}

func TestCLICommands_ArtifactGates(t *testing.T) {
	// The artifact declaration owns the command's default output; a default
	// jq projection alongside it would be dead configuration.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    jq: .id
    output:
      artifact:
        contentPointer: $.steps[*].content[*]
        kind: image
        defaultPath: "render-{timestamp}.{ext}"
`, "declares both jq and output.artifact")

	// output: on non-operation commands describes an invocation that never runs.
	requireDecodeError(t, `
version: 1
commands:
  render:
    planned: >-
      "render" is not part of this build.
    output:
      artifact:
        contentPointer: $.steps[*].content[*]
        kind: image
        defaultPath: "render-{timestamp}.{ext}"
`, "cannot declare output")

	// output: carries only artifact in v1; jq keeps its command-level spelling.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      jq: .id
`, "jq is declared at the command level")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artefact:
        contentPointer: $.steps[*].content[*]
`, `unknown key "artefact" (did you mean "artifact"?)`)

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output: {}
`, "output must declare artifact")

	// Flags owned by the artifact runtime are reserved for every command.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      out: {to: $.engine}
`, `flag name "out" is reserved`)

	for _, name := range []string{"async", "resume", "poll-interval", "poll-timeout"} {
		t.Run("reserved async flag "+name, func(t *testing.T) {
			requireDecodeError(t, fmt.Sprintf(`
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      %s: {to: $.engine}
`, name), fmt.Sprintf(`flag name %q is reserved`, name))
		})
	}
}

func TestCLICommands_ArtifactFieldValidation(t *testing.T) {
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artifact:
        kind: image
        defaultPath: "render-{timestamp}.{ext}"
`, "requires contentPointer")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artifact:
        contentPointer: $.steps[*].content[*]
        defaultPath: "render-{timestamp}.{ext}"
`, "requires kind")

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artifact:
        contentPointer: $.steps[*].content[*]
        kind: gif
        defaultPath: "render-{timestamp}.{ext}"
`, `unsupported kind "gif" (expected audio, image, video)`)

	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artifact:
        contentPointer: $.steps[*].content[*]
        kind: image
`, "requires defaultPath")
}

func TestCLICommands_ArtifactPointerGrammar(t *testing.T) {
	artifactManifest := func(pointer string) string {
		return `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artifact:
        contentPointer: "` + pointer + `"
        kind: image
        defaultPath: "render-{timestamp}.{ext}"
`
	}

	requireDecodeError(t, artifactManifest(`$.steps[0].content[*]`), "spell the segment as [*]")
	requireDecodeError(t, artifactManifest(`$..content`), "recursive descent")
	requireDecodeError(t, artifactManifest(`$.steps[?(@.type)]`), "filter expression")
	requireDecodeError(t, artifactManifest(`$.steps[*].*`), "dot wildcard")
	requireDecodeError(t, artifactManifest(`/steps/content`), "JSON Pointer syntax")
	requireDecodeError(t, artifactManifest(`$`), "addresses the document root")
}

func TestCLICommands_ArtifactDefaultPathValidation(t *testing.T) {
	artifactManifest := func(pattern string) string {
		return `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artifact:
        contentPointer: $.steps[*].content[*]
        kind: image
        defaultPath: "` + pattern + `"
`
	}

	requireDecodeError(t, artifactManifest(`render-{time}.{ext}`), "unknown placeholder {time}")
	requireDecodeError(t, artifactManifest(`render-{timestamp}.png`), "must contain the {ext} placeholder")
	requireDecodeError(t, artifactManifest(`/tmp/render-{timestamp}.{ext}`), "must be a relative path")
	requireDecodeError(t, artifactManifest(`../render-{timestamp}.{ext}`), "must not contain .. segments")
	requireDecodeError(t, artifactManifest(`render-{timestamp.{ext}`), "unknown placeholder {timestamp.{ext}")
}

func TestCLICommands_ArtifactSchemaLink(t *testing.T) {
	// The pointer must resolve: a misspelled field names the miss and suggests
	// the fix from the surviving views' properties.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artifact:
        contentPointer: $.step[*].content[*]
        kind: image
        defaultPath: "render-{timestamp}.{ext}"
`, `segment 1 (.step) does not resolve`)

	// [*] on a non-array is a targeted error.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artifact:
        contentPointer: $.id[*]
        kind: image
        defaultPath: "render-{timestamp}.{ext}"
`, "segment 2 ([*]) does not address an array")

	// The declared kind must exist among the terminal content blocks.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artifact:
        contentPointer: $.steps[*].content[*]
        kind: video
        defaultPath: "render-{timestamp}.{ext}"
`, `has no video block with a string "data" property (content types found: image, text)`)

	// A pointer that stops before the content blocks fails the terminal check.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    output:
      artifact:
        contentPointer: $.steps[*]
        kind: image
        defaultPath: "render-{timestamp}.{ext}"
`, `has no image block with a string "data" property`)

	// Operations without a JSON success response cannot host artifacts.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateNote
    output:
      artifact:
        contentPointer: $.steps[*].content[*]
        kind: image
        defaultPath: "render-{timestamp}.{ext}"
`, "has no application/json 2xx response")
}

// cliArtifactCustomShapeSpec is a response whose content blocks and job
// identity use none of the v1 default member names, so nothing about it can
// link (or extract at runtime) unless the declared bindings drive the walk.
const cliArtifactCustomShapeSpec = `openapi: 3.1.0
info:
  title: Render Service
  version: 1.0.0
paths:
  /renders:
    post:
      operationId: CreateRender
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                prompt:
                  type: string
              required: [prompt]
      responses:
        "200":
          description: render job
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/RenderJob'
components:
  schemas:
    RenderJob:
      type: object
      properties:
        job_ref:
          type: string
        attempt:
          type: integer
        phase:
          type: string
          enum: [queued, running, done, failed]
        outputs:
          type: array
          items:
            oneOf:
              - $ref: '#/components/schemas/CaptionBlock'
              - $ref: '#/components/schemas/PictureBlock'
    CaptionBlock:
      type: object
      properties:
        block_kind:
          const: caption
        caption:
          type: string
    PictureBlock:
      type: object
      properties:
        block_kind:
          const: image
        b64:
          type: string
        content_type:
          type: string
        content_uri:
          type: string
`

func cliArtifactCustomShapeManifest(blockAndIdentity string) string {
	return `
version: 1
commands:
  render:
    summary: Render an image
    op: CreateRender
    output:
      artifact:
        contentPointer: $.outputs[*]
        kind: image
        defaultPath: "render-{timestamp}-{rand}.{ext}"
` + blockAndIdentity
}

func TestCLICommands_ArtifactDeclaredShapeLinks(t *testing.T) {
	manifest, err := decodeCLITestSpec(t, cliArtifactCustomShapeSpec, cliArtifactCustomShapeManifest(`        block:
          typeField: block_kind
          dataField: b64
          mimeTypeField: content_type
          uriField: content_uri
        identity:
          idField: job_ref
          statusField: phase
          terminalStatus: done
`))
	require.NoError(t, err)

	artifact := manifest.Commands[0].Output.Artifact
	require.NotNil(t, artifact)
	assert.Equal(t, "block_kind", artifact.TypeField)
	assert.Equal(t, "b64", artifact.DataField)
	assert.Equal(t, "content_type", artifact.MimeTypeField)
	assert.Equal(t, "content_uri", artifact.URIField)
	assert.Equal(t, "job_ref", artifact.IDField)
	assert.Equal(t, "phase", artifact.StatusField)
	assert.Equal(t, "done", artifact.TerminalStatus)
	assert.True(t, artifact.blockExplicit)
	assert.True(t, artifact.identityExplicit)
}

func TestCLICommands_ArtifactDeclaredShapeRequired(t *testing.T) {
	// Without a block declaration the walk validates the v1 default shape,
	// which this response does not carry: the config must fail generation
	// instead of shipping a runtime that can never find its content.
	_, err := decodeCLITestSpec(t, cliArtifactCustomShapeSpec, cliArtifactCustomShapeManifest(""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `is not a typed content block (expected objects with a "type" const "image" and a string "data" property)`)
}

func TestCLICommands_ArtifactDeclaredShapeMismatches(t *testing.T) {
	// A misspelled data binding names the declared member, not the default.
	_, err := decodeCLITestSpec(t, cliArtifactCustomShapeSpec, cliArtifactCustomShapeManifest(`        block:
          typeField: block_kind
          dataField: b64data
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `has no image block with a string "b64data" property (content types found: caption, image)`)

	// A declared identity binding must resolve at the response root.
	_, err = decodeCLITestSpec(t, cliArtifactCustomShapeSpec, cliArtifactCustomShapeManifest(`        block:
          typeField: block_kind
          dataField: b64
        identity:
          idField: job_ref
          statusField: state
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `identity.statusField: property "state" does not resolve at the root of the 200 response of operation "CreateRender"`)

	// Declaring any identity member opts the whole group into strict linking:
	// a defaulted idField that does not resolve is named as the miss.
	_, err = decodeCLITestSpec(t, cliArtifactCustomShapeSpec, cliArtifactCustomShapeManifest(`        block:
          typeField: block_kind
          dataField: b64
        identity:
          statusField: phase
          terminalStatus: done
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `identity.idField: property "id" does not resolve at the root of the 200 response of operation "CreateRender"`)

	// A declared identity binding must be a string.
	_, err = decodeCLITestSpec(t, cliArtifactCustomShapeSpec, cliArtifactCustomShapeManifest(`        block:
          typeField: block_kind
          dataField: b64
        identity:
          idField: attempt
          statusField: phase
          terminalStatus: done
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `identity.idField: property "attempt" must be a string in every variant of the 200 response of operation "CreateRender"`)

	// A closed status enum must contain the declared terminal status.
	_, err = decodeCLITestSpec(t, cliArtifactCustomShapeSpec, cliArtifactCustomShapeManifest(`        block:
          typeField: block_kind
          dataField: b64
        identity:
          idField: job_ref
          statusField: phase
          terminalStatus: complete
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `identity.terminalStatus "complete" is not an enum member of "phase" in the 200 response of operation "CreateRender"`)
}

func TestCLICommands_ArtifactDeclaredURIBindingMissIsNamed(t *testing.T) {
	// When the only disqualifier of an otherwise viable block is the declared
	// URI binding, the error names that member instead of blaming dataField.
	spec := strings.Replace(cliArtifactCustomShapeSpec, "        content_uri:\n          type: string", "        content_uri:\n          type: integer", 1)
	require.NotEqual(t, cliArtifactCustomShapeSpec, spec, "fixture replacement must apply")
	_, err := decodeCLITestSpec(t, spec, cliArtifactCustomShapeManifest(`        block:
          typeField: block_kind
          dataField: b64
          uriField: content_uri
`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `the image block at $.outputs[*] has a "content_uri" property that is not a string`)
	assert.NotContains(t, err.Error(), `string "b64" property`)
}

func TestCLICommands_ArtifactTerminalStatusIsCheckedPerVariant(t *testing.T) {
	// The terminal status must be a member of the closed status enum in
	// every variant, not merely of the union of all variants' enums.
	spec := strings.Replace(cliArtifactCustomShapeSpec, "                $ref: '#/components/schemas/RenderJob'", `                oneOf:
                  - $ref: '#/components/schemas/RenderJob'
                  - $ref: '#/components/schemas/RenderJobLegacy'`, 1) + `    RenderJobLegacy:
      type: object
      properties:
        job_ref:
          type: string
        phase:
          type: string
          enum: [queued, halted]
        outputs:
          type: array
          items:
            oneOf:
              - $ref: '#/components/schemas/PictureBlock'
`
	require.NotEqual(t, cliArtifactCustomShapeSpec, spec, "fixture replacement must apply")
	declared := `        block:
          typeField: block_kind
          dataField: b64
        identity:
          idField: job_ref
          statusField: phase
          terminalStatus: done
`
	// "done" is an enum member of the first variant only: the legacy
	// variant's closed enum must reject it.
	_, err := decodeCLITestSpec(t, spec, cliArtifactCustomShapeManifest(declared))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `identity.terminalStatus "done" is not an enum member of "phase" in the 200 response of operation "CreateRender"`)

	// A terminal status present in every variant's enum links.
	_, err = decodeCLITestSpec(t, spec, cliArtifactCustomShapeManifest(strings.Replace(declared, "terminalStatus: done", "terminalStatus: queued", 1)))
	require.NoError(t, err)
}

func TestCLICommands_ArtifactIdentityDeclaredDefaultsAreLinked(t *testing.T) {
	// Declaring identity with the default names opts into strict linking on
	// the base spec (root id string + status enum containing completed).
	manifest, _, err := decodeCLITest(t, cliArtifactBaseCommand+`        identity:
          idField: id
          statusField: status
`)
	require.NoError(t, err)
	assert.True(t, manifest.Commands[0].Output.Artifact.identityExplicit)
	assert.Equal(t, "completed", manifest.Commands[0].Output.Artifact.TerminalStatus)

	// The defaulted terminal status is still validated once identity is
	// declared: a base-spec variant whose enum lacks it must be rejected.
	spec := strings.Replace(cliCommandsTestSpec, "enum: [in_progress, completed, failed, requires_action]", "enum: [in_progress, succeeded, failed]", 1)
	require.NotEqual(t, cliCommandsTestSpec, spec, "fixture replacement must apply")
	_, err = decodeCLITestSpec(t, spec, cliArtifactBaseCommand+`        identity:
          statusField: status
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `identity.terminalStatus "completed" is not an enum member of "status"`)
}

func TestCLICommands_ArtifactBlockIdentityGrammar(t *testing.T) {
	requireDecodeError(t, cliArtifactBaseCommand+`        block:
          dataFeld: b64
`, `output.artifact block has unknown key "dataFeld" (did you mean "dataField"?)`)

	requireDecodeError(t, cliArtifactBaseCommand+`        block: {}
`, "output.artifact block must declare at least one of")

	requireDecodeError(t, cliArtifactBaseCommand+`        identity: {}
`, "output.artifact identity must declare at least one of")

	requireDecodeError(t, cliArtifactBaseCommand+`        identity:
          idFeld: handle
`, `output.artifact identity has unknown key "idFeld" (did you mean "idField"?)`)

	// Bindings are member names, not paths.
	requireDecodeError(t, cliArtifactBaseCommand+`        block:
          dataField: $.data
`, `block dataField "$.data" must be the exact JSON member name of one property`)

	// Two block fields may not collapse onto one member. When both sides are
	// declared the message names them symmetrically...
	requireDecodeError(t, cliArtifactBaseCommand+`        block:
          typeField: payload
          dataField: payload
`, `typeField and dataField both bind property "payload"; each block field must bind a distinct property`)

	// ...but a collision with a filled-in default names only the binding the
	// author actually wrote, labeling the other side as the default.
	requireDecodeError(t, cliArtifactBaseCommand+`        block:
          dataField: type
`, `dataField binds property "type", which is also the default typeField binding`)

	requireDecodeError(t, cliArtifactBaseCommand+`        block:
          dataField: uri
`, `dataField binds property "uri", which is also the default uriField binding`)

	requireDecodeError(t, cliArtifactBaseCommand+`        identity:
          idField: status
`, `idField binds property "status", which is also the default statusField binding`)

	requireDecodeError(t, cliArtifactBaseCommand+`        identity:
          idField: shared
          statusField: shared
`, `idField and statusField both bind property "shared"; each identity field must bind a distinct property`)

	requireDecodeError(t, cliArtifactBaseCommand+`        identity:
          terminalStatus: ""
`, "identity terminalStatus must be a non-empty status value")

	// A non-string terminalStatus keeps the command attribution its sibling
	// binding errors carry.
	requireDecodeError(t, cliArtifactBaseCommand+`        identity:
          terminalStatus: [done]
`, `command "render" output.artifact: `)
	requireDecodeError(t, cliArtifactBaseCommand+`        identity:
          terminalStatus: [done]
`, "identity terminalStatus must be a string")
}

func TestCLICommands_ArtifactSkipsNoContentResponses(t *testing.T) {
	artifactSpec := func(responses string) string {
		return `openapi: 3.1.0
info:
  title: Artifact Service
  version: 1.0.0
paths:
  /artifact:
    get:
      operationId: GetArtifact
      responses:
` + responses + `
components:
  schemas:
    ArtifactResponse:
      type: object
      properties:
        result:
          $ref: '#/components/schemas/ImageBlock'
    ImageBlock:
      type: object
      properties:
        type:
          const: image
        data:
          type: string
`
	}
	manifest := `
version: 1
commands:
  artifact:
    op: GetArtifact
    output:
      artifact:
        contentPointer: $.result
        kind: image
        defaultPath: "artifact-{timestamp}.{ext}"
`
	schemaBearingResponse := func(status string) string {
		return `        "` + status + `":
          description: artifact
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ArtifactResponse'
`
	}

	t.Run("204 only", func(t *testing.T) {
		_, err := decodeCLITestSpec(t, artifactSpec(schemaBearingResponse("204")), manifest)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no application/json 2xx response")
	})

	t.Run("204 and 206", func(t *testing.T) {
		responses := schemaBearingResponse("204") + schemaBearingResponse("206")
		linked, err := decodeCLITestSpec(t, artifactSpec(responses), manifest)
		require.NoError(t, err)
		require.NotNil(t, linked.Commands[0].Output.Artifact)
		assert.Equal(t, "206", linked.Commands[0].Output.Artifact.ResponseCode)
	})
}

// Union arms composed with allOf (shared base properties beside a oneOf)
// must keep the shared properties visible on every branch of the walk.
func TestCLICommands_ArtifactSchemaLinkComposedUnion(t *testing.T) {
	spec := strings.Replace(cliCommandsTestSpec, `    TaskResult:
      type: object
      properties:
        id:
          type: string
        status:
          type: string
          enum: [in_progress, completed, failed, requires_action]
        steps:
          type: array
          items:
            oneOf:
              - $ref: '#/components/schemas/NoteStep'
              - $ref: '#/components/schemas/OutputStep'
`, `    TaskResult:
      allOf:
        - type: object
          properties:
            id:
              type: string
            status:
              type: string
              enum: [in_progress, completed, failed, requires_action]
        - type: object
          properties:
            steps:
              type: array
              items:
                allOf:
                  - type: object
                    properties:
                      content:
                        type: array
                        items:
                          oneOf:
                            - $ref: '#/components/schemas/TextBlock'
                            - $ref: '#/components/schemas/ImageBlock'
                  - oneOf:
                      - $ref: '#/components/schemas/NoteStep'
                      - type: object
                        properties:
                          type:
                            const: output
`, 1)
	require.NotEqual(t, cliCommandsTestSpec, spec, "fixture replacement must apply")

	fullYAML := spec + "x-speakeasy-cli-commands:\n" +
		"  version: 1\n" +
		"  commands:\n" +
		"    render:\n" +
		"      op: CreateTask#CreateRenderTaskParams\n" +
		"      output:\n" +
		"        artifact:\n" +
		"          contentPointer: $.steps[*].content[*]\n" +
		"          kind: image\n" +
		"          defaultPath: \"render-{timestamp}.{ext}\"\n"
	ctx := context.Background()
	doc, _, err := openapi.Unmarshal(ctx, bytes.NewReader([]byte(fullYAML)))
	require.NoError(t, err)
	docInfo := &document.DocumentInfo{Doc: doc, Schema: []byte(fullYAML), SchemaPath: "test.yaml"}
	for name, node := range doc.GetExtensions().All() {
		if name == "x-speakeasy-cli-commands" {
			manifest, _, err := DecodeCLICommandsManifest(ctx, docInfo, node)
			require.NoError(t, err)
			require.NotNil(t, manifest.Commands[0].Output.Artifact)
			assert.Equal(t, "200", manifest.Commands[0].Output.Artifact.ResponseCode)
			return
		}
	}
	t.Fatal("extension node not found")
}

// --- Output projection: output.stream -------------------------------------

func TestCLICommands_OutputStream(t *testing.T) {
	// output.stream.select coexists with the command-level jq sugar: the
	// projection owns stream-mode rendering, jq stays the default elsewhere.
	manifest, warnings, err := decodeCLITest(t, `
version: 1
commands:
  say:
    op: StreamTask
    args:
      prompt: {to: $.prompt, variadic: true}
    jq: .data
    output:
      stream:
        select: $.data.delta.text
  invite:
    op: CreateNote
    args:
      text: {to: $.text, variadic: true}
    jq: .id
	`)
	require.NoError(t, err)
	// The access itself distributes over the stream union. The remaining
	// warning is a schemaexec normalization gap: the concrete nested
	// type: [string, null] field is widened to an unconstrained schema.
	require.Equal(t, []string{
		`command "say": jq projection ".data" cannot be statically verified — unknown (unconstrained) schema at $.oneOf[0].properties.delta.oneOf[0].properties.text; it will only be checked at runtime`,
		`command "invite": jq projection ".id" cannot be statically verified — operation "CreateNote" declares no 2xx application/json response schema; it will only be checked at runtime`,
	}, warnings)
	require.Len(t, manifest.Commands, 2)

	say := manifest.Commands[0]
	require.NotNil(t, say.Output)
	require.NotNil(t, say.Output.Projection)
	assert.Equal(t, ".data", say.Output.Projection.JQ)
	require.NotNil(t, say.Output.Stream)
	assert.Equal(t, "$.data.delta.text", say.Output.Stream.Select)
	assert.Equal(t, "/data/delta/text", say.Output.Stream.Pointer)
	assert.Nil(t, say.Output.Artifact)

	invite := manifest.Commands[1]
	require.NotNil(t, invite.Output)
	assert.Equal(t, ".id", invite.Output.Projection.JQ)
	assert.Nil(t, invite.Output.Stream)

	// jq keeps its command-level spelling.
	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      jq: .b
      stream: {select: $.data.delta.text}
`, "jq is declared at the command level")

	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output: {}
`, "output must declare artifact, stream, or outcome")

	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      streem: {select: $.data}
`, `output has unknown key "streem" (did you mean "stream"?)`)

	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {selct: $.data}
`, `output.stream has unknown key "selct" (did you mean "select"?)`)

	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {}
`, "output.stream requires select:")

	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: ".data.text"}
`, "must start with $")

	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: "$.data[*].text"}
`, "wildcard")

	// A command's default output is a written file or a streamed projection.
	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: $.data.delta.text}
      artifact:
        contentPointer: $.data
        kind: image
        defaultPath: "x-{timestamp}.{ext}"
`, "declares both output.artifact and output.stream")

	// Output behavior describes an operation invocation.
	requireDecodeError(t, `
version: 1
commands:
  archive:
    planned: not yet
    output:
      stream: {select: $.data}
`, "cannot declare output")
}

func TestCLICommands_StreamProjectionLink(t *testing.T) {
	// The path resolves through the SSE envelope, the event union, and the
	// delta union to a string leaf: no warnings.
	_, warnings, err := decodeCLITest(t, `
version: 1
commands:
  say:
    op: StreamTask
    args:
      prompt: {to: $.prompt, variadic: true}
    output:
      stream: {select: $.data.delta.text}
	`)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	// Present in only some arms is fine (events lacking it are skipped).
	_, warnings, err = decodeCLITest(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: $.data.error.message}
`)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	// Array items via a bounded index.
	_, warnings, err = decodeCLITest(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: "$.data.delta.parts[0]"}
`)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	// Absent in every arm is a generation error with a did-you-mean.
	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: $.data.delta.txt}
`, `does not resolve in any event shape of the streaming response (segment "txt" (did you mean "text"?))`)

	// A leaf that is provably not a string in some arm is an error.
	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: $.data.index}
`, "resolves to an integer in one event shape")

	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: $.data.delta}
`, "resolves to an object in one event shape")

	// Leaf shapes that are strings: nullable type lists, string/null unions,
	// allOf-composed strings, string consts, $ref'd string enums.
	for _, sel := range []string{"$.data.delta.text", "$.data.note", "$.data.label", "$.data.event_type", "$.data.reason", "$.data.event_id", "$.data.delta.type"} {
		_, warnings, err = decodeCLITest(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: "`+sel+`"}
	`)
		require.NoError(t, err, sel)
		assert.Empty(t, warnings, sel)
	}

	// A union lifted through allOf resolves through every arm.
	_, warnings, err = decodeCLITest(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: $.data.error.detail}
`)
	require.NoError(t, err)
	assert.Empty(t, warnings)

	// A non-string const is provably not a string.
	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: $.data.version}
`, "resolves to a constant number in one event shape")

	// Traversing through a scalar, or past a closed object, is a miss.
	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: $.data.delta.text.inner}
`, "does not resolve in any event shape")

	requireDecodeError(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream: {select: $.data.error.mesage}
`, `(segment "mesage" (did you mean "message"?))`)

	// Operations without a streaming response cannot project a stream.
	requireDecodeError(t, `
version: 1
commands:
  note:
    op: CreateNote
    output:
      stream: {select: $.text}
`, "has no streaming (text/event-stream or JSONL) success response")

	// An open (unverifiable) stream schema warns instead of erroring.
	_, warnings, err = decodeCLITest(t, `
version: 1
commands:
  lines:
    op: StreamTaskLines
    output:
      stream: {select: $.text}
`)
	require.NoError(t, err)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "could not be verified")
}

func TestCLICommands_OperationDeclarations(t *testing.T) {
	manifest, warnings, err := decodeCLITest(t, `
version: 1
operations:
  StreamTask:
    output:
      stream: {select: $.data.delta.text}
    flags:
      stream:
        to: $.stream
        type: bool
        defaultFrom: schema
        summary: Stream events as they arrive
  CreateTask:
    flags:
      stream: {to: $.stream, defaultFrom: schema}
	`)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, manifest.Operations, 2)

	streamTask := manifest.Operations[0]
	assert.Equal(t, "StreamTask", streamTask.OperationID)
	assert.False(t, streamTask.BodyRequired)
	require.NotNil(t, streamTask.Output)
	assert.Equal(t, "/data/delta/text", streamTask.Output.Stream.Pointer)
	require.Len(t, streamTask.Flags, 1)
	assert.Equal(t, "bool", streamTask.Flags[0].Type)
	assert.Equal(t, true, streamTask.Flags[0].Default)

	createTask := manifest.Operations[1]
	assert.True(t, createTask.BodyRequired)
	require.Len(t, createTask.Flags, 1)
	assert.Equal(t, true, createTask.Flags[0].Default)
}

func TestCLICommands_OperationDeclarationsAreStrict(t *testing.T) {
	requireDecodeError(t, `
version: 1
operations:
  StreamTask:
    outputs: {stream: {select: $.data.delta.text}}
`, `operation "StreamTask" has unknown key "outputs"`)
	requireDecodeError(t, `
version: 1
operations:
  MissingOperation:
    output: {stream: {select: $.data.text}}
`, `operation "MissingOperation" does not exist`)
	requireDecodeError(t, `
version: 1
operations:
  StreamTask:
    flags:
      stream: {to: $.stream, required: true}
`, `flag "stream" cannot be required`)
	requireDecodeError(t, `
version: 1
operations:
  StreamTask:
    flags:
      stream: {to: $.stream, defaultFrom: schema}
      again: {to: $.stream}
`, `both bind /stream`)
	requireDecodeError(t, `
version: 1
operations:
  StreamTask:
    flags:
      nested: {to: $.stream.level}
`, `operation "StreamTask" flag "nested" binds /stream/level; operation flags bind top-level body fields only`)
	requireDecodeError(t, `
version: 1
operations:
  StreamTask:
    flags:
      all: {to: $.stream}
`, `flag "all" collides with the generated operation flag for pagination`)
	requireDecodeError(t, `
version: 1
operations:
  StreamTask:
    flags:
      stream: {to: $.stream, shorthand: a}
`, `shorthand -a collides with the generated pagination flag --all`)
	requireDecodeError(t, `
version: 1
operations:
  StreamTask:
    flags:
      prompt: {to: $.prompt, defaultFrom: schema}
`, `declares defaultFrom: schema, but the schema property at /prompt has no default`)
	requireDecodeError(t, `
version: 1
operations:
  CreateTask:
    flags:
      engine: {to: $.engine}
`, `property is absent from request body variant`)
}

// --- Variant selectors (partial-body preset merge) -------------------------

// decodeCLITestSpec is decodeCLITest against an arbitrary document instead
// of the shared fixture.
func decodeCLITestSpec(t *testing.T, spec, manifestYAML string) (*CLICommandManifest, error) {
	t.Helper()
	manifest, _, err := decodeCLITestSpecWarn(t, spec, manifestYAML)
	return manifest, err
}

// decodeCLITestSpecWarn is decodeCLITestSpec plus the decoder's warnings, for
// tests that assert on (or assert the absence of) generation warnings.
func decodeCLITestSpecWarn(t *testing.T, spec, manifestYAML string) (*CLICommandManifest, []string, error) {
	t.Helper()
	var indented strings.Builder
	indented.WriteString("x-speakeasy-cli-commands:\n")
	for _, line := range strings.Split(strings.TrimRight(manifestYAML, "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			indented.WriteString("\n")
			continue
		}
		indented.WriteString("  " + line + "\n")
	}
	fullYAML := spec + indented.String()
	ctx := context.Background()
	doc, _, err := openapi.Unmarshal(ctx, bytes.NewReader([]byte(fullYAML)))
	require.NoError(t, err)
	docInfo := &document.DocumentInfo{Doc: doc, Schema: []byte(fullYAML), SchemaPath: "test.yaml"}
	for name, node := range doc.GetExtensions().All() {
		if name == "x-speakeasy-cli-commands" {
			return DecodeCLICommandsManifest(ctx, docInfo, node)
		}
	}
	t.Fatalf("extension node not found in test document")
	return nil, nil, nil
}

const cliRouteDispatchSpec = `openapi: 3.1.0
info:
  title: Job Service
  version: 1.0.0
paths:
  /jobs:
    post:
      operationId: createJob
      requestBody:
        content:
          application/json:
            schema:
              oneOf:
                - $ref: '#/components/schemas/EngineJobParams'
                - $ref: '#/components/schemas/PipelineJobParams'
      responses:
        "200": {description: ok}
components:
  schemas:
    EngineJobParams:
      type: object
      required: [input]
      properties:
        engine:
          type: string
          default: text-1
          enum: [text-1, text-2, shared]
          description: Engine selection.
        input:
          type: string
          description: Input content.
        output_modes:
          type: string
        background:
          type: boolean
          description: Run in the background.
    PipelineJobParams:
      type: object
      required: [pipeline, input]
      properties:
        pipeline:
          type: string
          description: Pipeline selection.
        input:
          type: string
          description: Input content.
        pipeline_config:
          type: string
        background:
          type: boolean
          description: Run in the background.
`

const cliRouteDispatchManifest = `
version: 1
commands:
  jobs run:
    routes:
      engine:
        op: createJob#EngineJobParams
        default: true
        selector: engine
      pipeline:
        op: createJob#PipelineJobParams
    args:
      input: {to: $.input, variadic: true}
    flags:
      engine: {to: $.engine, defaultFrom: schema}
      pipeline: {to: $.pipeline}
      background: {to: $.background}
`

func cliRouteDispatchSpecWithThirdMember() string {
	spec := strings.Replace(cliRouteDispatchSpec,
		"                - $ref: '#/components/schemas/PipelineJobParams'\n",
		"                - $ref: '#/components/schemas/PipelineJobParams'\n                - $ref: '#/components/schemas/BatchJobParams'\n", 1)
	return spec + `    BatchJobParams:
      type: object
      required: [batch, input]
      properties:
        batch: {type: string}
        input: {type: string, description: Input content.}
`
}

func TestCLICommands_RouteDispatchPositive(t *testing.T) {
	manifest, warnings, err := decodeCLITestSpecWarn(t, cliRouteDispatchSpec, `
version: 1
commands:
  jobs run:
    routes:
      engine-route:
        label: Engine
        op: createJob#EngineJobParams
        default: true
        selector: engine
        preset: {$.background: true}
      pipeline:
        op: createJob#PipelineJobParams
        preset: {$.pipeline_config: standard}
    args:
      input: {to: $.input, variadic: true}
    flags:
      engine: {to: $.engine, defaultFrom: schema}
      pipeline: {to: $.pipeline}
      pipeline-config: {to: $.pipeline_config}
      background: {to: $.background}
    preset:
      $.background: false
`)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, manifest.Commands, 1)
	cmd := manifest.Commands[0]
	require.Len(t, cmd.Source.Routes, 2)

	engine, pipeline := cmd.Source.Routes[0], cmd.Source.Routes[1]
	assert.Equal(t, "engine-route", engine.ID)
	assert.Equal(t, "Engine", engine.Label)
	assert.True(t, engine.Default)
	assert.Equal(t, "engine", engine.Selector)
	assert.Equal(t, []string{"engine"}, engine.Selectors.Own)
	assert.Equal(t, []string{"pipeline"}, engine.Selectors.Foreign)
	assert.Equal(t, []CLICommandPreset{{Bind: CLICommandBind{In: "body", Pointer: "/background", Mode: "set"}, Value: true}}, engine.Presets)
	assert.Equal(t, "Pipeline", pipeline.Label)
	assert.Equal(t, "pipeline", pipeline.Selector)
	assert.Equal(t, []string{"pipeline"}, pipeline.Selectors.Own)
	assert.Equal(t, []string{"engine"}, pipeline.Selectors.Foreign)
	assert.Equal(t, []CLICommandPreset{
		{Bind: CLICommandBind{In: "body", Pointer: "/background", Mode: "set"}, Value: false},
		{Bind: CLICommandBind{In: "body", Pointer: "/pipeline_config", Mode: "set"}, Value: "standard"},
	}, pipeline.Presets)

	require.Len(t, cmd.Args, 1)
	assert.Equal(t, []string{"engine-route", "pipeline"}, cmd.Args[0].RouteIDs)
	assert.Equal(t, []string{"engine-route", "pipeline"}, cmd.Args[0].RequiredRouteIDs)
	assert.True(t, cmd.Args[0].Required)
	flags := map[string]CLICommandInput{}
	for _, flag := range cmd.Flags {
		flags[flag.Name] = flag
	}
	assert.Equal(t, []string{"engine-route"}, flags["engine"].RouteIDs)
	assert.Empty(t, flags["engine"].RequiredRouteIDs)
	assert.Equal(t, "text-1", flags["engine"].Default)
	assert.Equal(t, []string{"pipeline"}, flags["pipeline"].RouteIDs)
	assert.Equal(t, []string{"pipeline"}, flags["pipeline"].RequiredRouteIDs)
	assert.False(t, flags["pipeline"].Required)
	assert.Equal(t, []string{"pipeline"}, flags["pipeline-config"].RouteIDs)
	assert.Empty(t, flags["pipeline-config"].RequiredRouteIDs)
	assert.Equal(t, []string{"engine-route", "pipeline"}, flags["background"].RouteIDs)
	assert.Empty(t, flags["background"].RequiredRouteIDs)

	assert.Equal(t, []CLICommandDispatchKey{
		{Pointer: "/background", RouteIDs: []string{"engine-route", "pipeline"}},
		{Pointer: "/engine", RouteIDs: []string{"engine-route"}},
		{Pointer: "/input", RouteIDs: []string{"engine-route", "pipeline"}},
		{Pointer: "/output_modes", RouteIDs: []string{"engine-route"}},
		{Pointer: "/pipeline", RouteIDs: []string{"pipeline"}},
		{Pointer: "/pipeline_config", RouteIDs: []string{"pipeline"}},
	}, cmd.DispatchKeys)
}

func TestCLICommands_RouteDispatchAnchorErrors(t *testing.T) {
	withoutEngineSelector := strings.Replace(cliRouteDispatchManifest, "        selector: engine\n", "", 1)
	requireCLIErrorExact(t, cliRouteDispatchSpec, withoutEngineSelector,
		`command "jobs run": route "engine" has no unique required bound flag; set selector: <flag> to a unique bound flag backed by a required or schema-defaulted distinguishing property`)

	invalidSelector := strings.Replace(cliRouteDispatchManifest, "selector: engine", "selector: background", 1)
	requireCLIErrorExact(t, cliRouteDispatchSpec, invalidSelector,
		`command "jobs run": route "engine" selector "background" must name a declared flag unique to that route and backed by a required or schema-defaulted distinguishing property`)

	optionalSelector := strings.Replace(cliRouteDispatchManifest, "      pipeline: {to: $.pipeline}\n", "      pipeline: {to: $.pipeline}\n      pipeline-config: {to: $.pipeline_config}\n", 1)
	optionalSelector = strings.Replace(optionalSelector, "        op: createJob#PipelineJobParams\n", "        op: createJob#PipelineJobParams\n        selector: pipeline-config\n", 1)
	requireCLIErrorExact(t, cliRouteDispatchSpec, optionalSelector,
		`command "jobs run": route "pipeline" selector "pipeline-config" must name a declared flag unique to that route and backed by a required or schema-defaulted distinguishing property`)

	ambiguousSpec := strings.Replace(cliRouteDispatchSpec, "      required: [pipeline, input]\n", "      required: [pipeline, workspace, input]\n", 1)
	ambiguousSpec = strings.Replace(ambiguousSpec, "        pipeline_config:\n          type: string\n", "        workspace:\n          type: string\n        pipeline_config:\n          type: string\n", 1)
	ambiguousManifest := strings.Replace(cliRouteDispatchManifest, "      pipeline: {to: $.pipeline}\n", "      pipeline: {to: $.pipeline}\n      workspace: {to: $.workspace}\n", 1)
	requireCLIErrorExact(t, ambiguousSpec, ambiguousManifest,
		`command "jobs run": route "pipeline" has multiple unique required bound flags (--pipeline, --workspace); set selector: pipeline or selector: workspace`)
}

func TestCLICommands_RouteDispatchInputErrors(t *testing.T) {
	routeSpecificArg := strings.Replace(cliRouteDispatchManifest, "      input: {to: $.input, variadic: true}\n", "      output: {to: $.output_modes, variadic: true}\n", 1)
	requireCLIErrorExact(t, cliRouteDispatchSpec, routeSpecificArg,
		`command "jobs run": arg "output" is declared only by route "engine"; positional route selection is not part of v1`)

	missing := strings.Replace(cliRouteDispatchManifest, "      background: {to: $.background}\n", "      extra: {to: $.extra}\n", 1)
	requireCLIErrorExact(t, cliRouteDispatchSpec, missing,
		`command "jobs run": flag "extra" bind /extra is not declared by any routed variant; additionalProperties cannot establish route membership`)

	typeSpec := strings.Replace(cliRouteDispatchSpec, "        output_modes:\n          type: string\n", "        output_modes:\n          type: string\n        limit:\n          type: integer\n", 1)
	typeSpec = strings.Replace(typeSpec, "        pipeline_config:\n          type: string\n", "        pipeline_config:\n          type: string\n        limit:\n          type: string\n", 1)
	typeManifest := strings.Replace(cliRouteDispatchManifest, "      background: {to: $.background}\n", "      limit: {to: $.limit}\n", 1)
	requireCLIErrorExact(t, typeSpec, typeManifest,
		`command "jobs run": flag "limit" bind /limit has incompatible route types (engine: int, pipeline: string); one dispatch flag must have one scalar type`)

	unionSpec := strings.Replace(cliRouteDispatchSpec, "        output_modes:\n          type: string\n", "        output_modes:\n          type: string\n        config:\n          oneOf:\n            - {type: string}\n            - {type: boolean}\n", 1)
	unionSpec = strings.Replace(unionSpec, "        pipeline_config:\n          type: string\n", "        pipeline_config:\n          type: string\n        config:\n          type: string\n", 1)
	unionManifest := strings.Replace(cliRouteDispatchManifest, "      background: {to: $.background}\n", "      config: {to: $.config, type: string}\n", 1)
	requireCLIErrorExact(t, unionSpec, unionManifest,
		`command "jobs run": flag "config" bind /config has incompatible route shapes (engine: union, pipeline: string); union/scalar route bindings are not part of v1`)

	structuredSpec := strings.Replace(cliRouteDispatchSpec, "        output_modes:\n          type: string\n", "        output_modes:\n          type: array\n          items: {type: string}\n", 1)
	structuredManifest := strings.Replace(cliRouteDispatchManifest, "      background: {to: $.background}\n", "      modalities: {to: $.output_modes}\n", 1)
	requireCLIErrorExact(t, structuredSpec, structuredManifest,
		`command "jobs run" flag "modalities" bind /output_modes targets a array property; v1 inputs bind scalar fields only (structured construction requires the request-plan.payload-holes capability)`)
}

func TestCLICommands_RouteDispatchUnionShapeErrors(t *testing.T) {
	anyOfSpec := strings.Replace(cliRouteDispatchSpec, "              oneOf:\n", "              anyOf:\n", 1)
	requireCLIErrorExact(t, anyOfSpec, cliRouteDispatchManifest,
		`command "jobs run": operation "createJob" uses anyOf; mutually exclusive route dispatch requires oneOf`)

	discriminatorSpec := strings.Replace(cliRouteDispatchSpec,
		"                - $ref: '#/components/schemas/PipelineJobParams'\n",
		"                - $ref: '#/components/schemas/PipelineJobParams'\n              discriminator:\n                propertyName: kind\n", 1)
	requireCLIErrorExact(t, discriminatorSpec, cliRouteDispatchManifest,
		`command "jobs run": request body union uses discriminator "kind"; value-discriminated route dispatch is not part of v1`)

	valueSpec := strings.Replace(cliRouteDispatchSpec, "        engine:\n          type: string\n", "        kind: {type: string, enum: [engine]}\n        engine:\n          type: string\n", 1)
	valueSpec = strings.Replace(valueSpec, "        pipeline:\n          type: string\n", "        kind: {type: string, enum: [pipeline]}\n        pipeline:\n          type: string\n", 1)
	requireCLIErrorExact(t, valueSpec, cliRouteDispatchManifest,
		`command "jobs run": request variants "EngineJobParams" and "PipelineJobParams" are distinguished by values of shared property "kind"; value-discriminated route dispatch is not part of v1`)

	conflictSpec := strings.Replace(cliRouteDispatchSpec, "            schema:\n              oneOf:\n", "            schema:\n              properties:\n                background: {type: string}\n              oneOf:\n", 1)
	requireCLIErrorExact(t, conflictSpec, cliRouteDispatchManifest,
		`command "jobs run": request body union member 1: property "background" is defined beside oneOf and by EngineJobParams with conflicting schemas`)
}

func TestCLICommands_RouteDispatchPresetRules(t *testing.T) {
	commandPreset := strings.Replace(cliRouteDispatchManifest, "    args:\n", "    preset: {$.output_modes: text}\n    args:\n", 1)
	requireCLIErrorExact(t, cliRouteDispatchSpec, commandPreset,
		`command "jobs run": command preset /output_modes is not declared by route "pipeline" (PipelineJobParams); command presets must apply to every route`)

	thirdSpec := strings.Replace(cliRouteDispatchSpec,
		"                - $ref: '#/components/schemas/PipelineJobParams'\n",
		"                - $ref: '#/components/schemas/PipelineJobParams'\n                - $ref: '#/components/schemas/BatchJobParams'\n", 1)
	thirdSpec = strings.Replace(thirdSpec, "        output_modes:\n          type: string\n", "        output_modes:\n          type: string\n        batch:\n          type: string\n", 1)
	thirdSpec += `    BatchJobParams:
      type: object
      required: [batch, input]
      properties:
        batch: {type: string}
        input: {type: string, description: Input content.}
`
	routePreset := strings.Replace(cliRouteDispatchManifest, "        selector: engine\n", "        selector: engine\n        preset: {$.batch: nightly}\n", 1)
	requireCLIErrorExact(t, thirdSpec, routePreset,
		`command "jobs run" route "engine": preset batch is required by another member of the request body union but only optional on the pinned variant EngineJobParams, so a body built from the presets could be read as that other variant; drop that preset (bind an arg or flag instead) or add a discriminator to the union`)
}

// A command-level object preset over a union property shared by every route
// records its discriminator on each route's effective preset set.
func TestCLICommands_RouteDispatchPresetMergePoints(t *testing.T) {
	formatProp := "        format:\n          oneOf:\n            - {$ref: '#/components/schemas/PngFormat'}\n            - {$ref: '#/components/schemas/SvgFormat'}\n"
	spec := strings.ReplaceAll(cliRouteDispatchSpec, "        background:\n", formatProp+"        background:\n") + `    PngFormat:
      type: object
      properties:
        kind: {const: png}
        width: {type: integer}
    SvgFormat:
      type: object
      properties:
        kind: {const: svg}
        scale: {type: number}
`
	manifestYAML := strings.Replace(cliRouteDispatchManifest, "    args:\n", "    preset:\n      $.format: {kind: png}\n    args:\n", 1)
	manifest, err := decodeCLITestSpec(t, spec, manifestYAML)
	require.NoError(t, err)
	want := []CLIPresetMergePoint{{Pointer: "/format", Key: "kind", Values: []any{"png"}}}
	for _, route := range manifest.Commands[0].Source.Routes {
		assert.Equal(t, want, route.PresetMergePoints, route.ID)
	}
}

func TestCLICommands_RouteDispatchCrossRouteFacts(t *testing.T) {
	defaultSpec := strings.Replace(cliRouteDispatchSpec, "        output_modes:\n          type: string\n", "        output_modes:\n          type: string\n        mode:\n          type: string\n          default: one\n", 1)
	defaultSpec = strings.Replace(defaultSpec, "        pipeline_config:\n          type: string\n", "        pipeline_config:\n          type: string\n        mode:\n          type: string\n          default: two\n", 1)
	defaultManifest := strings.Replace(cliRouteDispatchManifest, "      background: {to: $.background}\n", "      mode: {to: $.mode, defaultFrom: schema}\n", 1)
	requireCLIErrorExact(t, defaultSpec, defaultManifest,
		`command "jobs run": flag "mode" defaultFrom: schema requires identical defaults across declaring routes; routes "engine" and "pipeline" disagree`)

	descriptionSpec := strings.Replace(cliRouteDispatchSpec, "        output_modes:\n          type: string\n", "        output_modes:\n          type: string\n        detail: {type: string, description: Engine detail.}\n", 1)
	descriptionSpec = strings.Replace(descriptionSpec, "        pipeline_config:\n          type: string\n", "        pipeline_config:\n          type: string\n        detail: {type: string, description: Pipeline detail.}\n", 1)
	descriptionManifest := strings.Replace(cliRouteDispatchManifest, "      background: {to: $.background}\n", "      detail: {to: $.detail}\n", 1)
	requireCLIErrorExact(t, descriptionSpec, descriptionManifest,
		`command "jobs run": flag "detail" bind /detail has different schema descriptions on routes "engine" and "pipeline"; set summary: explicitly`)

	authoredDefaultSpec := strings.Replace(cliRouteDispatchSpec, "        output_modes:\n          type: string\n", "        output_modes:\n          type: string\n        mode: {type: string, enum: [common, engine]}\n", 1)
	authoredDefaultSpec = strings.Replace(authoredDefaultSpec, "        pipeline_config:\n          type: string\n", "        pipeline_config:\n          type: string\n        mode: {type: string, enum: [common, pipeline]}\n", 1)
	authoredDefaultManifest := strings.Replace(cliRouteDispatchManifest, "      background: {to: $.background}\n", "      mode: {to: $.mode, default: engine}\n", 1)
	requireCLIErrorExact(t, authoredDefaultSpec, authoredDefaultManifest,
		`command "jobs run": flag "mode" default is invalid on route "pipeline": value engine is not in the closed enum (common, pipeline)`)

	enumManifest := strings.Replace(cliRouteDispatchManifest, "      background: {to: $.background}\n", "      mode: {to: $.mode}\n", 1)
	manifest, err := decodeCLITestSpec(t, authoredDefaultSpec, enumManifest)
	require.NoError(t, err)
	assert.Equal(t, []any{"common"}, manifest.Commands[0].Flags[2].Enum)

	normalizeSpec := strings.Replace(cliRouteDispatchSpec, "        output_modes:\n          type: string\n", "        output_modes:\n          type: string\n        labels:\n          type: array\n          items: {type: string}\n", 1)
	normalizeSpec = strings.Replace(normalizeSpec, "        pipeline_config:\n          type: string\n", "        pipeline_config:\n          type: string\n        labels:\n          type: string\n", 1)
	normalizeManifest := strings.Replace(cliRouteDispatchManifest, "    args:\n", "    preset: {$.labels: fixed}\n    args:\n", 1)
	requireCLIErrorExact(t, normalizeSpec, normalizeManifest,
		`command "jobs run": command preset /labels normalizes differently for routes "engine" and "pipeline"; use route-local presets instead`)
}

func TestCLICommands_RouteDispatchDefaultSafety(t *testing.T) {
	pipelineDefault := strings.Replace(cliRouteDispatchManifest, "        default: true\n", "", 1)
	pipelineDefault = strings.Replace(pipelineDefault, "        op: createJob#PipelineJobParams\n", "        op: createJob#PipelineJobParams\n        default: true\n", 1)
	requireCLIErrorExact(t, cliRouteDispatchSpec, pipelineDefault,
		`command "jobs run": default route "pipeline" is not the request union's schema-defaulted variant and its effective preset supplies no distinguishing key; either drop default: or move it to route "engine", whose selector carries the schema default, or preset a distinguishing key on this route`)

	presetDefault := strings.Replace(pipelineDefault, "        default: true\n", "        default: true\n        selector: pipeline\n        preset: {$.pipeline: fixed}\n", 1)
	_, err := decodeCLITestSpec(t, cliRouteDispatchSpec, presetDefault)
	require.NoError(t, err)

	noSchemaDefaultSpec := strings.Replace(cliRouteDispatchSpec, "          default: text-1\n", "", 1)
	noSchemaDefaultSpec = strings.Replace(noSchemaDefaultSpec, "      required: [input]\n", "      required: [engine, input]\n", 1)
	requireCLIErrorExact(t, noSchemaDefaultSpec, cliRouteDispatchManifest,
		`command "jobs run": flag "engine" defaultFrom: schema requires identical defaults across declaring routes; route "engine" has no schema default`)
	noSchemaDefaultManifest := strings.Replace(cliRouteDispatchManifest, "      engine: {to: $.engine, defaultFrom: schema}\n", "      engine: {to: $.engine}\n", 1)
	requireCLIErrorExact(t, noSchemaDefaultSpec, noSchemaDefaultManifest,
		`command "jobs run": default route "engine" is not the request union's schema-defaulted variant and its effective preset supplies no distinguishing key; either drop default: or preset a distinguishing key on this route (the union has no unique routed schema-defaulted variant to move default: to)`)

	// UnionMeta.DefaultJSON excludes optional null defaults from its selector
	// field sets. Dispatch default validation must make the same choice: a null
	// does not select a request variant on the wire.
	nullDefaultSpec := strings.Replace(noSchemaDefaultSpec,
		"        engine:\n          type: string\n",
		"        engine:\n          type: string\n          default: null\n", 1)
	requireCLIErrorExact(t, nullDefaultSpec, noSchemaDefaultManifest,
		`command "jobs run": default route "engine" is not the request union's schema-defaulted variant and its effective preset supplies no distinguishing key; either drop default: or preset a distinguishing key on this route (the union has no unique routed schema-defaulted variant to move default: to)`)
}

// A distinguishing key that carries a schema default but is declared by a
// subset of two members out of three does not select one variant on the wire
// (UnionMeta.DefaultJSON is keyed by that field alone), so a default route on
// either member is rejected unless its preset pins a distinguishing key.
func TestCLICommands_RouteDispatchDefaultSharedSubsetKey(t *testing.T) {
	spec := cliRouteDispatchSpecWithThirdMember()
	spec = strings.Replace(spec, "        batch: {type: string}\n", "        batch: {type: string}\n        engine:\n          type: string\n          default: text-1\n", 1)
	requireCLIErrorExact(t, spec, cliRouteDispatchManifest,
		`command "jobs run": default route "engine" is not the request union's schema-defaulted variant and its effective preset supplies no distinguishing key; either drop default: or preset a distinguishing key on this route (the union has no unique routed schema-defaulted variant to move default: to)`)
}

func TestCLICommands_RouteDispatchOverrideCoverage(t *testing.T) {
	thirdSpec := cliRouteDispatchSpecWithThirdMember()
	override := strings.Replace(cliRouteDispatchManifest, "  jobs run:\n", "  jobs run:\n    override: true\n", 1)
	requireCLIErrorExact(t, thirdSpec, override,
		`command "jobs run": override omits request variant "BatchJobParams"; overriding would remove that variant's only generated CLI surface`)

	full := strings.Replace(override, "      pipeline:\n        op: createJob#PipelineJobParams\n", "      pipeline:\n        op: createJob#PipelineJobParams\n      batch:\n        op: createJob#BatchJobParams\n", 1)
	full = strings.Replace(full, "      background: {to: $.background}\n", "      background: {to: $.background}\n      batch: {to: $.batch}\n", 1)
	manifest, err := decodeCLITestSpec(t, thirdSpec, full)
	require.NoError(t, err)
	assert.True(t, manifest.Commands[0].Override)
	assert.Equal(t, "batch", manifest.Commands[0].Source.Routes[2].Selector)

	inlineSpec := strings.Replace(thirdSpec, "                - $ref: '#/components/schemas/BatchJobParams'\n", "                - type: object\n                  required: [batch, input]\n                  properties:\n                    batch: {type: string}\n                    input: {type: string}\n", 1)
	requireCLIErrorExact(t, inlineSpec, override,
		`command "jobs run": override cannot cover inline member 3 because dispatch routes can pin only component-schema members`)
}

func TestCLICommands_RouteDispatchCompleteMembershipAndWarnings(t *testing.T) {
	thirdSpec := cliRouteDispatchSpecWithThirdMember()
	manifest, warnings, err := decodeCLITestSpecWarn(t, thirdSpec, cliRouteDispatchManifest)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	// The shared input is routed, so it is covered; the unrouted member is
	// named in DispatchKeys, not in routed satisfiability warnings.
	var batchKey, inputKey CLICommandDispatchKey
	for _, key := range manifest.Commands[0].DispatchKeys {
		switch key.Pointer {
		case "/batch":
			batchKey = key
		case "/input":
			inputKey = key
		}
	}
	assert.Empty(t, batchKey.RouteIDs)
	assert.Equal(t, []string{"BatchJobParams"}, batchKey.UnroutedVariants)
	assert.Equal(t, []string{"engine", "pipeline"}, inputKey.RouteIDs)
	assert.Equal(t, []string{"BatchJobParams"}, inputKey.UnroutedVariants)

	warningSpec := strings.Replace(cliRouteDispatchSpec, "      required: [pipeline, input]\n", "      required: [pipeline, input, token_budget]\n", 1)
	warningSpec = strings.Replace(warningSpec, "        pipeline_config:\n          type: string\n", "        pipeline_config:\n          type: string\n        token_budget:\n          type: integer\n", 1)
	_, warnings, err = decodeCLITestSpecWarn(t, warningSpec, cliRouteDispatchManifest)
	require.NoError(t, err)
	assert.Contains(t, warnings, `command "jobs run" route "pipeline": required field "token_budget" is not covered by an arg, flag, preset, or schema default; callers must supply it via --body`)
}

func TestCLICommands_RouteDispatchRequiresOneOf(t *testing.T) {
	directSpec := strings.Replace(cliRouteDispatchSpec,
		"              oneOf:\n                - $ref: '#/components/schemas/EngineJobParams'\n                - $ref: '#/components/schemas/PipelineJobParams'\n",
		"              $ref: '#/components/schemas/EngineJobParams'\n", 1)
	requireCLIErrorExact(t, directSpec, cliRouteDispatchManifest,
		`command "jobs run": route dispatch requires the operation request body to be a oneOf union of component schemas`)
}

func TestCLICommands_VariantSelectors(t *testing.T) {
	// Non-discriminated union: distinguishing keys are required-or-defaulted
	// in some member and absent from at least one. input is required by the
	// render variant (and merely optional on the workflow variant), engine
	// is required with a default, workflow is required by the other member.
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.engine: render-pro-2
  flow:
    op: CreateTask#CreateWorkflowTaskParams
  upload:
    op: CreateUpload
    preset:
      $.name: fixed
`)
	require.NoError(t, err)

	render := manifest.Commands[0].Source.Routes[0].Selectors
	require.NotNil(t, render)
	assert.Equal(t, []string{"input", "engine"}, render.Own)
	assert.Equal(t, []string{"workflow"}, render.Foreign)
	assert.Empty(t, render.DiscriminatorKey)

	flow := manifest.Commands[1].Source.Routes[0].Selectors
	require.NotNil(t, flow)
	// input is a (optional) property of the workflow variant too, so it is
	// never a foreign selector for it.
	assert.Equal(t, []string{"input", "workflow"}, flow.Own)
	assert.Equal(t, []string{"engine"}, flow.Foreign)

	// A direct-$ref (non-union) body has nothing to select.
	assert.Nil(t, manifest.Commands[2].Source.Routes[0].Selectors)

	// A preset that a sibling requires while the pinned variant merely
	// permits it (input: required by the render variant, optional on the
	// workflow variant) could be read as the sibling once merged into a body
	// that names no variant; without a discriminator nothing pins the
	// variant for certain, so the preset is rejected. Binding the same
	// property to an arg is fine: arguments never reach a partial body.
	requireDecodeError(t, `
version: 1
commands:
  flow:
    op: CreateTask#CreateWorkflowTaskParams
    preset:
      $.input: fixed
      $.workflow: nightly
`, `preset input is required by another member`)
	_, _, err = decodeCLITest(t, `
version: 1
commands:
  flow:
    op: CreateTask#CreateWorkflowTaskParams
    args:
      text: {to: $.input, variadic: true}
    preset:
      $.workflow: nightly
`)
	require.NoError(t, err)
}

const cliDiscriminatedSpec = `openapi: 3.1.0
info:
  title: Jobs
  version: 1.0.0
paths:
  /jobs:
    post:
      operationId: CreateJob
      requestBody:
        content:
          application/json:
            schema:
              oneOf:
                - $ref: '#/components/schemas/ModelJob'
                - $ref: '#/components/schemas/AgentJob'
              discriminator:
                propertyName: kind
                mapping:
                  model: '#/components/schemas/ModelJob'
                  agent: '#/components/schemas/AgentJob'
      responses:
        "200":
          description: ok
  /jobs/unmapped:
    post:
      operationId: CreateUnmappedJob
      requestBody:
        content:
          application/json:
            schema:
              oneOf:
                - $ref: '#/components/schemas/LooseModelJob'
                - $ref: '#/components/schemas/LooseAgentJob'
              discriminator:
                propertyName: kind
      responses:
        "200":
          description: ok
  /jobs/aliased:
    post:
      operationId: CreateAliasedJob
      requestBody:
        content:
          application/json:
            schema:
              oneOf:
                - $ref: '#/components/schemas/ModelJob'
                - $ref: '#/components/schemas/AgentJob'
              discriminator:
                propertyName: kind
                mapping:
                  model: '#/components/schemas/ModelJob'
                  llm: '#/components/schemas/ModelJob'
                  agent: '#/components/schemas/AgentJob'
      responses:
        "200":
          description: ok
  /jobs/mixed:
    post:
      operationId: CreateMixedJob
      requestBody:
        content:
          application/json:
            schema:
              oneOf:
                - $ref: '#/components/schemas/ModelJob'
                - type: object
                  required: [agent, input]
                  properties:
                    agent: {type: string}
                    input: {type: string}
      responses:
        "200":
          description: ok
  /jobs/ref:
    post:
      operationId: CreateRefJob
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/JobUnion'
      responses:
        "200":
          description: ok
components:
  schemas:
    JobUnion:
      oneOf:
        - $ref: '#/components/schemas/ModelJob'
        - $ref: '#/components/schemas/AgentJob'
    ModelJob:
      type: object
      required: [kind, model, input]
      properties:
        kind: {type: string, enum: [model]}
        model: {type: string}
        input: {type: string}
        shared: {type: string}
    AgentJob:
      type: object
      required: [kind, agent, input]
      properties:
        kind: {type: string, enum: [agent]}
        agent: {type: string}
        input: {type: string}
        shared: {type: string}
    LooseModelJob:
      type: object
      required: [kind, input]
      properties:
        kind: {type: string}
        input: {type: string}
    LooseAgentJob:
      type: object
      required: [kind, input]
      properties:
        kind: {type: string}
        input: {type: string}
`

func TestCLICommands_VariantSelectorsDiscriminated(t *testing.T) {
	manifest, err := decodeCLITestSpec(t, cliDiscriminatedSpec, `
version: 1
commands:
  model:
    op: CreateJob#ModelJob
    preset:
      $.model: standard-2
  loose:
    op: CreateUnmappedJob#LooseAgentJob
    preset:
      $.kind: LooseAgentJob
  aliased:
    op: CreateAliasedJob#ModelJob
  mixed:
    op: CreateMixedJob#ModelJob
`)
	require.NoError(t, err)

	// The single-value enum discriminator is reported by key+value (a
	// different value under the same key is the conflict), never as a
	// distinguishing key; the remaining required keys split as usual.
	model := manifest.Commands[0].Source.Routes[0].Selectors
	require.NotNil(t, model)
	assert.Equal(t, "kind", model.DiscriminatorKey)
	assert.Equal(t, "model", model.DiscriminatorValue)
	assert.Equal(t, []any{"model"}, model.DiscriminatorAliases)
	assert.Equal(t, []string{"model"}, model.Own)
	assert.Equal(t, []string{"agent"}, model.Foreign)

	// No mapping and no const property: the implicit mapping applies (the
	// value is the component name), and a preset must agree with it.
	loose := manifest.Commands[1].Source.Routes[0].Selectors
	require.NotNil(t, loose)
	assert.Equal(t, "kind", loose.DiscriminatorKey)
	assert.Equal(t, "LooseAgentJob", loose.DiscriminatorValue)
	assert.Empty(t, loose.Own)
	assert.Empty(t, loose.Foreign)
	_, err = decodeCLITestSpec(t, cliDiscriminatedSpec, `
version: 1
commands:
  loose:
    op: CreateUnmappedJob#LooseAgentJob
    preset:
      $.kind: agent
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not select the pinned variant LooseAgentJob")

	// Mapping aliases: every value that maps to the pinned variant is
	// accepted; the schema's own const value is the fill value, the implicit
	// component name is not accepted (a mapping entry exists).
	aliased := manifest.Commands[2].Source.Routes[0].Selectors
	require.NotNil(t, aliased)
	assert.Equal(t, "model", aliased.DiscriminatorValue)
	assert.Equal(t, []any{"model", "llm"}, aliased.DiscriminatorAliases)

	// An inline sibling still contributes its selectors.
	mixed := manifest.Commands[3].Source.Routes[0].Selectors
	require.NotNil(t, mixed)
	assert.Equal(t, []string{"model"}, mixed.Own)
	assert.Equal(t, []string{"agent"}, mixed.Foreign)

	// A body that references a union component (rather than declaring the
	// oneOf inline) is the same union: pinning works and selectors resolve.
	viaRef, err := decodeCLITestSpec(t, cliDiscriminatedSpec, `
version: 1
commands:
  ref-model:
    op: CreateRefJob#ModelJob
`)
	require.NoError(t, err)
	refSel := viaRef.Commands[0].Source.Routes[0].Selectors
	require.NotNil(t, refSel)
	assert.Equal(t, []string{"model"}, refSel.Own)
	assert.Equal(t, []string{"agent"}, refSel.Foreign)
	_, err = decodeCLITestSpec(t, cliDiscriminatedSpec, `
version: 1
commands:
  ref-bare:
    op: CreateRefJob
    preset:
      $.model: standard-2
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pin a variant")

}

const cliNestedPresetSpec = `openapi: 3.1.0
info:
  title: Drawings
  version: 1.0.0
paths:
  /drawings:
    post:
      operationId: CreateDrawing
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                prompt:
                  type: string
                format:
                  oneOf:
                    - $ref: '#/components/schemas/Format'
                    - type: array
                      items:
                        $ref: '#/components/schemas/Format'
                mapped:
                  $ref: '#/components/schemas/MappedFormat'
                described:
                  $ref: '#/components/schemas/MappedFormat'
                  description: Same union behind a $ref with sibling keywords
                implicit:
                  $ref: '#/components/schemas/ImplicitFormat'
                source:
                  oneOf:
                    - $ref: '#/components/schemas/UrlSource'
                    - $ref: '#/components/schemas/FileSource'
                versioned:
                  oneOf:
                    - $ref: '#/components/schemas/VersionedPng'
                    - $ref: '#/components/schemas/VersionedSvg'
                tiered:
                  oneOf:
                    - $ref: '#/components/schemas/VersionedPng'
                    - $ref: '#/components/schemas/VersionedSvg'
                    - $ref: '#/components/schemas/TieredPdf'
                custom:
                  oneOf:
                    - $ref: '#/components/schemas/PngFormat'
                    - type: object
                      additionalProperties:
                        type: string
                config:
                  type: object
                  properties:
                    dpi:
                      type: integer
                    serial:
                      type: string
                      readOnly: true
                    format:
                      $ref: '#/components/schemas/Format'
                palette:
                  type: object
                  additionalProperties:
                    $ref: '#/components/schemas/Format'
                layered:
                  allOf:
                    - type: object
                      additionalProperties:
                        $ref: '#/components/schemas/Format'
                inclusive:
                  anyOf:
                    - $ref: '#/components/schemas/PngFormat'
                    - $ref: '#/components/schemas/SvgFormat'
                    - type: object
                      additionalProperties: true
                nullable:
                  anyOf:
                    - $ref: '#/components/schemas/PngFormat'
                    - $ref: '#/components/schemas/SvgFormat'
                    - type: [object, "null"]
                      additionalProperties: true
                empty:
                  oneOf:
                    - $ref: '#/components/schemas/PngFormat'
                    - {}
                remapped:
                  oneOf:
                    - $ref: '#/components/schemas/SwappedFormat'
                    - $ref: '#/components/schemas/MappedFormat'
                scoped:
                  oneOf:
                    - $ref: '#/components/schemas/ImplicitFormat'
                    - $ref: '#/components/schemas/ApiFormat'
                composed:
                  allOf:
                    - type: object
                      properties:
                        format:
                          $ref: '#/components/schemas/Format'
                    - oneOf:
                        - type: object
                          properties:
                            mode:
                              const: fast
                        - type: object
                          properties:
                            mode:
                              const: slow
                labelled:
                  oneOf:
                    - $ref: '#/components/schemas/PngFormat'
                    - $ref: '#/components/schemas/SvgFormat'
                    - type: object
                      required: [label]
                      additionalProperties:
                        type: string
                constant:
                  oneOf:
                    - $ref: '#/components/schemas/PngFormat'
                    - const:
                        kind: svg
                wrapped:
                  allOf:
                    - type: object
                    - anyOf:
                        - $ref: '#/components/schemas/PngFormat'
                        - $ref: '#/components/schemas/SvgFormat'
                        - type: object
                          additionalProperties: true
                refined:
                  allOf:
                    - type: object
                      properties:
                        format:
                          $ref: '#/components/schemas/Format'
                    - oneOf:
                        - type: object
                          properties:
                            mode:
                              const: fast
                            format:
                              type: object
                        - type: object
                          properties:
                            mode:
                              const: slow
                            format:
                              type: object
                listed:
                  enum:
                    - kind: png
                    - preset: vector
                typedListed:
                  type: object
                  enum:
                    - kind: png
                    - preset: vector
                listedMember:
                  oneOf:
                    - enum:
                        - kind: png
                        - preset: vector
                    - type: string
                mixed:
                  oneOf:
                    - $ref: '#/components/schemas/PngFormat'
                    - type: string
                  discriminator:
                    propertyName: kind
                listedFields:
                  type: object
                  properties:
                    kind:
                      type: string
                    width:
                      type: integer
                  enum:
                    - kind: png
                    - kind: svg
                      width: 10
                listedUnion:
                  enum:
                    - kind: png
                    - kind: svg
                  oneOf:
                    - $ref: '#/components/schemas/PngFormat'
                    - $ref: '#/components/schemas/SvgFormat'
                overlap:
                  oneOf:
                    - $ref: '#/components/schemas/PngFormat'
                    - $ref: '#/components/schemas/SvgFormat'
                    - type: object
                      properties:
                        kind:
                          type: string
                          enum: [png, raw]
                        label:
                          type: string
                plain:
                  type: object
                  properties:
                    kind:
                      type: string
                    width:
                      type: integer
      responses:
        "200":
          description: OK
components:
  schemas:
    Format:
      oneOf:
        - $ref: '#/components/schemas/PngFormat'
        - $ref: '#/components/schemas/SvgFormat'
        - type: object
          additionalProperties: true
    SwappedFormat:
      oneOf:
        - $ref: '#/components/schemas/VersionedSvg'
      discriminator:
        propertyName: kind
        mapping:
          portable: '#/components/schemas/VersionedSvg'
    ApiFormat:
      oneOf:
        - $ref: '#/components/schemas/VersionedSvg'
      discriminator:
        propertyName: api
        mapping:
          png: '#/components/schemas/VersionedSvg'
    MappedFormat:
      oneOf:
        - $ref: '#/components/schemas/PngFormat'
        - $ref: '#/components/schemas/SvgFormat'
      discriminator:
        propertyName: kind
        mapping:
          png: '#/components/schemas/PngFormat'
          portable: '#/components/schemas/PngFormat'
          svg: '#/components/schemas/SvgFormat'
    ImplicitFormat:
      oneOf:
        - $ref: '#/components/schemas/PngFormat'
        - $ref: '#/components/schemas/SvgFormat'
      discriminator:
        propertyName: kind
    UrlSource:
      type: object
      required: [url]
      properties:
        url:
          type: string
    FileSource:
      type: object
      required: [path]
      properties:
        path:
          type: string
    VersionedPng:
      type: object
      properties:
        api:
          const: v1
        kind:
          const: png
    VersionedSvg:
      type: object
      properties:
        api:
          const: v1
        kind:
          const: svg
    TieredPdf:
      type: object
      properties:
        api:
          const: v2
        kind:
          const: pdf
    PngFormat:
      type: object
      required: [kind]
      properties:
        kind:
          const: png
        width:
          type: integer
    SvgFormat:
      type: object
      required: [kind]
      properties:
        kind:
          type: string
          enum: [svg]
        scale:
          type: number
`

// An object-valued preset records a merge point for every object of its value
// the schema proves safe to fill into a caller's object: plain objects and
// map entries, and union members the preset identifies uniquely, guarded by
// their discriminator. Unions without one (no discriminator, overlapping
// accepted values, map-shaped members) get none.
func TestCLICommands_PresetMergePoints(t *testing.T) {
	manifest, err := decodeCLITestSpec(t, cliNestedPresetSpec, `
version: 1
commands:
  draw:
    op: CreateDrawing
    preset:
      $.format:
        kind: png
      $.mapped:
        kind: png
        width: 10
      $.plain:
        kind: png
      $.described:
        kind: png
      $.implicit:
        kind: png
      $.source:
        url: https://example.com/a
      $.versioned:
        api: v1
        kind: png
      $.tiered:
        api: v1
        kind: png
      $.custom:
        kind: png
      $.config:
        dpi: 72
        format:
          kind: png
          width: 10
      $.palette:
        logo:
          kind: png
          width: 10
      $.layered:
        logo:
          kind: png
      $.nullable:
        kind: png
      $.inclusive:
        kind: png
      $.empty:
        kind: png
      $.remapped:
        kind: png
      $.scoped:
        kind: png
      $.composed:
        mode: fast
        format:
          kind: png
          width: 10
      $.labelled:
        kind: png
        width: 10
      $.constant:
        kind: png
      $.wrapped:
        kind: png
      $.refined:
        mode: fast
        format:
          kind: png
          width: 10
      $.listed:
        kind: png
      $.typedListed:
        kind: png
      $.listedMember:
        kind: png
      $.listedUnion:
        kind: png
      $.overlap:
        kind: png
`)
	require.NoError(t, err)
	got := map[string]CLIPresetMergePoint{}
	for _, disc := range manifest.Commands[0].Source.Routes[0].PresetMergePoints {
		got[disc.Pointer] = disc
	}
	assert.Equal(t, map[string]CLIPresetMergePoint{
		"/format":          {Pointer: "/format", Key: "kind", Values: []any{"png"}},
		"/mapped":          {Pointer: "/mapped", Key: "kind", Values: []any{"png", "portable"}},
		"/described":       {Pointer: "/described", Key: "kind", Values: []any{"png", "portable"}},
		"/implicit":        {Pointer: "/implicit", Key: "kind", Values: []any{"png", "PngFormat"}},
		"/versioned":       {Pointer: "/versioned", Key: "kind", Values: []any{"png"}},
		"/tiered":          {Pointer: "/tiered", Key: "kind", Values: []any{"png"}},
		"/config/format":   {Pointer: "/config/format", Key: "kind", Values: []any{"png"}},
		"/plain":           {Pointer: "/plain"},
		"/config":          {Pointer: "/config"},
		"/palette":         {Pointer: "/palette"},
		"/palette/logo":    {Pointer: "/palette/logo", Key: "kind", Values: []any{"png"}},
		"/layered":         {Pointer: "/layered"},
		"/layered/logo":    {Pointer: "/layered/logo", Key: "kind", Values: []any{"png"}},
		"/scoped":          {Pointer: "/scoped", Key: "kind", Values: []any{"png", "PngFormat"}},
		"/remapped":        {Pointer: "/remapped", Key: "kind", Values: []any{"png"}},
		"/composed":        {Pointer: "/composed", Key: "mode", Values: []any{"fast"}},
		"/composed/format": {Pointer: "/composed/format", Key: "kind", Values: []any{"png"}},
		"/inclusive":       {Pointer: "/inclusive", Key: "kind", Values: []any{"png"}},
		"/nullable":        {Pointer: "/nullable", Key: "kind", Values: []any{"png"}},
		"/wrapped":         {Pointer: "/wrapped", Key: "kind", Values: []any{"png"}},
		"/refined":         {Pointer: "/refined", Key: "mode", Values: []any{"fast"}},
		"/refined/format":  {Pointer: "/refined/format", Key: "kind", Values: []any{"png"}},
	}, got)
}

func TestCLICommands_NestedPresetBinds(t *testing.T) {
	manifest, err := decodeCLITestSpec(t, cliNestedPresetSpec, `
version: 1
commands:
  draw:
    op: CreateDrawing
    preset:
      $.format:
        kind: png
      $.mapped:
        kind: png
        width: 10
      $.plain:
        kind: png
    flags:
      width: {to: $.format.width}
      mapped-width: {to: $.mapped.width}
      plain-width: {to: $.plain.width}
`)
	require.NoError(t, err)
	flags := map[string]CLICommandInput{}
	for _, flag := range manifest.Commands[0].Flags {
		flags[flag.Name] = flag
	}
	assert.Equal(t, "/format/width", flags["width"].Bind.Pointer)
	assert.Equal(t, "int", flags["width"].Type)
	assert.Nil(t, flags["width"].Default)
	assert.Equal(t, "int", flags["mapped-width"].Type)
	assert.Equal(t, "preset", flags["mapped-width"].DefaultFrom)
	assert.EqualValues(t, 10, flags["mapped-width"].Default)
	assert.Equal(t, "int", flags["plain-width"].Type)
}

func TestCLICommands_NestedPresetBindErrors(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"no preset", "flags:\n      dpi: {to: $.config.dpi}", `flag "dpi" bind /config/dpi nests beneath /config, which no object preset sets`},
		{"scalar preset", "preset:\n      $.prompt: hello\n    flags:\n      x: {to: $.prompt.x}", `flag "x" bind /prompt/x nests beneath /prompt, which no object preset sets`},
		{"discriminator", "preset:\n      $.format: {kind: png}\n    flags:\n      kind: {to: $.format.kind}", `targets the discriminator of the union at /format, which the preset pins`},
		{"discriminator of a single object member", "preset:\n      $.mixed: {kind: png}\n    flags:\n      kind: {to: $.mixed.kind, type: string}", `targets the discriminator of the union at /mixed, which the preset pins`},
		{"no unique member", "preset:\n      $.source: {url: https://example.com/a}\n    flags:\n      url: {to: $.source.url}", `the preset at /source does not select exactly one member of its union`},
		{"other member field", "preset:\n      $.format: {kind: png}\n    flags:\n      scale: {to: $.format.scale}", `flag "scale" bind beneath /format: /scale does not resolve in PngFormat`},
		{"typo", "preset:\n      $.format: {kind: png}\n    flags:\n      width: {to: $.format.widht}", `did you mean "width"`},
		{"readOnly field", "preset:\n      $.config: {dpi: 72}\n    flags:\n      serial: {to: $.config.serial}", `flag "serial" bind beneath /config: /config/serial targets a readOnly property`},
		{"const/enum object", "preset:\n      $.listedFields: {kind: png}\n    flags:\n      width: {to: $.listedFields.width}", `flag "width" bind /listedFields/width nests beneath /listedFields, whose schema restricts the object to const/enum values`},
		{"positional", "preset:\n      $.format: {kind: png}\n    args:\n      w: {to: $.format.width, variadic: true, type: string}", `arg "w" binds /format/width; the positional binds a top-level body field`},
		{"parent flag first", "preset:\n      $.format: {kind: png}\n    flags:\n      format: {to: $.format, type: string}\n      width: {to: $.format.width}", `input "format" binds /format and input "width" binds /format/width beneath it`},
		{"nested flag first", "preset:\n      $.format: {kind: png}\n    flags:\n      width: {to: $.format.width}\n      format: {to: $.format, type: string}", `input "format" binds /format and input "width" binds /format/width beneath it`},
		{"positional parent", "preset:\n      $.format: {kind: png}\n    args:\n      format: {to: $.format, variadic: true, type: string}\n    flags:\n      width: {to: $.format.width}", `input "format" binds /format and input "width" binds /format/width beneath it`},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := decodeCLITestSpec(t, cliNestedPresetSpec, "\nversion: 1\ncommands:\n  draw:\n    op: CreateDrawing\n    "+testCase.input+"\n")
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.want)
		})
	}
}

func TestCLICommands_NestedBindRejectedOnRouteDispatch(t *testing.T) {
	formatProp := "        format:\n          oneOf:\n            - {$ref: '#/components/schemas/PngFormat'}\n            - {$ref: '#/components/schemas/SvgFormat'}\n"
	spec := strings.ReplaceAll(cliRouteDispatchSpec, "        background:\n", formatProp+"        background:\n") + `    PngFormat:
      type: object
      properties:
        kind: {const: png}
        width: {type: integer}
    SvgFormat:
      type: object
      properties:
        kind: {const: svg}
        scale: {type: number}
`
	manifestYAML := strings.Replace(cliRouteDispatchManifest, "    args:\n", "    preset:\n      $.format: {kind: png}\n    args:\n", 1) + "      width: {to: $.format.width}\n"
	_, err := decodeCLITestSpec(t, spec, manifestYAML)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `flag "width" binds /format/width; nested binds are not part of route dispatch`)
}

func TestCLICommands_InputNamePatternHyphens(t *testing.T) {
	for _, name := range []string{"a", "a1", "a-b", "a-b-c", "a1-2b", "engine-mode"} {
		assert.True(t, cliInputNamePattern.MatchString(name), name)
	}
	for _, name := range []string{"", "-", "a-", "-a", "a--b", "a-b-", "1a", "A", "a_b", "a b"} {
		assert.False(t, cliInputNamePattern.MatchString(name), name)
	}
}

func TestCLICommands_HumanizeRouteIDNeverPanics(t *testing.T) {
	tests := map[string]string{
		"":           "",
		"-":          "",
		"a":          "A",
		"engine":     "Engine",
		"a-b":        "A B",
		"a--b":       "A B",
		"a-":         "A",
		"-a":         "A",
		"text-to-3d": "Text To 3d",
	}
	for id, want := range tests {
		assert.NotPanics(t, func() {
			assert.Equal(t, want, cliHumanizeRouteID(id), id)
		}, id)
	}
}

// A defaultFrom: schema default is emitted into Go source exactly like an
// explicit default, so a schema default whose YAML type disagrees with the
// property's declared type must be a validation error on every path that
// resolves it: operation-level flags, single-route command inputs, and
// dispatch command inputs.
func TestCLICommands_SchemaDefaultTypeMismatchIsAnError(t *testing.T) {
	mistypedSpec := strings.Replace(cliCommandsTestSpec,
		"            priority:\n              type: integer\n",
		"            priority:\n              type: integer\n              default: high\n", 1)
	require.NotEqual(t, cliCommandsTestSpec, mistypedSpec)

	t.Run("operation flag", func(t *testing.T) {
		// Operation flags must bind in every request body variant, so the
		// shared boolean stream property carries the mistyped default.
		spec := strings.ReplaceAll(cliCommandsTestSpec, "type: boolean\n              default: true\n", "type: boolean\n              default: 1\n")
		spec = strings.ReplaceAll(spec, "type: boolean\n          default: true\n", "type: boolean\n          default: 1\n")
		require.NotEqual(t, cliCommandsTestSpec, spec)
		requireCLIErrorExact(t, spec, `
version: 1
operations:
  CreateTask:
    flags:
      stream: {to: $.stream, defaultFrom: schema}
`, `operation "CreateTask" flag "stream" defaultFrom: schema: the schema default at /stream does not match type bool: value 1 is not a boolean`)
	})

	t.Run("single-route command flag", func(t *testing.T) {
		requireCLIErrorExact(t, mistypedSpec, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      priority: {to: $.priority, defaultFrom: schema}
`, `command "render" flag "priority" defaultFrom: schema: the schema default at /priority does not match type int: value high is not an integer`)
	})

	t.Run("dispatch command flag", func(t *testing.T) {
		spec := strings.Replace(cliRouteDispatchSpec,
			"        output_modes:\n          type: string\n",
			"        output_modes:\n          type: string\n        retries:\n          type: integer\n          default: three\n", 1)
		spec = strings.Replace(spec,
			"        pipeline_config:\n          type: string\n",
			"        pipeline_config:\n          type: string\n        retries:\n          type: integer\n          default: three\n", 1)
		require.NotEqual(t, cliRouteDispatchSpec, spec)
		manifest := strings.Replace(cliRouteDispatchManifest,
			"      background: {to: $.background}\n",
			"      retries: {to: $.retries, defaultFrom: schema}\n", 1)
		requireCLIErrorExact(t, spec, manifest,
			`command "jobs run" flag "retries" defaultFrom: schema: the schema default at /retries does not match type int: value three is not an integer`)

		// The well-typed case is normalized to the explicit-default
		// representation on the dispatch path too.
		wellTyped := strings.ReplaceAll(spec, "default: three", "default: 2")
		linked, _, err := decodeCLITestSpecWarn(t, wellTyped, manifest)
		require.NoError(t, err)
		require.Len(t, linked.Commands, 1)
		var retries *CLICommandInput
		for i := range linked.Commands[0].Flags {
			if linked.Commands[0].Flags[i].Name == "retries" {
				retries = &linked.Commands[0].Flags[i]
			}
		}
		require.NotNil(t, retries, "the retries flag must survive linking")
		assert.Equal(t, int64(2), retries.Default)
	})

	t.Run("well-typed schema default still links", func(t *testing.T) {
		spec := strings.Replace(cliCommandsTestSpec,
			"            priority:\n              type: integer\n",
			"            priority:\n              type: integer\n              default: 3\n", 1)
		manifest, _, err := decodeCLITestSpecWarn(t, spec, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      priority: {to: $.priority, defaultFrom: schema}
`)
		require.NoError(t, err)
		require.Len(t, manifest.Commands, 1)
		require.Len(t, manifest.Commands[0].Flags, 1)
		assert.Equal(t, int64(3), manifest.Commands[0].Flags[0].Default)
	})
}

// Dispatch commands lint their jq projection exactly as single-route commands
// do: a projection that cannot parse, or that provably selects nothing from
// the shared operation's response schema, is a generation error.
func TestCLICommands_RouteDispatchProjectionIsLinted(t *testing.T) {
	t.Run("parse failure", func(t *testing.T) {
		manifest := strings.TrimSuffix(cliRouteDispatchManifest, "\n") + "\n    jq: '.id |'\n"
		requireCLIErrorExact(t, cliRouteDispatchSpec, manifest,
			`command "jobs run": jq projection ".id |" is not valid jq: unexpected EOF`)
	})

	t.Run("missing field against the response schema", func(t *testing.T) {
		spec := strings.Replace(cliRouteDispatchSpec,
			"        \"200\": {description: ok}\n",
			"        \"200\":\n          description: ok\n          content:\n            application/json:\n              schema:\n                type: object\n                properties:\n                  id: {type: string}\n", 1)
		require.NotEqual(t, cliRouteDispatchSpec, spec)
		manifest := strings.TrimSuffix(cliRouteDispatchManifest, "\n") + "\n    jq: .identifier\n"
		requireCLIErrorExact(t, spec, manifest,
			`command "jobs run": jq projection ".identifier" always returns nothing — ".identifier" does not exist in the response`)

		sound := strings.TrimSuffix(cliRouteDispatchManifest, "\n") + "\n    jq: .id\n"
		_, warnings, err := decodeCLITestSpecWarn(t, spec, sound)
		require.NoError(t, err)
		assert.Empty(t, warnings)
	})

	t.Run("no response schema warns instead of failing", func(t *testing.T) {
		manifest := strings.TrimSuffix(cliRouteDispatchManifest, "\n") + "\n    jq: .id\n"
		_, warnings, err := decodeCLITestSpecWarn(t, cliRouteDispatchSpec, manifest)
		require.NoError(t, err)
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], `command "jobs run": jq projection ".id" cannot be statically verified`)
	})
}

func TestCLICommands_InputSuggestions(t *testing.T) {
	// A declared suggestions: list replaces the schema-derived one. It is
	// display-only, so values outside the schema enum (open or closed) are
	// accepted; only the input type bounds it.
	manifest, warnings, err := decodeCLITest(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    args:
      prompt: {to: $.input, type: string, variadic: true, suggestions: [hello, world]}
    flags:
      engine: {to: $.engine, defaultFrom: schema, suggestions: [render-pro-2, render-next-1]}
      rate: {to: $.sample_rate, suggestions: [8000, 96000]}
      priority: {to: $.priority, suggestions: []}
      background: {to: $.background}
  workflow:
    op: CreateTask#CreateWorkflowTaskParams
    flags:
      workflow: {to: $.workflow, suggestions: [nightly, hourly]}
      mode: {to: $.mode, suggestions: [turbo]}
  workflow-plain:
    op: CreateTask#CreateWorkflowTaskParams
    flags:
      workflow: {to: $.workflow}
      mode: {to: $.mode}
`)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, manifest.Commands, 3)

	render := manifest.Commands[0]
	require.Len(t, render.Args, 1)
	assert.Equal(t, []any{"hello", "world"}, render.Args[0].Enum)
	assert.True(t, render.Args[0].SuggestionsDeclared)
	require.Len(t, render.Flags, 4)
	assert.Equal(t, []any{"render-pro-2", "render-next-1"}, render.Flags[0].Enum)
	assert.True(t, render.Flags[0].SuggestionsDeclared)
	assert.Equal(t, "render-standard-2", render.Flags[0].Default)
	assert.Equal(t, []any{int64(8000), int64(96000)}, render.Flags[1].Enum)
	assert.Equal(t, "int", render.Flags[1].Type)
	assert.Empty(t, render.Flags[2].Enum)
	assert.True(t, render.Flags[2].SuggestionsDeclared)
	assert.Empty(t, render.Flags[3].Enum)
	assert.False(t, render.Flags[3].SuggestionsDeclared)

	workflow := manifest.Commands[1]
	require.Len(t, workflow.Flags, 2)
	assert.Equal(t, []any{"nightly", "hourly"}, workflow.Flags[0].Enum)
	assert.Equal(t, []any{"turbo"}, workflow.Flags[1].Enum)
	plain := manifest.Commands[2]
	require.Len(t, plain.Flags, 2)
	assert.Equal(t, []any{"batch", "streaming"}, plain.Flags[1].Enum)
	assert.False(t, plain.Flags[1].SuggestionsDeclared)

	// Values must match the bound property's type, including on open schemas.
	requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      priority: {to: $.priority, suggestions: [high]}
`, `command "render" flag "priority" bind /priority suggestions: value high is not an integer`)
	requireDecodeError(t, `
version: 1
commands:
  note:
    op: CreateNote
    flags:
      extra: {to: $.extra, type: int, suggestions: [fast]}
`, `command "note" flag "extra" bind /extra suggestions: value fast is not an integer`)

	// Manifest shape.
	for _, tc := range []struct{ suggestions, want string }{
		{"fast", `input "engine": suggestions must be a list of scalar values`},
		{"{a: b}", `input "engine": suggestions must be a list of scalar values`},
		{"[{name: fast}]", `input "engine": suggestions entries must be scalar values`},
		{"[fast, ~]", `input "engine": suggestions entries cannot be null`},
		{"[fast, fast]", `input "engine": suggestions lists fast more than once`},
	} {
		requireDecodeError(t, `
version: 1
commands:
  render:
    op: CreateTask#CreateRenderTaskParams
    flags:
      engine: {to: $.engine, suggestions: `+tc.suggestions+`}
`, tc.want)
	}
}

func TestCLICommands_InputSuggestionsOperationsAndDispatch(t *testing.T) {
	manifest, warnings, err := decodeCLITest(t, `
version: 1
operations:
  CreateUpload:
    flags:
      name: {to: $.name, suggestions: [alpha, beta]}
`)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, manifest.Operations, 1)
	require.Len(t, manifest.Operations[0].Flags, 1)
	assert.Equal(t, []any{"alpha", "beta"}, manifest.Operations[0].Flags[0].Enum)
	assert.True(t, manifest.Operations[0].Flags[0].SuggestionsDeclared)

	requireDecodeError(t, `
version: 1
operations:
  CreateUpload:
    flags:
      name: {to: $.name, suggestions: [1]}
`, `operation "CreateUpload" flag "name" suggestions: value 1 is not a string`)

	// Route dispatch: a declared list wins over the cross-route intersection.
	spec := strings.Replace(cliRouteDispatchSpec, "        output_modes:\n          type: string\n", "        output_modes:\n          type: string\n        mode: {type: string, enum: [common, engine]}\n", 1)
	spec = strings.Replace(spec, "        pipeline_config:\n          type: string\n", "        pipeline_config:\n          type: string\n        mode: {type: string, enum: [common, pipeline]}\n", 1)
	authored := strings.Replace(cliRouteDispatchManifest, "      background: {to: $.background}\n", "      mode: {to: $.mode, suggestions: [common, engine]}\n      engine-alias: {to: $.engine, suggestions: [text-2]}\n", 1)
	authored = strings.Replace(authored, "      engine: {to: $.engine, defaultFrom: schema}\n", "", 1)
	authored = strings.Replace(authored, "        selector: engine\n", "        selector: engine-alias\n", 1)
	dispatch, err := decodeCLITestSpec(t, spec, authored)
	require.NoError(t, err)
	require.Len(t, dispatch.Commands[0].Flags, 3)
	assert.Equal(t, "mode", dispatch.Commands[0].Flags[1].Name)
	assert.Equal(t, []any{"common", "engine"}, dispatch.Commands[0].Flags[1].Enum)
	assert.True(t, dispatch.Commands[0].Flags[1].SuggestionsDeclared)
	assert.Equal(t, "engine-alias", dispatch.Commands[0].Flags[2].Name)
	assert.Equal(t, []any{"text-2"}, dispatch.Commands[0].Flags[2].Enum)
}

func TestCLICommands_PresetShownAsFlagDefault(t *testing.T) {
	// The effective preset is what the request carries when the flag is
	// omitted, so it is the displayed default regardless of defaultFrom or
	// an authored default; schema suggestions are kept.
	manifest, warnings, err := decodeCLITest(t, `
version: 1
commands:
  image:
    op: CreateTask#CreateRenderTaskParams
    preset:
      $.engine: render-image-9
      $.background: true
      $.priority: 3
    args:
      prompt: {to: $.input, type: string, variadic: true}
    flags:
      engine: {to: $.engine, defaultFrom: schema}
      background: {to: $.background}
      priority: {to: $.priority, default: 1}
`)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	flags := manifest.Commands[0].Flags
	require.Len(t, flags, 3)
	assert.Equal(t, "preset", flags[0].DefaultFrom)
	assert.Equal(t, "render-image-9", flags[0].Default)
	assert.Equal(t, []any{"render-standard-2", "render-pro-2", "render-mini-1"}, flags[0].Enum)
	assert.Equal(t, "preset", flags[1].DefaultFrom)
	assert.Equal(t, true, flags[1].Default)
	assert.Equal(t, "preset", flags[2].DefaultFrom)
	assert.Equal(t, int64(3), flags[2].Default)

	// An explicitly open schema: the preset still becomes the default when
	// it matches the declared flag type, and is ignored when it does not.
	manifest, _, err = decodeCLITest(t, `
version: 1
commands:
  note:
    op: CreateNote
    preset:
      $.extra: fast
      $.level: 5
    flags:
      extra: {to: $.extra, type: string}
      level: {to: $.level, type: string}
`)
	require.NoError(t, err)
	flags = manifest.Commands[0].Flags
	require.Len(t, flags, 2)
	assert.Equal(t, "preset", flags[0].DefaultFrom)
	assert.Equal(t, "fast", flags[0].Default)
	assert.Empty(t, flags[1].DefaultFrom)
	assert.Nil(t, flags[1].Default)

	// Dispatch: a command preset or agreeing route presets become the
	// default; a preset on only one declaring route does not.
	commandPreset := strings.Replace(cliRouteDispatchManifest, "    args:\n", "    preset: {$.background: true}\n    args:\n", 1)
	dispatch, err := decodeCLITestSpec(t, cliRouteDispatchSpec, commandPreset)
	require.NoError(t, err)
	background := dispatch.Commands[0].Flags[2]
	assert.Equal(t, "background", background.Name)
	assert.Equal(t, "preset", background.DefaultFrom)
	assert.Equal(t, true, background.Default)

	engineOnly := strings.Replace(cliRouteDispatchManifest, "        selector: engine\n", "        selector: engine\n        preset: {$.background: true}\n", 1)
	dispatch, err = decodeCLITestSpec(t, cliRouteDispatchSpec, engineOnly)
	require.NoError(t, err)
	background = dispatch.Commands[0].Flags[2]
	assert.Equal(t, "background", background.Name)
	assert.Empty(t, background.DefaultFrom)
	assert.Nil(t, background.Default)
}

func TestCLICommands_StreamMetadata(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream:
        select: $.data.delta.text
        metadata:
          select: $.data.error.message
          label: Resource ID
`)
	require.NoError(t, err)
	assert.Equal(t, []CLICommandStreamMetadata{{
		Select:   []string{"$.data.error.message"},
		Pointers: []string{"/data/error/message"},
		Label:    "Resource ID",
	}}, manifest.Commands[0].Output.Stream.Metadata)

	manifest, _, err = decodeCLITest(t, `
version: 1
commands:
  say:
    op: StreamTask
    output:
      stream:
        select: $.data.delta.text
        metadata:
          - select: $.data.event_id
            label: Event ID
            when: incomplete
          - select: [$.data.reason, $.data.error.message]
            label: Status
            complete: done
`)
	require.NoError(t, err)
	assert.Equal(t, []CLICommandStreamMetadata{
		{Select: []string{"$.data.event_id"}, Pointers: []string{"/data/event_id"}, Label: "Event ID", When: "incomplete"},
		{Select: []string{"$.data.reason", "$.data.error.message"}, Pointers: []string{"/data/reason", "/data/error/message"}, Label: "Status", Complete: []string{"done"}},
	}, manifest.Commands[0].Output.Stream.Metadata)
}

func TestCLICommands_StreamMetadataValidation(t *testing.T) {
	for _, tc := range []struct{ name, metadata, want string }{
		{"missing select", "{label: ID}", "requires nonempty select and label"},
		{"missing label", "{select: $.data.error.message}", "requires nonempty select and label"},
		{"empty label", "{select: $.data.error.message, label: '  '}", "requires nonempty select and label"},
		{"label type", "{select: $.data.error.message, label: 12}", "metadata.label"},
		{"select type", "{select: [], label: ID}", "metadata.select"},
		{"unknown key", "{select: $.data.error.message, label: ID, labels: ID}", "metadata has unknown key"},
		{"wildcard", "{select: '$.data[*]', label: ID}", "metadata.select"},
		{"absent path", "{select: $.data.missing, label: ID}", "output.stream.metadata.select $.data.missing does not resolve"},
		{"number", "{select: $.data.index, label: ID}", "output.stream.metadata.select $.data.index resolves to an integer"},
		{"object", "{select: $.data.delta, label: ID}", "output.stream.metadata.select $.data.delta resolves to an object"},
		{"newline label", `{select: $.data.error.message, label: "ID\nInjected"}`, "metadata.label must contain only printable"},
		{"escape label", `{select: $.data.error.message, label: "ID\x1b"}`, "metadata.label must contain only printable"},
		{"bidi label", `{select: $.data.error.message, label: "ID\u202e"}`, "metadata.label must contain only printable"},
		{"separator label", `{select: $.data.error.message, label: "ID\u2028"}`, "metadata.label must contain only printable"},
		{"empty list", "[]", "metadata must list at least one field"},
		{"absent fallback path", "{select: [$.data.reason, $.data.missing], label: ID}", "output.stream.metadata.select $.data.missing does not resolve"},
		{"repeated path", "{select: [$.data.reason, $.data.reason], label: ID}", "lists \"$.data.reason\" more than once"},
		{"duplicate label", "[{select: $.data.reason, label: ID}, {select: $.data.event_id, label: ID}]", "label \"ID\" is declared more than once"},
		{"complete type", "{select: $.data.reason, label: Status, complete: {done: true}}", "metadata.complete must be a string or a list of strings"},
		{"two status fields", "[{select: $.data.reason, label: Status, complete: done}, {select: $.data.event_id, label: Event, complete: x}]", "declares complete on more than one field"},
		{"when without status", "[{select: $.data.event_id, label: Event, when: incomplete}]", "when: incomplete requires a status field"},
		{"when value", "[{select: $.data.reason, label: Status, complete: done}, {select: $.data.event_id, label: Event, when: always}]", "metadata.when must be incomplete"},
		{"status with when", "{select: $.data.reason, label: Status, complete: done, when: incomplete}", "already reported only when incomplete"},
		{"misspelled key", "{select: $.data.reason, label: Status, complet: done}", `unknown key "complet" (did you mean "complete"?)`},
		{"empty complete", "{select: $.data.reason, label: Status, complete: []}", "metadata.complete must list at least one string"},
		{"empty complete value", `{select: $.data.reason, label: Status, complete: [done, ""]}`, "metadata.complete must not contain empty strings"},
		{"repeated complete value", "{select: $.data.reason, label: Status, complete: [done, done]}", `metadata.complete lists "done" more than once`},
		{"complete member type", "{select: $.data.reason, label: Status, complete: [done, 42]}", "metadata.complete must be a string"},
		{"empty fallback path", `{select: [$.data.reason, ""], label: ID}`, "metadata.select must not contain empty strings"},
		{"fallback path type", "{select: [$.data.reason, 42], label: ID}", "metadata.select must be a string"},
		{"non-string fallback path", "{select: [$.data.reason, $.data.index], label: ID}", "output.stream.metadata.select $.data.index resolves to an integer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireDecodeError(t, "version: 1\ncommands:\n  say:\n    op: StreamTask\n    output:\n      stream:\n        select: $.data.delta.text\n        metadata: "+tc.metadata+"\n", tc.want)
		})
	}
}

func TestCLICommands_StreamMetadataOpenSchema(t *testing.T) {
	manifest, warnings, err := decodeCLITest(t, `
version: 1
operations:
  StreamTaskLines:
    output:
      stream:
        select: $.text
        metadata:
          select: $['resource/id~key']
          label: Resource ID
`)
	require.NoError(t, err)
	require.Len(t, manifest.Operations, 1)
	require.Len(t, manifest.Operations[0].Output.Stream.Metadata, 1)
	assert.Equal(t, []string{"/resource~1id~0key"}, manifest.Operations[0].Output.Stream.Metadata[0].Pointers)
	require.Len(t, warnings, 2)
	assert.Contains(t, warnings[1], "output.stream.metadata.select")
	assert.Contains(t, warnings[1], "could not be verified")
}

const validOutcomeManifest = `
version: 1
commands:
  check:
    op: GetTask
    output:
      outcome:
        statePointer: $.status
        states:
          in_progress: pending
          completed: success
          failed: failure
          requires_action: handoff
  follow:
    op: StreamTask
    args:
      prompt: {to: $.prompt, variadic: true}
    output:
      stream:
        select: $.data.delta.text
      outcome:
        streamStatePointer: $.data.reason
        streamErrorPointer: $.data.error
        states:
          done: success
          cancelled: failure
operations:
  GetTask:
    output:
      outcome:
        statePointer: $.status
        errorPointer: $.steps
        states: {in_progress: pending, completed: success, failed: failure, requires_action: handoff}
`

func TestCLICommands_OutcomeDecodeAndLink(t *testing.T) {
	manifest, _, err := decodeCLITest(t, validOutcomeManifest)
	require.NoError(t, err)
	require.Len(t, manifest.Commands, 2)

	check := manifest.Commands[0].Output.Outcome
	require.NotNil(t, check)
	assert.Equal(t, "$.status", check.StateFrom)
	assert.Equal(t, "/status", check.StatePointer)
	assert.Empty(t, check.StreamStatePointer)
	assert.Equal(t, map[string]string{"in_progress": "pending", "completed": "success", "failed": "failure", "requires_action": "handoff"}, check.States)
	assert.Empty(t, check.ErrorPointer)
	assert.Empty(t, check.ErrorMessageField)

	follow := manifest.Commands[1].Output.Outcome
	require.NotNil(t, follow)
	assert.Empty(t, follow.StatePointer)
	assert.Equal(t, "$.data.reason", follow.StreamStateFrom)
	assert.Equal(t, "/data/reason", follow.StreamStatePointer)
	assert.Equal(t, "/data/error", follow.StreamErrorPointer)
	assert.Equal(t, "message", follow.ErrorMessageField)
	assert.Equal(t, "/data/delta/text", manifest.Commands[1].Output.Stream.Pointer)

	require.Len(t, manifest.Operations, 1)
	op := manifest.Operations[0].Output.Outcome
	require.NotNil(t, op)
	assert.Equal(t, "/status", op.StatePointer)
	assert.Equal(t, "/steps", op.ErrorPointer)
	assert.Equal(t, "message", op.ErrorMessageField)
}

func TestCLICommands_OutcomeNullableLeaves(t *testing.T) {
	spec := strings.Replace(cliCommandsTestSpec, "        status:\n          type: string\n          enum: [in_progress, completed, failed, requires_action]\n        steps:\n          type: array\n", "        status:\n          type: [string, \"null\"]\n          enum: [in_progress, completed, failed, requires_action, null]\n        steps:\n          type: [array, \"null\"]\n", 1)
	require.NotEqual(t, cliCommandsTestSpec, spec)
	manifest, _, err := decodeCLIWithSpec(t, spec, validOutcomeManifest)
	require.NoError(t, err)
	require.Len(t, manifest.Operations, 1)
	assert.Equal(t, "/steps", manifest.Operations[0].Output.Outcome.ErrorPointer)
}

func TestCLICommands_OutcomeErrors(t *testing.T) {
	base := validOutcomeManifest
	for _, tc := range []struct {
		name     string
		manifest string
		want     string
	}{
		{"no pointer", strings.Replace(base, "        statePointer: $.status\n        states:\n          in_progress: pending", "        states:\n          in_progress: pending", 1), "requires statePointer (the state in the JSON response) or streamStatePointer"},
		{"no failure class", strings.Replace(base, "          failed: failure\n          requires_action: handoff\n  follow", "          failed: success\n          requires_action: handoff\n  follow", 1), "must classify at least one state as failure"},
		{"bad classification", strings.Replace(base, "failed: failure\n          requires_action", "failed: fatal\n          requires_action", 1), `output.outcome.states["failed"] has unsupported classification "fatal"`},
		{"unknown key", strings.Replace(base, "        statePointer: $.status\n        states:", "        pointer: $.status\n        states:", 1), `output.outcome has unknown key "pointer"`},
		{"not an enum member", strings.Replace(base, "          requires_action: handoff\n  follow", "          requires_action: handoff\n          exploded: failure\n  follow", 1), `classifies "exploded", which is not an enum member at $.status`},
		{"unclassified enum member", strings.Replace(base, "          requires_action: handoff\n  follow", "  follow", 1), `does not classify enum member "requires_action"`},
		{"state pointer not an enum", strings.Replace(base, "        statePointer: $.status\n        states:\n          in_progress", "        statePointer: $.id\n        states:\n          in_progress", 1), "selected state must declare a non-empty enum"},
		{"stream pointer on discrete operation", strings.Replace(base, "        statePointer: $.status\n        states:\n          in_progress", "        streamStatePointer: $.status\n        states:\n          in_progress", 1), "has no streaming (text/event-stream or JSONL) success response"},
		{"stream pointer misses every event", strings.Replace(base, "streamStatePointer: $.data.reason", "streamStatePointer: $.data.outcome", 1), "does not resolve in any event shape"},
		{"error pointer must be object or array", strings.Replace(base, "errorPointer: $.steps", "errorPointer: $.id", 1), "selected failure detail must be an object or an array"},
		{"error pointer misses every response variant", strings.Replace(base, "errorPointer: $.steps", "errorPointer: $.step", 1), `output.outcome.errorPointer $.step does not resolve in any application/json 2xx response variant of operation "GetTask" (did you mean "steps"?)`},
		{"stream state not an enum member", strings.Replace(base, "          done: success\n          cancelled: failure", "          done: success\n          canceled: failure", 1), `output.outcome.states classifies "canceled", which is not an enum member at $.data.reason in operation "StreamTask" (did you mean "cancelled"?)`},
		{"stream enum member unclassified", strings.Replace(base, "          done: success\n          cancelled: failure", "          cancelled: failure", 1), `output.outcome.states does not classify enum member "done" from $.data.reason in operation "StreamTask"`},
		{"stream error pointer misses every event", strings.Replace(base, "streamErrorPointer: $.data.error", "streamErrorPointer: $.data.failure", 1), "output.outcome.streamErrorPointer $.data.failure does not resolve in any event shape"},
		{"stream error pointer must be object or array", strings.Replace(base, "streamErrorPointer: $.data.error", "streamErrorPointer: $.data.event_id", 1), "resolves to a string in one event shape; the failure detail must be an object or an array"},
		{"error pointer without state pointer", strings.Replace(base, "        streamStatePointer: $.data.reason\n", "        streamStatePointer: $.data.reason\n        errorPointer: $.data.reason\n", 1), "output.outcome.errorPointer requires statePointer"},
		{"message field without pointer", strings.Replace(base, "        statePointer: $.status\n        states:\n          in_progress", "        statePointer: $.status\n        errorMessageField: detail\n        states:\n          in_progress", 1), "errorMessageField requires errorPointer or streamErrorPointer"},
		{"async conflict", strings.TrimSuffix(validAsyncManifest, "\n") + "\n    output:\n      outcome:\n        statePointer: $.status\n        states: {in_progress: pending, completed: success, failed: failure, requires_action: handoff}\n", "declares both async and output.outcome"},
		{"planned command", "version: 1\ncommands:\n  later:\n    planned: later\n    output:\n      outcome:\n        statePointer: $.status\n        states: {failed: failure}\n", "planned command and cannot declare output"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := decodeCLITest(t, tc.manifest)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

const cliOutcomeUnionTaskResult = `    TaskResult:
      oneOf:
        - $ref: '#/components/schemas/TaskSettled'
        - $ref: '#/components/schemas/TaskFailed'
    TaskSettled:
      type: object
      properties:
        id:
          type: string
        status:
          type: string
          enum: [in_progress, completed, failed, requires_action]
    TaskFailed:
      type: object
      properties:
        id:
          type: string
        status:
          type: string
          enum: [in_progress, completed, failed, requires_action]
        steps:
          type: array
          items:
            oneOf:
              - $ref: '#/components/schemas/NoteStep'
              - $ref: '#/components/schemas/OutputStep'
`

func TestCLICommands_OutcomeErrorPointerAnyVariant(t *testing.T) {
	start := strings.Index(cliCommandsTestSpec, "    TaskResult:\n")
	end := strings.Index(cliCommandsTestSpec, "    NoteStep:\n")
	require.True(t, start >= 0 && end > start)
	spec := cliCommandsTestSpec[:start] + cliOutcomeUnionTaskResult + cliCommandsTestSpec[end:]

	manifest, _, err := decodeCLIWithSpec(t, spec, validOutcomeManifest)
	require.NoError(t, err)
	require.Len(t, manifest.Operations, 1)
	assert.Equal(t, "/steps", manifest.Operations[0].Output.Outcome.ErrorPointer)

	_, _, err = decodeCLIWithSpec(t, spec, strings.Replace(validOutcomeManifest, "errorPointer: $.steps", "errorPointer: $.id", 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "selected failure detail must be an object or an array")

	_, _, err = decodeCLIWithSpec(t, spec, strings.Replace(validOutcomeManifest, "errorPointer: $.steps", "errorPointer: $.detail", 1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `output.outcome.errorPointer $.detail does not resolve in any application/json 2xx response variant of operation "GetTask"`)
}

func TestCLICommands_OutcomeStreamStatesWithoutEnum(t *testing.T) {
	manifest, warnings, err := decodeCLITest(t, strings.Replace(validOutcomeManifest, "streamStatePointer: $.data.reason", "streamStatePointer: $.data.event_id", 1))
	require.NoError(t, err)
	require.Len(t, manifest.Commands, 2)
	assert.Equal(t, "/data/event_id", manifest.Commands[1].Output.Outcome.StreamStatePointer)
	require.Len(t, warnings, 1, warnings)
	assert.Equal(t, `command "follow" output.outcome.streamStatePointer $.data.event_id in operation "StreamTask" does not select a closed string enum in every event shape; states cannot be verified at generation time, and a state missing from the table keeps exit 0 at runtime`, warnings[0])

	_, warnings, err = decodeCLITest(t, validOutcomeManifest)
	require.NoError(t, err)
	assert.Empty(t, warnings)
}

func TestCLICommands_OutcomeInheritedFromOperation(t *testing.T) {
	manifest, _, err := decodeCLITest(t, `
version: 1
commands:
  check:
    op: GetTask
  own:
    op: GetTask
    output:
      outcome:
        statePointer: $.status
        states: {in_progress: pending, completed: success, failed: failure, requires_action: failure}
operations:
  GetTask:
    output:
      outcome:
        statePointer: $.status
        errorPointer: $.steps
        states: {in_progress: pending, completed: success, failed: failure, requires_action: handoff}
`)
	require.NoError(t, err)
	require.Len(t, manifest.Commands, 2)
	require.Len(t, manifest.Operations, 1)

	inherited := manifest.Commands[0].Output.Outcome
	require.NotNil(t, inherited)
	assert.Equal(t, manifest.Operations[0].Output.Outcome, inherited)
	assert.Equal(t, "/steps", inherited.ErrorPointer)

	own := manifest.Commands[1].Output.Outcome
	require.NotNil(t, own)
	assert.Equal(t, "failure", own.States["requires_action"])
	assert.Empty(t, own.ErrorPointer)

	asyncManifest := strings.Replace(validAsyncManifest, "      params:\n        stream: false\n", "", 1)
	async, _, err := decodeCLITest(t, strings.TrimSuffix(asyncManifest, "\n")+`
operations:
  CreateTask:
    output:
      outcome:
        statePointer: $.status
        states: {in_progress: pending, completed: success, failed: failure, requires_action: handoff}
`)
	require.NoError(t, err)
	require.Len(t, async.Commands, 1)
	require.NotNil(t, async.Commands[0].Async)
	if async.Commands[0].Output != nil {
		assert.Nil(t, async.Commands[0].Output.Outcome, "an async intent keeps async.states as its only classification")
	}
}

func TestCLICommands_OutcomeStreamStateAllOfIntersectsEnums(t *testing.T) {
	const stopReason = "    StopReason:\n      type: string\n      enum: [done, cancelled]\n"
	require.Contains(t, cliCommandsTestSpec, stopReason)
	typoTable := strings.Replace(validOutcomeManifest, "          done: success\n          cancelled: failure", "          done: success\n          canceled: failure", 1)

	for _, tc := range []struct {
		name   string
		schema string
	}{
		{"type-only arm", "    StopReason:\n      allOf:\n        - type: string\n        - enum: [done, cancelled]\n"},
		{"two enum arms", "    StopReason:\n      allOf:\n        - type: string\n          enum: [done, cancelled, extra]\n        - enum: [done, cancelled]\n"},
		{"enum beside ref", "    StopReason:\n      $ref: '#/components/schemas/StopReasonBase'\n      enum: [done, cancelled]\n    StopReasonBase:\n      type: string\n      enum: [done, cancelled, extra]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := strings.Replace(cliCommandsTestSpec, stopReason, tc.schema, 1)

			_, warnings, err := decodeCLIWithSpec(t, spec, validOutcomeManifest)
			require.NoError(t, err)
			assert.Empty(t, warnings)

			_, _, err = decodeCLIWithSpec(t, spec, typoTable)
			require.Error(t, err)
			assert.Contains(t, err.Error(), `output.outcome.states classifies "canceled", which is not an enum member at $.data.reason in operation "StreamTask" (did you mean "cancelled"?)`)
		})
	}
}

const cliContinuationTestSpec = `openapi: 3.1.0
info:
  title: Widget Service
  version: 1.0.0
paths:
  /widgets/compose:
    post:
      operationId: ComposeWidget
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ComposeRequest'
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Widget'
  /widgets/compose/stream:
    post:
      operationId: StreamWidget
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ComposeRequest'
      responses:
        "200":
          description: events
          content:
            text/event-stream:
              schema:
                type: object
                required: [data]
                properties:
                  data:
                    $ref: '#/components/schemas/WidgetEvent'
  /widgets/{id}:
    get:
      operationId: GetWidget
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Widget'
components:
  schemas:
    ComposeRequest:
      type: object
      properties:
        prompt:
          type: string
        resumeToken:
          type: string
        previousId:
          type: string
        budget:
          type: integer
        options:
          type: object
          properties:
            cursor:
              type: string
      required: [prompt]
    WidgetStatus:
      type: string
      enum: [in_progress, more_required, budget_reached, completed, failed]
    Widget:
      type: object
      properties:
        id:
          type: string
        status:
          $ref: '#/components/schemas/WidgetStatus'
        resumeToken:
          type: string
        steps:
          type: array
          items:
            type: object
            properties:
              text:
                type: string
        usage:
          type: object
          properties:
            inputUnits:
              type: integer
            outputUnits:
              type: integer
    WidgetEvent:
      type: object
      properties:
        event_type:
          type: string
        resumeToken:
          type: string
        delta:
          type: object
          properties:
            text:
              type: string
        widget:
          $ref: '#/components/schemas/Widget'
`

const validContinuationManifest = `
version: 1
commands:
  compose:
    op: ComposeWidget
    args:
      prompt: {to: $.prompt, variadic: true}
    flags:
      resume-token: {to: $.resumeToken}
      previous-id: {to: $.previousId}
  watch:
    op: StreamWidget
    args:
      prompt: {to: $.prompt, variadic: true}
    output:
      stream:
        select: $.data.delta.text
      outcome:
        streamStatePointer: $.data.widget.status
        states: {in_progress: pending, more_required: continue, budget_reached: success, completed: success, failed: failure}
    continuation:
      bind:
        - streamFrom: [$.data.resumeToken, $.data.widget.resumeToken]
          to: $.resumeToken
        - streamFrom: $.data.widget.id
          to: $.previousId
          optional: true
      maxHops: 4
      default: false
operations:
  ComposeWidget:
    output:
      outcome:
        statePointer: $.status
        states: {in_progress: pending, more_required: continue, budget_reached: success, completed: success, failed: failure}
    continuation:
      bind:
        - from: $.resumeToken
          to: $.resumeToken
        - from: $.id
          to: $.previousId
          optional: true
      merge: $.steps
      sum: [$.usage.inputUnits, $.usage.outputUnits]
`

func TestCLICommands_ContinuationDecodeAndLink(t *testing.T) {
	manifest, warnings, err := decodeCLIWithSpec(t, cliContinuationTestSpec, validContinuationManifest)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, manifest.Operations, 1)
	require.Len(t, manifest.Commands, 2)

	op := manifest.Operations[0].Continuation
	require.NotNil(t, op)
	assert.Equal(t, "ComposeWidget", op.OperationID)
	assert.Equal(t, 32, op.MaxHops)
	assert.True(t, op.Default)
	assert.Equal(t, []string{"/steps"}, op.Merge)
	assert.Equal(t, []string{"/usage/inputUnits", "/usage/outputUnits"}, op.Sum)
	require.Len(t, op.Bind, 2)
	assert.Equal(t, CLICommandContinuationBinding{From: "$.resumeToken", Pointer: "/resumeToken", To: "$.resumeToken", ToPointer: "/resumeToken", ToPath: []string{"resumeToken"}}, op.Bind[0])
	assert.True(t, op.Bind[1].Optional)
	assert.Empty(t, op.Bind[1].Flag, "the operation entry declares no flags")

	compose := manifest.Commands[0].Continuation
	require.NotNil(t, compose, "a single-route intent inherits its operation entry's recipe")
	assert.Equal(t, "resume-token", compose.Bind[0].Flag)
	assert.Equal(t, "previous-id", compose.Bind[1].Flag)
	assert.NotSame(t, &op.Bind[0], &compose.Bind[0])

	watch := manifest.Commands[1].Continuation
	require.NotNil(t, watch)
	assert.Equal(t, 4, watch.MaxHops)
	assert.False(t, watch.Default)
	assert.Equal(t, []string{"/data/resumeToken", "/data/widget/resumeToken"}, watch.Bind[0].StreamPointers)
	assert.Equal(t, []string{"/data/widget/id"}, watch.Bind[1].StreamPointers)
}

func TestCLICommands_ContinuationNestedTarget(t *testing.T) {
	manifest, _, err := decodeCLIWithSpec(t, cliContinuationTestSpec, strings.Replace(validContinuationManifest, "        - from: $.resumeToken\n          to: $.resumeToken\n", "        - from: $.resumeToken\n          to: $.options.cursor\n", 1))
	require.NoError(t, err)
	assert.Equal(t, []string{"options", "cursor"}, manifest.Operations[0].Continuation.Bind[0].ToPath)
	assert.Equal(t, "/options/cursor", manifest.Operations[0].Continuation.Bind[0].ToPointer)
}

func TestCLICommands_ContinuationErrors(t *testing.T) {
	base := validContinuationManifest
	opOutcome := "    output:\n      outcome:\n        statePointer: $.status\n        states: {in_progress: pending, more_required: continue, budget_reached: success, completed: success, failed: failure}\n"
	for _, tc := range []struct {
		name     string
		manifest string
		want     string
	}{
		{"unknown key", strings.Replace(base, "      maxHops: 4\n", "      maxHops: 4\n      hops: 2\n", 1), `continuation has unknown key "hops"`},
		{"unknown bind key", strings.Replace(base, "          to: $.previousId\n          optional: true\n      maxHops", "          to: $.previousId\n          required: false\n      maxHops", 1), `continuation.bind has unknown key "required"`},
		{"missing bind", strings.Replace(base, "      bind:\n        - from: $.resumeToken\n          to: $.resumeToken\n        - from: $.id\n          to: $.previousId\n          optional: true\n", "", 1), "continuation requires bind"},
		{"only optional", strings.Replace(base, "        - from: $.resumeToken\n          to: $.resumeToken\n", "        - from: $.resumeToken\n          to: $.resumeToken\n          optional: true\n", 1), "at least one binding that is not optional"},
		{"duplicate target", strings.Replace(base, "          to: $.previousId\n          optional: true\n      merge", "          to: $.resumeToken\n          optional: true\n      merge", 1), "writes $.resumeToken more than once"},
		{"missing to", strings.Replace(base, "        - from: $.resumeToken\n          to: $.resumeToken\n", "        - from: $.resumeToken\n", 1), "requires to (the request-body path"},
		{"missing from", strings.Replace(base, "        - from: $.resumeToken\n          to: $.resumeToken\n", "        - to: $.resumeToken\n", 1), "requires from (JSON response) or streamFrom"},
		{"indexed to", strings.Replace(base, "          to: $.resumeToken\n        - from: $.id", "          to: $.options[0]\n        - from: $.id", 1), "must name object properties only"},
		{"zero hops", strings.Replace(base, "maxHops: 4", "maxHops: 0", 1), "continuation.maxHops must be at least 1"},
		{"hops not an int", strings.Replace(base, "maxHops: 4", "maxHops: many", 1), "continuation.maxHops must be an integer"},
		{"no outcome", strings.Replace(base, opOutcome, "", 1), "declares continuation but no output.outcome"},
		{"no continue state", strings.Replace(base, "states: {in_progress: pending, more_required: continue, budget_reached: success, completed: success, failed: failure}\n    continuation", "states: {in_progress: pending, more_required: success, budget_reached: success, completed: success, failed: failure}\n    continuation", 1), "classify at least one state as continue"},
		{"from not a string", strings.Replace(base, "        - from: $.resumeToken\n", "        - from: $.steps\n", 1), "selected value must be a string"},
		{"from on stream op", strings.Replace(base, "        - streamFrom: [$.data.resumeToken, $.data.widget.resumeToken]\n", "        - from: $.resumeToken\n", 1), "has no application/json 2xx response; use streamFrom"},
		{"streamFrom misses", strings.Replace(base, "streamFrom: $.data.widget.id", "streamFrom: $.data.widget.ident", 1), "does not resolve in any event shape"},
		{"to absent", strings.Replace(base, "        - from: $.resumeToken\n          to: $.resumeToken\n", "        - from: $.resumeToken\n          to: $.resumeTokn\n", 1), `property "resumeTokn" is absent from request body variant`},
		{"to not a string", strings.Replace(base, "        - from: $.resumeToken\n          to: $.resumeToken\n", "        - from: $.resumeToken\n          to: $.budget\n", 1), "must name a string property"},
		{"merge not an array", strings.Replace(base, "merge: $.steps", "merge: $.id", 1), "continuation.merge $.id in operation \"ComposeWidget\" response 200 must select an array"},
		{"sum not a number", strings.Replace(base, "sum: [$.usage.inputUnits, $.usage.outputUnits]", "sum: [$.id]", 1), "must select a number"},
		{"merge on stream op", strings.Replace(base, "      maxHops: 4\n", "      maxHops: 4\n      merge: $.data.widget.steps\n", 1), "apply to application/json responses"},
		{"bad classification", strings.Replace(base, "more_required: continue, budget_reached: success, completed: success, failed: failure}\n    continuation", "more_required: resume, budget_reached: success, completed: success, failed: failure}\n    continuation", 1), `has unsupported classification "resume"`},
		{"reserved flag", strings.Replace(base, "      previous-id: {to: $.previousId}\n", "      previous-id: {to: $.previousId}\n      auto-continue: {to: $.prompt}\n", 1), "auto-continue"},
		{"async conflict", strings.Replace(base, "  watch:\n", "  start:\n    op: ComposeWidget\n    async:\n      op: GetWidget\n      id: {from: $.id, to: {in: path, name: id}}\n      statePointer: $.status\n      states: {in_progress: pending, more_required: pending, budget_reached: success, completed: success, failed: failure}\n      interval: 1s\n      maxInterval: 1s\n      timeout: 1m\n  watch:\n", 1), `async command "start" also uses as its create operation`},
		{"async and continuation", strings.Replace(base, "  watch:\n    op: StreamWidget\n", "  watch:\n    op: StreamWidget\n    async:\n      op: GetWidget\n      id: {from: $.id, to: {in: path, name: id}}\n      statePointer: $.status\n      states: {completed: success}\n", 1), "declares both async and output.stream"},
		{"planned command", "version: 1\ncommands:\n  later:\n    planned: later\n    continuation:\n      bind: [{from: $.id, to: $.previousId}]\n", "planned command and cannot declare continuation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := decodeCLIWithSpec(t, cliContinuationTestSpec, tc.manifest)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestCLICommands_ContinuationAsyncStatesRejectContinue(t *testing.T) {
	_, _, err := decodeCLIWithSpec(t, cliContinuationTestSpec, `
version: 1
commands:
  start:
    op: ComposeWidget
    async:
      op: GetWidget
      id: {from: $.id, to: {in: path, name: id}}
      statePointer: $.status
      states: {in_progress: pending, more_required: continue, budget_reached: success, completed: success, failed: failure}
      interval: 1s
      maxInterval: 1s
      timeout: 1m
`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `async.states["more_required"] has unsupported classification "continue"`)
}
