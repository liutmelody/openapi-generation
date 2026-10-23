package extensions

import (
	"context"
	stderrors "errors"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/speakeasy-api/openapi-generation/v2/internal/contenttypes"
	"github.com/speakeasy-api/openapi-generation/v2/internal/resolution"
	"github.com/speakeasy-api/openapi-generation/v2/internal/types"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/openapi"
)

// This file links a decoded x-speakeasy-cli-commands manifest against the
// OpenAPI document: operations and request variants must resolve, every bind
// and preset pointer must land on a real schema property, structural facts
// (scalar types, summaries, enum suggestions, schema defaults, requiredness)
// are inferred from the variant schema, and preset values are validated
// against it. Structural drift therefore surfaces at generation time as a
// hard error naming the fix — never as a runtime 400.

const cliComponentRefPrefix = "#/components/schemas/"

// cliSchemaIndex caches decoded component schemas and per-operation facts.
type cliSchemaIndex struct {
	components map[string]any
	operations map[string]*cliOperationInfo
	opIDs      []string
}

type cliOperationInfo struct {
	method         string
	bodySchema     any  // raw decoded application/json request schema (nil when absent)
	bodyRequired   bool // requestBody.required; controls safe op-flag synthesis
	hasRequestBody bool
	params         map[string][]string // location → parameter names (path-item + operation level)
	parameterInfo  map[string]*cliParameterInfo
	// responseSchema is the raw decoded application/json schema of the 2xx
	// response ("200" preferred, else the lowest 2xx code that has one).
	// Artifact declarations validate their content pointer against it.
	responseSchema   any
	responseSchemaJS *oas3.JSONSchema[oas3.Referenceable]
	responseCode     string
	// jsonResponses retains every successful application/json schema for
	// async contracts, whose pointers must be valid across every status code.
	jsonResponses []cliJSONResponseSchema
	// streamSchemas are the raw decoded schemas of every successful (2XX)
	// streaming response media type (text/event-stream, JSONL/JSON-seq), in
	// document order. Non-nil length > 0 means the CLI iterates this
	// operation's response as a stream. A streaming media type declared
	// without a schema is recorded as a nil entry (unverifiable).
	streamSchemas   []any
	streamSchemaJS  []*oas3.JSONSchema[oas3.Referenceable]
	streamMediaSeen bool
}

type cliJSONResponseSchema struct {
	code   string
	schema any
}

type cliParameterInfo struct {
	name     string
	in       string
	required bool
	global   bool
	schema   any
}

// cliResolvedSchema is the composed view of an object schema after following
// $ref chains and merging allOf branches.
type cliResolvedSchema struct {
	refName      string // component name when the schema came from a $ref
	properties   map[string]any
	propOrder    []string
	required     map[string]bool
	explicitOpen bool // additionalProperties explicitly permits unknown keys
	// additionalProperties: false somewhere in the composition; a closed
	// layer constrains the conjunction, so it wins over an open one.
	explicitClosed bool
	unionMembers   []any          // oneOf/anyOf member schemas (raw), when present
	additional     map[string]any // additionalProperties schema, across allOf layers
	// const or enum somewhere in the composition: the value must equal a
	// listed one, so filling keys into it can only break that equality.
	enumerated bool
	raw        map[string]any
}

// cliPropertyFacts is the classification of a single property schema used for
// inference and preset validation.
type cliPropertyFacts struct {
	kinds           []string // resolved scalar kind(s): string,int,float,bool — or array,object
	enum            []any
	catalogCommands []string
	openEnum        bool
	defaultVal      any
	hasDefault      bool
	constVal        any
	hasConst        bool
	description     string
	readOnly        bool
	arrayOfStr      bool // exactly array<string> (for bounded promotion)
	items           *cliPropertyFacts
	itemsErr        error
	unionArms       []any // raw member schemas when the property is a union
}

func (d *cliManifestDecoder) linkManifest(manifest *CLICommandManifest) error {
	if err := d.buildSchemaIndex(); err != nil {
		return err
	}

	for i := range manifest.Commands {
		cmd := &manifest.Commands[i]
		if cmd.Source.Type != CLICommandSourceOperation {
			continue
		}
		if err := d.linkCommand(cmd); err != nil {
			return err
		}
	}
	for i := range manifest.Operations {
		if err := d.linkOperation(&manifest.Operations[i]); err != nil {
			return err
		}
	}
	inheritOperationOutcomes(manifest)
	inheritOperationContinuations(manifest)
	return d.linkContinuations(manifest)
}

// inheritOperationContinuations gives a single-route, non-async intent
// without its own continuation the recipe its operation entry declared, as
// output.outcome is inherited. Bindings are copied because hint flags are
// resolved against each command's own flags.
func inheritOperationContinuations(manifest *CLICommandManifest) {
	recipes := map[string]*CLICommandContinuation{}
	for i := range manifest.Operations {
		if op := &manifest.Operations[i]; op.Continuation != nil {
			recipes[op.OperationID] = op.Continuation
		}
	}
	if len(recipes) == 0 {
		return
	}
	for i := range manifest.Commands {
		cmd := &manifest.Commands[i]
		if cmd.Source.Type != CLICommandSourceOperation || len(cmd.Source.Routes) != 1 || cmd.Async != nil || cmd.Continuation != nil {
			continue
		}
		recipe, ok := recipes[cmd.Source.Routes[0].OperationID]
		if !ok {
			continue
		}
		copied := *recipe
		copied.Bind = append([]CLICommandContinuationBinding(nil), recipe.Bind...)
		cmd.Continuation = &copied
	}
}

func (d *cliManifestDecoder) linkContinuations(manifest *CLICommandManifest) error {
	asyncCreates := map[string]string{}
	for i := range manifest.Commands {
		cmd := &manifest.Commands[i]
		if cmd.Async != nil && len(cmd.Source.Routes) > 0 {
			asyncCreates[cmd.Source.Routes[0].OperationID] = strings.Join(cmd.Path, " ")
		}
	}
	for i := range manifest.Operations {
		op := &manifest.Operations[i]
		if op.Continuation == nil {
			continue
		}
		var outcome *CLICommandOutcome
		if op.Output != nil {
			outcome = op.Output.Outcome
		}
		if err := d.linkContinuation(fmt.Sprintf("operation %q", op.OperationID), op.Continuation, op.OperationID, outcome, op.Flags, asyncCreates); err != nil {
			return err
		}
	}
	for i := range manifest.Commands {
		cmd := &manifest.Commands[i]
		if cmd.Continuation == nil || cmd.Source.Type != CLICommandSourceOperation {
			continue
		}
		var outcome *CLICommandOutcome
		if cmd.Output != nil {
			outcome = cmd.Output.Outcome
		}
		owner := fmt.Sprintf("command %q", strings.Join(cmd.Path, " "))
		if err := d.linkContinuation(owner, cmd.Continuation, cmd.Source.Routes[0].OperationID, outcome, cmd.Flags, asyncCreates); err != nil {
			return err
		}
	}
	return nil
}

func (d *cliManifestDecoder) linkContinuation(owner string, recipe *CLICommandContinuation, opID string, outcome *CLICommandOutcome, flags []CLICommandInput, asyncCreates map[string]string) error {
	opInfo, ok := d.schemaIndex.operations[opID]
	if !ok {
		return fmt.Errorf("%s continuation: operation %q does not exist in the document%s", owner, opID, cliDidYouMean(opID, d.schemaIndex.opIDs))
	}
	recipe.OperationID = opID
	if asyncCommand, ok := asyncCreates[opID]; ok {
		return fmt.Errorf("%s declares continuation on operation %q, which async command %q also uses as its create operation; continuation and async cannot share an operation in v1", owner, opID, asyncCommand)
	}
	if outcome == nil {
		return fmt.Errorf("%s declares continuation but no output.outcome; declare a states table with at least one continue state on the command or its operation entry", owner)
	}
	hasContinue := false
	for _, class := range outcome.States {
		hasContinue = hasContinue || class == "continue"
	}
	if !hasContinue {
		return fmt.Errorf("%s continuation requires output.outcome.states to classify at least one state as continue", owner)
	}
	hasJSON, hasStream := len(opInfo.jsonResponses) > 0, opInfo.streamMediaSeen
	if !hasJSON && !hasStream {
		return fmt.Errorf("%s continuation: operation %q has no application/json or streaming 2xx response to read resume values from", owner, opID)
	}
	if hasJSON && outcome.StatePointer == "" {
		return fmt.Errorf("%s continuation: operation %q returns application/json, so output.outcome must declare statePointer", owner, opID)
	}
	if hasStream && outcome.StreamStatePointer == "" {
		return fmt.Errorf("%s continuation: operation %q streams its response, so output.outcome must declare streamStatePointer", owner, opID)
	}
	if opInfo.bodySchema == nil {
		return fmt.Errorf("%s continuation: operation %q has no application/json request body to carry resume values", owner, opID)
	}

	for i := range recipe.Bind {
		binding := &recipe.Bind[i]
		if binding.From != "" {
			if !hasJSON {
				return fmt.Errorf("%s continuation.bind.from %s: operation %q has no application/json 2xx response; use streamFrom for streamed events", owner, binding.From, opID)
			}
			for _, response := range opInfo.jsonResponses {
				if _, err := d.linkAsyncResponsePointer(binding.From, response.schema, false); err != nil {
					return fmt.Errorf("%s continuation.bind.from %s in operation %q response %s: %w", owner, binding.From, opID, response.code, err)
				}
			}
		} else if hasJSON && !binding.Optional {
			return fmt.Errorf("%s continuation.bind entry for %s requires from: operation %q returns application/json", owner, binding.To, opID)
		}
		if len(binding.StreamFrom) > 0 {
			if !hasStream {
				return fmt.Errorf("%s continuation.bind.streamFrom: operation %q has no streaming 2xx response", owner, opID)
			}
			for _, path := range binding.StreamFrom {
				if err := d.linkStreamSelection(owner, "continuation.bind.streamFrom", path, opInfo); err != nil {
					return err
				}
			}
		} else if hasStream && !binding.Optional {
			return fmt.Errorf("%s continuation.bind entry for %s requires streamFrom: operation %q streams its response", owner, binding.To, opID)
		}
		if err := d.linkContinuationTarget(owner, opID, binding, opInfo.bodySchema); err != nil {
			return err
		}
		binding.Flag = ""
		for _, flag := range flags {
			if flag.Bind != nil && flag.Bind.In == "body" && flag.Bind.Pointer == binding.ToPointer {
				binding.Flag = flag.Name
				break
			}
		}
	}

	if (len(recipe.Merge) > 0 || len(recipe.Sum) > 0) && !hasJSON {
		return fmt.Errorf("%s continuation.merge and continuation.sum apply to application/json responses, and operation %q has none", owner, opID)
	}
	for _, path := range recipe.MergeFrom {
		if err := d.linkContinuationResultPath(owner, "continuation.merge", path, opID, opInfo, func(kind string) bool { return kind == "array" }, "an array"); err != nil {
			return err
		}
	}
	for _, path := range recipe.SumFrom {
		if err := d.linkContinuationResultPath(owner, "continuation.sum", path, opID, opInfo, func(kind string) bool { return kind == "int" || kind == "float" }, "a number"); err != nil {
			return err
		}
	}
	return nil
}

func (d *cliManifestDecoder) linkContinuationResultPath(owner, field, path, opID string, opInfo *cliOperationInfo, accept func(string) bool, expected string) error {
	for _, response := range opInfo.jsonResponses {
		views, err := d.walkResponsePath(path, response.schema)
		if err != nil {
			return fmt.Errorf("%s %s %s in operation %q response %s: %w", owner, field, path, opID, response.code, err)
		}
		facts, err := d.responseLeafFacts(views)
		if err != nil {
			return fmt.Errorf("%s %s %s in operation %q response %s: %w", owner, field, path, opID, response.code, err)
		}
		for _, propertyFacts := range facts {
			if len(propertyFacts.kinds) != 1 || !accept(propertyFacts.kinds[0]) {
				return fmt.Errorf("%s %s %s in operation %q response %s must select %s in every response variant", owner, field, path, opID, response.code, expected)
			}
		}
	}
	return nil
}

// linkContinuationTarget requires the bound request-body path to be a
// writable string property in every request-body variant, so a resume value
// always lands in a field the API accepts.
func (d *cliManifestDecoder) linkContinuationTarget(owner, opID string, binding *CLICommandContinuationBinding, bodySchema any) error {
	bodyMap, _ := bodySchema.(map[string]any)
	unionMap, _ := d.unionBodySchema(bodyMap)
	rawVariants := cliUnionMembers(unionMap)
	if len(rawVariants) == 0 {
		rawVariants = []any{bodySchema}
	}
	for i, raw := range rawVariants {
		current := raw
		for depth, name := range binding.ToPath {
			variant, err := d.resolveObjectSchema(current, nil)
			if err != nil {
				return fmt.Errorf("%s continuation.bind.to %s: request body variant %d of operation %q: %w", owner, binding.To, i+1, opID, err)
			}
			prop, exists := variant.properties[name]
			if !exists {
				return fmt.Errorf("%s continuation.bind.to %s: property %q is absent from request body variant %s of operation %q%s", owner, binding.To, name, cliSchemaLabel(variant), opID, cliDidYouMean(name, variant.propOrder))
			}
			if depth < len(binding.ToPath)-1 {
				current = prop
				continue
			}
			facts, err := d.propertyFacts(prop, nil)
			if err != nil {
				return fmt.Errorf("%s continuation.bind.to %s in request body variant %s of operation %q: %w", owner, binding.To, cliSchemaLabel(variant), opID, err)
			}
			if facts.readOnly {
				return fmt.Errorf("%s continuation.bind.to %s names readOnly property %q in request body variant %s of operation %q", owner, binding.To, name, cliSchemaLabel(variant), opID)
			}
			if len(facts.kinds) != 1 || facts.kinds[0] != "string" {
				return fmt.Errorf("%s continuation.bind.to %s must name a string property in request body variant %s of operation %q", owner, binding.To, cliSchemaLabel(variant), opID)
			}
		}
	}
	return nil
}

// inheritOperationOutcomes gives a single-route intent without its own
// output.outcome the table its operation entry declared, so a failure state
// classified for the operation command also fails the intent. Async intents
// keep async.states as their only classification.
func inheritOperationOutcomes(manifest *CLICommandManifest) {
	outcomes := map[string]*CLICommandOutcome{}
	for i := range manifest.Operations {
		if op := &manifest.Operations[i]; op.Output != nil && op.Output.Outcome != nil {
			outcomes[op.OperationID] = op.Output.Outcome
		}
	}
	if len(outcomes) == 0 {
		return
	}
	for i := range manifest.Commands {
		cmd := &manifest.Commands[i]
		if cmd.Source.Type != CLICommandSourceOperation || len(cmd.Source.Routes) != 1 || cmd.Async != nil {
			continue
		}
		if cmd.Output != nil && cmd.Output.Outcome != nil {
			continue
		}
		outcome, ok := outcomes[cmd.Source.Routes[0].OperationID]
		if !ok {
			continue
		}
		if cmd.Output == nil {
			cmd.Output = &CLICommandOutput{}
		}
		cmd.Output.Outcome = outcome
	}
}

func (d *cliManifestDecoder) buildSchemaIndex() error {
	if d.schemaIndex != nil {
		return nil
	}
	index := &cliSchemaIndex{
		components: map[string]any{},
		operations: map[string]*cliOperationInfo{},
	}
	doc := d.docInfo.Doc
	globalParams := map[string]bool{}
	globals, err := New(types.NewTargetFromTemplate("cli")).HandleGlobalsExtension(d.ctx, doc)
	if err != nil {
		return fmt.Errorf("decode x-speakeasy-globals while linking CLI commands: %w", err)
	}
	if globals != nil {
		for _, paramRef := range globals.Parameters {
			if param := d.resolveParameter(paramRef); param != nil {
				globalParams[cliParameterKey(string(param.GetIn()), param.GetName())] = true
			}
		}
	}

	if doc.Components != nil && doc.Components.Schemas != nil {
		for name, schemaRef := range doc.Components.Schemas.All() {
			if schemaRef == nil || schemaRef.GetSchema() == nil {
				continue
			}
			node := schemaRef.GetSchema().GetRootNode()
			if node == nil {
				continue
			}
			decoded, err := decodeYAMLNode(node)
			if err != nil {
				continue
			}
			index.components[name] = decoded
		}
	}

	if doc.Paths != nil {
		for _, pathItemRef := range doc.Paths.All() {
			if pathItemRef == nil {
				continue
			}
			pathItem := pathItemRef.GetObject()
			if pathItem == nil || !pathItem.IsInitialized() {
				continue
			}

			pathParams := map[string][]string{}
			pathParamInfo := map[string]*cliParameterInfo{}
			for _, paramRef := range pathItem.GetParameters() {
				if param := d.resolveParameter(paramRef); param != nil {
					in := string(param.GetIn())
					pathParams[in] = append(pathParams[in], param.GetName())
					pathParamInfo[cliParameterKey(in, param.GetName())] = d.cliParameterInfo(param, globalParams)
				}
			}

			for method, op := range pathItem.All() {
				if op == nil || op.GetOperationID() == "" {
					continue
				}
				info := &cliOperationInfo{method: method.String(), params: map[string][]string{}, parameterInfo: map[string]*cliParameterInfo{}}
				for in, names := range pathParams {
					info.params[in] = append(info.params[in], names...)
				}
				for key, param := range pathParamInfo {
					info.parameterInfo[key] = param
				}
				for _, paramRef := range op.GetParameters() {
					if param := d.resolveParameter(paramRef); param != nil {
						in := string(param.GetIn())
						info.params[in] = append(info.params[in], param.GetName())
						info.parameterInfo[cliParameterKey(in, param.GetName())] = d.cliParameterInfo(param, globalParams)
					}
				}

				if responses := op.GetResponses(); responses != nil {
					for statusCode, respRef := range responses.All() {
						if respRef == nil || len(statusCode) != 3 || statusCode[0] != '2' {
							continue
						}
						resp := respRef.GetObject()
						if resp == nil || resp.GetContent() == nil {
							continue
						}
						for contentType, mediaType := range resp.GetContent().All() {
							if !cliIsStreamingContentType(contentType) {
								continue
							}
							info.streamMediaSeen = true
							var decoded any
							var schemaJS *oas3.JSONSchema[oas3.Referenceable]
							if mediaType != nil {
								schemaJS = mediaType.GetSchema()
								if schemaRef := schemaJS; schemaRef != nil && schemaRef.GetSchema() != nil {
									if node := schemaRef.GetSchema().GetRootNode(); node != nil {
										if d, err := decodeYAMLNode(node); err == nil {
											decoded = d
										}
									}
								}
							}
							info.streamSchemas = append(info.streamSchemas, decoded)
							info.streamSchemaJS = append(info.streamSchemaJS, schemaJS)
						}
					}
				}

				if rbRef := op.GetRequestBody(); rbRef != nil {
					info.hasRequestBody = true
					if rb := rbRef.GetObject(); rb != nil {
						info.bodyRequired = rb.GetRequired()
						if content := rb.GetContent(); content != nil {
							if mediaType, ok := content.Get("application/json"); ok && mediaType != nil {
								if schemaRef := mediaType.GetSchema(); schemaRef != nil && schemaRef.GetSchema() != nil {
									if node := schemaRef.GetSchema().GetRootNode(); node != nil {
										if decoded, err := decodeYAMLNode(node); err == nil {
											info.bodySchema = decoded
										}
									}
								}
							}
						}
					}
				}

				if responses := op.GetResponses(); responses != nil {
					chosenCode := ""
					for statusCode, respRef := range responses.All() {
						if len(statusCode) != 3 || statusCode[0] != '2' || respRef == nil {
							continue
						}
						resp := respRef.GetObject()
						if resp == nil {
							continue
						}
						content := resp.GetContent()
						if content == nil {
							continue
						}
						mediaType, ok := content.Get("application/json")
						if !ok || mediaType == nil {
							continue
						}
						schemaRef := mediaType.GetSchema()
						if schemaRef == nil || schemaRef.GetSchema() == nil {
							continue
						}
						node := schemaRef.GetSchema().GetRootNode()
						if node == nil {
							continue
						}
						decoded, err := decodeYAMLNode(node)
						if err != nil {
							continue
						}
						info.jsonResponses = append(info.jsonResponses, cliJSONResponseSchema{code: statusCode, schema: decoded})
						// RFC 9110 forbids response content for 204 and 205. Async
						// validation still sees an authored JSON schema, but artifact
						// selection must never choose a body Go strips from the wire.
						if statusCode == "204" || statusCode == "205" {
							continue
						}
						// Deterministic choice: "200" wins, else the lowest code.
						if chosenCode == "" || statusCode < chosenCode {
							chosenCode = statusCode
							info.responseSchema = decoded
							info.responseSchemaJS = schemaRef
							info.responseCode = statusCode
						}
					}
					sort.SliceStable(info.jsonResponses, func(i, j int) bool {
						return info.jsonResponses[i].code < info.jsonResponses[j].code
					})
				}

				opID := op.GetOperationID()
				if _, exists := index.operations[opID]; !exists {
					index.opIDs = append(index.opIDs, opID)
				}
				index.operations[opID] = info
			}
		}
	}

	d.schemaIndex = index
	return nil
}

// resolveParameter returns the parameter behind a possibly-referenced entry.
// Globals and operations commonly point at #/components/parameters via $ref;
// an unresolved reference must not silently drop the parameter (a global
// would then look like an uncovered required parameter, or vice versa).
func (d *cliManifestDecoder) resolveParameter(paramRef *openapi.ReferencedParameter) *openapi.Parameter {
	if paramRef == nil {
		return nil
	}
	if param := paramRef.GetObject(); param != nil {
		return param
	}
	if d.docInfo == nil {
		return nil
	}
	ctx := d.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	param, err := resolution.Resolve(ctx, paramRef, d.docInfo)
	if err != nil {
		return nil
	}
	return param
}

func cliParameterKey(in, name string) string {
	if in == "header" {
		name = strings.ToLower(name)
	}
	return in + string(rune(0)) + name
}

func (d *cliManifestDecoder) cliParameterInfo(param *openapi.Parameter, globalParams map[string]bool) *cliParameterInfo {
	in := string(param.GetIn())
	info := &cliParameterInfo{
		name:     param.GetName(),
		in:       in,
		required: param.GetRequired(),
		global:   globalParams[cliParameterKey(in, param.GetName())],
	}
	if schema := param.GetSchema(); schema != nil && schema.GetRootNode() != nil {
		info.schema, _ = decodeYAMLNode(schema.GetRootNode())
	}
	return info
}

func (d *cliManifestDecoder) linkCommand(cmd *CLICommand) error {
	if len(cmd.Source.Routes) > 1 {
		return d.linkDispatchCommand(cmd)
	}
	return d.linkSingleRouteCommand(cmd)
}

// linkSingleRouteCommand is the pre-dispatch linker path. Keep its behavior
// isolated so dispatch-specific inference cannot change established command
// manifests.
func (d *cliManifestDecoder) linkSingleRouteCommand(cmd *CLICommand) error {
	key := strings.Join(cmd.Path, " ")
	owner := fmt.Sprintf("command %q", key)
	route := cmd.Source.Routes[0]

	opInfo, ok := d.schemaIndex.operations[route.OperationID]
	if !ok {
		return fmt.Errorf("command %q: operation %q does not exist in the document%s", key, route.OperationID, cliDidYouMean(route.OperationID, d.schemaIndex.opIDs))
	}

	if cmd.Output != nil && cmd.Output.Stream != nil {
		if err := d.linkStreamProjection(owner, cmd.Output.Stream, route.OperationID, opInfo); err != nil {
			return err
		}
	}
	if cmd.Output != nil && cmd.Output.Outcome != nil {
		if err := d.linkOutcome(owner, cmd.Output.Outcome, route.OperationID, opInfo); err != nil {
			return err
		}
	}
	artifactOpInfo := opInfo
	artifactOpID := route.OperationID
	projectionSchema := cliOperationProjectionSchema(opInfo)
	if cmd.Async != nil {
		pollInfo, err := d.linkAsync(key, cmd, opInfo)
		if err != nil {
			return err
		}
		artifactOpInfo = pollInfo
		artifactOpID = cmd.Async.OperationID
		projectionSchema = pollInfo.responseSchemaJS
	}
	if cmd.Output != nil && cmd.Output.Projection != nil {
		unavailableReason := fmt.Sprintf("operation %q declares no 2xx application/json response schema", artifactOpID)
		if err := d.lintProjection(owner, "jq", cmd.Output.Projection.JQ, unavailableReason, projectionSchema, false); err != nil {
			return err
		}
	}

	// Non-body binds: validate the parameter reference, then gate on the
	// renderer capability — v1 renders body binds only, and a silently
	// ignored bind would be worse than an error that names the gap.
	for _, list := range [][]CLICommandInput{cmd.Args, cmd.Flags} {
		for _, input := range list {
			if input.Bind == nil || input.Bind.In == "body" {
				continue
			}
			paramName := cliPointerPropertyName(input.Bind.Pointer)
			if err := d.checkParameterExists(key, input.Name, input.Bind.In, paramName, opInfo); err != nil {
				return err
			}
			return fmt.Errorf("command %q input %q binds the %s parameter %q; parameter binds require the parameter-bind renderer capability, which is not part of v1 (the operation's own parameter flags remain available on the command)", key, input.Name, input.Bind.In, paramName)
		}
	}

	if cmd.Output != nil && cmd.Output.Artifact != nil {
		if err := d.linkArtifact(key, cmd.Output.Artifact, artifactOpInfo, artifactOpID); err != nil {
			return err
		}
	}

	needsBody := len(cmd.Presets) > 0 || route.RequestVariant != ""
	for _, list := range [][]CLICommandInput{cmd.Args, cmd.Flags} {
		for _, input := range list {
			if input.Bind != nil && input.Bind.In == "body" {
				needsBody = true
			}
		}
	}
	if !needsBody {
		return nil // pure alias command: nothing to link beyond the operation
	}

	if opInfo.bodySchema == nil {
		return fmt.Errorf("command %q binds into the request body, but operation %q has no application/json request body", key, route.OperationID)
	}

	variant, err := d.resolveVariant(key, route, opInfo.bodySchema)
	if err != nil {
		return err
	}

	presetPointers := map[string]bool{}
	presetValues := map[string]any{}
	for _, preset := range cmd.Presets {
		presetPointers[preset.Bind.Pointer] = true
		presetValues[preset.Bind.Pointer] = preset.Value
	}

	for i := range cmd.Presets {
		if err := d.linkPreset(key, &cmd.Presets[i], variant); err != nil {
			return err
		}
	}

	selectors, err := d.variantSelectors(key, route, opInfo.bodySchema, variant, cmd.Presets)
	if err != nil {
		return err
	}
	cmd.Source.Routes[0].Selectors = selectors
	cmd.Source.Routes[0].PresetMergePoints = d.presetMergePoints(variant, cmd.Presets)
	for i := range cmd.Args {
		if err := d.linkInput(key, &cmd.Args[i], variant, presetPointers, presetValues, true); err != nil {
			return err
		}
	}
	for i := range cmd.Flags {
		if err := d.linkInput(key, &cmd.Flags[i], variant, presetPointers, presetValues, false); err != nil {
			return err
		}
	}

	d.checkSatisfiability(key, cmd, variant, presetPointers)
	defaults, err := d.catalogDefaultsForPresets(cmd.Presets, variant)
	if err != nil {
		return err
	}
	cmd.CatalogDefaults = cliLabelCatalogDefaults(cmd, [][]CLICatalogDefault{defaults})
	return nil
}

func (d *cliManifestDecoder) linkOperation(op *CLIOperation) error {
	opInfo, ok := d.schemaIndex.operations[op.OperationID]
	if !ok {
		return fmt.Errorf("operation %q does not exist in the document%s", op.OperationID, cliDidYouMean(op.OperationID, d.schemaIndex.opIDs))
	}
	op.BodyRequired = opInfo.bodyRequired
	if op.Output != nil && op.Output.Stream != nil {
		if err := d.linkStreamProjection(fmt.Sprintf("operation %q", op.OperationID), op.Output.Stream, op.OperationID, opInfo); err != nil {
			return err
		}
	}
	if op.Output != nil && op.Output.Outcome != nil {
		if err := d.linkOutcome(fmt.Sprintf("operation %q", op.OperationID), op.Output.Outcome, op.OperationID, opInfo); err != nil {
			return err
		}
	}
	if len(op.Flags) == 0 {
		return nil
	}
	if opInfo.bodySchema == nil {
		return fmt.Errorf("operation %q declares flags, but has no application/json request body", op.OperationID)
	}
	for i := range op.Flags {
		if err := d.linkOperationFlag(op.OperationID, &op.Flags[i], opInfo.bodySchema); err != nil {
			return err
		}
	}
	return nil
}

type cliDispatchRouteLink struct {
	route          *CLICommandRoute
	variant        *cliResolvedSchema
	presetPointers map[string]bool
	presetValues   map[string]any
}

type cliDispatchMember struct {
	ref     string
	label   string
	variant *cliResolvedSchema
}

func (d *cliManifestDecoder) linkDispatchCommand(cmd *CLICommand) error {
	key := strings.Join(cmd.Path, " ")
	operationID := cmd.Source.Routes[0].OperationID
	for i := 1; i < len(cmd.Source.Routes); i++ {
		if cmd.Source.Routes[i].OperationID != operationID {
			return fmt.Errorf("command %q: routes must all use one operation; route %q uses %q and route %q uses %q (multi-operation dispatch is not part of v1)", key, cmd.Source.Routes[0].ID, operationID, cmd.Source.Routes[i].ID, cmd.Source.Routes[i].OperationID)
		}
	}
	opInfo, ok := d.schemaIndex.operations[operationID]
	if !ok {
		return fmt.Errorf("command %q: operation %q does not exist in the document%s", key, operationID, cliDidYouMean(operationID, d.schemaIndex.opIDs))
	}
	if cmd.Output != nil && cmd.Output.Stream != nil {
		if err := d.linkStreamProjection(fmt.Sprintf("command %q", key), cmd.Output.Stream, operationID, opInfo); err != nil {
			return err
		}
	}
	if cmd.Output != nil && cmd.Output.Outcome != nil {
		if err := d.linkOutcome(fmt.Sprintf("command %q", key), cmd.Output.Outcome, operationID, opInfo); err != nil {
			return err
		}
	}
	// An async recipe links exactly as for a pinned intent: the poll operation
	// is validated and an artifact extracts from the terminal poll response.
	artifactOpInfo := opInfo
	artifactOpID := operationID
	projectionSchema := cliOperationProjectionSchema(opInfo)
	if cmd.Async != nil {
		pollInfo, err := d.linkAsync(key, cmd, opInfo)
		if err != nil {
			return err
		}
		artifactOpInfo = pollInfo
		artifactOpID = cmd.Async.OperationID
		projectionSchema = pollInfo.responseSchemaJS
	}
	// The jq projection is linted exactly as for a single-route command: the
	// routes share one operation, so the response schema is the same.
	if cmd.Output != nil && cmd.Output.Projection != nil {
		unavailableReason := fmt.Sprintf("operation %q declares no 2xx application/json response schema", artifactOpID)
		if err := d.lintProjection(fmt.Sprintf("command %q", key), "jq", cmd.Output.Projection.JQ, unavailableReason, projectionSchema, false); err != nil {
			return err
		}
	}
	for _, list := range [][]CLICommandInput{cmd.Args, cmd.Flags} {
		for _, input := range list {
			if input.Bind == nil || input.Bind.In == "body" {
				continue
			}
			paramName := cliPointerPropertyName(input.Bind.Pointer)
			if err := d.checkParameterExists(key, input.Name, input.Bind.In, paramName, opInfo); err != nil {
				return err
			}
			return fmt.Errorf("command %q input %q binds the %s parameter %q; parameter binds require the parameter-bind renderer capability, which is not part of v1 (the operation's own parameter flags remain available on the command)", key, input.Name, input.Bind.In, paramName)
		}
	}
	if cmd.Output != nil && cmd.Output.Artifact != nil {
		if err := d.linkArtifact(key, cmd.Output.Artifact, artifactOpInfo, artifactOpID); err != nil {
			return err
		}
	}
	if opInfo.bodySchema == nil {
		return fmt.Errorf("command %q binds into the request body, but operation %q has no application/json request body", key, operationID)
	}

	rawBody, _ := opInfo.bodySchema.(map[string]any)
	unionMap, _ := d.unionBodySchema(rawBody)
	if unionMap == nil {
		return fmt.Errorf("command %q: route dispatch requires the operation request body to be a oneOf union of component schemas", key)
	}
	if anyOf, ok := unionMap["anyOf"].([]any); ok && len(anyOf) > 0 {
		return fmt.Errorf("command %q: operation %q uses anyOf; mutually exclusive route dispatch requires oneOf", key, operationID)
	}
	membersRaw, _ := unionMap["oneOf"].([]any)
	if len(membersRaw) < 2 {
		return fmt.Errorf("command %q: route dispatch requires the operation request body to be a oneOf union of component schemas", key)
	}
	if disc, ok := unionMap["discriminator"].(map[string]any); ok {
		if propertyName, _ := disc["propertyName"].(string); propertyName != "" {
			return fmt.Errorf("command %q: request body union uses discriminator %q; value-discriminated route dispatch is not part of v1", key, propertyName)
		}
	}

	parent, err := d.resolveObjectSchema(unionMap, nil)
	if err != nil {
		return fmt.Errorf("command %q: request body union: %w", key, err)
	}
	members := make([]cliDispatchMember, 0, len(membersRaw))
	memberByRef := map[string]cliDispatchMember{}
	for i, raw := range membersRaw {
		resolved, err := d.resolveObjectSchema(raw, nil)
		if err != nil {
			return fmt.Errorf("command %q: request body union member %d: %w", key, i+1, err)
		}
		resolved, err = cliMergeDispatchParent(resolved, parent)
		if err != nil {
			return fmt.Errorf("command %q: request body union member %d: %w", key, i+1, err)
		}
		ref := ""
		if m, ok := raw.(map[string]any); ok {
			ref, _ = m["$ref"].(string)
		}
		label := fmt.Sprintf("inline member %d", i+1)
		if ref != "" {
			label = strings.TrimPrefix(ref, cliComponentRefPrefix)
		}
		members = append(members, cliDispatchMember{ref: ref, label: label, variant: resolved})
		if ref != "" {
			memberByRef[ref] = members[len(members)-1]
		}
	}

	links := make([]cliDispatchRouteLink, 0, len(cmd.Source.Routes))
	routeByRef := map[string]string{}
	for i := range cmd.Source.Routes {
		route := &cmd.Source.Routes[i]
		member, found := memberByRef[route.RequestVariant]
		if !found {
			return fmt.Errorf("command %q: request variant %q is not a member of the operation's request body union (members: %s)", key, route.RequestVariant, cliCandidateList(cliDispatchMemberLabels(members)))
		}
		if firstID := routeByRef[route.RequestVariant]; firstID != "" {
			return fmt.Errorf("command %q: routes %q and %q both pin request variant %q; each dispatch route must pin a distinct variant", key, firstID, route.ID, member.label)
		}
		routeByRef[route.RequestVariant] = route.ID
		links = append(links, cliDispatchRouteLink{route: route, variant: member.variant})
	}
	if cmd.Override {
		for _, member := range members {
			if member.ref == "" {
				return fmt.Errorf("command %q: override cannot cover %s because dispatch routes can pin only component-schema members", key, member.label)
			}
			if routeByRef[member.ref] == "" {
				return fmt.Errorf("command %q: override omits request variant %q; overriding would remove that variant's only generated CLI surface", key, member.label)
			}
		}
	}
	if err := d.checkDispatchValueDiscrimination(key, members); err != nil {
		return err
	}
	if err := d.linkDispatchPresets(key, cmd, links); err != nil {
		return err
	}
	for i := range links {
		selectors, err := d.variantSelectors(key, *links[i].route, opInfo.bodySchema, links[i].variant, links[i].route.Presets)
		if err != nil {
			return err
		}
		links[i].route.Selectors = selectors
		links[i].route.PresetMergePoints = d.presetMergePoints(links[i].variant, links[i].route.Presets)
	}

	for i := range cmd.Args {
		if err := d.linkDispatchInput(key, &cmd.Args[i], links, true); err != nil {
			return err
		}
	}
	for i := range cmd.Flags {
		if err := d.linkDispatchInput(key, &cmd.Flags[i], links, false); err != nil {
			return err
		}
	}
	if err := d.linkDispatchAnchors(key, cmd, links); err != nil {
		return err
	}
	if err := d.checkDispatchDefault(key, links, members); err != nil {
		return err
	}
	cmd.DispatchKeys = cliBuildDispatchKeys(cmd.Source.Routes, members)
	catalogDefaults := make([][]CLICatalogDefault, len(links))
	for i, link := range links {
		d.checkDispatchSatisfiability(key, cmd, link)
		defaults, err := d.catalogDefaultsForPresets(link.route.Presets, link.variant)
		if err != nil {
			return err
		}
		catalogDefaults[i] = defaults
	}
	cmd.CatalogDefaults = cliLabelCatalogDefaults(cmd, catalogDefaults)
	return nil
}

func cliMergeDispatchParent(variant, parent *cliResolvedSchema) (*cliResolvedSchema, error) {
	if parent == nil {
		return variant, nil
	}
	if len(parent.properties) == 0 {
		// No properties to merge, but a propertyless parent can still close
		// the composition (additionalProperties: false).
		if parent.explicitClosed && !variant.explicitClosed {
			merged := *variant
			merged.explicitClosed = true
			merged.explicitOpen = false
			return &merged, nil
		}
		return variant, nil
	}
	merged := &cliResolvedSchema{
		refName:    variant.refName,
		properties: map[string]any{},
		required:   map[string]bool{},
		explicitOpen: (variant.explicitOpen || parent.explicitOpen) &&
			!variant.explicitClosed && !parent.explicitClosed,
		explicitClosed: variant.explicitClosed || parent.explicitClosed,
		raw:            variant.raw,
	}
	for _, name := range parent.propOrder {
		merged.properties[name] = parent.properties[name]
		merged.propOrder = append(merged.propOrder, name)
	}
	for _, name := range variant.propOrder {
		if existing, ok := merged.properties[name]; ok {
			if reflect.DeepEqual(existing, variant.properties[name]) {
				continue
			}
			return nil, fmt.Errorf("property %q is defined beside oneOf and by %s with conflicting schemas", name, cliSchemaLabel(variant))
		}
		if _, exists := merged.properties[name]; !exists {
			merged.propOrder = append(merged.propOrder, name)
		}
		merged.properties[name] = variant.properties[name]
	}
	for name := range parent.required {
		merged.required[name] = true
	}
	for name := range variant.required {
		merged.required[name] = true
	}
	return merged, nil
}

func cliDispatchMemberLabels(members []cliDispatchMember) []string {
	labels := make([]string, 0, len(members))
	for _, member := range members {
		labels = append(labels, member.label)
	}
	return labels
}

func (d *cliManifestDecoder) checkDispatchValueDiscrimination(cmdKey string, members []cliDispatchMember) error {
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			left, right := members[i], members[j]
			for _, name := range left.variant.propOrder {
				rightSchema, shared := right.variant.properties[name]
				if !shared {
					continue
				}
				leftFacts, leftErr := d.propertyFacts(left.variant.properties[name], nil)
				rightFacts, rightErr := d.propertyFacts(rightSchema, nil)
				if leftErr != nil || rightErr != nil {
					continue
				}
				leftValue, leftSingle := cliSingletonPropertyValue(leftFacts)
				rightValue, rightSingle := cliSingletonPropertyValue(rightFacts)
				if leftSingle && rightSingle && !cliValuesEqual(leftValue, rightValue) {
					return fmt.Errorf("command %q: request variants %q and %q are distinguished by values of shared property %q; value-discriminated route dispatch is not part of v1", cmdKey, left.label, right.label, name)
				}
			}
		}
	}
	return nil
}

// linkOperationFlag proves that a declared operation flag can be merged into
// every possible request body. Union bodies are deliberately stricter than
// intent bindings: the property must exist with the same scalar type in every
// member so the flag cannot silently select or invalidate a variant.
func (d *cliManifestDecoder) linkOperationFlag(opID string, input *CLICommandInput, bodySchema any) error {
	if _, _, nested := cliSplitNestedPointer(input.Bind.Pointer); nested {
		return fmt.Errorf("operation %q flag %q binds %s; operation flags bind top-level body fields only (nested binds are for intent commands with an object preset)", opID, input.Name, input.Bind.Pointer)
	}
	bodyMap, _ := bodySchema.(map[string]any)
	unionMap, _ := d.unionBodySchema(bodyMap)
	rawVariants := cliUnionMembers(unionMap)
	if len(rawVariants) == 0 {
		rawVariants = []any{bodySchema}
	}

	var commonKind string
	var commonDefault any
	haveCommonDefault := false
	for i, raw := range rawVariants {
		variant, err := d.resolveObjectSchema(raw, nil)
		if err != nil {
			return fmt.Errorf("operation %q flag %q: request body variant %d: %w", opID, input.Name, i+1, err)
		}
		name := cliPointerPropertyName(input.Bind.Pointer)
		prop, exists := variant.properties[name]
		if !exists {
			return fmt.Errorf("operation %q flag %q binds %s, but that property is absent from request body variant %s%s", opID, input.Name, input.Bind.Pointer, cliSchemaLabel(variant), cliDidYouMean(name, variant.propOrder))
		}
		facts, err := d.propertyFacts(prop, nil)
		if err != nil {
			return fmt.Errorf("operation %q flag %q bind %s in variant %s: %w", opID, input.Name, input.Bind.Pointer, cliSchemaLabel(variant), err)
		}
		if facts.readOnly {
			return fmt.Errorf("operation %q flag %q binds readOnly property %s in variant %s", opID, input.Name, input.Bind.Pointer, cliSchemaLabel(variant))
		}
		if facts.unionArms != nil || len(facts.kinds) != 1 || !cliIsScalarKind(facts.kinds[0]) {
			return fmt.Errorf("operation %q flag %q binds %s in variant %s, which is not a single scalar property", opID, input.Name, input.Bind.Pointer, cliSchemaLabel(variant))
		}
		kind := facts.kinds[0]
		if commonKind == "" {
			commonKind = kind
		} else if commonKind != kind {
			return fmt.Errorf("operation %q flag %q binds %s with incompatible types across request body variants: %s and %s", opID, input.Name, input.Bind.Pointer, commonKind, kind)
		}
		if input.Type != "" && input.Type != kind {
			return fmt.Errorf("operation %q flag %q declares type %s, but %s is %s in variant %s", opID, input.Name, input.Type, input.Bind.Pointer, kind, cliSchemaLabel(variant))
		}
		if input.Summary == "" && facts.description != "" {
			input.Summary = cliFirstSentence(facts.description)
		}
		if input.SuggestionsDeclared {
			if err := cliCheckSuggestions(input.Enum, kind); err != nil {
				return fmt.Errorf("operation %q flag %q suggestions: %w", opID, input.Name, err)
			}
		} else if len(rawVariants) == 1 && len(facts.enum) > 0 {
			input.Enum = facts.enum
		}
		if input.DefaultFrom == "schema" {
			if !facts.hasDefault {
				return fmt.Errorf("operation %q flag %q declares defaultFrom: schema, but the schema property at %s has no default in variant %s", opID, input.Name, input.Bind.Pointer, cliSchemaLabel(variant))
			}
			if !haveCommonDefault {
				commonDefault, haveCommonDefault = facts.defaultVal, true
			} else if !cliValuesEqual(commonDefault, facts.defaultVal) {
				return fmt.Errorf("operation %q flag %q declares defaultFrom: schema, but %s has different defaults across request body variants", opID, input.Name, input.Bind.Pointer)
			}
		}
	}
	if input.Type == "" {
		input.Type = commonKind
	}
	if input.DefaultFrom == "schema" {
		normalized, err := cliCheckSchemaDefault(commonDefault, input.Type)
		if err != nil {
			return fmt.Errorf("operation %q flag %q defaultFrom: schema: the schema default at %s does not match type %s: %w", opID, input.Name, input.Bind.Pointer, input.Type, err)
		}
		input.Default = normalized
	} else if input.Default != nil {
		if err := cliCheckScalarValue(input.Default, input.Type); err != nil {
			return fmt.Errorf("operation %q flag %q default: %w", opID, input.Name, err)
		}
	}
	return nil
}

func (d *cliManifestDecoder) linkAsync(cmdKey string, cmd *CLICommand, createInfo *cliOperationInfo) (*cliOperationInfo, error) {
	recipe := cmd.Async
	pollInfo, ok := d.schemaIndex.operations[recipe.OperationID]
	if !ok {
		return nil, fmt.Errorf("command %q async.op: operation %q does not exist in the document%s", cmdKey, recipe.OperationID, cliDidYouMean(recipe.OperationID, d.schemaIndex.opIDs))
	}
	if pollInfo.method != "get" {
		return nil, fmt.Errorf("command %q async.op %q uses %s; polling operations must use GET so foreground polling cannot repeat a mutating request", cmdKey, recipe.OperationID, strings.ToUpper(pollInfo.method))
	}
	if pollInfo.hasRequestBody {
		return nil, fmt.Errorf("command %q async.op %q declares a request body; polling operations must be bodyless GET operations", cmdKey, recipe.OperationID)
	}
	if len(createInfo.jsonResponses) == 0 {
		return nil, fmt.Errorf("command %q declares async, but create operation %q has no application/json 2xx response from which to read id.from %s", cmdKey, cmd.Source.Routes[0].OperationID, recipe.ID.From)
	}
	for _, response := range createInfo.jsonResponses {
		if _, err := d.linkAsyncResponsePointer(recipe.ID.From, response.schema, false); err != nil {
			return nil, fmt.Errorf("command %q async.id.from %s in create operation %q response %s: %w", cmdKey, recipe.ID.From, cmd.Source.Routes[0].OperationID, response.code, err)
		}
	}

	targetKey := cliParameterKey(recipe.ID.To.In, recipe.ID.To.Name)
	target, ok := pollInfo.parameterInfo[targetKey]
	if !ok {
		return nil, fmt.Errorf("command %q async.id.to: operation %q has no %s parameter %q%s", cmdKey, recipe.OperationID, recipe.ID.To.In, recipe.ID.To.Name, cliDidYouMean(recipe.ID.To.Name, pollInfo.params[recipe.ID.To.In]))
	}
	if !target.required {
		return nil, fmt.Errorf("command %q async.id.to names optional %s parameter %q on operation %q; the handle target must be required", cmdKey, recipe.ID.To.In, recipe.ID.To.Name, recipe.OperationID)
	}
	if err := d.requireStringSchema(target.schema); err != nil {
		return nil, fmt.Errorf("command %q async.id.to %s parameter %q on operation %q must have a string schema: %w", cmdKey, recipe.ID.To.In, recipe.ID.To.Name, recipe.OperationID, err)
	}
	if format := d.schemaFormat(target.schema, nil); cliAsyncTypedFormats[format] {
		return nil, fmt.Errorf("command %q async.id.to %s parameter %q on operation %q has format %s; the handle target must be a plain string", cmdKey, recipe.ID.To.In, recipe.ID.To.Name, recipe.OperationID, format)
	}
	covered := map[string]bool{targetKey: true}
	paramNames := make([]string, 0, len(recipe.Params))
	for name := range recipe.Params {
		paramNames = append(paramNames, name)
	}
	sort.Strings(paramNames)
	for _, name := range paramNames {
		if name == recipe.ID.To.Name {
			return nil, fmt.Errorf("command %q async.params.%s duplicates the handle parameter declared by async.id.to", cmdKey, name)
		}
		query := pollInfo.parameterInfo[cliParameterKey("query", name)]
		header := pollInfo.parameterInfo[cliParameterKey("header", name)]
		if query != nil && header != nil {
			return nil, fmt.Errorf("command %q async.params.%s is ambiguous: poll operation %q has both query and header parameters with that name; location-qualified async params are not part of v1", cmdKey, name, recipe.OperationID)
		}
		param := query
		if param == nil {
			param = header
		}
		if param == nil {
			if pollInfo.parameterInfo[cliParameterKey("path", name)] != nil {
				return nil, fmt.Errorf("command %q async.params.%s names a path parameter on poll operation %q; path parameters must be bound by async.id.to", cmdKey, name, recipe.OperationID)
			}
			candidates := append([]string{}, pollInfo.params["query"]...)
			candidates = append(candidates, pollInfo.params["header"]...)
			return nil, fmt.Errorf("command %q async.params.%s: poll operation %q has no query or header parameter %q%s", cmdKey, name, recipe.OperationID, name, cliDidYouMean(name, candidates))
		}
		if param.global {
			return nil, fmt.Errorf("command %q async.params.%s names global %s parameter %q on poll operation %q; configure global parameters through their generated CLI surface", cmdKey, name, param.in, param.name, recipe.OperationID)
		}
		if param.schema == nil {
			return nil, fmt.Errorf("command %q async.params.%s names %s parameter %q on poll operation %q, but it has no schema to validate", cmdKey, name, param.in, param.name, recipe.OperationID)
		}
		facts, err := d.propertyFacts(param.schema, nil)
		if err != nil {
			return nil, fmt.Errorf("command %q async.params.%s: inspect %s parameter %q on poll operation %q: %w", cmdKey, name, param.in, param.name, recipe.OperationID, err)
		}
		if len(facts.kinds) != 1 || !cliContains([]string{"string", "int", "float", "bool"}, facts.kinds[0]) {
			return nil, fmt.Errorf("command %q async.params.%s targets %s parameter %q on poll operation %q, which must have a scalar string, integer, number, or boolean schema", cmdKey, name, param.in, param.name, recipe.OperationID)
		}
		if err := cliCheckScalarValue(recipe.Params[name], facts.kinds[0]); err != nil {
			return nil, fmt.Errorf("command %q async.params.%s: %w", cmdKey, name, err)
		}
		if err := cliCheckEnumValue(recipe.Params[name], facts); err != nil {
			return nil, fmt.Errorf("command %q async.params.%s: %w", cmdKey, name, err)
		}
		if err := cliCheckFormattedValue(recipe.Params[name], d.schemaFormat(param.schema, nil)); err != nil {
			return nil, fmt.Errorf("command %q async.params.%s: %w", cmdKey, name, err)
		}
		recipe.ResolvedParams = append(recipe.ResolvedParams, CLICommandAsyncResolvedParameter{In: param.in, Name: param.name, Value: recipe.Params[name]})
		covered[cliParameterKey(param.in, param.name)] = true
	}
	for _, param := range pollInfo.parameterInfo {
		if !param.required || param.global || covered[cliParameterKey(param.in, param.name)] {
			continue
		}
		return nil, fmt.Errorf("command %q async.op %q has required non-global %s parameter %q not covered by async.id.to or async.params", cmdKey, recipe.OperationID, param.in, param.name)
	}

	if len(pollInfo.jsonResponses) == 0 {
		if pollInfo.streamMediaSeen {
			return nil, fmt.Errorf("command %q async.op %q is streaming-only; polling requires an application/json 2xx response", cmdKey, recipe.OperationID)
		}
		return nil, fmt.Errorf("command %q async.op %q has no application/json 2xx response to inspect", cmdKey, recipe.OperationID)
	}
	enumStates := map[string]bool{}
	for _, response := range pollInfo.jsonResponses {
		stateFacts, err := d.linkAsyncResponsePointer(recipe.StateFrom, response.schema, true)
		if err != nil {
			return nil, fmt.Errorf("command %q async.statePointer %s in poll operation %q response %s: %w", cmdKey, recipe.StateFrom, recipe.OperationID, response.code, err)
		}
		for _, facts := range stateFacts {
			if facts.openEnum {
				return nil, fmt.Errorf("command %q async.statePointer %s in poll operation %q response %s is an open enum; every possible state must be classifiable at generation time", cmdKey, recipe.StateFrom, recipe.OperationID, response.code)
			}
			for _, value := range facts.enum {
				state, ok := value.(string)
				if !ok {
					return nil, fmt.Errorf("command %q async.statePointer %s in poll operation %q response %s has a non-string enum member %v", cmdKey, recipe.StateFrom, recipe.OperationID, response.code, value)
				}
				enumStates[state] = true
			}
		}
	}
	for state := range recipe.States {
		if !enumStates[state] {
			return nil, fmt.Errorf("command %q async.states classifies %q, which is not an enum member at %s in poll operation %q%s", cmdKey, state, recipe.StateFrom, recipe.OperationID, cliDidYouMean(state, cliSortedKeys(enumStates)))
		}
	}
	for state := range enumStates {
		if _, classified := recipe.States[state]; !classified {
			return nil, fmt.Errorf("command %q async.states does not classify enum member %q from %s in poll operation %q; classify every member as pending, success, failure, or handoff", cmdKey, state, recipe.StateFrom, recipe.OperationID)
		}
	}
	if recipe.errorExplicit {
		for _, response := range pollInfo.jsonResponses {
			if err := d.linkAsyncErrorBindings(cmdKey, recipe, response.schema, response.code); err != nil {
				return nil, err
			}
		}
	}
	recipe.CreateResponseCode = createInfo.responseCode
	recipe.ResponseCode = pollInfo.responseCode
	return pollInfo, nil
}

// linkOutcome checks the state pointers against the operation's responses:
// the JSON state must be a closed string enum that the states table covers
// exactly, and the stream pointers must resolve in at least one event shape.
func (d *cliManifestDecoder) linkOutcome(owner string, outcome *CLICommandOutcome, opID string, opInfo *cliOperationInfo) error {
	if outcome.StatePointer != "" {
		if len(opInfo.jsonResponses) == 0 {
			return fmt.Errorf("%s output.outcome.statePointer %s: operation %q has no application/json 2xx response to inspect; a streaming-only operation declares streamStatePointer instead", owner, outcome.StateFrom, opID)
		}
		enumStates := map[string]bool{}
		for _, response := range opInfo.jsonResponses {
			stateFacts, err := d.linkAsyncResponsePointer(outcome.StateFrom, response.schema, true)
			if err != nil {
				return fmt.Errorf("%s output.outcome.statePointer %s in operation %q response %s: %w", owner, outcome.StateFrom, opID, response.code, err)
			}
			for _, facts := range stateFacts {
				if facts.openEnum {
					return fmt.Errorf("%s output.outcome.statePointer %s in operation %q response %s is an open enum; every possible state must be classifiable at generation time", owner, outcome.StateFrom, opID, response.code)
				}
				for _, value := range facts.enum {
					if value == nil {
						continue
					}
					state, ok := value.(string)
					if !ok {
						return fmt.Errorf("%s output.outcome.statePointer %s in operation %q response %s has a non-string enum member %v", owner, outcome.StateFrom, opID, response.code, value)
					}
					enumStates[state] = true
				}
			}
		}
		for state := range outcome.States {
			if !enumStates[state] {
				return fmt.Errorf("%s output.outcome.states classifies %q, which is not an enum member at %s in operation %q%s", owner, state, outcome.StateFrom, opID, cliDidYouMean(state, cliSortedKeys(enumStates)))
			}
		}
		for state := range enumStates {
			if _, classified := outcome.States[state]; !classified {
				return fmt.Errorf("%s output.outcome.states does not classify enum member %q from %s in operation %q; classify every member as pending, success, failure, or handoff", owner, state, outcome.StateFrom, opID)
			}
		}
		if outcome.ErrorPointer != "" {
			resolved := false
			candidates := map[string]bool{}
			for _, response := range opInfo.jsonResponses {
				found, names, err := d.linkOutcomeErrorPointer(outcome.ErrorFrom, response.schema)
				if err != nil {
					return fmt.Errorf("%s output.outcome.errorPointer %s in operation %q response %s: %w", owner, outcome.ErrorFrom, opID, response.code, err)
				}
				resolved = resolved || found
				for _, name := range names {
					candidates[name] = true
				}
			}
			if !resolved {
				return fmt.Errorf("%s output.outcome.errorPointer %s does not resolve in any application/json 2xx response variant of operation %q%s", owner, outcome.ErrorFrom, opID, cliDidYouMean(cliLastPathSegmentName(outcome.ErrorFrom), cliSortedKeys(candidates)))
			}
		}
	}
	if outcome.StreamStatePointer != "" {
		if !opInfo.streamMediaSeen {
			return fmt.Errorf("%s declares output.outcome.streamStatePointer, but operation %q has no streaming (text/event-stream or JSONL) success response", owner, opID)
		}
		walker, err := d.walkStreamSelection(owner, "output.outcome.streamStatePointer", outcome.StreamStateFrom, opInfo)
		if err != nil {
			return err
		}
		if err := d.linkOutcomeStreamStates(owner, outcome, opID, walker); err != nil {
			return err
		}
		if outcome.StreamErrorPointer != "" {
			if err := d.linkStreamSelection(owner, cliStreamDetailField, outcome.StreamErrorFrom, opInfo); err != nil {
				return err
			}
		}
	}
	return nil
}

const cliStreamDetailField = "output.outcome.streamErrorPointer"

// linkOutcomeErrorPointer accepts failure detail that exists in only some
// response variants (a failed union arm, say); the runtime treats absence as
// no detail. It reports whether the path resolved anywhere in this response
// and, when it did not, the property names seen where it went missing.
func (d *cliManifestDecoder) linkOutcomeErrorPointer(path string, responseSchema any) (bool, []string, error) {
	views, candidates, err := d.walkResponsePathViews(path, responseSchema, true)
	if err != nil {
		return false, nil, err
	}
	if len(views) == 0 {
		return false, candidates, nil
	}
	facts, err := d.responseLeafFacts(views)
	if err != nil {
		return false, nil, err
	}
	for _, propertyFacts := range facts {
		if len(propertyFacts.kinds) != 1 || (propertyFacts.kinds[0] != "object" && propertyFacts.kinds[0] != "array") {
			return false, nil, stderrors.New("selected failure detail must be an object or an array in every response variant")
		}
	}
	return true, nil, nil
}

// linkOutcomeStreamStates holds the states table to the enum the stream
// pointer selects. A stream-only table must match that enum exactly; when
// statePointer already pinned the table to the JSON enum, the stream enum may
// be a subset of it, but every member must still be classified.
func (d *cliManifestDecoder) linkOutcomeStreamStates(owner string, outcome *CLICommandOutcome, opID string, walker *cliStreamPathWalker) error {
	if walker.stateOpen {
		return fmt.Errorf("%s output.outcome.streamStatePointer %s in operation %q is an open enum; every possible state must be classifiable at generation time", owner, outcome.StreamStateFrom, opID)
	}
	if walker.resolved == 0 {
		return nil
	}
	if walker.stateNoEnum || len(walker.stateEnum) == 0 {
		d.warnf("%s output.outcome.streamStatePointer %s in operation %q does not select a closed string enum in every event shape; states cannot be verified at generation time, and a state missing from the table keeps exit 0 at runtime", owner, outcome.StreamStateFrom, opID)
		return nil
	}
	if outcome.StatePointer == "" {
		for _, state := range slices.Sorted(maps.Keys(outcome.States)) {
			if !walker.stateEnum[state] {
				return fmt.Errorf("%s output.outcome.states classifies %q, which is not an enum member at %s in operation %q%s", owner, state, outcome.StreamStateFrom, opID, cliDidYouMean(state, cliSortedKeys(walker.stateEnum)))
			}
		}
	}
	for _, state := range cliSortedKeys(walker.stateEnum) {
		if _, classified := outcome.States[state]; !classified {
			return fmt.Errorf("%s output.outcome.states does not classify enum member %q from %s in operation %q; classify every member as pending, success, failure, or handoff", owner, state, outcome.StreamStateFrom, opID)
		}
	}
	return nil
}

func cliLastPathSegmentName(path string) string {
	segments, err := cliParseSingularPath(path)
	if err != nil || len(segments) == 0 || segments[len(segments)-1].IsIndex {
		return ""
	}
	return segments[len(segments)-1].Name
}

// linkAsyncErrorBindings validates explicitly declared failure-detail
// bindings against a poll response schema. The errorField member must exist
// at the root of at least one traversable variant (failure detail
// legitimately lives on only the failure arm of a response union), and
// everywhere it resolves it must be an object whose every object view
// carries errorMessageField as a string — so a misspelled binding fails
// generation with the member named instead of silently degrading every
// failure message at runtime. Defaulted bindings are deliberately not
// linked, keeping the v1 best-effort reads for undeclared recipes.
func (d *cliManifestDecoder) linkAsyncErrorBindings(cmdKey string, recipe *CLICommandAsync, responseSchema any, responseCode string) error {
	views, err := d.artifactExpand(responseSchema, 0)
	if err != nil {
		return fmt.Errorf("command %q async.errorField: %w", cmdKey, err)
	}
	if len(views) == 0 {
		return fmt.Errorf("command %q async.errorField: the %s response of poll operation %q has no traversable object variant to bind %q against", cmdKey, responseCode, recipe.OperationID, recipe.ErrorField)
	}
	rootCandidates := map[string]bool{}
	resolved := false
	for _, view := range views {
		prop, ok := view.properties[recipe.ErrorField]
		if !ok {
			for _, name := range view.propOrder {
				rootCandidates[name] = true
			}
			continue
		}
		resolved = true
		errorViews, err := d.artifactExpand(prop, 0)
		if err != nil {
			return fmt.Errorf("command %q async.errorField %q: %w", cmdKey, recipe.ErrorField, err)
		}
		objectSeen := false
		for _, errorView := range errorViews {
			if !cliAsyncErrorViewIsObject(errorView) {
				// Non-object arms (a nullable union's null arm, say) cannot
				// carry the message member and are skipped, like the
				// non-object union arms an artifact pointer walks past.
				continue
			}
			objectSeen = true
			messageProp, ok := errorView.properties[recipe.ErrorMessageField]
			if !ok {
				return fmt.Errorf("command %q async.errorMessageField: property %q does not resolve inside the %q member of the %s response of poll operation %q%s", cmdKey, recipe.ErrorMessageField, recipe.ErrorField, responseCode, recipe.OperationID, cliDidYouMean(recipe.ErrorMessageField, errorView.propOrder))
			}
			facts, err := d.propertyFacts(messageProp, nil)
			if err != nil {
				return fmt.Errorf("command %q async.errorMessageField: %w", cmdKey, err)
			}
			if len(facts.kinds) != 1 || facts.kinds[0] != "string" {
				return fmt.Errorf("command %q async.errorMessageField: property %q inside the %q member of the %s response of poll operation %q must be a string", cmdKey, recipe.ErrorMessageField, recipe.ErrorField, responseCode, recipe.OperationID)
			}
		}
		if !objectSeen {
			return fmt.Errorf("command %q async.errorField: property %q in the %s response of poll operation %q is not an object; the failure-detail binding must name the object member that carries the %q message", cmdKey, recipe.ErrorField, responseCode, recipe.OperationID, recipe.ErrorMessageField)
		}
	}
	if !resolved {
		return fmt.Errorf("command %q async.errorField: property %q does not resolve at the root of the %s response of poll operation %q%s", cmdKey, recipe.ErrorField, responseCode, recipe.OperationID, cliDidYouMean(recipe.ErrorField, cliSortedKeys(rootCandidates)))
	}
	return nil
}

// cliAsyncErrorViewIsObject reports whether an expanded view of the bound
// failure-detail member can carry object properties (scalar and null union
// arms cannot).
func cliAsyncErrorViewIsObject(view *cliResolvedSchema) bool {
	if len(view.propOrder) > 0 {
		return true
	}
	if view.raw == nil {
		return false
	}
	switch declared := view.raw["type"].(type) {
	case string:
		return declared == "object"
	case []any:
		for _, member := range declared {
			if member == "object" {
				return true
			}
		}
		return false
	}
	return view.raw["properties"] != nil || view.raw["additionalProperties"] != nil
}

// linkAsyncResponsePointer is deliberately stricter than artifact linking:
// a handle or state missing from even one response-union arm would otherwise
// become a data-dependent runtime failure.
func (d *cliManifestDecoder) linkAsyncResponsePointer(path string, responseSchema any, requireEnum bool) ([]*cliPropertyFacts, error) {
	views, err := d.walkResponsePath(path, responseSchema)
	if err != nil {
		return nil, err
	}
	facts, err := d.responseLeafFacts(views)
	if err != nil {
		return nil, err
	}
	for _, propertyFacts := range facts {
		if len(propertyFacts.kinds) != 1 || propertyFacts.kinds[0] != "string" {
			return nil, stderrors.New("selected value must be a string in every response variant")
		}
		if requireEnum && len(propertyFacts.enum) == 0 {
			return nil, stderrors.New("selected state must declare a non-empty enum in every response variant")
		}
	}
	return facts, nil
}

// walkResponsePath follows a singular path through every traversable variant
// of a response schema and returns the leaf views.
func (d *cliManifestDecoder) walkResponsePath(path string, responseSchema any) ([]*cliResolvedSchema, error) {
	views, _, err := d.walkResponsePathViews(path, responseSchema, false)
	return views, err
}

// walkResponsePathViews is walkResponsePath with a lenient mode that drops a
// variant the path does not resolve in instead of failing; it then also
// returns the property names available where the last variants were lost,
// for did-you-mean when nothing resolved.
func (d *cliManifestDecoder) walkResponsePathViews(path string, responseSchema any, lenient bool) ([]*cliResolvedSchema, []string, error) {
	segments, err := cliParseSingularPath(path)
	if err != nil {
		return nil, nil, err
	}
	views, err := d.artifactExpand(responseSchema, 0)
	if err != nil {
		return nil, nil, err
	}
	if len(views) == 0 {
		return nil, nil, stderrors.New("response schema has no traversable object variant")
	}
	var candidates []string
	for i, segment := range segments {
		var next []*cliResolvedSchema
		candidates = nil
		for _, view := range views {
			var child any
			if segment.IsIndex {
				if view.raw == nil || view.raw["type"] != "array" {
					if lenient {
						continue
					}
					return nil, nil, fmt.Errorf("segment %d ([%d]) does not address an array in every response variant", i+1, segment.Index)
				}
				child = view.raw["items"]
				if child == nil {
					return nil, nil, fmt.Errorf("segment %d ([%d]) addresses an array without an items schema", i+1, segment.Index)
				}
			} else {
				var ok bool
				child, ok = view.properties[segment.Name]
				if !ok {
					if lenient {
						candidates = append(candidates, view.propOrder...)
						continue
					}
					return nil, nil, fmt.Errorf("segment %d (.%s) does not resolve in every response variant%s", i+1, segment.Name, cliDidYouMean(segment.Name, view.propOrder))
				}
			}
			expanded, err := d.artifactExpand(child, 0)
			if err != nil {
				return nil, nil, fmt.Errorf("segment %d: %w", i+1, err)
			}
			if len(expanded) == 0 {
				if lenient {
					continue
				}
				return nil, nil, fmt.Errorf("segment %d does not resolve to a schema in every response variant", i+1)
			}
			next = append(next, expanded...)
		}
		views = next
		if len(views) == 0 {
			break
		}
	}
	return views, candidates, nil
}

// responseLeafFacts reports the facts of each non-null leaf view; a null
// union arm or a "null" entry in a type list is accepted as absence.
func (d *cliManifestDecoder) responseLeafFacts(views []*cliResolvedSchema) ([]*cliPropertyFacts, error) {
	facts := make([]*cliPropertyFacts, 0, len(views))
	for _, view := range views {
		raw := cliWithoutNullType(view.raw)
		if raw == nil {
			continue
		}
		propertyFacts, err := d.propertyFacts(raw, nil)
		if err != nil {
			return nil, err
		}
		facts = append(facts, propertyFacts)
	}
	if len(facts) == 0 {
		return nil, stderrors.New("selected value is always null")
	}
	return facts, nil
}

func cliWithoutNullType(schema map[string]any) map[string]any {
	switch typ := schema["type"].(type) {
	case string:
		if typ == "null" {
			return nil
		}
	case []any:
		kept := make([]any, 0, len(typ))
		for _, t := range typ {
			if t != "null" {
				kept = append(kept, t)
			}
		}
		if len(kept) == len(typ) {
			return schema
		}
		if len(kept) == 0 {
			return nil
		}
		stripped := maps.Clone(schema)
		if len(kept) == 1 {
			stripped["type"] = kept[0]
		} else {
			stripped["type"] = kept
		}
		return stripped
	}
	return schema
}

func (d *cliManifestDecoder) requireStringSchema(schema any) error {
	if schema == nil {
		return stderrors.New("parameter schema is missing")
	}
	facts, err := d.propertyFacts(schema, nil)
	if err != nil {
		return err
	}
	if len(facts.kinds) != 1 || facts.kinds[0] != "string" {
		if len(facts.kinds) == 0 {
			return stderrors.New("schema is untyped")
		}
		return fmt.Errorf("got %s", strings.Join(facts.kinds, " or "))
	}
	return nil
}

func (d *cliManifestDecoder) schemaFormat(schema any, seen []string) string {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return ""
	}
	if format, ok := schemaMap["format"].(string); ok && format != "" {
		return format
	}
	if ref, ok := schemaMap["$ref"].(string); ok && strings.HasPrefix(ref, cliComponentRefPrefix) {
		name := strings.TrimPrefix(ref, cliComponentRefPrefix)
		if component, ok := d.schemaIndex.components[name]; ok && !cliContains(seen, name) {
			if format := d.schemaFormat(component, append(seen, name)); format != "" {
				return format
			}
		}
	}
	if allOf, ok := schemaMap["allOf"].([]any); ok {
		for _, branch := range allOf {
			if format := d.schemaFormat(branch, seen); format != "" {
				return format
			}
		}
	}
	return ""
}

var cliAsyncTypedFormats = map[string]bool{"date": true, "date-time": true, "bigint": true, "decimal": true}

func cliCheckFormattedValue(value any, format string) error {
	str, _ := value.(string)
	switch format {
	case "date-time":
		if _, err := time.Parse(time.RFC3339Nano, str); err != nil {
			if _, err := time.Parse(time.RFC3339, str); err != nil {
				return fmt.Errorf("value %v is not an RFC 3339 date-time (parameter format is date-time)", value)
			}
		}
	case "date":
		if _, err := time.Parse("2006-01-02", str); err != nil {
			return fmt.Errorf("value %v is not a YYYY-MM-DD date (parameter format is date)", value)
		}
	case "bigint", "decimal":
		return fmt.Errorf("parameter format %s is not supported for async params", format)
	}
	return nil
}

func cliSingletonPropertyValue(facts *cliPropertyFacts) (any, bool) {
	if facts.hasConst {
		return facts.constVal, true
	}
	if len(facts.enum) == 1 {
		return facts.enum[0], true
	}
	return nil, false
}

func (d *cliManifestDecoder) linkDispatchPresets(cmdKey string, cmd *CLICommand, links []cliDispatchRouteLink) error {
	for i := range cmd.Presets {
		name := cliPointerPropertyName(cmd.Presets[i].Bind.Pointer)
		var normalized any
		for routeIndex := range links {
			link := &links[routeIndex]
			if _, declared := link.variant.properties[name]; !declared {
				return fmt.Errorf("command %q: command preset %s is not declared by route %q (%s); command presets must apply to every route", cmdKey, cmd.Presets[i].Bind.Pointer, link.route.ID, cliSchemaLabel(link.variant))
			}
			candidate := cmd.Presets[i]
			if err := d.linkPreset(cmdKey, &candidate, link.variant); err != nil {
				return err
			}
			if routeIndex == 0 {
				normalized = candidate.Value
			} else if !reflect.DeepEqual(normalized, candidate.Value) {
				return fmt.Errorf("command %q: command preset %s normalizes differently for routes %q and %q; use route-local presets instead", cmdKey, candidate.Bind.Pointer, links[0].route.ID, link.route.ID)
			}
		}
		cmd.Presets[i].Value = normalized
	}

	for i := range links {
		link := &links[i]
		routeLocal := append([]CLICommandPreset(nil), link.route.Presets...)
		for j := range routeLocal {
			if err := d.linkPreset(cmdKey, &routeLocal[j], link.variant); err != nil {
				return err
			}
		}
		link.route.Presets = cliMergeDispatchPresets(cmd.Presets, routeLocal)
		link.presetPointers = map[string]bool{}
		link.presetValues = map[string]any{}
		for _, preset := range link.route.Presets {
			link.presetPointers[preset.Bind.Pointer] = true
			link.presetValues[preset.Bind.Pointer] = preset.Value
		}
	}
	return nil
}

func cliMergeDispatchPresets(command, route []CLICommandPreset) []CLICommandPreset {
	effective := append([]CLICommandPreset(nil), command...)
	index := map[string]int{}
	for i := range effective {
		index[effective[i].Bind.Pointer] = i
	}
	for _, preset := range route {
		if i, exists := index[preset.Bind.Pointer]; exists {
			effective[i] = preset
			continue
		}
		index[preset.Bind.Pointer] = len(effective)
		effective = append(effective, preset)
	}
	return effective
}

type cliDispatchInputFacts struct {
	link       *cliDispatchRouteLink
	facts      *cliPropertyFacts
	normalized CLICommandInput
}

func (d *cliManifestDecoder) linkDispatchInput(cmdKey string, input *CLICommandInput, links []cliDispatchRouteLink, isArg bool) error {
	name := cliPointerPropertyName(input.Bind.Pointer)
	what := "flag"
	if isArg {
		what = "arg"
	}
	if _, _, nested := cliSplitNestedPointer(input.Bind.Pointer); nested {
		return fmt.Errorf("command %q: %s %q binds %s; nested binds are not part of route dispatch (routes: commands bind top-level body fields only)", cmdKey, what, input.Name, input.Bind.Pointer)
	}
	var declaring []cliDispatchInputFacts
	for i := range links {
		link := &links[i]
		prop, declared := link.variant.properties[name]
		if !declared {
			continue
		}
		facts, err := d.propertyFacts(prop, nil)
		if err != nil {
			return fmt.Errorf("command %q %s %q bind %s: %w", cmdKey, what, input.Name, input.Bind.Pointer, err)
		}
		declaring = append(declaring, cliDispatchInputFacts{link: link, facts: facts})
		input.RouteIDs = append(input.RouteIDs, link.route.ID)
	}
	if len(declaring) == 0 {
		return fmt.Errorf("command %q: %s %q bind %s is not declared by any routed variant; additionalProperties cannot establish route membership", cmdKey, what, input.Name, input.Bind.Pointer)
	}
	if isArg && len(declaring) != len(links) {
		return fmt.Errorf("command %q: arg %q is declared only by route %q; positional route selection is not part of v1", cmdKey, input.Name, declaring[0].link.route.ID)
	}

	for _, declared := range declaring {
		shape := cliDispatchPropertyShape(declared.facts)
		if shape == "array" || shape == "object" {
			candidate := *input
			return d.linkInput(cmdKey, &candidate, declared.link.variant, declared.link.presetPointers, declared.link.presetValues, isArg)
		}
	}
	for i := 1; i < len(declaring); i++ {
		leftShape := cliDispatchPropertyShape(declaring[0].facts)
		rightShape := cliDispatchPropertyShape(declaring[i].facts)
		if (leftShape == "union") != (rightShape == "union") {
			return fmt.Errorf("command %q: %s %q bind %s has incompatible route shapes (%s: %s, %s: %s); union/scalar route bindings are not part of v1", cmdKey, what, input.Name, input.Bind.Pointer, declaring[0].link.route.ID, leftShape, declaring[i].link.route.ID, rightShape)
		}
		if leftShape != "union" && leftShape != rightShape {
			return fmt.Errorf("command %q: %s %q bind %s has incompatible route types (%s: %s, %s: %s); one dispatch flag must have one scalar type", cmdKey, what, input.Name, input.Bind.Pointer, declaring[0].link.route.ID, leftShape, declaring[i].link.route.ID, rightShape)
		}
	}
	if input.Summary == "" {
		firstDescription := strings.TrimSpace(declaring[0].facts.description)
		for i := 1; i < len(declaring); i++ {
			if strings.TrimSpace(declaring[i].facts.description) != firstDescription {
				return fmt.Errorf("command %q: %s %q bind %s has different schema descriptions on routes %q and %q; set summary: explicitly", cmdKey, what, input.Name, input.Bind.Pointer, declaring[0].link.route.ID, declaring[i].link.route.ID)
			}
		}
	}

	defaultFromSchema := input.DefaultFrom == "schema"
	if defaultFromSchema {
		first := declaring[0]
		if !first.facts.hasDefault {
			return fmt.Errorf("command %q: %s %q defaultFrom: schema requires identical defaults across declaring routes; route %q has no schema default", cmdKey, what, input.Name, first.link.route.ID)
		}
		for i := 1; i < len(declaring); i++ {
			current := declaring[i]
			if !current.facts.hasDefault || !reflect.DeepEqual(first.facts.defaultVal, current.facts.defaultVal) {
				return fmt.Errorf("command %q: %s %q defaultFrom: schema requires identical defaults across declaring routes; routes %q and %q disagree", cmdKey, what, input.Name, first.link.route.ID, current.link.route.ID)
			}
		}
	}

	for i := range declaring {
		candidate := *input
		if defaultFromSchema {
			candidate.DefaultFrom = ""
			candidate.Default = nil
		}
		if err := d.linkInput(cmdKey, &candidate, declaring[i].link.variant, declaring[i].link.presetPointers, declaring[i].link.presetValues, isArg); err != nil {
			return err
		}
		if input.Default != nil {
			defaultFacts := declaring[i].facts
			if defaultFacts.unionArms != nil {
				selected, selectErr := d.selectUnionArm(defaultFacts.unionArms, candidate.Type)
				if selectErr != nil {
					return fmt.Errorf("command %q: %s %q default cannot be validated on route %q: %w", cmdKey, what, input.Name, declaring[i].link.route.ID, selectErr)
				}
				defaultFacts = selected
			}
			if err := cliCheckEnumValue(input.Default, defaultFacts); err != nil {
				return fmt.Errorf("command %q: %s %q default is invalid on route %q: %w", cmdKey, what, input.Name, declaring[i].link.route.ID, err)
			}
		}
		declaring[i].normalized = candidate
	}
	input.Type = declaring[0].normalized.Type
	if input.Summary == "" {
		input.Summary = declaring[0].normalized.Summary
	}
	if !input.SuggestionsDeclared {
		input.Enum = cliIntersectDispatchEnums(declaring)
	}

	if !isArg {
		if value, ok := cliDispatchPresetDefault(declaring, input.Bind.Pointer, input.Type); ok {
			input.DefaultFrom = "preset"
			input.Default = value
			defaultFromSchema = false
		}
	}
	if defaultFromSchema {
		presetCovered := false
		for _, declared := range declaring {
			if declared.link.presetPointers[input.Bind.Pointer] {
				presetCovered = true
				break
			}
		}
		if presetCovered {
			input.DefaultFrom = ""
			input.Default = nil
			d.warnf("command %q flag %q: schema-default display suppressed because an effective route preset sets %s", cmdKey, input.Name, input.Bind.Pointer)
		} else {
			normalized, err := cliCheckSchemaDefault(declaring[0].facts.defaultVal, input.Type)
			if err != nil {
				return fmt.Errorf("command %q flag %q defaultFrom: schema: the schema default at %s does not match type %s: %w", cmdKey, input.Name, input.Bind.Pointer, input.Type, err)
			}
			input.Default = normalized
		}
	}

	for _, declared := range declaring {
		var required bool
		if input.requiredExplicit {
			required = input.Required
		} else {
			required = declared.link.variant.required[name] && !declared.facts.hasDefault
		}
		if declared.link.presetPointers[input.Bind.Pointer] {
			required = false
		}
		if required {
			input.RequiredRouteIDs = append(input.RequiredRouteIDs, declared.link.route.ID)
		}
	}
	input.Required = len(input.RequiredRouteIDs) == len(links)
	return nil
}

func cliDispatchPropertyShape(facts *cliPropertyFacts) string {
	if facts.unionArms != nil {
		return "union"
	}
	if len(facts.kinds) == 1 {
		return facts.kinds[0]
	}
	return "untyped"
}

// cliDispatchPresetDefault reports the value the request carries at pointer
// when the caller omits the flag: the effective preset, provided every
// declaring route sets the same scalar.
func cliDispatchPresetDefault(declaring []cliDispatchInputFacts, pointer, kind string) (any, bool) {
	var value any
	for i, declared := range declaring {
		candidate, ok := declared.link.presetValues[pointer]
		if !ok || !cliIsPresetDefault(candidate, kind) {
			return nil, false
		}
		if i > 0 && !cliValuesEqual(value, candidate) {
			return nil, false
		}
		value = candidate
	}
	return value, len(declaring) > 0
}

func cliIntersectDispatchEnums(declaring []cliDispatchInputFacts) []any {
	var intersection []any
	started := false
	for _, declared := range declaring {
		enum := declared.normalized.Enum
		if len(enum) == 0 {
			continue
		}
		if !started {
			intersection = append([]any(nil), enum...)
			started = true
			continue
		}
		filtered := intersection[:0]
		for _, candidate := range intersection {
			if cliContainsValue(enum, candidate) {
				filtered = append(filtered, candidate)
			}
		}
		intersection = filtered
	}
	return intersection
}

func (d *cliManifestDecoder) linkDispatchAnchors(cmdKey string, cmd *CLICommand, links []cliDispatchRouteLink) error {
	for i := range links {
		link := &links[i]
		if link.route.Selector != "" {
			flag := cliFindDispatchFlag(cmd.Flags, link.route.Selector)
			if flag == nil || len(flag.RouteIDs) != 1 || flag.RouteIDs[0] != link.route.ID {
				return fmt.Errorf("command %q: route %q selector %q must name a declared flag unique to that route and backed by a required or schema-defaulted distinguishing property", cmdKey, link.route.ID, link.route.Selector)
			}
			name := cliPointerPropertyName(flag.Bind.Pointer)
			facts, _ := d.propertyFacts(link.variant.properties[name], nil)
			if facts == nil || (!link.variant.required[name] && !facts.hasDefault) || link.route.Selectors == nil || !cliContains(link.route.Selectors.Own, name) {
				return fmt.Errorf("command %q: route %q selector %q must name a declared flag unique to that route and backed by a required or schema-defaulted distinguishing property", cmdKey, link.route.ID, link.route.Selector)
			}
			continue
		}

		var candidates []string
		for j := range cmd.Flags {
			flag := &cmd.Flags[j]
			if len(flag.RouteIDs) != 1 || flag.RouteIDs[0] != link.route.ID {
				continue
			}
			name := cliPointerPropertyName(flag.Bind.Pointer)
			facts, err := d.propertyFacts(link.variant.properties[name], nil)
			if err != nil || facts.hasDefault || !link.variant.required[name] || link.presetPointers[flag.Bind.Pointer] {
				continue
			}
			if link.route.Selectors == nil || !cliContains(link.route.Selectors.Own, name) {
				continue
			}
			candidates = append(candidates, flag.Name)
		}
		switch len(candidates) {
		case 0:
			return fmt.Errorf("command %q: route %q has no unique required bound flag; set selector: <flag> to a unique bound flag backed by a required or schema-defaulted distinguishing property", cmdKey, link.route.ID)
		case 1:
			link.route.Selector = candidates[0]
		default:
			choices := make([]string, 0, len(candidates))
			for _, candidate := range candidates {
				choices = append(choices, "selector: "+candidate)
			}
			return fmt.Errorf("command %q: route %q has multiple unique required bound flags (--%s); set %s", cmdKey, link.route.ID, strings.Join(candidates, ", --"), strings.Join(choices, " or "))
		}
	}
	return nil
}

func cliFindDispatchFlag(flags []CLICommandInput, name string) *CLICommandInput {
	for i := range flags {
		if flags[i].Name == name {
			return &flags[i]
		}
	}
	return nil
}

func (d *cliManifestDecoder) checkDispatchDefault(cmdKey string, links []cliDispatchRouteLink, members []cliDispatchMember) error {
	var declaredDefault *cliDispatchRouteLink
	for i := range links {
		if links[i].route.Default {
			declaredDefault = &links[i]
			break
		}
	}
	if declaredDefault == nil {
		return nil
	}

	counts := map[string]int{}
	selecting := make([]map[string]*cliPropertyFacts, len(members))
	for i, member := range members {
		selecting[i] = map[string]*cliPropertyFacts{}
		for _, name := range member.variant.propOrder {
			facts, err := d.propertyFacts(member.variant.properties[name], nil)
			if err != nil || facts.hasConst || len(facts.enum) == 1 || (!member.variant.required[name] && (!facts.hasDefault || facts.defaultVal == nil)) {
				continue
			}
			selecting[i][name] = facts
			counts[name]++
		}
	}
	// Mirror the generated UnionMeta.DefaultJSON derivation (metadata.ts):
	// walk the members in order, consider each distinguishing key once (its
	// first declaring member), and collect a selector default when that
	// first declaration carries a non-null schema default. Exactly one such
	// default makes the union zero-config; the defaulted variant is only
	// certain when the key is exclusive to that member.
	seenKey := map[string]bool{}
	defaultRef := ""
	defaultKey := ""
	defaultCount := 0
	for i, member := range members {
		for _, name := range member.variant.propOrder {
			facts, selects := selecting[i][name]
			if !selects || counts[name] == len(members) || seenKey[name] {
				continue
			}
			seenKey[name] = true
			if facts.hasDefault && facts.defaultVal != nil {
				defaultCount++
				defaultKey = name
				defaultRef = member.ref
			}
		}
	}
	if defaultCount != 1 || counts[defaultKey] != 1 {
		defaultRef = ""
	}
	if defaultRef != "" && declaredDefault.route.RequestVariant == defaultRef {
		return nil
	}
	if declaredDefault.route.Selectors != nil {
		for _, own := range declaredDefault.route.Selectors.Own {
			pointer := "/" + strings.ReplaceAll(strings.ReplaceAll(own, "~", "~0"), "/", "~1")
			if declaredDefault.presetPointers[pointer] {
				return nil
			}
		}
	}

	moveTo := ""
	for _, link := range links {
		if defaultRef != "" && link.route.RequestVariant == defaultRef {
			moveTo = link.route.ID
			break
		}
	}
	if moveTo == "" {
		return fmt.Errorf("command %q: default route %q is not the request union's schema-defaulted variant and its effective preset supplies no distinguishing key; either drop default: or preset a distinguishing key on this route (the union has no unique routed schema-defaulted variant to move default: to)", cmdKey, declaredDefault.route.ID)
	}
	return fmt.Errorf("command %q: default route %q is not the request union's schema-defaulted variant and its effective preset supplies no distinguishing key; either drop default: or move it to route %q, whose selector carries the schema default, or preset a distinguishing key on this route", cmdKey, declaredDefault.route.ID, moveTo)
}

func cliBuildDispatchKeys(routes []CLICommandRoute, members []cliDispatchMember) []CLICommandDispatchKey {
	routeByRef := map[string]string{}
	for _, route := range routes {
		routeByRef[route.RequestVariant] = route.ID
	}
	type membership struct {
		routes   map[string]bool
		unrouted map[string]bool
	}
	byPointer := map[string]*membership{}
	for _, member := range members {
		for _, name := range member.variant.propOrder {
			pointer := "/" + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
			m := byPointer[pointer]
			if m == nil {
				m = &membership{routes: map[string]bool{}, unrouted: map[string]bool{}}
				byPointer[pointer] = m
			}
			if routeID := routeByRef[member.ref]; routeID != "" {
				m.routes[routeID] = true
			} else {
				m.unrouted[member.label] = true
			}
		}
	}
	pointers := make([]string, 0, len(byPointer))
	for pointer := range byPointer {
		pointers = append(pointers, pointer)
	}
	sort.Strings(pointers)
	out := make([]CLICommandDispatchKey, 0, len(pointers))
	for _, pointer := range pointers {
		m := byPointer[pointer]
		key := CLICommandDispatchKey{Pointer: pointer}
		for _, route := range routes {
			if m.routes[route.ID] {
				key.RouteIDs = append(key.RouteIDs, route.ID)
			}
		}
		for variant := range m.unrouted {
			key.UnroutedVariants = append(key.UnroutedVariants, variant)
		}
		sort.Strings(key.UnroutedVariants)
		out = append(out, key)
	}
	return out
}

func (d *cliManifestDecoder) checkDispatchSatisfiability(cmdKey string, cmd *CLICommand, link cliDispatchRouteLink) {
	if len(cmd.Args) == 0 && len(cmd.Flags) == 0 {
		return
	}
	bound := map[string]bool{}
	for _, list := range [][]CLICommandInput{cmd.Args, cmd.Flags} {
		for _, input := range list {
			if input.Bind != nil && input.Bind.In == "body" && cliContains(input.RouteIDs, link.route.ID) {
				bound[input.Bind.Pointer] = true
			}
		}
	}
	names := make([]string, 0, len(link.variant.required))
	for name := range link.variant.required {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		pointer := "/" + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
		if bound[pointer] || link.presetPointers[pointer] {
			continue
		}
		if propSchema, ok := link.variant.properties[name]; ok {
			if facts, err := d.propertyFacts(propSchema, nil); err == nil && facts.hasDefault {
				continue
			}
		}
		d.warnf("command %q route %q: required field %q is not covered by an arg, flag, preset, or schema default; callers must supply it via --body", cmdKey, link.route.ID, name)
	}
}

// cliIsStreamingContentType mirrors the generator's response classification:
// text/event-stream (SSE) and JSONL / JSON-seq media types are iterated as
// streams by the generated CLI (StreamResult), everything else is a single
// result.
func cliIsStreamingContentType(contentType string) bool {
	return contenttypes.IsEventStream(contentType) || contenttypes.IsJsonL(contentType) || contenttypes.IsJsonSeq(contentType)
}

// linkStreamProjection validates output.stream.select against the
// operation's streaming response schemas. The walk is union-aware: every
// oneOf/anyOf arm (and every successful streaming response) is a candidate
// event shape; a segment must resolve in at least one candidate (events
// lacking it are skipped at runtime), and every candidate leaf it does
// resolve to must be a string (or null). Constructs the walk cannot see
// through (open schemas, untyped nodes, cycles, non-component refs) make the
// path unverifiable: a warning, not an error, since streams are polymorphic.
// After the path walk succeeds, the same path is lowered to jq and sent
// through the shared symbolic executor for escalation only: proven-broken
// results error, while executor warnings are suppressed because the singular
// path walker is authoritative for unverifiable arms.
func (d *cliManifestDecoder) linkStreamProjection(owner string, stream *CLICommandStreamProjection, opID string, opInfo *cliOperationInfo) error {
	if !opInfo.streamMediaSeen {
		return fmt.Errorf("%s declares output.stream, but operation %q has no streaming (text/event-stream or JSONL) success response; the CLI only streams events for operations that declare one", owner, opID)
	}
	if err := d.linkStreamSelection(owner, "output.stream.select", stream.Select, opInfo); err != nil {
		return err
	}
	for _, field := range stream.Metadata {
		for _, path := range field.Select {
			if err := d.linkStreamSelection(owner, "output.stream.metadata.select", path, opInfo); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *cliManifestDecoder) linkStreamSelection(owner, field, selectPath string, opInfo *cliOperationInfo) error {
	_, err := d.walkStreamSelection(owner, field, selectPath, opInfo)
	return err
}

// walkStreamSelection is linkStreamSelection returning the walker, so a
// caller can inspect what the resolved leaves declared.
func (d *cliManifestDecoder) walkStreamSelection(owner, field, selectPath string, opInfo *cliOperationInfo) (*cliStreamPathWalker, error) {
	segments, err := cliParseSingularPath(selectPath)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", owner, field, err)
	}

	walker := &cliStreamPathWalker{d: d, owner: owner, selectPath: selectPath, field: field, detail: field == cliStreamDetailField}
	for _, schema := range opInfo.streamSchemas {
		if schema == nil {
			walker.unverifiable("the streaming response declares no schema")
			continue
		}
		walker.walk(schema, segments, nil)
	}
	if walker.err != nil {
		return nil, walker.err
	}
	expected := "a string"
	if walker.detail {
		expected = "an object or array"
	}
	if walker.resolved == 0 {
		if walker.unverified == 0 {
			return nil, fmt.Errorf("%s %s %s does not resolve in any event shape of the streaming response%s", owner, field, selectPath, walker.missSuggestion())
		}
		d.warnf("%s %s %s could not be verified against the streaming response schema (%s); events without %s at that path are skipped at runtime", owner, field, selectPath, walker.firstReason, expected)
		return walker, nil
	}
	if err := d.lintProjection(owner, field, cliSingularPathToJQ(segments), "the streaming response declares no event schema", cliOperationProjectionSchema(opInfo), true); err != nil {
		return nil, err
	}
	if walker.unverified > 0 {
		d.warnf("%s %s %s resolves in %d event shape(s) but could not be verified in every arm (%s); unverified events without %s at that path are skipped at runtime", owner, field, selectPath, walker.resolved, walker.firstReason, expected)
	}
	return walker, nil
}

func cliSingularPathToJQ(segments []cliPathSegment) string {
	var jq strings.Builder
	for _, segment := range segments {
		if segment.IsIndex {
			jq.WriteByte('[')
			jq.WriteString(strconv.Itoa(segment.Index))
			jq.WriteByte(']')
			continue
		}
		if cliIsPathIdentifier(segment.Name) {
			jq.WriteByte('.')
			jq.WriteString(segment.Name)
			continue
		}
		jq.WriteString(".[")
		jq.WriteString(strconv.Quote(segment.Name))
		jq.WriteByte(']')
	}
	return jq.String()
}

// cliStreamPathWalker accumulates the outcome of walking a select path
// through every candidate event shape.
type cliStreamPathWalker struct {
	d          *cliManifestDecoder
	owner      string
	selectPath string
	field      string
	detail     bool // the leaf must be an object or array (failure detail)

	resolved    int // arms where the full path resolved to an accepted leaf
	unverified  int // arms the walk could not see through
	firstReason string
	err         error

	stateEnum   map[string]bool // closed enum members declared at resolved string leaves
	stateNoEnum bool            // a resolved string leaf declared no closed enum
	stateOpen   bool            // a resolved string leaf declared an open enum

	// missCandidates records, for the deepest segment that failed to resolve,
	// the property names that were available (for did-you-mean).
	missDepth      int
	missSegment    string
	missCandidates []string
}

func (w *cliStreamPathWalker) unverifiable(reason string) {
	w.unverified++
	if w.firstReason == "" {
		w.firstReason = reason
	}
}

func (w *cliStreamPathWalker) recordMiss(depth int, segment string, candidates []string) {
	if depth < w.missDepth {
		return
	}
	if depth > w.missDepth {
		w.missCandidates = nil
	}
	w.missDepth = depth
	w.missSegment = segment
	for _, c := range candidates {
		if !cliContains(w.missCandidates, c) {
			w.missCandidates = append(w.missCandidates, c)
		}
	}
}

func (w *cliStreamPathWalker) missSuggestion() string {
	if w.missSegment == "" {
		return ""
	}
	sort.Strings(w.missCandidates)
	return fmt.Sprintf(" (segment %q%s)", w.missSegment, cliDidYouMean(w.missSegment, w.missCandidates))
}

// walk descends schema by segments. A schema's own (composed) properties are
// consulted first; when the segment is not among them, every union arm —
// declared directly (oneOf/anyOf) or lifted through allOf branches — is
// walked independently so polymorphic events count separately.
func (w *cliStreamPathWalker) walk(schema any, segments []cliPathSegment, seen []string) {
	if w.err != nil {
		return
	}
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		if b, isBool := schema.(bool); isBool && b {
			w.unverifiable("the schema accepts any value")
			return
		}
		w.unverifiable("the schema is not an object schema")
		return
	}

	// Follow component references (with cycle protection).
	if ref, ok := schemaMap["$ref"].(string); ok {
		name, isComponent := strings.CutPrefix(ref, cliComponentRefPrefix)
		if !isComponent || name == "" || strings.Contains(name, "/") {
			w.unverifiable(fmt.Sprintf("reference %q is not a top-level component schema reference", ref))
			return
		}
		for _, ancestor := range seen {
			if ancestor == name {
				w.unverifiable("schema reference cycle at " + name)
				return
			}
		}
		component, ok := w.d.schemaIndex.components[name]
		if !ok {
			w.err = fmt.Errorf("%s %s %s: referenced schema %q does not exist in components.schemas%s", w.owner, w.field, w.selectPath, name, cliDidYouMean(name, cliComponentNames(w.d.schemaIndex.components)))
			return
		}
		if len(segments) == 0 && cliRefHasSiblingConstraints(schemaMap) {
			w.checkLeaf(schemaMap, seen)
			return
		}
		w.walk(component, segments, append(seen, name))
		return
	}

	if len(segments) == 0 {
		w.checkLeaf(schemaMap, seen)
		return
	}

	seg := segments[0]
	types := cliSchemaTypes(schemaMap)
	arms := w.unionArms(schemaMap, seen)

	if seg.IsIndex {
		items, hasItems := schemaMap["items"]
		switch {
		case hasItems:
			w.walk(items, segments[1:], seen)
		case cliContains(types, "array"):
			w.unverifiable("the array declares no items schema")
		case len(arms) > 0:
			for _, arm := range arms {
				w.walk(arm, segments, seen)
			}
		case len(types) > 0:
			// A definite non-array (object or scalar) cannot be indexed.
			w.recordMiss(w.depthOf(segments), strconv.Itoa(seg.Index), nil)
		default:
			w.unverifiable("an index segment was applied to an untyped schema")
		}
		return
	}

	// Scalars and arrays have no named properties: a definite miss.
	if len(types) > 0 && !cliContains(types, "object") {
		w.recordMiss(w.depthOf(segments), seg.Name, nil)
		return
	}

	// Composed own properties (direct + allOf) win over union fan-out: a
	// property declared beside a oneOf applies to every arm.
	resolved, err := w.d.resolveObjectSchema(schemaMap, seen)
	if err != nil {
		w.unverifiable(err.Error())
		return
	}
	if prop, ok := resolved.properties[seg.Name]; ok {
		w.walk(prop, segments[1:], seen)
		return
	}
	if len(arms) > 0 {
		for _, arm := range arms {
			w.walk(arm, segments, seen)
		}
		return
	}

	additional, hasAdditional := schemaMap["additionalProperties"]
	switch {
	case hasAdditional && additional == false:
		// Explicitly closed: the property provably does not exist.
		w.recordMiss(w.depthOf(segments), seg.Name, resolved.propOrder)
	case len(resolved.properties) > 0 && !resolved.explicitOpen:
		// Declared properties without an explicit opening: treat a miss as a
		// provable typo (the same heuristic the bind linker applies).
		w.recordMiss(w.depthOf(segments), seg.Name, resolved.propOrder)
	default:
		w.unverifiable(fmt.Sprintf("%q is not a declared property and the schema accepts unknown keys", seg.Name))
	}
}

// unionArms collects the candidate arm schemas of a schema: its own
// oneOf/anyOf members plus those lifted through allOf branches (following
// component references).
func (w *cliStreamPathWalker) unionArms(schemaMap map[string]any, seen []string) []any {
	arms := append([]any(nil), cliUnionMembers(schemaMap)...)
	allOf, _ := schemaMap["allOf"].([]any)
	for _, branch := range allOf {
		branchMap, ok := branch.(map[string]any)
		if !ok {
			continue
		}
		branchSeen := seen
		for {
			ref, isRef := branchMap["$ref"].(string)
			if !isRef {
				break
			}
			name := strings.TrimPrefix(ref, cliComponentRefPrefix)
			if !strings.HasPrefix(ref, cliComponentRefPrefix) || cliContains(branchSeen, name) {
				branchMap = nil
				break
			}
			component, ok := w.d.schemaIndex.components[name].(map[string]any)
			if !ok {
				branchMap = nil
				break
			}
			branchSeen = append(branchSeen, name)
			branchMap = component
		}
		if branchMap != nil {
			arms = append(arms, w.unionArms(branchMap, branchSeen)...)
		}
	}
	return arms
}

// depthOf converts remaining segments into a depth (deeper misses win the
// did-you-mean so the message names the segment nearest the leaf).
func (w *cliStreamPathWalker) depthOf(remaining []cliPathSegment) int {
	return 1000 - len(remaining)
}

// leafClass is the classification of one candidate leaf schema.
type leafClass int

const (
	leafUnknown leafClass = iota // untyped: nothing provable
	leafString                   // string or null
	leafOther                    // provably not the expected kind
	leafDetail                   // object or array (failure detail)
)

// checkLeaf classifies the schema the full path resolved to.
func (w *cliStreamPathWalker) checkLeaf(schemaMap map[string]any, seen []string) {
	var (
		class leafClass
		kind  string
	)
	if w.detail {
		class, kind = w.classifyDetailLeaf(schemaMap, seen)
	} else {
		class, kind = w.classifyLeaf(schemaMap, seen)
	}
	switch class {
	case leafString, leafDetail:
		w.resolved++
		if !w.detail {
			w.recordLeafEnum(schemaMap, seen)
		}
	case leafOther:
		switch {
		case w.detail:
			w.err = fmt.Errorf("%s %s %s resolves to %s in one event shape; the failure detail must be an object or an array", w.owner, w.field, w.selectPath, kind)
		case w.field == "output.stream.metadata.select":
			w.err = fmt.Errorf("%s %s %s resolves to %s in one event shape; metadata must address a string field", w.owner, w.field, w.selectPath, kind)
		default:
			w.err = fmt.Errorf("%s %s %s resolves to %s in one event shape; the streamed projection writes string values raw, so select must address a string field", w.owner, w.field, w.selectPath, kind)
		}
	default:
		w.unverifiable("the selected field is untyped")
	}
}

func (w *cliStreamPathWalker) recordLeafEnum(schemaMap map[string]any, seen []string) {
	values, closed, open := w.leafEnum(schemaMap, seen)
	switch {
	case open:
		w.stateOpen = true
	case !closed:
		w.stateNoEnum = true
	default:
		if w.stateEnum == nil {
			w.stateEnum = map[string]bool{}
		}
		for _, value := range values {
			w.stateEnum[value] = true
		}
	}
}

// leafEnum reports the string members a string leaf is closed over. Every
// keyword on the schema constrains conjunctively: a const, a closed enum, the
// referenced component, each allOf arm, and a oneOf/anyOf (whose non-null arms
// must all be closed) each contribute a member set, and the leaf is closed
// over their intersection. Arms that only set a type constrain nothing.
func (w *cliStreamPathWalker) leafEnum(schemaMap map[string]any, seen []string) (values []string, closed, open bool) {
	if unknownValues, ok := schemaMap["x-speakeasy-unknown-values"].(string); ok && unknownValues == "allow" {
		return nil, false, true
	}
	var sources [][]string
	if ref, ok := schemaMap["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, cliComponentRefPrefix)
		if strings.HasPrefix(ref, cliComponentRefPrefix) && !cliContains(seen, name) {
			if component, ok := w.d.schemaIndex.components[name].(map[string]any); ok {
				refValues, refClosed, refOpen := w.leafEnum(component, append(seen, name))
				if refOpen {
					return nil, false, true
				}
				if refClosed {
					sources = append(sources, refValues)
				}
			}
		}
	}
	if constVal, ok := schemaMap["const"]; ok {
		switch s := constVal.(type) {
		case string:
			sources = append(sources, []string{s})
		case nil:
			sources = append(sources, nil)
		default:
			return nil, false, false
		}
	}
	if enum, ok := schemaMap["enum"].([]any); ok && len(enum) > 0 {
		var members []string
		for _, value := range enum {
			if s, isString := value.(string); isString {
				members = append(members, s)
			}
		}
		sources = append(sources, members)
	}
	if allOf, ok := schemaMap["allOf"].([]any); ok {
		for _, arm := range allOf {
			armMap, ok := arm.(map[string]any)
			if !ok {
				continue
			}
			armValues, armClosed, armOpen := w.leafEnum(armMap, seen)
			if armOpen {
				return nil, false, true
			}
			if armClosed {
				sources = append(sources, armValues)
			}
		}
	}
	if members := cliUnionMembers(schemaMap); members != nil {
		unionClosed := true
		var unionValues []string
		for _, member := range members {
			memberMap, ok := member.(map[string]any)
			if !ok {
				continue
			}
			if types := cliSchemaTypes(memberMap); len(types) == 1 && types[0] == "null" {
				continue
			}
			if cliSchemaIsAnnotationOnly(memberMap) {
				continue
			}
			memberValues, memberClosed, memberOpen := w.leafEnum(memberMap, seen)
			if memberOpen {
				return nil, false, true
			}
			unionClosed = unionClosed && memberClosed
			unionValues = append(unionValues, memberValues...)
		}
		if !unionClosed {
			return nil, false, false
		}
		sources = append(sources, unionValues)
	}
	if len(sources) == 0 {
		return nil, false, false
	}
	return cliIntersectStrings(sources), true, false
}

// cliIntersectStrings keeps the members of the first set present in every
// other set, in first-set order.
func cliIntersectStrings(sets [][]string) []string {
	var out []string
	for _, candidate := range sets[0] {
		if cliContains(out, candidate) {
			continue
		}
		keep := true
		for _, other := range sets[1:] {
			if !cliContains(other, candidate) {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, candidate)
		}
	}
	return out
}

// cliRefHasSiblingConstraints reports a reference whose siblings narrow the
// referenced schema, so the leaf is checked here rather than at the target.
func cliRefHasSiblingConstraints(schemaMap map[string]any) bool {
	siblings := make(map[string]any, len(schemaMap))
	for key, value := range schemaMap {
		if key != "$ref" {
			siblings[key] = value
		}
	}
	return !cliSchemaIsAnnotationOnly(siblings)
}

// cliSchemaIsAnnotationOnly reports a schema that constrains nothing (an
// allOf branch carrying only a description, say).
func cliSchemaIsAnnotationOnly(schemaMap map[string]any) bool {
	for _, key := range []string{"$ref", "type", "enum", "const", "oneOf", "anyOf", "allOf", "properties", "items", "x-speakeasy-unknown-values"} {
		if _, present := schemaMap[key]; present {
			return false
		}
	}
	return true
}

// classifyDetailLeaf accepts object and array leaves; a null arm is neutral.
func (w *cliStreamPathWalker) classifyDetailLeaf(schemaMap map[string]any, seen []string) (leafClass, string) {
	if ref, ok := schemaMap["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, cliComponentRefPrefix)
		if !strings.HasPrefix(ref, cliComponentRefPrefix) || cliContains(seen, name) {
			return leafUnknown, ""
		}
		component, ok := w.d.schemaIndex.components[name].(map[string]any)
		if !ok {
			return leafUnknown, ""
		}
		return w.classifyDetailLeaf(component, append(seen, name))
	}

	var branches []any
	if members := cliUnionMembers(schemaMap); members != nil {
		branches = append(branches, members...)
	}
	if allOf, ok := schemaMap["allOf"].([]any); ok {
		branches = append(branches, allOf...)
	}
	if len(branches) > 0 {
		result := leafUnknown
		for _, branch := range branches {
			branchMap, ok := branch.(map[string]any)
			if !ok {
				continue
			}
			class, kind := w.classifyDetailLeaf(branchMap, seen)
			switch class {
			case leafOther:
				return leafOther, kind
			case leafDetail:
				result = leafDetail
			}
		}
		return result, ""
	}

	types := cliSchemaTypes(schemaMap)
	_, hasProperties := schemaMap["properties"]
	_, hasItems := schemaMap["items"]
	if cliContains(types, "object") || cliContains(types, "array") || hasProperties || hasItems {
		return leafDetail, ""
	}
	if constVal, ok := schemaMap["const"]; ok {
		return leafOther, "a constant " + jsonValueKind(constVal)
	}
	if enum, ok := schemaMap["enum"].([]any); ok && len(enum) > 0 {
		return leafOther, "an enum"
	}
	for _, typ := range types {
		if typ != "null" {
			return leafOther, cliKindArticle(cliSchemaTypeKind(typ))
		}
	}
	return leafUnknown, ""
}

// classifyLeaf inspects a leaf schema: const/enum values, type (scalar or
// list), union arms (every arm must agree), and allOf branches (any typed
// branch decides; conflicting branches are an error).
func (w *cliStreamPathWalker) classifyLeaf(schemaMap map[string]any, seen []string) (leafClass, string) {
	if ref, ok := schemaMap["$ref"].(string); ok {
		name := strings.TrimPrefix(ref, cliComponentRefPrefix)
		if !strings.HasPrefix(ref, cliComponentRefPrefix) || cliContains(seen, name) {
			return leafUnknown, ""
		}
		component, ok := w.d.schemaIndex.components[name].(map[string]any)
		if !ok {
			return leafUnknown, ""
		}
		return w.classifyLeaf(component, append(seen, name))
	}

	if constVal, ok := schemaMap["const"]; ok {
		if _, isString := constVal.(string); isString || constVal == nil {
			return leafString, ""
		}
		return leafOther, "a constant " + jsonValueKind(constVal)
	}

	if members := cliUnionMembers(schemaMap); members != nil {
		result := leafUnknown
		for _, member := range members {
			memberMap, ok := member.(map[string]any)
			if !ok {
				continue
			}
			class, kind := w.classifyLeaf(memberMap, seen)
			switch class {
			case leafOther:
				return leafOther, kind
			case leafString:
				result = leafString
			}
		}
		return result, ""
	}

	if allOf, ok := schemaMap["allOf"].([]any); ok && len(allOf) > 0 {
		result := leafUnknown
		for _, branch := range allOf {
			branchMap, ok := branch.(map[string]any)
			if !ok {
				continue
			}
			class, kind := w.classifyLeaf(branchMap, seen)
			switch class {
			case leafOther:
				return leafOther, kind
			case leafString:
				result = leafString
			}
		}
		if result == leafUnknown {
			if _, hasProps := schemaMap["properties"]; hasProps {
				return leafOther, "an object"
			}
		}
		return result, ""
	}

	switch typ := schemaMap["type"].(type) {
	case string:
		switch typ {
		case "string", "null":
			return leafString, ""
		case "":
		default:
			return leafOther, cliKindArticle(cliSchemaTypeKind(typ))
		}
	case []any:
		for _, t := range typ {
			if s, _ := t.(string); s != "string" && s != "null" {
				return leafOther, cliKindArticle(cliSchemaTypeKind(s))
			}
		}
		if len(typ) > 0 {
			return leafString, ""
		}
	}

	if enum, ok := schemaMap["enum"].([]any); ok && len(enum) > 0 {
		for _, value := range enum {
			if _, isString := value.(string); !isString && value != nil {
				return leafOther, "an enum of " + jsonValueKind(value) + " values"
			}
		}
		return leafString, ""
	}
	if _, hasProps := schemaMap["properties"]; hasProps {
		return leafOther, "an object"
	}
	if _, hasItems := schemaMap["items"]; hasItems {
		return leafOther, "an array"
	}
	return leafUnknown, ""
}

// cliSchemaTypes returns the schema's declared type name(s): a single string
// or a type list (nil when untyped).
func cliSchemaTypes(schemaMap map[string]any) []string {
	switch typ := schemaMap["type"].(type) {
	case string:
		if typ != "" {
			return []string{typ}
		}
	case []any:
		var out []string
		for _, t := range typ {
			if s, ok := t.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// cliSchemaTypeKind maps a JSON Schema type name onto the linker's kind
// vocabulary (shared with cliKindArticle).
func cliSchemaTypeKind(typ string) string {
	switch typ {
	case "integer":
		return "int"
	case "number":
		return "float"
	case "boolean":
		return "bool"
	}
	return typ
}

// jsonValueKind names the JSON kind of a decoded YAML/JSON value.
func jsonValueKind(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case int, int64, float64:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return fmt.Sprintf("%T", value)
}

func cliKindArticle(kind string) string {
	switch kind {
	case "int":
		return "an integer"
	case "float":
		return "a number"
	case "bool":
		return "a boolean"
	case "array":
		return "an array"
	case "object":
		return "an object"
	}
	return "a " + kind
}

func (d *cliManifestDecoder) checkParameterExists(cmdKey, inputName, in, paramName string, opInfo *cliOperationInfo) error {
	names := opInfo.params[in]
	for _, name := range names {
		if name == paramName || (in == "header" && strings.EqualFold(name, paramName)) {
			return nil
		}
	}
	return fmt.Errorf("command %q input %q: operation has no %s parameter %q%s", cmdKey, inputName, in, paramName, cliDidYouMean(paramName, names))
}

// resolveVariant selects and composes the request-body schema the command
// binds against. A pinned variant must be an actual member of the body (its
// direct reference or a union member); an unpinned union must be pinned.
func (d *cliManifestDecoder) resolveVariant(cmdKey string, route CLICommandRoute, bodySchema any) (*cliResolvedSchema, error) {
	bodyMap, _ := bodySchema.(map[string]any)
	unionMap, directRef := d.unionBodySchema(bodyMap)
	members := cliUnionMemberRefs(unionMap)

	if route.RequestVariant != "" {
		if !strings.HasPrefix(route.RequestVariant, cliComponentRefPrefix) {
			return nil, fmt.Errorf("command %q: request variant %q must reference a component schema (%s<Name>)", cmdKey, route.RequestVariant, cliComponentRefPrefix)
		}
		name := strings.TrimPrefix(route.RequestVariant, cliComponentRefPrefix)
		if _, ok := d.schemaIndex.components[name]; !ok {
			return nil, fmt.Errorf("command %q: request variant %q does not exist in components.schemas%s", cmdKey, name, cliDidYouMean(name, cliComponentNames(d.schemaIndex.components)))
		}
		switch {
		case len(members) > 0:
			if !cliContains(members, route.RequestVariant) && directRef != route.RequestVariant {
				return nil, fmt.Errorf("command %q: request variant %q is not a member of the operation's request body union (members: %s)", cmdKey, route.RequestVariant, cliCandidateList(cliTrimRefPrefixes(members)))
			}
		case directRef != "":
			if directRef != route.RequestVariant {
				return nil, fmt.Errorf("command %q: operation's request body is %s, not a union; the variant must match it or be omitted", cmdKey, strings.TrimPrefix(directRef, cliComponentRefPrefix))
			}
		default:
			return nil, fmt.Errorf("command %q: operation's request body is an inline schema, not a union of component references; drop the #%s variant", cmdKey, name)
		}
		// Resolve the actual schema node carrying the reference: the union
		// member (or the body itself for a direct reference) may declare
		// keywords beside $ref, which apply conjunctively and must not be
		// discarded by reconstructing a bare reference.
		var raw map[string]any
		if unionMap != nil {
			for _, member := range cliUnionMembers(unionMap) {
				memberMap, ok := member.(map[string]any)
				if !ok {
					continue
				}
				if ref, _ := memberMap["$ref"].(string); ref == route.RequestVariant {
					raw = memberMap
					break
				}
			}
		}
		if raw == nil && directRef == route.RequestVariant && bodyMap != nil {
			raw = bodyMap
		}
		if raw == nil {
			raw = map[string]any{"$ref": route.RequestVariant}
		}
		return d.resolveObjectSchema(raw, nil)
	}

	if len(members) > 0 {
		return nil, fmt.Errorf("command %q: operation's request body is a union; pin a variant with op: OperationID#Variant (members: %s)", cmdKey, cliCandidateList(cliTrimRefPrefixes(members)))
	}
	return d.resolveObjectSchema(bodySchema, nil)
}

// variantSelectors derives, for a command pinned to one member of a union
// request body, the keys that tell the members apart: distinguishing keys are
// those required (or schema-defaulted) in some member but not present in
// every member — the same rule the generated runtime uses to pick a variant
// for a bare JSON body — split into keys the pinned variant declares (own)
// and keys only other members declare (foreign). Const/single-enum
// properties (discriminators) are excluded from that set and reported via
// DiscriminatorKey/DiscriminatorValue instead, since there the conflict is a
// different value under the same key. Inline members take part like
// referenced ones. Returns nil for non-union bodies.
func (d *cliManifestDecoder) variantSelectors(cmdKey string, route CLICommandRoute, bodySchema any, variant *cliResolvedSchema, presets []CLICommandPreset) (*CLIVariantSelectors, error) {
	rawBody, _ := bodySchema.(map[string]any)
	bodyMap, _ := d.unionBodySchema(rawBody)
	members := cliUnionMembers(bodyMap)
	if len(members) < 2 || route.RequestVariant == "" {
		return nil, nil
	}

	counts := map[string]int{}
	var order []string
	pinnedIndex := -1
	memberSelecting := make([]map[string]bool, len(members))
	for i, raw := range members {
		if m, ok := raw.(map[string]any); ok {
			if ref, ok := m["$ref"].(string); ok && ref == route.RequestVariant {
				pinnedIndex = i
			}
		}
		member, err := d.resolveObjectSchema(raw, nil)
		if err != nil {
			return nil, fmt.Errorf("command %q: request body union member %d: %w", cmdKey, i+1, err)
		}
		memberSelecting[i] = map[string]bool{}
		for _, name := range member.propOrder {
			facts, err := d.propertyFacts(member.properties[name], nil)
			if err != nil {
				facts = &cliPropertyFacts{}
			}
			prop, _ := member.properties[name].(map[string]any)
			if cliIsConstProperty(prop) || len(facts.enum) == 1 {
				continue
			}
			if !member.required[name] && !facts.hasDefault {
				continue
			}
			memberSelecting[i][name] = true
			if _, seen := counts[name]; !seen {
				order = append(order, name)
			}
			counts[name]++
		}
	}

	out := &CLIVariantSelectors{}
	for _, name := range order {
		if counts[name] == len(members) {
			continue // shared by every member: selects nothing
		}
		if _, declared := variant.properties[name]; declared {
			out.Own = append(out.Own, name)
		} else {
			out.Foreign = append(out.Foreign, name)
		}
	}

	if disc, ok := bodyMap["discriminator"].(map[string]any); ok {
		if key, ok := disc["propertyName"].(string); ok && key != "" {
			out.DiscriminatorKey = key
			accepted := d.pinnedDiscriminatorValues(disc, key, route.RequestVariant, variant)
			out.DiscriminatorValue = accepted[0]
			out.DiscriminatorAliases = accepted
			// A preset on the discriminator must name the pinned variant:
			// otherwise even the argument path (no --body) would send the
			// other variant under this command's name.
			pointer := "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
			for _, preset := range presets {
				if preset.Bind.Pointer != pointer {
					continue
				}
				if !cliContainsValue(accepted, preset.Value) {
					return nil, fmt.Errorf("command %q preset %s: value %v does not select the pinned variant %s (accepted: %s)", cmdKey, pointer, preset.Value, strings.TrimPrefix(route.RequestVariant, cliComponentRefPrefix), cliCandidateList(cliEnumStrings(accepted)))
				}
			}
		}
	}

	// A preset must not be able to select a sibling: a preset key that a
	// sibling requires (or defaults) while the pinned variant merely permits
	// it would, merged into a body that names no variant, be read as that
	// sibling. Only a discriminator pins the variant for certain (the
	// generated union unmarshal ignores unknown keys and scores candidates,
	// so no key the pinned variant alone declares can exclude a sibling), so
	// without one such a preset is a generation error.
	if pinnedIndex >= 0 && out.DiscriminatorValue == nil {
		var presetSibling []string
		for _, preset := range presets {
			name := cliPointerPropertyName(preset.Bind.Pointer)
			if memberSelecting[pinnedIndex][name] {
				continue
			}
			for j := range members {
				if j != pinnedIndex && memberSelecting[j][name] {
					presetSibling = append(presetSibling, name)
					break
				}
			}
		}
		if len(presetSibling) > 0 {
			context := fmt.Sprintf("command %q", cmdKey)
			if route.ID != "" {
				context += fmt.Sprintf(" route %q", route.ID)
			}
			return nil, fmt.Errorf("%s: preset %s is required by another member of the request body union but only optional on the pinned variant %s, so a body built from the presets could be read as that other variant; drop that preset (bind an arg or flag instead) or add a discriminator to the union", context, cliCandidateList(presetSibling), strings.TrimPrefix(route.RequestVariant, cliComponentRefPrefix))
		}
	}

	if len(out.Own) == 0 && len(out.Foreign) == 0 && out.DiscriminatorKey == "" {
		d.warnf("command %q: the request body union members share every required key; a partial --body cannot be checked against the pinned variant and presets fill in unconditionally", cmdKey)
	}
	return out, nil
}

// pinnedDiscriminatorValues lists every discriminator value that selects the
// pinned variant, most authoritative first: the pinned schema's own
// const/single-enum discriminator property (what the server accepts on the
// wire), explicit mapping entries that point at the variant, and — absent any
// mapping entry — the implicit mapping the OpenAPI discriminator object
// defines (the value is the component name). The first entry is the value
// filled into a body that omits the discriminator.
func (d *cliManifestDecoder) pinnedDiscriminatorValues(disc map[string]any, key, pinnedRef string, variant *cliResolvedSchema) []any {
	var accepted []any
	add := func(value any) {
		if value == nil || cliContainsValue(accepted, value) {
			return
		}
		accepted = append(accepted, value)
	}
	if prop, ok := variant.properties[key].(map[string]any); ok {
		if value, ok := prop["const"]; ok {
			add(value)
		}
		if enum, ok := prop["enum"].([]any); ok && len(enum) == 1 {
			add(enum[0])
		}
	}
	mapped := false
	if mapping, ok := disc["mapping"].(map[string]any); ok {
		names := make([]string, 0, len(mapping))
		for name := range mapping {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if ref, ok := mapping[name].(string); ok && ref == pinnedRef {
				add(name)
				mapped = true
			}
		}
	}
	if !mapped {
		add(strings.TrimPrefix(pinnedRef, cliComponentRefPrefix))
	}
	return accepted
}

// cliFlatUnion is a union property flattened through nested oneOf/anyOf
// layers: its object members (each carrying the properties declared beside
// the unions above it), the discriminator keys the layers declare, and their
// explicit mappings (key -> value -> member reference; "" when layers map the
// same value to different members). fallbacks marks members that accept every
// object on their own, so filling keys into an object can never invalidate it.
type cliFlatUnion struct {
	members   []*cliResolvedSchema
	keys      []string
	mappings  map[string]map[string]string
	fallbacks map[*cliResolvedSchema]bool
}

func newCLIFlatUnion() *cliFlatUnion {
	return &cliFlatUnion{mappings: map[string]map[string]string{}, fallbacks: map[*cliResolvedSchema]bool{}}
}

func (d *cliManifestDecoder) flattenUnion(schema any, depth int, inherited *cliResolvedSchema, out *cliFlatUnion) {
	if depth > 8 {
		return
	}
	resolved, err := d.resolveObjectSchema(schema, nil)
	if err != nil {
		return
	}
	if resolved.unionMembers == nil {
		if cliMayAcceptObject(resolved) {
			if cliAcceptsEveryObject(resolved) {
				out.fallbacks[resolved] = true
			}
			cliInheritProperties(resolved, inherited)
			out.members = append(out.members, resolved)
		}
		return
	}
	d.collectDiscriminators(schema, depth, out)
	cliInheritProperties(resolved, inherited)
	for _, member := range resolved.unionMembers {
		d.flattenUnion(member, depth+1, resolved, out)
	}
}

// cliMayAcceptObject reports whether a union member can validate a JSON
// object: it declares properties, is object-typed, leaves its type open, or
// lists an object among its const/enum values.
func cliMayAcceptObject(member *cliResolvedSchema) bool {
	if len(member.propOrder) > 0 || member.explicitOpen {
		return true
	}
	switch kind := member.raw["type"].(type) {
	case string:
		return kind == "object"
	case []any:
		return slices.Contains(kind, any("object"))
	case nil:
		if value, ok := member.raw["const"]; ok {
			_, isObject := value.(map[string]any)
			return isObject
		}
		if values, ok := member.raw["enum"].([]any); ok {
			return slices.ContainsFunc(values, func(value any) bool {
				_, isObject := value.(map[string]any)
				return isObject
			})
		}
		return true
	}
	return false
}

// cliAcceptsEveryObject reports whether a member, on its own, places no
// constraint on an object: no properties, required keys, value schema or
// other validation keyword.
func cliAcceptsEveryObject(member *cliResolvedSchema) bool {
	if len(member.propOrder) > 0 || len(member.required) > 0 || member.additional != nil || member.explicitClosed || member.unionMembers != nil {
		return false
	}
	for keyword, value := range member.raw {
		switch keyword {
		case "type":
			if types, ok := value.([]any); ok {
				if !slices.Contains(types, any("object")) {
					return false
				}
			} else if value != "object" {
				return false
			}
		case "additionalProperties":
			if value != true {
				if schema, ok := value.(map[string]any); !ok || len(schema) > 0 {
					return false
				}
			}
		case "title", "description", "nullable", "example", "examples", "deprecated":
		default:
			if !strings.HasPrefix(keyword, "x-") {
				return false
			}
		}
	}
	return true
}

// cliInheritProperties adds the properties and any const/enum restriction
// declared beside a union to one of its members; a property both declare is
// the conjunction of the two.
func cliInheritProperties(target, parent *cliResolvedSchema) {
	if parent == nil {
		return
	}
	if parent.enumerated {
		target.enumerated = true
	}
	for _, name := range parent.propOrder {
		existing, ok := target.properties[name]
		switch {
		case !ok:
			target.properties[name] = parent.properties[name]
			target.propOrder = append(target.propOrder, name)
		case !reflect.DeepEqual(existing, parent.properties[name]):
			target.properties[name] = map[string]any{"allOf": []any{parent.properties[name], existing}}
		}
	}
}

func cliAdditionalPropertiesSchema(object *cliResolvedSchema) (map[string]any, bool) {
	return object.additional, object.additional != nil
}

// excludesValue reports whether a property schema's enum rules out value.
func (d *cliManifestDecoder) excludesValue(schema, value any) bool {
	facts, err := d.propertyFacts(schema, nil)
	if err != nil {
		return false
	}
	return len(facts.enum) > 0 && !cliContainsValue(facts.enum, value)
}

// collectDiscriminators records the discriminator objects declared on a union
// node, including those reached through its $ref and allOf layers.
func (d *cliManifestDecoder) collectDiscriminators(schema any, depth int, out *cliFlatUnion) {
	schemaMap, ok := schema.(map[string]any)
	if !ok || depth > 8 {
		return
	}
	if disc, ok := schemaMap["discriminator"].(map[string]any); ok {
		key, _ := disc["propertyName"].(string)
		if key != "" && !cliContains(out.keys, key) {
			out.keys = append(out.keys, key)
		}
		if mapping, ok := disc["mapping"].(map[string]any); ok && key != "" {
			if out.mappings[key] == nil {
				out.mappings[key] = map[string]string{}
			}
			for value, ref := range mapping {
				ref, ok := ref.(string)
				if !ok {
					continue
				}
				if existing, seen := out.mappings[key][value]; seen && existing != ref {
					ref = ""
				}
				out.mappings[key][value] = ref
			}
		}
	}
	if ref, ok := schemaMap["$ref"].(string); ok {
		d.collectDiscriminators(d.schemaIndex.components[strings.TrimPrefix(ref, cliComponentRefPrefix)], depth+1, out)
	}
	if allOf, ok := schemaMap["allOf"].([]any); ok {
		for _, branch := range allOf {
			d.collectDiscriminators(branch, depth+1, out)
		}
	}
}

func (d *cliManifestDecoder) constPropertyValue(schema any) (any, bool) {
	facts, err := d.propertyFacts(schema, nil)
	if err != nil {
		return nil, false
	}
	if facts.hasConst {
		return facts.constVal, true
	}
	if len(facts.enum) == 1 {
		return facts.enum[0], true
	}
	return nil, false
}

// presetMergePoints walks each object-valued preset alongside its schema
// and records, keyed by JSON pointer, every object of the preset value that
// may fill a caller's object at the same pointer: plain objects and map
// entries unconditionally, union members only when the preset identifies one
// member, guarded by the discriminator it sets. Elsewhere, including objects
// restricted to const/enum values, the caller's object is kept whole.
func (d *cliManifestDecoder) presetMergePoints(variant *cliResolvedSchema, presets []CLICommandPreset) []CLIPresetMergePoint {
	var out []CLIPresetMergePoint
	for _, preset := range presets {
		value, ok := preset.Value.(map[string]any)
		if !ok {
			continue
		}
		propSchema, ok := variant.properties[cliPointerPropertyName(preset.Bind.Pointer)]
		if !ok {
			continue
		}
		d.collectPresetMergePoints(preset.Bind.Pointer, propSchema, value, 0, &out)
	}
	return out
}

func (d *cliManifestDecoder) collectPresetMergePoints(pointer string, schema any, value map[string]any, depth int, out *[]CLIPresetMergePoint) {
	if depth > 8 {
		return
	}
	flat := newCLIFlatUnion()
	d.flattenUnion(schema, 0, nil, flat)
	point := CLIPresetMergePoint{Pointer: pointer}
	var object *cliResolvedSchema
	switch {
	case len(flat.members) == 1:
		object = flat.members[0]
	case len(flat.members) > 1:
		point, object = d.presetUnionMergePoint(pointer, flat, value)
	}
	if object == nil || object.enumerated {
		return
	}
	*out = append(*out, point)
	names := make([]string, 0, len(value))
	for name := range value {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		child, ok := value[name].(map[string]any)
		if !ok {
			continue
		}
		childSchema, ok := object.properties[name]
		if !ok {
			childSchema, ok = cliAdditionalPropertiesSchema(object)
		}
		if !ok {
			continue
		}
		token := strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
		d.collectPresetMergePoints(pointer+"/"+token, childSchema, child, depth+1, out)
	}
}

// presetUnionMergePoint finds the discriminator a preset object sets on a
// union: a key the union declares as its discriminator, else a key whose
// const/single-enum values differ across members, whose preset value is the
// const of exactly one member, and which no other member can also accept: a
// member declaring it must rule the value out by enum, a member not declaring
// it must be closed or accept every object. The accepted values are the preset's value plus every value
// mapped (explicitly, or implicitly by component name) to the member it
// selects. member is nil when the preset does not select exactly one member.
func (d *cliManifestDecoder) presetUnionMergePoint(pointer string, flat *cliFlatUnion, value map[string]any) (guard CLIPresetMergePoint, member *cliResolvedSchema) {
	guard.Pointer = pointer
	key := ""
	for _, candidate := range flat.keys {
		if _, ok := value[candidate]; ok {
			key = candidate
			break
		}
	}
	declared := key != ""
	if !declared {
		names := make([]string, 0, len(value))
		for name := range value {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			var distinct []any
			matching := 0
			overlapping := false
			for _, m := range flat.members {
				prop, declares := m.properties[name]
				if !declares {
					overlapping = overlapping || (!m.explicitClosed && !flat.fallbacks[m])
					continue
				}
				v, ok := d.constPropertyValue(prop)
				if !ok {
					overlapping = overlapping || !d.excludesValue(prop, value[name])
					continue
				}
				if !cliContainsValue(distinct, v) {
					distinct = append(distinct, v)
				}
				if cliValuesEqual(v, value[name]) {
					matching++
				}
			}
			if len(distinct) >= 2 && matching == 1 && !overlapping {
				key = name
				break
			}
		}
	}
	if key == "" {
		return guard, nil
	}

	pinned := value[key]
	mappedRef, hasMapped := flat.mappings[key][fmt.Sprint(pinned)]
	if hasMapped && mappedRef == "" {
		return guard, nil
	}
	selectedRefs := map[string]bool{}
	if hasMapped {
		selectedRefs[mappedRef] = true
	}
	var selected []*cliResolvedSchema
	for _, m := range flat.members {
		ref := ""
		if m.refName != "" {
			ref = cliComponentRefPrefix + m.refName
		}
		v, ok := d.constPropertyValue(m.properties[key])
		if (ok && cliValuesEqual(v, pinned)) || (declared && ref != "" && (ref == mappedRef || m.refName == pinned)) {
			selected = append(selected, m)
			if ref != "" {
				selectedRefs[ref] = true
			}
		}
	}

	values := []any{pinned}
	aliases := make([]string, 0, len(flat.mappings[key]))
	mapped := map[string]bool{}
	for alias, ref := range flat.mappings[key] {
		if ref == "" {
			continue
		}
		mapped[ref] = true
		if selectedRefs[ref] {
			aliases = append(aliases, alias)
		}
	}
	if declared {
		for ref := range selectedRefs {
			if !mapped[ref] {
				aliases = append(aliases, strings.TrimPrefix(ref, cliComponentRefPrefix))
			}
		}
	}
	sort.Strings(aliases)
	for _, alias := range aliases {
		if !cliContainsValue(values, alias) {
			values = append(values, alias)
		}
	}
	guard.Key = key
	guard.Values = values
	if len(selected) == 1 {
		member = selected[0]
	}
	return guard, member
}

func cliContainsValue(list []any, want any) bool {
	for _, v := range list {
		if cliValuesEqual(v, want) {
			return true
		}
	}
	return false
}

// cliValuesEqual compares the numeric representations emitted by the schema
// and manifest decoders without routing integers through float64. Integer and
// float values compare equal only when the float represents that integer
// exactly and lies inside the integer type's range.
func cliValuesEqual(a, b any) bool {
	switch v := a.(type) {
	case int:
		return cliSignedValueEqual(int64(v), b)
	case int64:
		return cliSignedValueEqual(v, b)
	case uint64:
		return cliUnsignedValueEqual(v, b)
	case float64:
		return cliFloatValueEqual(v, b)
	default:
		return reflect.DeepEqual(a, b)
	}
}

func cliSignedValueEqual(a int64, b any) bool {
	switch v := b.(type) {
	case int:
		return a == int64(v)
	case int64:
		return a == v
	case uint64:
		return a >= 0 && uint64(a) == v
	case float64:
		return v >= float64(-1<<63) && v < float64(1<<63) && int64(v) == a && v == float64(a)
	default:
		return false
	}
}

func cliUnsignedValueEqual(a uint64, b any) bool {
	switch v := b.(type) {
	case int:
		return v >= 0 && a == uint64(v)
	case int64:
		return v >= 0 && a == uint64(v)
	case uint64:
		return a == v
	case float64:
		return v >= 0 && v < float64(1<<64) && uint64(v) == a && v == float64(a)
	default:
		return false
	}
}

func cliFloatValueEqual(a float64, b any) bool {
	switch v := b.(type) {
	case int:
		return cliSignedValueEqual(int64(v), a)
	case int64:
		return cliSignedValueEqual(v, a)
	case uint64:
		return cliUnsignedValueEqual(v, a)
	case float64:
		return a == v
	default:
		return false
	}
}

// cliIsConstProperty reports whether a property schema admits exactly one
// value (const or a single-value enum) — the discriminator shape.
func cliIsConstProperty(prop map[string]any) bool {
	if prop == nil {
		return false
	}
	if _, ok := prop["const"]; ok {
		return true
	}
	if enum, ok := prop["enum"].([]any); ok && len(enum) == 1 {
		return true
	}
	return false
}

// unionBodySchema locates the union behind a request body: the body itself
// when it declares oneOf/anyOf inline, or the component it references when
// that component is the union (a `$ref` to a oneOf schema — the common way
// to declare a union body). directRef is the body's own reference when it
// points at a non-union component (a plain object body).
func (d *cliManifestDecoder) unionBodySchema(bodyMap map[string]any) (unionMap map[string]any, directRef string) {
	if bodyMap == nil {
		return nil, ""
	}
	if len(cliUnionMembers(bodyMap)) > 0 {
		return bodyMap, ""
	}
	ref, ok := bodyMap["$ref"].(string)
	if !ok {
		return nil, ""
	}
	current := ref
	for depth := 0; depth < 8; depth++ {
		component, ok := d.schemaIndex.components[strings.TrimPrefix(current, cliComponentRefPrefix)].(map[string]any)
		if !ok {
			break
		}
		if len(cliUnionMembers(component)) > 0 {
			return component, ""
		}
		next, ok := component["$ref"].(string)
		if !ok {
			break
		}
		current = next
	}
	return nil, ref
}

// cliUnionMemberRefs lists the component references of a schema's oneOf/anyOf
// members. Inline members are not pinnable and are omitted.
func cliUnionMemberRefs(schema map[string]any) []string {
	if schema == nil {
		return nil
	}
	var refs []string
	for _, unionKey := range []string{"oneOf", "anyOf"} {
		if list, ok := schema[unionKey].([]any); ok {
			for _, member := range list {
				if m, ok := member.(map[string]any); ok {
					if ref, ok := m["$ref"].(string); ok {
						refs = append(refs, ref)
					}
				}
			}
		}
	}
	return refs
}

func cliTrimRefPrefixes(refs []string) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, strings.TrimPrefix(ref, cliComponentRefPrefix))
	}
	return out
}

func cliComponentNames(components map[string]any) []string {
	names := make([]string, 0, len(components))
	for name := range components {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resolveObjectSchema follows $ref chains and composes allOf branches into a
// single object view. Cycles terminate with an error naming the chain.
func (d *cliManifestDecoder) resolveObjectSchema(schema any, seen []string) (*cliResolvedSchema, error) {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil, stderrors.New("schema is not an object schema")
	}

	if ref, ok := schemaMap["$ref"].(string); ok {
		if !strings.HasPrefix(ref, cliComponentRefPrefix) {
			return nil, fmt.Errorf("reference %q is not a component schema reference; only %s<Name> references are supported", ref, cliComponentRefPrefix)
		}
		name := strings.TrimPrefix(ref, cliComponentRefPrefix)
		for _, ancestor := range seen {
			if ancestor == name {
				return nil, fmt.Errorf("schema reference cycle: %s", strings.Join(append(seen, name), " -> "))
			}
		}
		component, ok := d.schemaIndex.components[name]
		if !ok {
			return nil, fmt.Errorf("referenced schema %q does not exist in components.schemas%s", name, cliDidYouMean(name, cliComponentNames(d.schemaIndex.components)))
		}
		if len(schemaMap) > 1 {
			// JSON Schema 2020-12: keywords beside $ref apply conjunctively.
			// Rewrite as an implicit allOf so the existing composition (and
			// its cycle detection via seen) handles the reference branch
			// alongside the siblings instead of discarding them.
			branches := []any{map[string]any{"$ref": ref}}
			rest := map[string]any{}
			for k, v := range schemaMap {
				switch k {
				case "$ref":
				case "allOf":
					if list, ok := v.([]any); ok {
						branches = append(branches, list...)
					}
				default:
					rest[k] = v
				}
			}
			rest["allOf"] = branches
			resolved, err := d.resolveObjectSchema(rest, seen)
			if err != nil {
				return nil, err
			}
			if resolved.refName == "" {
				resolved.refName = name
			}
			return resolved, nil
		}
		resolved, err := d.resolveObjectSchema(component, append(seen, name))
		if err != nil {
			return nil, err
		}
		if resolved.refName == "" {
			resolved.refName = name
		}
		return resolved, nil
	}

	_, hasConst := schemaMap["const"]
	_, hasEnum := schemaMap["enum"]
	out := &cliResolvedSchema{
		properties: map[string]any{},
		required:   map[string]bool{},
		enumerated: hasConst || hasEnum,
		raw:        schemaMap,
	}

	if allOf, ok := schemaMap["allOf"].([]any); ok {
		for _, branch := range allOf {
			resolvedBranch, err := d.resolveObjectSchema(branch, seen)
			if err != nil {
				return nil, err
			}
			for _, name := range resolvedBranch.propOrder {
				branchProp := resolvedBranch.properties[name]
				if existing, dup := out.properties[name]; dup {
					if !reflect.DeepEqual(existing, branchProp) {
						// allOf intersects constraints, so a redeclared
						// property is a refinement, not a conflict. Keep
						// both layers; fact derivation intersects them and
						// reports genuinely unsatisfiable combinations.
						out.properties[name] = map[string]any{"allOf": []any{existing, branchProp}}
					}
					continue
				}
				out.properties[name] = branchProp
				out.propOrder = append(out.propOrder, name)
			}
			for name := range resolvedBranch.required {
				out.required[name] = true
			}
			if resolvedBranch.explicitOpen {
				out.explicitOpen = true
			}
			if resolvedBranch.explicitClosed {
				out.explicitClosed = true
			}
			if resolvedBranch.enumerated {
				out.enumerated = true
			}
			out.additional = cliConjoinSchemas(out.additional, resolvedBranch.additional)
			// A union nested in an allOf branch constrains the composed
			// object; the artifact walk expands it against the merged view.
			if resolvedBranch.unionMembers != nil && out.unionMembers == nil {
				out.unionMembers = resolvedBranch.unionMembers
			}
		}
	}

	if props, ok := schemaMap["properties"].(map[string]any); ok {
		names := make([]string, 0, len(props))
		for name := range props {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if existing, dup := out.properties[name]; dup {
				if !reflect.DeepEqual(existing, props[name]) {
					// Direct redeclaration refines the allOf layer the same
					// way sibling allOf branches refine each other.
					out.properties[name] = map[string]any{"allOf": []any{existing, props[name]}}
				}
				continue
			}
			out.properties[name] = props[name]
			out.propOrder = append(out.propOrder, name)
		}
	}

	if required, ok := schemaMap["required"].([]any); ok {
		for _, name := range required {
			if s, ok := name.(string); ok {
				out.required[s] = true
			}
		}
	}

	switch additional := schemaMap["additionalProperties"].(type) {
	case bool:
		out.explicitOpen = additional
		if !additional {
			// Never un-close: a closed layer anywhere in the composition
			// constrains the conjunction even when an enclosing layer
			// declares additionalProperties: true.
			out.explicitClosed = true
		}
	case map[string]any:
		out.explicitOpen = true
		if len(additional) > 0 {
			out.additional = cliConjoinSchemas(out.additional, additional)
		}
	}
	if out.explicitClosed {
		// The conjunction of an open and a closed layer is closed.
		out.explicitOpen = false
	}

	if members := cliUnionMembers(schemaMap); members != nil {
		out.unionMembers = members
	}
	return out, nil
}

func cliConjoinSchemas(a, b map[string]any) map[string]any {
	switch {
	case a == nil:
		return b
	case b == nil || reflect.DeepEqual(a, b):
		return a
	}
	return map[string]any{"allOf": []any{a, b}}
}

func cliUnionMembers(schema map[string]any) []any {
	for _, unionKey := range []string{"oneOf", "anyOf"} {
		if list, ok := schema[unionKey].([]any); ok && len(list) > 0 {
			return list
		}
	}
	return nil
}

// mergePropertyFacts composes the facts of a referenced schema with facts
// derived from the reference's sibling keywords, conjunctively: validation
// keywords intersect (a contradiction is an error rather than an override),
// annotations take the nearest (sibling) value, and readOnly is effective
// when either layer declares it.
func mergePropertyFacts(ref, sibling *cliPropertyFacts) (*cliPropertyFacts, error) {
	out := *ref
	out.catalogCommands = slices.Clone(ref.catalogCommands)
	for _, command := range sibling.catalogCommands {
		if !slices.Contains(out.catalogCommands, command) {
			out.catalogCommands = append(out.catalogCommands, command)
		}
	}
	if sibling.kinds != nil {
		switch {
		case out.kinds == nil:
			out.kinds = sibling.kinds
		case slices.Equal(out.kinds, sibling.kinds):
		case len(out.kinds) == 1 && len(sibling.kinds) == 1 &&
			(out.kinds[0] == "int" && sibling.kinds[0] == "float" || out.kinds[0] == "float" && sibling.kinds[0] == "int"):
			// integer refines number: the intersection is integer.
			out.kinds = []string{"int"}
		default:
			return nil, fmt.Errorf("conjunctive schema composition declares type %q alongside %q; the constraints cannot both hold", strings.Join(sibling.kinds, ","), strings.Join(out.kinds, ","))
		}
	}
	if sibling.enum != nil {
		if out.enum == nil {
			out.enum = sibling.enum
		} else {
			intersection := make([]any, 0, len(out.enum))
			for _, v := range out.enum {
				for _, s := range sibling.enum {
					if cliValuesEqual(v, s) {
						intersection = append(intersection, v)
						break
					}
				}
			}
			if len(intersection) == 0 {
				return nil, stderrors.New("conjunctive schema composition declares enums with no values in common; the constraints cannot both hold")
			}
			out.enum = intersection
		}
	}
	if sibling.openEnum {
		out.openEnum = true
	}
	if sibling.hasDefault {
		out.defaultVal = sibling.defaultVal
		out.hasDefault = true
	}
	if sibling.hasConst {
		if out.hasConst && !cliValuesEqual(out.constVal, sibling.constVal) {
			return nil, stderrors.New("conjunctive schema composition declares contradictory const values")
		}
		out.constVal = sibling.constVal
		out.hasConst = true
	}
	if sibling.description != "" {
		out.description = sibling.description
	}
	if sibling.readOnly {
		out.readOnly = true
	}
	if sibling.items != nil {
		if out.items != nil {
			merged, err := mergePropertyFacts(out.items, sibling.items)
			if err != nil {
				return nil, fmt.Errorf("items: %w", err)
			}
			out.items = merged
			out.arrayOfStr = len(merged.kinds) == 1 && merged.kinds[0] == "string"
			return &out, nil
		}
		out.items = sibling.items
		out.arrayOfStr = sibling.arrayOfStr
		out.itemsErr = sibling.itemsErr
	}
	if sibling.unionArms != nil {
		if out.unionArms != nil {
			return nil, stderrors.New("reference siblings declare a union alongside a referenced union schema; move the members into one schema")
		}
		out.unionArms = sibling.unionArms
	}
	return &out, nil
}

// propertyFacts classifies the property schema behind a single-segment body
// pointer, following references and unwrapping single-scalar facts.
func (d *cliManifestDecoder) propertyFacts(schema any, seen []string) (*cliPropertyFacts, error) {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil, stderrors.New("property schema is not an object")
	}

	if ref, ok := schemaMap["$ref"].(string); ok {
		if !strings.HasPrefix(ref, cliComponentRefPrefix) {
			return nil, fmt.Errorf("reference %q is not a component schema reference", ref)
		}
		name := strings.TrimPrefix(ref, cliComponentRefPrefix)
		for _, ancestor := range seen {
			if ancestor == name {
				return nil, fmt.Errorf("schema reference cycle: %s", strings.Join(append(seen, name), " -> "))
			}
		}
		component, ok := d.schemaIndex.components[name]
		if !ok {
			return nil, fmt.Errorf("referenced schema %q does not exist in components.schemas%s", name, cliDidYouMean(name, cliComponentNames(d.schemaIndex.components)))
		}
		refFacts, err := d.propertyFacts(component, append(seen, name))
		if err != nil {
			return nil, err
		}
		if len(schemaMap) == 1 {
			return refFacts, nil
		}
		// JSON Schema 2020-12: keywords beside $ref apply conjunctively —
		// {$ref, readOnly: true} is read-only, sibling defaults/enums count.
		siblings := map[string]any{}
		for k, v := range schemaMap {
			if k != "$ref" {
				siblings[k] = v
			}
		}
		siblingFacts, err := d.propertyFacts(siblings, seen)
		if err != nil {
			return nil, err
		}
		return mergePropertyFacts(refFacts, siblingFacts)
	}

	if allOf, ok := schemaMap["allOf"].([]any); ok && len(allOf) > 0 {
		// allOf intersects: fold every branch's facts through the same
		// conjunctive merge the $ref-sibling path uses, then apply keywords
		// declared beside allOf last — they are the nearest layer, so their
		// annotations win over every branch.
		rest := map[string]any{}
		for k, v := range schemaMap {
			if k != "allOf" {
				rest[k] = v
			}
		}
		var merged *cliPropertyFacts
		for _, branch := range allOf {
			branchFacts, err := d.propertyFacts(branch, seen)
			if err != nil {
				return nil, err
			}
			if merged == nil {
				merged = branchFacts
				continue
			}
			// Peer branches have no "nearest": a genuine annotation
			// disagreement between them is ambiguous, never last-wins.
			if merged.hasDefault && branchFacts.hasDefault && !cliValuesEqual(merged.defaultVal, branchFacts.defaultVal) {
				return nil, stderrors.New("allOf branches declare different defaults; move the default into one schema")
			}
			merged, err = mergePropertyFacts(merged, branchFacts)
			if err != nil {
				return nil, err
			}
		}
		restFacts, err := d.propertyFacts(rest, seen)
		if err != nil {
			return nil, err
		}
		if merged == nil {
			return restFacts, nil
		}
		return mergePropertyFacts(merged, restFacts)
	}

	facts := &cliPropertyFacts{}
	if catalog, ok := schemaMap["x-speakeasy-cli-catalog"].(map[string]any); ok {
		if command, ok := catalog["command"].(string); ok && command != "" {
			facts.catalogCommands = []string{command}
		}
	}
	if desc, ok := schemaMap["description"].(string); ok {
		facts.description = desc
	}
	if readOnly, ok := schemaMap["readOnly"].(bool); ok {
		facts.readOnly = readOnly
	}
	if defaultVal, present := schemaMap["default"]; present {
		facts.defaultVal = defaultVal
		facts.hasDefault = true
	}
	if constVal, present := schemaMap["const"]; present {
		facts.constVal = constVal
		facts.hasConst = true
	}
	if enum, ok := schemaMap["enum"].([]any); ok {
		facts.enum = enum
	}
	// Open-enum marker: only the explicit "allow" value opens the enum,
	// matching the IsOpenEnum semantics ("", "disallow" = closed).
	if unknownValues, ok := schemaMap["x-speakeasy-unknown-values"].(string); ok && unknownValues == "allow" {
		facts.openEnum = true
	}

	if members := cliUnionMembers(schemaMap); members != nil {
		facts.unionArms = members
		return facts, nil
	}

	typeStr, _ := schemaMap["type"].(string)
	switch typeStr {
	case "string":
		facts.kinds = []string{"string"}
	case "integer":
		facts.kinds = []string{"int"}
	case "number":
		facts.kinds = []string{"float"}
	case "boolean":
		facts.kinds = []string{"bool"}
	case "array":
		facts.kinds = []string{"array"}
		if items, ok := schemaMap["items"].(map[string]any); ok {
			itemFacts, err := d.propertyFacts(items, seen)
			if err != nil {
				facts.itemsErr = err
				break
			}
			facts.items = itemFacts
			if len(itemFacts.kinds) == 1 && itemFacts.kinds[0] == "string" {
				facts.arrayOfStr = true
				if facts.enum == nil {
					facts.enum = itemFacts.enum
				}
				if itemFacts.openEnum {
					facts.openEnum = true
				}
			}
		}
	case "object":
		facts.kinds = []string{"object"}
	case "":
		if _, hasProps := schemaMap["properties"]; hasProps {
			facts.kinds = []string{"object"}
		} else if allOf, hasAllOf := schemaMap["allOf"].([]any); hasAllOf && len(allOf) > 0 {
			facts.kinds = []string{"object"}
		} else {
			// Untyped schema: accept anything, infer nothing.
			facts.kinds = nil
		}
	default:
		facts.kinds = []string{typeStr}
	}
	return facts, nil
}

// lookupProperty resolves a single-segment pointer inside the composed
// variant schema. A miss on a closed schema is an error with a did-you-mean;
// a miss on an explicitly-open schema is a warning that disables inference.
func (d *cliManifestDecoder) lookupProperty(cmdKey, what, pointer string, variant *cliResolvedSchema) (*cliPropertyFacts, bool, error) {
	name := cliPointerPropertyName(pointer)
	propSchema, ok := variant.properties[name]
	if !ok {
		if variant.explicitOpen {
			d.warnf("command %q %s %s does not name a declared property of %s (the schema explicitly accepts unknown keys); no validation or inference applies", cmdKey, what, pointer, cliSchemaLabel(variant))
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("command %q %s %s does not resolve in %s%s", cmdKey, what, pointer, cliSchemaLabel(variant), cliDidYouMean(name, variant.propOrder))
	}
	facts, err := d.propertyFacts(propSchema, nil)
	if err != nil {
		return nil, false, fmt.Errorf("command %q %s %s: %w", cmdKey, what, pointer, err)
	}
	return facts, true, nil
}

func cliSplitNestedPointer(pointer string) (parent, leaf string, ok bool) {
	head, tail, found := strings.Cut(strings.TrimPrefix(pointer, "/"), "/")
	if !found {
		return "", "", false
	}
	return "/" + head, "/" + tail, true
}

// nestedBindScope resolves the object a nested flag bind writes into: the
// property at parent, or the member of its union the object preset selects.
func (d *cliManifestDecoder) nestedBindScope(cmdKey, what, parent, leaf string, variant *cliResolvedSchema, presetValues map[string]any) (*cliResolvedSchema, map[string]any, error) {
	pointer := parent + leaf
	presetObj, ok := presetValues[parent].(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("command %q %s %s nests beneath %s, which no object preset sets; a nested bind targets a field of an object preset (preset: {$.%s: {...}})", cmdKey, what, pointer, parent, cliPointerPropertyName(parent))
	}
	flat := newCLIFlatUnion()
	d.flattenUnion(variant.properties[cliPointerPropertyName(parent)], 0, nil, flat)
	field := cliPointerPropertyName(leaf)
	var scope *cliResolvedSchema
	pinned := ""
	switch {
	case len(flat.members) == 1:
		scope = flat.members[0]
		if _, ok := presetObj[field]; ok && cliContains(flat.keys, field) {
			pinned = field
		}
	case len(flat.members) > 1:
		point, member := d.presetUnionMergePoint(parent, flat, presetObj)
		if member == nil {
			return nil, nil, fmt.Errorf("command %q %s %s: the preset at %s does not select exactly one member of its union, so the field cannot be resolved; pin the member's discriminator in the preset", cmdKey, what, pointer, parent)
		}
		pinned = point.Key
		scope = member
	}
	if pinned != "" && pinned == field {
		return nil, nil, fmt.Errorf("command %q %s %s targets the discriminator of the union at %s, which the preset pins; a flag there would switch members", cmdKey, what, pointer, parent)
	}
	if scope == nil {
		return nil, nil, fmt.Errorf("command %q %s %s nests beneath %s, which is not an object property", cmdKey, what, pointer, parent)
	}
	if scope.enumerated {
		return nil, nil, fmt.Errorf("command %q %s %s nests beneath %s, whose schema restricts the object to const/enum values; a field set by a flag would match none of them", cmdKey, what, pointer, parent)
	}
	leafPresets := map[string]any{}
	if value, ok := presetObj[field]; ok {
		leafPresets[leaf] = value
	}
	return scope, leafPresets, nil
}

func cliPointerPropertyName(pointer string) string {
	name := strings.TrimPrefix(pointer, "/")
	name = strings.ReplaceAll(name, "~1", "/")
	return strings.ReplaceAll(name, "~0", "~")
}

func cliSchemaLabel(variant *cliResolvedSchema) string {
	if variant.refName != "" {
		return variant.refName
	}
	return "the request body schema"
}

func (d *cliManifestDecoder) linkInput(cmdKey string, input *CLICommandInput, variant *cliResolvedSchema, presetPointers map[string]bool, presetValues map[string]any, isArg bool) error {
	what := fmt.Sprintf("flag %q bind", input.Name)
	if isArg {
		what = fmt.Sprintf("arg %q bind", input.Name)
	}

	// The sole positional is variadic: the renderer joins argv into one
	// string, so only string-typed bindings can be honest — including
	// explicitly typed bindings into open schemas.
	if isArg && input.Type != "" && input.Type != "string" {
		return fmt.Errorf("command %q arg %q resolves to type %s; the variadic positional joins argv into a single string, so only string bindings are part of v1 (declare a flag instead)", cmdKey, input.Name, input.Type)
	}

	pointer := input.Bind.Pointer
	parent, leaf, nested := cliSplitNestedPointer(pointer)
	if nested {
		if isArg {
			return fmt.Errorf("command %q arg %q binds %s; the positional binds a top-level body field (declare a flag for a nested field)", cmdKey, input.Name, pointer)
		}
		scope, leafPresets, err := d.nestedBindScope(cmdKey, what, parent, leaf, variant, presetValues)
		if err != nil {
			return err
		}
		variant, presetValues, pointer = scope, leafPresets, leaf
		what = fmt.Sprintf("flag %q bind beneath %s:", input.Name, parent)
	}

	facts, found, err := d.lookupProperty(cmdKey, what, pointer, variant)
	if err != nil {
		return err
	}
	if nested && found && facts.readOnly {
		return fmt.Errorf("command %q %s %s targets a readOnly property; the server would reject or ignore it", cmdKey, what, input.Bind.Pointer)
	}
	if !found {
		if input.Type == "" {
			return fmt.Errorf("command %q %s %s targets an undeclared property; declare an explicit type: (string, int, float, or bool)", cmdKey, what, input.Bind.Pointer)
		}
		if input.SuggestionsDeclared {
			if err := cliCheckSuggestions(input.Enum, input.Type); err != nil {
				return fmt.Errorf("command %q %s %s suggestions: %w", cmdKey, what, input.Bind.Pointer, err)
			}
		}
		if value, ok := presetValues[pointer]; ok && !isArg && cliIsPresetDefault(value, input.Type) {
			input.DefaultFrom = "preset"
			input.Default = value
		}
		return nil
	}

	// Type: infer single-scalar targets, check explicit declarations. For
	// union targets an explicit type is a checked arm selection.
	if facts.unionArms != nil {
		if input.Type == "" {
			kinds, err := d.unionArmKinds(facts.unionArms)
			if err != nil {
				return fmt.Errorf("command %q %s %s: %w", cmdKey, what, input.Bind.Pointer, err)
			}
			return fmt.Errorf("command %q %s %s targets a union; declare an explicit type: selecting one arm (arms: %s)", cmdKey, what, input.Bind.Pointer, cliCandidateList(kinds))
		}
		arm, err := d.selectUnionArm(facts.unionArms, input.Type)
		if err != nil {
			return fmt.Errorf("command %q %s %s: %w", cmdKey, what, input.Bind.Pointer, err)
		}
		if input.SuggestionsDeclared {
			if err := cliCheckSuggestions(input.Enum, input.Type); err != nil {
				return fmt.Errorf("command %q %s %s suggestions: %w", cmdKey, what, input.Bind.Pointer, err)
			}
		} else if len(arm.enum) > 0 {
			input.Enum = arm.enum
		}
		if input.Summary == "" {
			input.Summary = cliFirstSentence(facts.description)
		}
	} else {
		switch {
		case len(facts.kinds) == 1 && cliIsScalarKind(facts.kinds[0]):
			if input.Type == "" {
				input.Type = facts.kinds[0]
			} else if input.Type != facts.kinds[0] {
				return fmt.Errorf("command %q %s %s declares type %s, but the schema property is %s", cmdKey, what, input.Bind.Pointer, input.Type, facts.kinds[0])
			}
		case len(facts.kinds) == 0:
			if input.Type == "" {
				return fmt.Errorf("command %q %s %s targets an untyped schema; declare an explicit type:", cmdKey, what, input.Bind.Pointer)
			}
		default:
			return fmt.Errorf("command %q %s %s targets a %s property; v1 inputs bind scalar fields only (structured construction requires the request-plan.payload-holes capability)", cmdKey, what, input.Bind.Pointer, facts.kinds[0])
		}
		if input.SuggestionsDeclared {
			if err := cliCheckSuggestions(input.Enum, input.Type); err != nil {
				return fmt.Errorf("command %q %s %s suggestions: %w", cmdKey, what, input.Bind.Pointer, err)
			}
		} else if len(facts.enum) > 0 {
			input.Enum = facts.enum
		}
		if input.Summary == "" {
			input.Summary = cliFirstSentence(facts.description)
		}
	}

	// Re-check after inference: a schema-inferred positional type must be a
	// string for the same reason as an explicit one.
	if isArg && input.Type != "" && input.Type != "string" {
		return fmt.Errorf("command %q arg %q resolves to type %s; the variadic positional joins argv into a single string, so only string bindings are part of v1 (declare a flag instead)", cmdKey, input.Name, input.Type)
	}

	// Requiredness: args inherit the schema's composed required[] unless the
	// author decided, minus pointers a preset already satisfies. Flags are
	// never required-inferred.
	if isArg && !input.requiredExplicit {
		name := cliPointerPropertyName(input.Bind.Pointer)
		if variant.required[name] && !presetPointers[input.Bind.Pointer] && !facts.hasDefault {
			input.Required = true
		}
	}

	// Display-only defaults. A preset at the same pointer is what the request
	// carries when the flag is omitted, so it is the default help shows.
	presetValue, presetCovered := presetValues[pointer]
	switch {
	case isArg:
	case presetCovered && cliIsPresetDefault(presetValue, input.Type):
		input.DefaultFrom = "preset"
		input.Default = presetValue
	case input.DefaultFrom == "schema" && presetCovered:
		input.DefaultFrom = ""
		input.Default = nil
		d.warnf("command %q flag %q: schema-default display suppressed because a preset sets %s", cmdKey, input.Name, input.Bind.Pointer)
	case input.DefaultFrom == "schema" && !facts.hasDefault:
		return fmt.Errorf("command %q flag %q declares defaultFrom: schema, but the schema property at %s has no default", cmdKey, input.Name, input.Bind.Pointer)
	case input.DefaultFrom == "schema":
		normalized, err := cliCheckSchemaDefault(facts.defaultVal, input.Type)
		if err != nil {
			return fmt.Errorf("command %q flag %q defaultFrom: schema: the schema default at %s does not match type %s: %w", cmdKey, input.Name, input.Bind.Pointer, input.Type, err)
		}
		input.Default = normalized
	case input.Default != nil && input.Type != "":
		if err := cliCheckScalarValue(input.Default, input.Type); err != nil {
			return fmt.Errorf("command %q flag %q default: %w", cmdKey, input.Name, err)
		}
	}

	return nil
}

func (d *cliManifestDecoder) unionArmKinds(arms []any) ([]string, error) {
	var kinds []string
	for _, arm := range arms {
		facts, err := d.propertyFacts(arm, nil)
		if err != nil {
			return nil, err
		}
		switch {
		case facts.unionArms != nil:
			kinds = append(kinds, "union")
		case len(facts.kinds) == 1:
			kinds = append(kinds, facts.kinds[0])
		default:
			kinds = append(kinds, "untyped")
		}
	}
	return kinds, nil
}

// selectUnionArm finds the union arm matching an explicitly declared scalar
// type. The declaration is a checked selection, not a blind override.
func (d *cliManifestDecoder) selectUnionArm(arms []any, declaredType string) (*cliPropertyFacts, error) {
	var matches []*cliPropertyFacts
	for _, arm := range arms {
		facts, err := d.propertyFacts(arm, nil)
		if err != nil {
			return nil, err
		}
		if len(facts.kinds) == 1 && facts.kinds[0] == declaredType {
			matches = append(matches, facts)
		}
	}
	if len(matches) == 0 {
		kinds, err := d.unionArmKinds(arms)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("declared type %s selects no union arm (arms: %s)", declaredType, cliCandidateList(kinds))
	}
	return matches[0], nil
}

// linkArtifact validates an output.artifact declaration against the
// operation's 2xx application/json response schema: the content pointer must
// walk to a set of schemas, and at least one union branch must survive the
// COMPLETE walk to a content block matching the DECLARED shape (an object
// whose typeField property is a const equal to the kind and whose dataField
// property is a string; the v1 names type/data/mime_type/uri are the
// defaults). Structural drift in the response schema therefore fails
// generation with the fix named instead of producing a runtime that can
// never find its content. Declared identity bindings are validated against
// the response root the same way (defaulted identity stays the v1
// best-effort enrichment and is deliberately not linked, so existing
// declarations cannot start failing generation).
func (d *cliManifestDecoder) linkArtifact(cmdKey string, artifact *CLICommandArtifact, opInfo *cliOperationInfo, opID string) error {
	if opInfo.responseSchema == nil {
		return fmt.Errorf("command %q declares output.artifact, but operation %q has no application/json 2xx response to extract %s content from", cmdKey, opID, artifact.Kind)
	}

	artifact.ResponseCode = opInfo.responseCode

	views, err := d.artifactExpand(opInfo.responseSchema, 0)
	if err != nil {
		return fmt.Errorf("command %q output.artifact: %w", cmdKey, err)
	}

	for i, seg := range artifact.Segments {
		var next []*cliResolvedSchema
		fieldCandidates := map[string]bool{}
		for _, view := range views {
			if seg.Wild {
				if view.raw == nil || view.raw["type"] != "array" {
					continue
				}
				items, ok := view.raw["items"]
				if !ok {
					continue
				}
				expanded, err := d.artifactExpand(items, 0)
				if err != nil {
					return fmt.Errorf("command %q output.artifact contentPointer %s segment %d ([*]): %w", cmdKey, artifact.ContentPointer, i+1, err)
				}
				next = append(next, expanded...)
				continue
			}
			for _, name := range view.propOrder {
				fieldCandidates[name] = true
			}
			prop, ok := view.properties[seg.Field]
			if !ok {
				continue
			}
			expanded, err := d.artifactExpand(prop, 0)
			if err != nil {
				return fmt.Errorf("command %q output.artifact contentPointer %s segment %d (.%s): %w", cmdKey, artifact.ContentPointer, i+1, seg.Field, err)
			}
			next = append(next, expanded...)
		}
		if len(next) == 0 {
			if seg.Wild {
				return fmt.Errorf("command %q output.artifact contentPointer %s: segment %d ([*]) does not address an array in the response schema of operation %q", cmdKey, artifact.ContentPointer, i+1, opID)
			}
			return fmt.Errorf("command %q output.artifact contentPointer %s: segment %d (.%s) does not resolve in the response schema of operation %q%s", cmdKey, artifact.ContentPointer, i+1, seg.Field, opID, cliDidYouMean(seg.Field, cliSortedKeys(fieldCandidates)))
		}
		views = next
	}

	var foundTypes []string
	uriRejected := false
	for _, view := range views {
		typeConst := d.artifactTypeConst(view, artifact.TypeField)
		if typeConst != "" && !cliContains(foundTypes, typeConst) {
			foundTypes = append(foundTypes, typeConst)
		}
		if typeConst != artifact.Kind {
			continue
		}
		dataProp, ok := view.properties[artifact.DataField]
		if !ok {
			continue
		}
		dataFacts, err := d.propertyFacts(dataProp, nil)
		if err != nil || len(dataFacts.kinds) != 1 || dataFacts.kinds[0] != "string" {
			continue
		}
		if mimeProp, ok := view.properties[artifact.MimeTypeField]; ok {
			mimeFacts, err := d.propertyFacts(mimeProp, nil)
			if err != nil || len(mimeFacts.kinds) != 1 || mimeFacts.kinds[0] != "string" {
				continue
			}
		}
		if artifact.blockExplicit {
			// A declared block validates its URI binding too; the defaulted
			// "uri" stays a best-effort runtime read as in v1.
			if uriProp, ok := view.properties[artifact.URIField]; ok {
				uriFacts, err := d.propertyFacts(uriProp, nil)
				if err != nil || len(uriFacts.kinds) != 1 || uriFacts.kinds[0] != "string" {
					uriRejected = true
					continue
				}
			}
		}
		if err := d.linkArtifactIdentity(cmdKey, artifact, opInfo, opID); err != nil {
			return err
		}
		return nil // a viable content-block branch survives the whole walk
	}
	if uriRejected {
		// The only disqualifier of an otherwise viable block was the URI
		// binding: name it instead of blaming the data binding.
		return fmt.Errorf("command %q output.artifact: the %s block at %s has a %q property that is not a string; the declared block uriField binding must be a string property", cmdKey, artifact.Kind, artifact.ContentPointer, artifact.URIField)
	}
	if len(foundTypes) > 0 {
		return fmt.Errorf("command %q output.artifact: the content at %s has no %s block with a string %q property (content types found: %s)", cmdKey, artifact.ContentPointer, artifact.Kind, artifact.DataField, cliCandidateList(foundTypes))
	}
	return fmt.Errorf("command %q output.artifact: the content at %s is not a typed content block (expected objects with a %q const %q and a string %q property)", cmdKey, artifact.ContentPointer, artifact.TypeField, artifact.Kind, artifact.DataField)
}

// linkArtifactIdentity validates explicitly declared identity bindings
// against the root of the linked response schema. Deliberately as strict as
// async pointer linking: a declared binding missing from some response
// variant would be a silently absent enrichment at runtime, so every
// traversable variant must carry the property as a string, and a closed
// status enum must contain the declared terminal status.
func (d *cliManifestDecoder) linkArtifactIdentity(cmdKey string, artifact *CLICommandArtifact, opInfo *cliOperationInfo, opID string) error {
	if !artifact.identityExplicit {
		return nil
	}
	views, err := d.artifactExpand(opInfo.responseSchema, 0)
	if err != nil {
		return fmt.Errorf("command %q output.artifact identity: %w", cmdKey, err)
	}
	if len(views) == 0 {
		return fmt.Errorf("command %q output.artifact identity: the %s response of operation %q has no traversable object variant to bind identity fields against", cmdKey, opInfo.responseCode, opID)
	}
	for _, binding := range []struct{ key, field string }{
		{"idField", artifact.IDField},
		{"statusField", artifact.StatusField},
	} {
		for _, view := range views {
			prop, ok := view.properties[binding.field]
			if !ok {
				return fmt.Errorf("command %q output.artifact identity.%s: property %q does not resolve at the root of the %s response of operation %q in every variant%s", cmdKey, binding.key, binding.field, opInfo.responseCode, opID, cliDidYouMean(binding.field, view.propOrder))
			}
			facts, err := d.propertyFacts(prop, nil)
			if err != nil {
				return fmt.Errorf("command %q output.artifact identity.%s: %w", cmdKey, binding.key, err)
			}
			if len(facts.kinds) != 1 || facts.kinds[0] != "string" {
				return fmt.Errorf("command %q output.artifact identity.%s: property %q must be a string in every variant of the %s response of operation %q", cmdKey, binding.key, binding.field, opInfo.responseCode, opID)
			}
			if binding.key == "statusField" && !facts.openEnum && len(facts.enum) > 0 {
				// Per-variant, matching the string-kind strictness above: a
				// closed enum in ANY variant must contain the terminal
				// status, or that variant could never deliver content.
				variantStates := map[string]bool{}
				for _, member := range facts.enum {
					if state, ok := member.(string); ok {
						variantStates[state] = true
					}
				}
				if !variantStates[artifact.TerminalStatus] {
					return fmt.Errorf("command %q output.artifact identity.terminalStatus %q is not an enum member of %q in the %s response of operation %q%s", cmdKey, artifact.TerminalStatus, artifact.StatusField, opInfo.responseCode, opID, cliDidYouMean(artifact.TerminalStatus, cliSortedKeys(variantStates)))
				}
			}
		}
	}
	return nil
}

// artifactExpand resolves $ref/allOf and flattens oneOf/anyOf members into the
// concrete schema views a content pointer can traverse. Unresolvable branches
// are dropped (some union arms are legitimately non-object); depth bounds
// pathological nesting.
func (d *cliManifestDecoder) artifactExpand(schema any, depth int) ([]*cliResolvedSchema, error) {
	if depth > 16 {
		return nil, stderrors.New("union nesting exceeds 16 levels")
	}
	resolved, resolveErr := d.resolveObjectSchema(schema, nil)
	if resolveErr != nil {
		// Non-object branch (e.g. a bare scalar union arm): not traversable,
		// and dropping it is the point of the walk, not an error.
		return nil, nil //nolint:nilerr
	}
	if resolved.unionMembers == nil {
		return []*cliResolvedSchema{resolved}, nil
	}
	var out []*cliResolvedSchema
	for _, member := range resolved.unionMembers {
		expanded, err := d.artifactExpand(member, depth+1)
		if err != nil {
			return nil, err
		}
		for _, branch := range expanded {
			// Properties declared beside the union (or in sibling allOf
			// branches) apply to every arm: merge them into the branch view
			// without overriding what the arm itself declares.
			out = append(out, artifactMergeParent(branch, resolved))
		}
	}
	return out, nil
}

// artifactMergeParent returns branch extended with the parent's properties
// (arm-declared properties win). Only property visibility matters for the
// pointer walk; required/open flags are irrelevant here.
func artifactMergeParent(branch, parent *cliResolvedSchema) *cliResolvedSchema {
	if len(parent.properties) == 0 {
		return branch
	}
	merged := &cliResolvedSchema{
		refName:    branch.refName,
		properties: map[string]any{},
		required:   map[string]bool{},
		explicitOpen: (branch.explicitOpen || parent.explicitOpen) &&
			!branch.explicitClosed && !parent.explicitClosed,
		explicitClosed: branch.explicitClosed || parent.explicitClosed,
		raw:            branch.raw,
	}
	for _, name := range branch.propOrder {
		merged.properties[name] = branch.properties[name]
		merged.propOrder = append(merged.propOrder, name)
	}
	for _, name := range parent.propOrder {
		if _, dup := merged.properties[name]; dup {
			continue
		}
		merged.properties[name] = parent.properties[name]
		merged.propOrder = append(merged.propOrder, name)
	}
	for name := range branch.required {
		merged.required[name] = true
	}
	return merged
}

// artifactTypeConst extracts the discriminating const of a content block's
// declared typeField property ("const: image" or a single-value enum), or
// "". It derives the composed facts rather than reading the raw map, so a
// type refined through allOf (including the composition wrapper duplicate
// properties merge into) or carried behind $ref siblings still discriminates.
func (d *cliManifestDecoder) artifactTypeConst(view *cliResolvedSchema, typeField string) string {
	prop, ok := view.properties[typeField]
	if !ok {
		return ""
	}
	facts, err := d.propertyFacts(prop, nil)
	if err != nil {
		return ""
	}
	if facts.hasConst {
		if s, ok := facts.constVal.(string); ok {
			return s
		}
	}
	if len(facts.enum) == 1 {
		if s, ok := facts.enum[0].(string); ok {
			return s
		}
	}
	return ""
}

func cliSortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (d *cliManifestDecoder) linkPreset(cmdKey string, preset *CLICommandPreset, variant *cliResolvedSchema) error {
	facts, found, err := d.lookupProperty(cmdKey, "preset "+preset.Bind.Pointer, preset.Bind.Pointer, variant)
	if err != nil {
		return err
	}
	if !found {
		return nil // explicitly-open schema: warned, value passes through unvalidated
	}
	if facts.readOnly {
		return fmt.Errorf("command %q preset %s targets a readOnly property; the server would reject or ignore it", cmdKey, preset.Bind.Pointer)
	}

	value, err := d.checkPresetValue(cmdKey, preset.Bind.Pointer, preset.Value, facts)
	if err != nil {
		return err
	}
	preset.Value = value
	return nil
}

// checkPresetValue validates a preset value against the target property and
// applies the one bounded, top-level coercion in v1: a scalar string whose
// target is unambiguously array<string> promotes to a one-element array.
func (d *cliManifestDecoder) checkPresetValue(cmdKey, pointer string, value any, facts *cliPropertyFacts) (any, error) {
	rawErr := d.checkPresetValueStrict(cmdKey, pointer, value, facts)
	if rawErr == nil {
		return value, nil
	}

	str, isString := value.(string)
	if !isString {
		return nil, rawErr
	}

	if facts.unionArms != nil {
		var promoteTarget *cliPropertyFacts
		promoteCandidates := 0
		for _, arm := range facts.unionArms {
			armFacts, err := d.propertyFacts(arm, nil)
			if err != nil {
				return nil, fmt.Errorf("command %q preset %s: %w", cmdKey, pointer, err)
			}
			if armFacts.arrayOfStr {
				promoteCandidates++
				promoteTarget = armFacts
			}
		}
		if promoteCandidates == 1 {
			promoted := []any{str}
			if err := d.checkPresetValueStrict(cmdKey, pointer, promoted, promoteTarget); err == nil {
				d.warnf("command %q preset %s: scalar %q promoted to a one-element array (target arm is array<string>)", cmdKey, pointer, str)
				return promoted, nil
			}
		}
		return nil, rawErr
	}

	if facts.arrayOfStr {
		promoted := []any{str}
		if err := d.checkPresetValueStrict(cmdKey, pointer, promoted, facts); err != nil {
			return nil, err
		}
		d.warnf("command %q preset %s: scalar %q promoted to a one-element array (target is array<string>)", cmdKey, pointer, str)
		return promoted, nil
	}
	return nil, rawErr
}

// checkPresetValueStrict recursively validates without coercion. In
// particular, array elements cannot trigger scalar-to-array promotion.
func (d *cliManifestDecoder) checkPresetValueStrict(cmdKey, pointer string, value any, facts *cliPropertyFacts) error {
	if facts.unionArms != nil {
		for _, arm := range facts.unionArms {
			armFacts, err := d.propertyFacts(arm, nil)
			if err != nil {
				return fmt.Errorf("command %q preset %s: %w", cmdKey, pointer, err)
			}
			if err := d.checkPresetValueStrict(cmdKey, pointer, value, armFacts); err == nil {
				return nil
			}
		}
		kinds, err := d.unionArmKinds(facts.unionArms)
		if err != nil {
			return fmt.Errorf("command %q preset %s: %w", cmdKey, pointer, err)
		}
		return fmt.Errorf("command %q preset %s: value %v matches no union arm (arms: %s)", cmdKey, pointer, value, cliCandidateList(kinds))
	}

	if len(facts.kinds) == 0 {
		return nil // untyped target: pass through
	}

	switch facts.kinds[0] {
	case "string", "int", "float", "bool":
		if err := cliCheckScalarValue(value, facts.kinds[0]); err != nil {
			return fmt.Errorf("command %q preset %s: %w", cmdKey, pointer, err)
		}
		if err := cliCheckEnumValue(value, facts); err != nil {
			return fmt.Errorf("command %q preset %s: %w", cmdKey, pointer, err)
		}
	case "array":
		list, ok := value.([]any)
		if !ok {
			return fmt.Errorf("command %q preset %s: value %v is not an array (and only array<string> targets promote scalars)", cmdKey, pointer, value)
		}
		if facts.itemsErr != nil {
			return fmt.Errorf("command %q preset %s: array item schema: %w", cmdKey, pointer, facts.itemsErr)
		}
		if facts.items != nil {
			for i, item := range list {
				itemPointer := pointer + "/" + strconv.Itoa(i)
				if err := d.checkPresetValueStrict(cmdKey, itemPointer, item, facts.items); err != nil {
					return err
				}
			}
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return fmt.Errorf("command %q preset %s: value %v is not an object", cmdKey, pointer, value)
		}
	}
	return nil
}

// cliCheckSchemaDefault validates a default lifted from the schema (a raw
// YAML/JSON scalar, so integers may arrive as int, uint64, or an integral
// float64) against the input's resolved kind, and returns it in the same
// representation an explicit manifest default would have. The result is
// emitted into Go source verbatim, so a schema default of the wrong type is a
// validation error here rather than a compile failure in the generated CLI.
func cliCheckSchemaDefault(value any, kind string) (any, error) {
	switch kind {
	case "int":
		switch v := value.(type) {
		case int:
			value = int64(v)
		case uint64:
			if v <= math.MaxInt64 {
				value = int64(v)
			}
		case float64:
			if v == math.Trunc(v) && v >= math.MinInt64 && v < math.MaxInt64 {
				value = int64(v)
			}
		}
	case "float":
		switch v := value.(type) {
		case int:
			value = int64(v)
		case uint64:
			value = float64(v)
		}
	}
	if err := cliCheckScalarValue(value, kind); err != nil {
		return nil, err
	}
	return value, nil
}

func cliCheckScalarValue(value any, kind string) error {
	switch kind {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("value %v is not a string", value)
		}
	case "int":
		if _, ok := value.(int64); !ok {
			return fmt.Errorf("value %v is not an integer", value)
		}
	case "float":
		switch value.(type) {
		case float64, int64:
		default:
			return fmt.Errorf("value %v is not a number", value)
		}
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("value %v is not a boolean", value)
		}
	}
	return nil
}

// cliCheckEnumValue enforces closed enums only. Open enums (annotated to
// accept unknown values) never gate: upstream registries evolve faster than
// specs, and a pinned value must not brick generation.
func cliCheckEnumValue(value any, facts *cliPropertyFacts) error {
	if len(facts.enum) == 0 || facts.openEnum {
		return nil
	}
	for _, allowed := range facts.enum {
		if cliValuesEqual(value, allowed) {
			return nil
		}
	}
	return fmt.Errorf("value %v is not in the closed enum (%s)", value, cliCandidateList(cliEnumStrings(facts.enum)))
}

// cliCheckSuggestions validates a manifest suggestions: list against the
// input type. The list is display-only, so the schema enum does not bound it:
// the server owns the value space.
func cliCheckSuggestions(values []any, kind string) error {
	for _, value := range values {
		if err := cliCheckScalarValue(value, kind); err != nil {
			return err
		}
	}
	return nil
}

// cliIsPresetDefault reports whether a preset value can stand as the
// displayed default of a flag of the given type: a scalar of that type.
func cliIsPresetDefault(value any, kind string) bool {
	switch value.(type) {
	case string, bool, int64, float64:
		return cliCheckScalarValue(value, kind) == nil
	}
	return false
}

func cliEnumStrings(enum []any) []string {
	out := make([]string, 0, len(enum))
	for _, v := range enum {
		out = append(out, fmt.Sprintf("%v", v))
	}
	return out
}

// checkSatisfiability warns when a required schema field is reachable only
// via --body. This is a diagnostic, not an error: --body/stdin are universal
// escape hatches and a command may legitimately document itself as body-only.
func (d *cliManifestDecoder) checkSatisfiability(cmdKey string, cmd *CLICommand, variant *cliResolvedSchema, presetPointers map[string]bool) {
	if len(cmd.Args) == 0 && len(cmd.Flags) == 0 {
		return // pure alias/preset command: the generated flags surface applies
	}
	bound := map[string]bool{}
	for _, list := range [][]CLICommandInput{cmd.Args, cmd.Flags} {
		for _, input := range list {
			if input.Bind != nil && input.Bind.In == "body" {
				bound[input.Bind.Pointer] = true
			}
		}
	}
	names := make([]string, 0, len(variant.required))
	for name := range variant.required {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		pointer := "/" + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
		if bound[pointer] || presetPointers[pointer] {
			continue
		}
		if propSchema, ok := variant.properties[name]; ok {
			if facts, err := d.propertyFacts(propSchema, nil); err == nil && facts.hasDefault {
				continue
			}
		}
		d.warnf("command %q: required field %q is not covered by an arg, flag, preset, or schema default; callers must supply it via --body", cmdKey, name)
	}
}

// cliFirstSentence extracts the first sentence of a schema description for
// inferred input summaries.
func cliFirstSentence(description string) string {
	trimmed := strings.TrimSpace(description)
	if trimmed == "" {
		return ""
	}
	if idx := strings.Index(trimmed, ". "); idx != -1 {
		return trimmed[:idx+1]
	}
	if idx := strings.Index(trimmed, ".\n"); idx != -1 {
		return trimmed[:idx+1]
	}
	return trimmed
}

func cliIsScalarKind(kind string) bool {
	switch kind {
	case "string", "int", "float", "bool":
		return true
	}
	return false
}
