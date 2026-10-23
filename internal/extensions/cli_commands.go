package extensions

import (
	"context"
	stderrors "errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/speakeasy-api/openapi-generation/v2/internal/document"
	"github.com/speakeasy-api/openapi-generation/v2/pkg/errors"
	"github.com/speakeasy-api/openapi-generation/v2/pkg/logging"
	"gopkg.in/yaml.v3"
)

// x-speakeasy-cli-commands declares a curated, tier-1 CLI command surface in
// the OpenAPI document. Version 1 is the first shipped version of the
// extension: commands are a map keyed by command name, each command names
// exactly one source (an operation with an optional request variant, a
// planned placeholder, a group tag, or the long-form source object), and
// inputs bind into the request body via singular JSONPaths that are
// normalized to RFC 6901 pointers in the decoded IR.
//
// The decode is deliberately strict: every construct the renderer would have
// to silently ignore is a decode error that names the missing capability, so
// authored configuration can never be silently dropped.

// CLICommandBind describes where a CLI input value lands in the request.
// v1 renders body bindings addressed by RFC 6901 JSON Pointer.
type CLICommandBind struct {
	In      string `json:"in" yaml:"in"`           // "body" (v1)
	Pointer string `json:"pointer" yaml:"pointer"` // e.g. "/input"
	Mode    string `json:"mode" yaml:"mode"`       // "set" (v1)
}

// CLICommandInput is a positional argument or flag on a declared command.
type CLICommandInput struct {
	ID          string          `json:"id" yaml:"id"`
	Name        string          `json:"name" yaml:"name"`
	Summary     string          `json:"summary" yaml:"summary"`
	Type        string          `json:"type" yaml:"type"` // string | int | float | bool
	Required    bool            `json:"required" yaml:"required"`
	Variadic    bool            `json:"variadic" yaml:"variadic"`
	Shorthand   string          `json:"shorthand" yaml:"shorthand"`
	Default     any             `json:"default" yaml:"default"`
	DefaultFrom string          `json:"defaultFrom" yaml:"defaultFrom"` // "schema" | "preset"
	Bind        *CLICommandBind `json:"bind" yaml:"bind"`
	// Enum carries value suggestions only: the manifest's suggestions: list
	// when declared, else the schema enum. They are never enforced locally:
	// upstream registries evolve faster than specs, so enum drift must not
	// brick otherwise-valid invocations.
	Enum []any `json:"enum,omitempty" yaml:"enum,omitempty"`
	// SuggestionsDeclared records that Enum came from the manifest, so help
	// shows the whole authored list instead of the capped schema sample.
	SuggestionsDeclared bool `json:"suggestionsDeclared,omitempty" yaml:"suggestionsDeclared,omitempty"`
	// RouteIDs lists the dispatch routes whose request variant declares this
	// input's bound property. RequiredRouteIDs is the subset on which the
	// property remains required after schema defaults and effective presets.
	// Both are nil for ordinary single-route commands.
	RouteIDs         []string `json:"routeIds,omitempty" yaml:"routeIds,omitempty"`
	RequiredRouteIDs []string `json:"requiredRouteIds,omitempty" yaml:"requiredRouteIds,omitempty"`

	// requiredExplicit records whether the author set required: themselves,
	// so schema-required inference never overrides an explicit decision.
	requiredExplicit bool
}

// CLICommandPreset is a fixed request value applied before user inputs.
type CLICommandPreset struct {
	Bind  CLICommandBind `json:"bind" yaml:"bind"`
	Value any            `json:"value" yaml:"value"`
}

// CLICommandRoute binds a command to an operation.
type CLICommandRoute struct {
	ID             string `json:"id,omitempty" yaml:"id,omitempty"`
	Label          string `json:"label,omitempty" yaml:"label,omitempty"`
	Default        bool   `json:"default,omitempty" yaml:"default,omitempty"`
	OperationID    string `json:"operationId" yaml:"operationId"`
	RequestVariant string `json:"requestVariant" yaml:"requestVariant"`
	// Selector is the declared or inferred flag that identifies this route.
	Selector string `json:"selector,omitempty" yaml:"selector,omitempty"`
	// Presets contains route-local presets after decoding and the effective
	// command-then-route preset set after linking.
	Presets []CLICommandPreset `json:"presets,omitempty" yaml:"presets,omitempty"`

	// Selectors describes how the pinned request variant is told apart from
	// the other members of a union request body. The generated runtime uses
	// it when a caller supplies a partial body (--body/--body-param) to an
	// intent command: presets fill the gaps of a body that stays inside the
	// pinned variant, and a body that names another variant's selector is a
	// usage error instead of a silently re-targeted request. Nil when the
	// request body is not a union.
	Selectors *CLIVariantSelectors `json:"selectors,omitempty" yaml:"selectors,omitempty"`

	// PresetMergePoints lists the JSON pointers at which an object-valued
	// preset fills the missing keys of a caller's object. A caller's object
	// at any other pointer is kept whole.
	PresetMergePoints []CLIPresetMergePoint `json:"presetMergePoints,omitempty" yaml:"presetMergePoints,omitempty"`
}

// CLIPresetMergePoint is a pointer at which a preset object fills a caller's
// object. On a union, Key is the discriminator the preset sets and Values
// every value selecting the preset's member: a caller's object setting Key to
// any other value is kept whole.
type CLIPresetMergePoint struct {
	Pointer string `json:"pointer" yaml:"pointer"`
	Key     string `json:"key,omitempty" yaml:"key,omitempty"`
	Values  []any  `json:"values,omitempty" yaml:"values,omitempty"`
}

// CLICommandDispatchKey records body-key membership across the complete
// request union. Empty RouteIDs means that only unrouted variants declare the
// key, which lets the runtime return a typed escape-path error rather than
// silently selecting a routed variant.
type CLICommandDispatchKey struct {
	Pointer          string   `json:"pointer" yaml:"pointer"`
	RouteIDs         []string `json:"routeIds,omitempty" yaml:"routeIds,omitempty"`
	UnroutedVariants []string `json:"unroutedVariants,omitempty" yaml:"unroutedVariants,omitempty"`
}

// CLIVariantSelectors is the generation-time knowledge needed to keep a
// partial user body inside a pinned union variant.
type CLIVariantSelectors struct {
	// Own lists distinguishing keys (required or defaulted in some members,
	// absent from at least one) that the pinned variant declares.
	Own []string `json:"own,omitempty" yaml:"own,omitempty"`
	// Foreign lists distinguishing keys the pinned variant does not declare:
	// their presence in a body selects a different variant.
	Foreign []string `json:"foreign,omitempty" yaml:"foreign,omitempty"`
	// DiscriminatorKey/DiscriminatorValue name the pinned variant's
	// discriminator when the union declares one; DiscriminatorValue is the
	// value filled into a body that omits it (explicit mapping entry, then
	// the pinned schema's const/single-enum property, then the implicit
	// mapping — the component name), DiscriminatorAliases every value that
	// selects the pinned variant (mapping aliases included).
	DiscriminatorKey     string `json:"discriminatorKey,omitempty" yaml:"discriminatorKey,omitempty"`
	DiscriminatorValue   any    `json:"discriminatorValue,omitempty" yaml:"discriminatorValue,omitempty"`
	DiscriminatorAliases []any  `json:"discriminatorAliases,omitempty" yaml:"discriminatorAliases,omitempty"`
}

// CLICommandSourceType discriminates what a declared command does.
type CLICommandSourceType string

const (
	CLICommandSourceOperation CLICommandSourceType = "operation"
	CLICommandSourceGroup     CLICommandSourceType = "group"
	CLICommandSourcePlanned   CLICommandSourceType = "planned"
	CLICommandSourceCustom    CLICommandSourceType = "custom"
)

// CLICommandSource discriminates what a declared command does.
type CLICommandSource struct {
	Type   CLICommandSourceType `json:"type" yaml:"type"`
	Routes []CLICommandRoute    `json:"routes" yaml:"routes"`
	Note   string               `json:"note" yaml:"note"` // planned: message explaining availability
}

// CLICommandProjection is the default output projection for a command.
type CLICommandProjection struct {
	JQ     string `json:"jq" yaml:"jq"`
	Format string `json:"format" yaml:"format"`
}

// CLICommandArtifactSegment is one step of an artifact content-pointer walk:
// either a named object field or a [*] fan-out over every array item.
type CLICommandArtifactSegment struct {
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
	Wild  bool   `json:"wild,omitempty" yaml:"wild,omitempty"`
}

// CLICommandArtifact declares that the command's semantic result is a media
// file: the renderer generates --out/--raw-response flags and a runtime that
// extracts the last content block of the declared kind (document order),
// writes it to disk, and reports the path. The content-block shape is
// declaration-driven: block: binds the member names the runtime reads
// (discriminating kind, base64 payload, MIME type, URI), and identity: binds
// the response-root members used to enrich the reported envelope and
// not-ready diagnostics.
// Both default to the v1 shape ({type, data, mime_type, uri} blocks and
// {id, status} roots with terminal status "completed") so declarations that
// spell nothing keep their existing linked shape and runtime behavior.
// v1 handles inline base64 content only; URI-delivered content is a targeted
// runtime error naming the missing download capability.
type CLICommandArtifact struct {
	ContentPointer string                      `json:"contentPointer" yaml:"contentPointer"`
	Segments       []CLICommandArtifactSegment `json:"segments" yaml:"segments"`
	Kind           string                      `json:"kind" yaml:"kind"` // image | audio | video
	DefaultPath    string                      `json:"defaultPath" yaml:"defaultPath"`
	// Content-block field bindings (block:). Defaults are the v1 shape.
	TypeField     string `json:"typeField" yaml:"typeField"`         // default "type"
	DataField     string `json:"dataField" yaml:"dataField"`         // default "data"
	MimeTypeField string `json:"mimeTypeField" yaml:"mimeTypeField"` // default "mime_type"
	URIField      string `json:"uriField" yaml:"uriField"`           // default "uri"
	// Response-root identity bindings (identity:). Defaults are the v1 shape.
	IDField        string `json:"idField" yaml:"idField"`               // default "id"
	StatusField    string `json:"statusField" yaml:"statusField"`       // default "status"
	TerminalStatus string `json:"terminalStatus" yaml:"terminalStatus"` // default "completed"
	// ResponseCode is the 2xx status code whose application/json schema the
	// content pointer was validated against (set during linking).
	ResponseCode string `json:"responseCode" yaml:"responseCode"`

	// blockExplicit/identityExplicit record whether the author declared the
	// group. A declared group opts into strict linking of every binding in it
	// (a defaulted group keeps the v1 best-effort runtime reads unchanged, so
	// existing declarations cannot start failing generation).
	blockExplicit    bool
	identityExplicit bool
	// explicitBindings records the individual binding keys the author wrote,
	// so diagnostics never claim a defaulted binding was declared.
	explicitBindings map[string]bool
}

type CLICommandStreamMetadata struct {
	Select   []string `json:"select" yaml:"select"`
	Pointers []string `json:"pointers" yaml:"pointers"`
	Label    string   `json:"label" yaml:"label"`
	Complete []string `json:"complete,omitempty" yaml:"complete,omitempty"`
	When     string   `json:"when,omitempty" yaml:"when,omitempty"`
}

// CLICommandStreamProjection selects the field of each streamed event whose
// string value is written raw to stdout as the event arrives (stream mode
// only). Select is the authored singular JSONPath from the event root as the
// CLI sees each event (the same root a per-event --jq filter sees); Pointer
// is its RFC 6901 lowering, which the generated runtime evaluates.
type CLICommandStreamProjection struct {
	Metadata []CLICommandStreamMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Select   string                     `json:"select" yaml:"select"`
	Pointer  string                     `json:"pointer" yaml:"pointer"`
}

// CLICommandOutput groups output behavior for a declared command.
type CLICommandOutput struct {
	Projection *CLICommandProjection       `json:"projection" yaml:"projection"`
	Artifact   *CLICommandArtifact         `json:"artifact,omitempty" yaml:"artifact,omitempty"`
	Stream     *CLICommandStreamProjection `json:"stream,omitempty" yaml:"stream,omitempty"`
	Outcome    *CLICommandOutcome          `json:"outcome,omitempty" yaml:"outcome,omitempty"`
}

// CLICommandOutcome classifies the terminal state carried inside a
// successful response so the generated command can fail (exit 1) when the
// API reports failure with an HTTP success code. StatePointer addresses the
// state in the discrete JSON response and StreamStatePointer the same state
// inside streamed events; at least one is declared. States is keyed by the
// API's enum value, exactly as async.states, and only the "failure"
// classification changes runtime behavior. ErrorPointer optionally addresses
// the failure detail (an object or an array of objects) whose
// ErrorMessageField string is appended to the failure message. A
// single-route intent without its own table inherits its operation entry's.
type CLICommandOutcome struct {
	StateFrom          string            `json:"stateFrom,omitempty" yaml:"stateFrom,omitempty"`
	StatePointer       string            `json:"statePointer,omitempty" yaml:"statePointer,omitempty"`
	StreamStateFrom    string            `json:"streamStateFrom,omitempty" yaml:"streamStateFrom,omitempty"`
	StreamStatePointer string            `json:"streamStatePointer,omitempty" yaml:"streamStatePointer,omitempty"`
	States             map[string]string `json:"states" yaml:"states"`
	ErrorFrom          string            `json:"errorFrom,omitempty" yaml:"errorFrom,omitempty"`
	ErrorPointer       string            `json:"errorPointer,omitempty" yaml:"errorPointer,omitempty"`
	StreamErrorFrom    string            `json:"streamErrorFrom,omitempty" yaml:"streamErrorFrom,omitempty"`
	StreamErrorPointer string            `json:"streamErrorPointer,omitempty" yaml:"streamErrorPointer,omitempty"`
	ErrorMessageField  string            `json:"errorMessageField,omitempty" yaml:"errorMessageField,omitempty"`
}

// CLIOperation augments the generated command for one operation. Unlike an
// intent it does not create a command or choose a request variant: it adds an
// opt-in streamed projection and, when necessary, scalar flags that merge into
// a body property shared by every request-body union member.
type CLIOperation struct {
	OperationID  string                  `json:"operationId" yaml:"operationId"`
	Output       *CLICommandOutput       `json:"output,omitempty" yaml:"output,omitempty"`
	Flags        []CLICommandInput       `json:"flags,omitempty" yaml:"flags,omitempty"`
	Continuation *CLICommandContinuation `json:"continuation,omitempty" yaml:"continuation,omitempty"`
	BodyRequired bool                    `json:"bodyRequired,omitempty" yaml:"bodyRequired,omitempty"`
}

// CLICommandContinuationBinding copies one resume value from the response
// that stopped into the next request. From addresses the value in a discrete
// JSON response and StreamFrom in streamed events (the last present value
// wins); To is the request-body path that receives it. An optional binding
// is skipped when the value is absent, leaving the original request value.
// Flag is the command's declared flag bound to the same body path, resolved
// during linking for resume hints.
type CLICommandContinuationBinding struct {
	From           string   `json:"from,omitempty" yaml:"from,omitempty"`
	Pointer        string   `json:"pointer,omitempty" yaml:"pointer,omitempty"`
	StreamFrom     []string `json:"streamFrom,omitempty" yaml:"streamFrom,omitempty"`
	StreamPointers []string `json:"streamPointers,omitempty" yaml:"streamPointers,omitempty"`
	To             string   `json:"to" yaml:"to"`
	ToPointer      string   `json:"toPointer" yaml:"toPointer"`
	ToPath         []string `json:"toPath" yaml:"toPath"`
	Optional       bool     `json:"optional,omitempty" yaml:"optional,omitempty"`
	Flag           string   `json:"flag,omitempty" yaml:"flag,omitempty"`
}

// CLICommandContinuation re-sends a command's request while the response's
// output.outcome state classifies as continue, filling Bind values from the
// response that stopped. Merge paths are arrays concatenated across hops and
// Sum paths numbers summed across hops; every other field of a discrete
// result comes from the last hop.
type CLICommandContinuation struct {
	Bind        []CLICommandContinuationBinding `json:"bind" yaml:"bind"`
	MergeFrom   []string                        `json:"mergeFrom,omitempty" yaml:"mergeFrom,omitempty"`
	Merge       []string                        `json:"merge,omitempty" yaml:"merge,omitempty"`
	SumFrom     []string                        `json:"sumFrom,omitempty" yaml:"sumFrom,omitempty"`
	Sum         []string                        `json:"sum,omitempty" yaml:"sum,omitempty"`
	MaxHops     int                             `json:"maxHops" yaml:"maxHops"`
	Default     bool                            `json:"default" yaml:"default"`
	OperationID string                          `json:"operationId,omitempty" yaml:"operationId,omitempty"`
}

// CLICommandAsyncParameter identifies the poll-operation parameter that
// receives the handle returned by the create operation.
type CLICommandAsyncParameter struct {
	In   string `json:"in" yaml:"in"` // path | query
	Name string `json:"name" yaml:"name"`
}

// CLICommandAsyncPathSegment preserves the unambiguous shape of an authored
// singular JSONPath for downstream renderers. Pointer remains the runtime form.
type CLICommandAsyncPathSegment struct {
	Field   string `json:"field,omitempty" yaml:"field,omitempty"`
	Index   int    `json:"index,omitempty" yaml:"index,omitempty"`
	IsIndex bool   `json:"isIndex,omitempty" yaml:"isIndex,omitempty"`
}

// CLICommandAsyncResolvedParameter is a poll-operation parameter pinned by
// async.params after its location and schema have been linked.
type CLICommandAsyncResolvedParameter struct {
	In    string `json:"in" yaml:"in"` // query | header
	Name  string `json:"name" yaml:"name"`
	Value any    `json:"value" yaml:"value"`
}

// CLICommandAsyncID connects the create response to the poll request. From is
// retained for diagnostics; Pointer is its RFC 6901 lowering for the runtime.
type CLICommandAsyncID struct {
	From     string                       `json:"from" yaml:"from"`
	Pointer  string                       `json:"pointer" yaml:"pointer"`
	Segments []CLICommandAsyncPathSegment `json:"segments" yaml:"segments"`
	To       CLICommandAsyncParameter     `json:"to" yaml:"to"`
}

// CLICommandAsync declares a foreground polling recipe. States is keyed by
// the API's enum value so overlays replace classifications instead of
// appending duplicate set members. The failure-detail shape is
// declaration-driven: ErrorField binds the poll-response root member holding
// the failure detail object and ErrorMessageField the string member inside
// it that carries the human message. Both default to the v1 shape
// ("error"/"message"), and only declaring either opts the pair into strict
// linking against the poll response schema (a defaulted pair keeps the v1
// best-effort runtime reads, so existing recipes cannot start failing
// generation).
type CLICommandAsync struct {
	OperationID        string                             `json:"operationId" yaml:"operationId"`
	ID                 CLICommandAsyncID                  `json:"id" yaml:"id"`
	Params             map[string]any                     `json:"params,omitempty" yaml:"params,omitempty"`
	ResolvedParams     []CLICommandAsyncResolvedParameter `json:"resolvedParams,omitempty" yaml:"resolvedParams,omitempty"`
	StateFrom          string                             `json:"stateFrom" yaml:"stateFrom"`
	StatePointer       string                             `json:"statePointer" yaml:"statePointer"`
	StateSegments      []CLICommandAsyncPathSegment       `json:"stateSegments" yaml:"stateSegments"`
	States             map[string]string                  `json:"states" yaml:"states"`
	Interval           string                             `json:"interval" yaml:"interval"`
	Backoff            float64                            `json:"backoff" yaml:"backoff"`
	MaxInterval        string                             `json:"maxInterval" yaml:"maxInterval"`
	Timeout            string                             `json:"timeout" yaml:"timeout"`
	ErrorField         string                             `json:"errorField" yaml:"errorField"`               // default "error"
	ErrorMessageField  string                             `json:"errorMessageField" yaml:"errorMessageField"` // default "message"
	CreateResponseCode string                             `json:"createResponseCode" yaml:"createResponseCode"`
	ResponseCode       string                             `json:"responseCode" yaml:"responseCode"`

	// errorExplicit records whether the author declared either failure
	// binding; only a declared pair is linked (see linkAsyncErrorBindings).
	errorExplicit bool
}

// CLICommandExample is a labeled runnable invocation. The authoring-side slug
// (the map key, used as overlay-patchable identity) is dropped at
// normalization; document order is preserved.
type CLICommandExample struct {
	Summary string `json:"summary" yaml:"summary"`
	Command string `json:"command" yaml:"command"`
}

// CLICommandHelp carries concise human-facing help prose. Values are the text
// after their respective labels and are deliberately limited to one line so a
// manifest value cannot inject or impersonate another help section.
type CLICommandHelp struct {
	Defaults string `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Learn    string `json:"learn,omitempty" yaml:"learn,omitempty"`
	Escalate string `json:"escalate,omitempty" yaml:"escalate,omitempty"`
}

type CLICatalogDefault struct {
	CatalogCommand string `json:"catalogCommand" yaml:"catalogCommand"`
	Value          string `json:"value" yaml:"value"`
	Label          string `json:"label" yaml:"label"`
}

// CLICommand is one declared intent command.
type CLICommand struct {
	ID              string                  `json:"id" yaml:"id"`
	Path            []string                `json:"path" yaml:"path"`
	Category        string                  `json:"category" yaml:"category"`
	Summary         string                  `json:"summary" yaml:"summary"`
	Tagline         string                  `json:"tagline" yaml:"tagline"`
	Description     string                  `json:"description" yaml:"description"`
	Source          CLICommandSource        `json:"source" yaml:"source"`
	Args            []CLICommandInput       `json:"args" yaml:"args"`
	Flags           []CLICommandInput       `json:"flags" yaml:"flags"`
	Presets         []CLICommandPreset      `json:"presets" yaml:"presets"`
	CatalogDefaults []CLICatalogDefault     `json:"catalogDefaults,omitempty" yaml:"catalogDefaults,omitempty"`
	Async           *CLICommandAsync        `json:"async,omitempty" yaml:"async,omitempty"`
	Continuation    *CLICommandContinuation `json:"continuation,omitempty" yaml:"continuation,omitempty"`
	Output          *CLICommandOutput       `json:"output" yaml:"output"`
	Examples        []CLICommandExample     `json:"examples" yaml:"examples"`
	Help            *CLICommandHelp         `json:"help,omitempty" yaml:"help,omitempty"`
	Override        bool                    `json:"override,omitempty" yaml:"override,omitempty"`
	DispatchKeys    []CLICommandDispatchKey `json:"dispatchKeys,omitempty" yaml:"dispatchKeys,omitempty"`
	// Hints maps an error reason to agent-mode hint lines that are merged
	// into the reason-first error envelope. CLI_* reasons are the closed
	// namespace the generated runtime itself produces; external (server)
	// reasons pass through verbatim.
	Hints map[string][]string `json:"hints,omitempty" yaml:"hints,omitempty"`
}

// CLICommandManifest is the decoded x-speakeasy-cli-commands document
// extension. Commands preserve document order; Categories preserve the
// declared help-section order.
type CLICommandManifest struct {
	Version    int            `json:"version" yaml:"version"`
	Categories []string       `json:"categories,omitempty" yaml:"categories,omitempty"`
	Commands   []CLICommand   `json:"commands" yaml:"commands"`
	Operations []CLIOperation `json:"operations,omitempty" yaml:"operations,omitempty"`
}

var (
	cliCommandKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*( [a-z][a-z0-9-]*)*$`)
	cliInputNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	cliExampleSlugRegexp = regexp.MustCompile(`^[a-z0-9-]+$`)
	cliShorthandPattern  = regexp.MustCompile(`^[a-zA-Z0-9]$`)
	cliHintReasonPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
	cliCategoryIDPattern = regexp.MustCompile(`[^a-z0-9]+`)
)

// cliCategoryID mirrors intentCategoryID in
// templates/templates/cli/includes/intents.ts, which derives cobra group IDs
// from category titles. The two must normalize identically so the decoder
// rejects exactly the titles that would collide (or vanish) at render time.
func cliCategoryID(category string) string {
	return strings.Trim(cliCategoryIDPattern.ReplaceAllString(strings.ToLower(category), "-"), "-")
}

// cliReservedFlagNames are flag names owned by the generated runtime; a
// declared flag may not shadow them.
var cliReservedFlagNames = map[string]bool{
	"body":       true,
	"body-param": true,
	"schema":     true,
	"usage":      true,
	"jq":         true,
	"output":     true,
	"help":       true,
	// Static persistent flags are registered in
	// templates/templates/cli/root.go.stmpl, the source of truth.
	"output-format":   true,
	"color":           true,
	"raw-output":      true,
	"server-url":      true,
	"server":          true,
	"header":          true,
	"include-headers": true,
	"timeout":         true,
	"no-interactive":  true,
	// Registered when interactive mode or interactive auth is enabled;
	// reserved unconditionally so a manifest stays valid across gen.yaml
	// interactivity changes.
	"interactive":             true,
	"dry-run":                 true,
	"debug":                   true,
	"agent-mode":              true,
	"no-retries":              true,
	"retry-max-elapsed-time":  true,
	"retry-connection-errors": true,
	"retry-config":            true,
	// Owned by the artifact runtime (registered on commands that declare
	// output.artifact); reserved unconditionally so a manifest stays valid
	// when an artifact declaration is added later.
	"out":          true,
	"raw-response": true,
	// Owned by async intent commands. It is reserved unconditionally so
	// adding an async recipe cannot turn a previously valid flag into a
	// generated pflag collision.
	"async":         true,
	"resume":        true,
	"poll-interval": true,
	"poll-timeout":  true,
	// Owned by continuation commands, reserved for the same reason.
	"auto-continue":     true,
	"max-continuations": true,
}

// cliReservedFlagShorthands are owned by the generated runtime's persistent
// flags. The source of truth is templates/templates/cli/root.go.stmpl; a
// declared flag shadowing one would panic pflag when inherited flags merge.
// Cobra also owns -h for help even though it is not registered in the root
// template.
var cliReservedFlagShorthands = map[string]string{
	"o": "output-format",
	"q": "jq",
	"H": "header",
	"d": "debug",
	"h": "help",
}

// cliArtifactKinds is the closed set of artifact kinds: it drives the
// content-type filter and the fallback file extension when the response
// carries no usable MIME type.
var cliArtifactKinds = []string{"image", "audio", "video"}

// cliArtifactPlaceholders are the placeholders a defaultPath pattern may use.
var cliArtifactPlaceholders = map[string]bool{
	"timestamp": true,
	"rand":      true,
	"ext":       true,
}

// The v1 artifact shape, kept as the schema default for every block/identity
// binding: annotations omit fields carrying these values (the generated
// runtime re-applies them on read), so a declaration that spells no block: or
// identity: renders byte-identical output.
const (
	cliArtifactDefaultTypeField      = "type"
	cliArtifactDefaultDataField      = "data"
	cliArtifactDefaultMimeTypeField  = "mime_type"
	cliArtifactDefaultURIField       = "uri"
	cliArtifactDefaultIDField        = "id"
	cliArtifactDefaultStatusField    = "status"
	cliArtifactDefaultTerminalStatus = "completed"
)

// The v1 async failure-detail shape, kept as the schema default for both
// error bindings so undeclared recipes keep their bytes and behavior.
const (
	cliAsyncDefaultErrorField        = "error"
	cliAsyncDefaultErrorMessageField = "message"
)

var (
	cliArtifactKeys         = []string{"contentPointer", "kind", "defaultPath", "block", "identity"}
	cliArtifactBlockKeys    = []string{"typeField", "dataField", "mimeTypeField", "uriField"}
	cliArtifactIdentityKeys = []string{"idField", "statusField", "terminalStatus"}
)

// CLIRuntimeHintReasons is the closed namespace of CLI_*-prefixed error
// reasons the generated runtime can emit; hints keyed by any other CLI_*
// reason would never fire and are therefore decode errors. The generated
// agent-mode error envelope mirrors this list.
var CLIRuntimeHintReasons = []string{
	"CLI_VALIDATION",
	"CLI_CONNECTION",
	"CLI_PROTOCOL",
	"CLI_RUNTIME",
	"CLI_UNAVAILABLE",
	"CLI_AUTHENTICATION",
	"CLI_ASYNC_FAILED",
	"CLI_ASYNC_TIMEOUT",
	"CLI_ASYNC_UNKNOWN_STATE",
	"CLI_OUTCOME_FAILED",
	"CLI_CONTINUATION_LIMIT",
}

// cliReservedCommandKeys are keys reserved for future capabilities. Each maps
// to the capability an author is asking for, so the error can name it instead
// of pretending the key is a typo.
var cliReservedCommandKeys = map[string]string{
	"payload": "payload templates require the request-plan.payload-holes capability, which is not part of v1; binds deeper than one level beneath an object preset stay hard errors until it ships",
	"via":     "via is reserved for a future routing capability and is not part of v1",
	"pos":     "pos is reserved for a future positional-layout capability and is not part of v1",
	"format":  "format is reserved for a future output-format capability and is not part of v1; use jq for projections",
}

var cliCommandKeys = []string{
	"category", "summary", "tagline", "description",
	"op", "routes", "planned", "custom", "group", "source", "override",
	"args", "flags", "preset", "hints", "examples", "help", "jq", "output", "async", "continuation",
}

var cliOutputKeys = []string{"artifact", "stream", "outcome"}

var cliOutcomeKeys = []string{"statePointer", "streamStatePointer", "states", "errorPointer", "streamErrorPointer", "errorMessageField"}

var cliOutputStreamKeys = []string{"select", "metadata"}

var cliOutputStreamMetadataKeys = []string{"select", "label", "complete", "when"}

var cliOperationKeys = []string{"output", "flags", "continuation"}

var (
	cliContinuationKeys        = []string{"bind", "merge", "sum", "maxHops", "default"}
	cliContinuationBindingKeys = []string{"from", "streamFrom", "to", "optional"}
	// cliOutcomeStateKind extends the async classifications with continue,
	// which only a continuation recipe acts on; async polling has no resume
	// value to send.
	cliOutcomeStateKind = []string{"pending", "success", "failure", "handoff", "continue"}
)

const cliContinuationDefaultMaxHops = 32

// These flags are conditionally registered by operation commands. Operation
// declarations reserve them even when a particular operation does not use the
// corresponding feature, so adding pagination or binary output later cannot
// turn a valid manifest into a pflag registration panic.
var cliOperationReservedFlagNames = map[string]string{
	"all":               "pagination",
	"max-pages":         "pagination",
	"output-file":       "binary response output",
	"output-b64":        "binary response output",
	"auto-continue":     "continuation",
	"max-continuations": "continuation",
}

var cliInputKeys = []string{
	"to", "type", "required", "variadic", "shorthand", "summary", "default", "defaultFrom", "suggestions",
}

var (
	cliAsyncKeys      = []string{"op", "id", "params", "statePointer", "states", "interval", "backoff", "maxInterval", "timeout", "errorField", "errorMessageField"}
	cliAsyncIDKeys    = []string{"from", "to"}
	cliAsyncIDToKeys  = []string{"in", "name"}
	cliAsyncStateKind = []string{"pending", "success", "failure", "handoff"}
)

// HandleCLICommandsExtension decodes the document-level
// x-speakeasy-cli-commands extension. Returns nil when the extension is
// absent. Warnings are reported through the generation logger; errors abort
// generation with the offending node attached.
func (e *Extensions) HandleCLICommandsExtension(ctx context.Context, docInfo *document.DocumentInfo) (*CLICommandManifest, error) {
	if docInfo == nil || docInfo.Doc == nil {
		return nil, nil
	}
	exts := docInfo.Doc.GetExtensions()
	if exts.Len() == 0 {
		return nil, nil
	}
	node, ok := e.findExtension(exts, ExtCLICommands)
	if !ok {
		return nil, nil
	}

	manifest, warnings, err := DecodeCLICommandsManifest(ctx, docInfo, node)
	for _, w := range warnings {
		logging.LogWarning(ctx, ExtCLICommands.Name(), fmt.Errorf("%s", w))
	}
	if err != nil {
		return nil, errors.NewValidationError(ExtCLICommands.Name()+": "+err.Error(), node, nil)
	}
	return manifest, nil
}

// DecodeCLICommandsManifest performs the full strict decode, schema link, and
// lowering of the extension node. It is exported for focused testing; use
// HandleCLICommandsExtension during generation.
func DecodeCLICommandsManifest(ctx context.Context, docInfo *document.DocumentInfo, node *yaml.Node) (*CLICommandManifest, []string, error) {
	d := &cliManifestDecoder{ctx: ctx, docInfo: docInfo}

	expanded, err := cliExpandNode(node, &cliExpandState{})
	if err != nil {
		return nil, d.warnings, err
	}

	manifest, err := d.decodeManifest(expanded)
	if err != nil {
		return nil, d.warnings, err
	}
	return manifest, d.warnings, nil
}

type cliManifestDecoder struct {
	ctx      context.Context //nolint:containedctx // decoder is request-scoped; ctx feeds schema resolution logging
	docInfo  *document.DocumentInfo
	warnings []string

	schemaIndex *cliSchemaIndex
}

func (d *cliManifestDecoder) warnf(format string, args ...any) {
	d.warnings = append(d.warnings, fmt.Sprintf(format, args...))
}

func (d *cliManifestDecoder) decodeManifest(node *yaml.Node) (*CLICommandManifest, error) {
	entries, err := cliMapEntries(node, "the extension value")
	if err != nil {
		return nil, err
	}

	manifest := &CLICommandManifest{}
	var commandsNode, operationsNode *yaml.Node
	versionSeen := false

	for _, entry := range entries {
		switch entry.Key.Value {
		case "version":
			versionSeen = true
			if entry.Value.Kind != yaml.ScalarNode || entry.Value.Tag != "!!int" {
				return nil, fmt.Errorf("line %d: version must be the integer 1 (this is the first shipped version of the extension)", entry.Value.Line)
			}
			if entry.Value.Value != "1" {
				return nil, fmt.Errorf("line %d: unsupported version %s (only version 1 exists)", entry.Value.Line, entry.Value.Value)
			}
			manifest.Version = 1
		case "categories":
			if entry.Value.Kind != yaml.SequenceNode {
				return nil, fmt.Errorf("line %d: categories must be a sequence of section names", entry.Value.Line)
			}
			seen := map[string]bool{}
			seenIDs := map[string]string{}
			for _, item := range entry.Value.Content {
				name, err := cliScalarString(item, "category name")
				if err != nil {
					return nil, err
				}
				if name == "" {
					return nil, fmt.Errorf("line %d: category names must be non-empty", item.Line)
				}
				if seen[name] {
					return nil, fmt.Errorf("line %d: duplicate category %q", item.Line, name)
				}
				seen[name] = true
				// The renderer derives cobra group IDs from titles with
				// cliCategoryID's normalization; a collision or an empty ID
				// in that space corrupts --help grouping, so it must fail
				// here rather than render twice (or swallow every ungrouped
				// command under a punctuation-only title).
				id := cliCategoryID(name)
				if id == "" {
					return nil, fmt.Errorf("line %d: category %q normalizes to an empty CLI group ID; use a name containing letters or digits", item.Line, name)
				}
				if previous, ok := seenIDs[id]; ok {
					return nil, fmt.Errorf("line %d: categories %q and %q normalize to the same CLI group ID %q; rename one", item.Line, previous, name, id)
				}
				seenIDs[id] = name
				manifest.Categories = append(manifest.Categories, name)
			}
		case "commands":
			if entry.Value.Kind == yaml.SequenceNode {
				return nil, fmt.Errorf("line %d: commands must be a map keyed by command name; the sequence shape only existed in a pre-release draft and is not supported", entry.Value.Line)
			}
			commandsNode = entry.Value
		case "operations":
			if entry.Value.Kind == yaml.SequenceNode {
				return nil, fmt.Errorf("line %d: operations must be a map keyed by operationId", entry.Value.Line)
			}
			operationsNode = entry.Value
		default:
			return nil, fmt.Errorf("line %d: unknown key %q%s (expected version, categories, commands, operations)", entry.Key.Line, entry.Key.Value, cliDidYouMean(entry.Key.Value, []string{"version", "categories", "commands", "operations"}))
		}
	}

	if !versionSeen {
		return nil, stderrors.New("version is required (the integer 1)")
	}
	if commandsNode == nil && operationsNode == nil {
		return nil, stderrors.New("at least one of commands or operations is required")
	}

	seenIDs := map[string]int{}
	seenCategoryIDs := map[string]string{}
	if commandsNode != nil {
		commandEntries, err := cliMapEntries(commandsNode, "commands")
		if err != nil {
			return nil, err
		}
		if len(commandEntries) == 0 {
			return nil, fmt.Errorf("line %d: commands must declare at least one command", commandsNode.Line)
		}
		for _, entry := range commandEntries {
			key := entry.Key.Value
			if !cliCommandKeyPattern.MatchString(key) {
				return nil, fmt.Errorf("line %d: command key %q is invalid (expected lowercase words of [a-z0-9-] separated by single spaces, e.g. %q)", entry.Key.Line, key, "agent run")
			}
			path := strings.Split(key, " ")
			id := strings.Join(path, "-")
			if prevLine, ok := seenIDs[id]; ok {
				return nil, fmt.Errorf("line %d: command %q derives id %q, which collides with the command on line %d", entry.Key.Line, key, id, prevLine)
			}
			seenIDs[id] = entry.Key.Line

			cmd, err := d.decodeCommand(key, path, id, entry.Value)
			if err != nil {
				return nil, err
			}
			if cmd.Category != "" && len(manifest.Categories) > 0 && !cliContains(manifest.Categories, cmd.Category) {
				return nil, fmt.Errorf("command %q declares category %q, which is not in the categories list (%s)", key, cmd.Category, cliCandidateList(manifest.Categories))
			}
			if cmd.Category != "" && len(manifest.Categories) == 0 {
				// Without a declared list, per-command categories still
				// become cobra group IDs: validate them in the renderer's
				// normalization space like declared ones.
				catID := cliCategoryID(cmd.Category)
				if catID == "" {
					return nil, fmt.Errorf("command %q declares category %q, which normalizes to an empty CLI group ID; use a name containing letters or digits", key, cmd.Category)
				}
				if previous, ok := seenCategoryIDs[catID]; ok && previous != cmd.Category {
					return nil, fmt.Errorf("commands declare categories %q and %q, which normalize to the same CLI group ID %q; rename one", previous, cmd.Category, catID)
				}
				seenCategoryIDs[catID] = cmd.Category
			}
			manifest.Commands = append(manifest.Commands, *cmd)
		}
	}

	if operationsNode != nil {
		operationEntries, err := cliMapEntries(operationsNode, "operations")
		if err != nil {
			return nil, err
		}
		if len(operationEntries) == 0 {
			return nil, fmt.Errorf("line %d: operations must declare at least one operation", operationsNode.Line)
		}
		for _, entry := range operationEntries {
			opID, err := cliScalarString(entry.Key, "operationId")
			if err != nil || strings.TrimSpace(opID) == "" {
				return nil, fmt.Errorf("line %d: operations keys must be non-empty operationIds", entry.Key.Line)
			}
			op, err := d.decodeOperation(opID, entry.Value)
			if err != nil {
				return nil, err
			}
			manifest.Operations = append(manifest.Operations, *op)
		}
	}

	if err := d.checkPathPrefixes(manifest); err != nil {
		return nil, err
	}

	if err := d.linkManifest(manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

func (d *cliManifestDecoder) decodeOperation(opID string, node *yaml.Node) (*CLIOperation, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("operation %q", opID))
	if err != nil {
		return nil, err
	}
	op := &CLIOperation{OperationID: opID}
	for _, entry := range entries {
		switch entry.Key.Value {
		case "output":
			outputEntries, err := cliMapEntries(entry.Value, fmt.Sprintf("operation %q output", opID))
			if err != nil {
				return nil, err
			}
			if len(outputEntries) == 0 {
				return nil, fmt.Errorf("line %d: operation %q output requires stream or outcome", entry.Value.Line, opID)
			}
			for _, outputEntry := range outputEntries {
				switch outputEntry.Key.Value {
				case "stream":
					stream, err := d.decodeStreamProjection("operation "+opID, outputEntry.Value)
					if err != nil {
						return nil, err
					}
					if op.Output == nil {
						op.Output = &CLICommandOutput{}
					}
					op.Output.Stream = stream
				case "outcome":
					outcome, err := d.decodeOutcome("operation "+opID, outputEntry.Value)
					if err != nil {
						return nil, err
					}
					if op.Output == nil {
						op.Output = &CLICommandOutput{}
					}
					op.Output.Outcome = outcome
				default:
					return nil, fmt.Errorf("line %d: operation %q output has unknown key %q%s", outputEntry.Key.Line, opID, outputEntry.Key.Value, cliDidYouMean(outputEntry.Key.Value, []string{"stream", "outcome"}))
				}
			}
		case "flags":
			flags, err := d.decodeInputs("operation "+opID, entry.Value, false)
			if err != nil {
				return nil, err
			}
			for _, flag := range flags {
				if owner, reserved := cliOperationReservedFlagNames[flag.Name]; reserved {
					return nil, fmt.Errorf("operation %q flag %q collides with the generated operation flag for %s", opID, flag.Name, owner)
				}
				if flag.Shorthand == "a" {
					return nil, fmt.Errorf("operation %q flag %q shorthand -a collides with the generated pagination flag --all", opID, flag.Name)
				}
				if flag.Required {
					return nil, fmt.Errorf("operation %q flag %q cannot be required; a whole request body must remain an alternative source", opID, flag.Name)
				}
				if flag.Bind == nil || flag.Bind.In != "body" {
					return nil, fmt.Errorf("operation %q flag %q must bind a request-body property with to: $.field", opID, flag.Name)
				}
			}
			if err := checkOperationInputs(opID, flags); err != nil {
				return nil, err
			}
			op.Flags = flags
		case "continuation":
			if op.Continuation, err = d.decodeContinuation(fmt.Sprintf("operation %q", opID), entry.Value); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("line %d: operation %q has unknown key %q%s", entry.Key.Line, opID, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliOperationKeys))
		}
	}
	if op.Output == nil && len(op.Flags) == 0 && op.Continuation == nil {
		return nil, fmt.Errorf("line %d: operation %q must declare output, flags, or continuation", node.Line, opID)
	}
	return op, nil
}

func checkOperationInputs(opID string, flags []CLICommandInput) error {
	shorthands := map[string]string{}
	pointers := map[string]string{}
	for _, flag := range flags {
		if flag.Shorthand != "" {
			if previous, ok := shorthands[flag.Shorthand]; ok {
				return fmt.Errorf("operation %q: flags %q and %q both use shorthand -%s", opID, previous, flag.Name, flag.Shorthand)
			}
			shorthands[flag.Shorthand] = flag.Name
		}
		if previous, ok := pointers[flag.Bind.Pointer]; ok {
			return fmt.Errorf("operation %q: flags %q and %q both bind %s", opID, previous, flag.Name, flag.Bind.Pointer)
		}
		pointers[flag.Bind.Pointer] = flag.Name
	}
	return nil
}

// checkPathPrefixes enforces the nesting contract: a declared command may nest
// beneath another declared command only when that parent is a group. Parents
// that are not declared at all are resolved by the renderer against the
// generated command tree (and fail generation with a named parent if absent).
func (d *cliManifestDecoder) checkPathPrefixes(manifest *CLICommandManifest) error {
	byKey := map[string]*CLICommand{}
	for i := range manifest.Commands {
		byKey[strings.Join(manifest.Commands[i].Path, " ")] = &manifest.Commands[i]
	}
	for i := range manifest.Commands {
		cmd := &manifest.Commands[i]
		for prefixLen := 1; prefixLen < len(cmd.Path); prefixLen++ {
			prefixKey := strings.Join(cmd.Path[:prefixLen], " ")
			parent, declared := byKey[prefixKey]
			if declared && parent.Source.Type != CLICommandSourceGroup {
				return fmt.Errorf("command %q nests beneath %q, but %q is a %s command; only group commands (or generated command groups) can host nested commands", strings.Join(cmd.Path, " "), prefixKey, prefixKey, parent.Source.Type)
			}
		}
	}
	return nil
}

func (d *cliManifestDecoder) decodeCommand(key string, path []string, id string, node *yaml.Node) (*CLICommand, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q", key))
	if err != nil {
		return nil, err
	}

	cmd := &CLICommand{ID: id, Path: path}
	var (
		opNode, routesNode, plannedNode, customNode, groupNode, sourceNode *yaml.Node
		argsNode, flagsNode, presetNode, hintsNode, exampNode, helpNode    *yaml.Node
		outputNode, asyncNode, overrideNode, continuationNode              *yaml.Node
	)

	for _, entry := range entries {
		keyName := entry.Key.Value
		if capabilityMsg, reserved := cliReservedCommandKeys[keyName]; reserved {
			return nil, fmt.Errorf("line %d: command %q: %s", entry.Key.Line, key, capabilityMsg)
		}
		switch keyName {
		case "category":
			if cmd.Category, err = cliScalarString(entry.Value, "category"); err != nil {
				return nil, err
			}
		case "summary":
			if cmd.Summary, err = cliScalarString(entry.Value, "summary"); err != nil {
				return nil, err
			}
		case "tagline":
			if cmd.Tagline, err = cliScalarString(entry.Value, "tagline"); err != nil {
				return nil, err
			}
		case "description":
			if cmd.Description, err = cliScalarString(entry.Value, "description"); err != nil {
				return nil, err
			}
		case "op":
			opNode = entry.Value
		case "routes":
			routesNode = entry.Value
		case "planned":
			plannedNode = entry.Value
		case "custom":
			customNode = entry.Value
		case "group":
			groupNode = entry.Value
		case "source":
			sourceNode = entry.Value
		case "override":
			overrideNode = entry.Value
			if cmd.Override, err = cliScalarBool(entry.Value, "override"); err != nil {
				return nil, err
			}
		case "args":
			argsNode = entry.Value
		case "flags":
			flagsNode = entry.Value
		case "preset":
			presetNode = entry.Value
		case "hints":
			hintsNode = entry.Value
		case "examples":
			exampNode = entry.Value
		case "help":
			helpNode = entry.Value
		case "jq":
			jq, err := cliScalarString(entry.Value, "jq")
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(jq) == "" {
				return nil, fmt.Errorf("line %d: command %q: jq must be a non-empty expression", entry.Value.Line, key)
			}
			cmd.Output = &CLICommandOutput{Projection: &CLICommandProjection{JQ: jq}}
		case "output":
			outputNode = entry.Value
		case "async":
			asyncNode = entry.Value
		case "continuation":
			continuationNode = entry.Value
		default:
			return nil, fmt.Errorf("line %d: command %q has unknown key %q%s", entry.Key.Line, key, keyName, cliDidYouMean(keyName, cliCommandKeys))
		}
	}

	if outputNode != nil {
		artifact, stream, outcome, err := d.decodeOutput(key, outputNode)
		if err != nil {
			return nil, err
		}
		if cmd.Output == nil {
			cmd.Output = &CLICommandOutput{}
		}
		cmd.Output.Artifact = artifact
		cmd.Output.Stream = stream
		cmd.Output.Outcome = outcome
	}
	if cmd.Output != nil && cmd.Output.Projection != nil && cmd.Output.Artifact != nil {
		return nil, fmt.Errorf("command %q declares both jq and output.artifact; the artifact declaration owns the command's default output (users can still project with --jq at runtime)", key)
	}
	if cmd.Output != nil && cmd.Output.Artifact != nil && cmd.Output.Stream != nil {
		return nil, fmt.Errorf("command %q declares both output.artifact and output.stream; a command's default output is either a written media file or a streamed projection, not both", key)
	}
	if asyncNode != nil {
		if cmd.Async, err = d.decodeAsync(key, asyncNode); err != nil {
			return nil, err
		}
		if cmd.Output != nil && cmd.Output.Stream != nil {
			return nil, fmt.Errorf("command %q declares both async and output.stream; async polling requires discrete JSON responses and cannot be combined with streamed output", key)
		}
		if cmd.Output != nil && cmd.Output.Outcome != nil {
			return nil, fmt.Errorf("command %q declares both async and output.outcome; async.states already classifies the terminal poll state", key)
		}
	}
	if continuationNode != nil {
		if cmd.Continuation, err = d.decodeContinuation(fmt.Sprintf("command %q", key), continuationNode); err != nil {
			return nil, err
		}
		if cmd.Async != nil {
			return nil, fmt.Errorf("command %q declares both async and continuation; a continuation re-sends the command's own request and cannot follow a poll", key)
		}
	}

	if err := d.decodeSource(cmd, key, node, opNode, routesNode, plannedNode, customNode, groupNode, sourceNode); err != nil {
		return nil, err
	}
	if overrideNode != nil && routesNode == nil {
		return nil, fmt.Errorf("line %d: command %q: override is only valid with a routes: dispatch map", overrideNode.Line, key)
	}
	if continuationNode != nil && len(cmd.Source.Routes) > 1 {
		return nil, fmt.Errorf("line %d: command %q: continuation requires a single-operation command; dispatch commands with several routes cannot declare it", continuationNode.Line, key)
	}

	// Group tags categorize top-level command groups (help categories are
	// root-level sections); the restriction applies to both the group: true
	// sugar and the long-form source object.
	if cmd.Source.Type == CLICommandSourceGroup && len(cmd.Path) > 1 {
		return nil, fmt.Errorf("line %d: command %q: group tags categorize top-level command groups; nested group tags are not part of v1", node.Line, key)
	}

	if cmd.Source.Type != CLICommandSourceOperation {
		var illegal []string
		for keyName, present := range map[string]*yaml.Node{
			"args": argsNode, "flags": flagsNode, "preset": presetNode, "hints": hintsNode,
		} {
			if present != nil {
				illegal = append(illegal, keyName)
			}
		}
		if cmd.Output != nil && cmd.Output.Projection != nil {
			illegal = append(illegal, "jq")
		}
		if cmd.Output != nil && (cmd.Output.Artifact != nil || cmd.Output.Stream != nil || cmd.Output.Outcome != nil) {
			illegal = append(illegal, "output")
		}
		if asyncNode != nil {
			illegal = append(illegal, "async")
		}
		if continuationNode != nil {
			illegal = append(illegal, "continuation")
		}
		if cmd.Source.Type == CLICommandSourceGroup && exampNode != nil {
			illegal = append(illegal, "examples")
		}
		if len(illegal) > 0 {
			sort.Strings(illegal)
			return nil, fmt.Errorf("command %q is a %s command and cannot declare %s (those keys describe an operation invocation)", key, cmd.Source.Type, strings.Join(illegal, ", "))
		}
	}

	if argsNode != nil {
		if cmd.Args, err = d.decodeInputs(key, argsNode, true); err != nil {
			return nil, err
		}
	}
	if flagsNode != nil {
		if cmd.Flags, err = d.decodeInputs(key, flagsNode, false); err != nil {
			return nil, err
		}
	}
	if presetNode != nil {
		if cmd.Presets, err = d.decodePresets(key, presetNode); err != nil {
			return nil, err
		}
	}
	if hintsNode != nil {
		if cmd.Hints, err = d.decodeHints(key, hintsNode); err != nil {
			return nil, err
		}
	}
	if exampNode != nil {
		if cmd.Examples, err = d.decodeExamples(key, exampNode); err != nil {
			return nil, err
		}
	}
	if helpNode != nil {
		if cmd.Help, err = d.decodeCommandHelp(key, helpNode); err != nil {
			return nil, err
		}
	}

	if err := d.checkCommandInputs(cmd, key); err != nil {
		return nil, err
	}

	return cmd, nil
}

func (d *cliManifestDecoder) decodeCommandHelp(cmdKey string, node *yaml.Node) (*CLICommandHelp, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q help", cmdKey))
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("line %d: command %q help must declare defaults, learn, or escalate", node.Line, cmdKey)
	}

	help := &CLICommandHelp{}
	for _, entry := range entries {
		if entry.Key.Value != "defaults" && entry.Key.Value != "learn" && entry.Key.Value != "escalate" {
			return nil, fmt.Errorf("line %d: command %q help has unknown key %q%s", entry.Key.Line, cmdKey, entry.Key.Value, cliDidYouMean(entry.Key.Value, []string{"defaults", "learn", "escalate"}))
		}
		value, err := cliScalarString(entry.Value, "help "+entry.Key.Value)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("line %d: command %q help %s must be a non-empty string", entry.Value.Line, cmdKey, entry.Key.Value)
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("line %d: command %q help %s must be a single line", entry.Value.Line, cmdKey, entry.Key.Value)
		}
		switch entry.Key.Value {
		case "defaults":
			help.Defaults = value
		case "learn":
			help.Learn = value
		case "escalate":
			help.Escalate = value
		}
	}
	return help, nil
}

func (d *cliManifestDecoder) decodeAsync(cmdKey string, node *yaml.Node) (*CLICommandAsync, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q async", cmdKey))
	if err != nil {
		return nil, err
	}

	recipe := &CLICommandAsync{
		States:      map[string]string{},
		Params:      map[string]any{},
		Interval:    "2s",
		Backoff:     1.5,
		MaxInterval: "30s",
		Timeout:     "10m",
	}
	var idNode *yaml.Node
	var paramsNode *yaml.Node
	for _, entry := range entries {
		switch entry.Key.Value {
		case "op":
			recipe.OperationID, err = cliScalarString(entry.Value, "async.op")
		case "id":
			idNode = entry.Value
		case "params":
			paramsNode = entry.Value
		case "statePointer":
			recipe.StateFrom, recipe.StatePointer, recipe.StateSegments, err = cliDecodeAsyncPointer(entry.Value, "statePointer")
		case "states":
			err = d.decodeAsyncStates(cmdKey, entry.Value, recipe.States)
		case "interval":
			recipe.Interval, err = cliDecodeDuration(entry.Value, "async.interval")
		case "backoff":
			recipe.Backoff, err = cliDecodeBackoff(entry.Value)
		case "maxInterval":
			recipe.MaxInterval, err = cliDecodeDuration(entry.Value, "async.maxInterval")
		case "timeout":
			recipe.Timeout, err = cliDecodeDuration(entry.Value, "async.timeout")
		case "errorField":
			recipe.ErrorField, err = cliDecodeArtifactFieldName(entry.Value, "async.errorField")
			recipe.errorExplicit = true
		case "errorMessageField":
			recipe.ErrorMessageField, err = cliDecodeArtifactFieldName(entry.Value, "async.errorMessageField")
			recipe.errorExplicit = true
		default:
			return nil, fmt.Errorf("line %d: command %q async has unknown key %q%s", entry.Key.Line, cmdKey, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliAsyncKeys))
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: command %q: %w", entry.Value.Line, cmdKey, err)
		}
	}

	if strings.TrimSpace(recipe.OperationID) == "" {
		return nil, fmt.Errorf("line %d: command %q async requires op (the GET operation used to poll the handle)", node.Line, cmdKey)
	}
	if idNode == nil {
		return nil, fmt.Errorf("line %d: command %q async requires id.from and id.to", node.Line, cmdKey)
	}
	if err := d.decodeAsyncID(cmdKey, idNode, &recipe.ID); err != nil {
		return nil, err
	}
	if paramsNode != nil {
		if err := d.decodeAsyncParams(cmdKey, paramsNode, recipe.Params); err != nil {
			return nil, err
		}
	}
	if recipe.StateFrom == "" {
		return nil, fmt.Errorf("line %d: command %q async requires statePointer", node.Line, cmdKey)
	}
	if len(recipe.States) == 0 {
		return nil, fmt.Errorf("line %d: command %q async requires states (a map from every API state to pending, success, failure, or handoff)", node.Line, cmdKey)
	}
	hasSuccess := false
	for _, class := range recipe.States {
		hasSuccess = hasSuccess || class == "success"
	}
	if !hasSuccess {
		return nil, fmt.Errorf("line %d: command %q async.states must classify at least one state as success", node.Line, cmdKey)
	}
	interval, _ := time.ParseDuration(recipe.Interval)
	maxInterval, _ := time.ParseDuration(recipe.MaxInterval)
	timeout, _ := time.ParseDuration(recipe.Timeout)
	if maxInterval < interval {
		return nil, fmt.Errorf("line %d: command %q async.maxInterval (%s) must be greater than or equal to async.interval (%s)", node.Line, cmdKey, recipe.MaxInterval, recipe.Interval)
	}
	if timeout < interval {
		return nil, fmt.Errorf("line %d: command %q async.timeout (%s) must be greater than or equal to async.interval (%s)", node.Line, cmdKey, recipe.Timeout, recipe.Interval)
	}
	// Fill the v1 failure-detail defaults so the linker and the annotation
	// serializer always see concrete member names (the serializer omits
	// values equal to these defaults, keeping undeclared recipes byte-stable).
	if recipe.ErrorField == "" {
		recipe.ErrorField = cliAsyncDefaultErrorField
	}
	if recipe.ErrorMessageField == "" {
		recipe.ErrorMessageField = cliAsyncDefaultErrorMessageField
	}
	return recipe, nil
}

func (d *cliManifestDecoder) decodeAsyncID(cmdKey string, node *yaml.Node, id *CLICommandAsyncID) error {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q async.id", cmdKey))
	if err != nil {
		return err
	}
	var toNode *yaml.Node
	for _, entry := range entries {
		switch entry.Key.Value {
		case "from":
			id.From, id.Pointer, id.Segments, err = cliDecodeAsyncPointer(entry.Value, "async.id.from")
		case "to":
			toNode = entry.Value
		default:
			return fmt.Errorf("line %d: command %q async.id has unknown key %q%s", entry.Key.Line, cmdKey, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliAsyncIDKeys))
		}
		if err != nil {
			return fmt.Errorf("line %d: command %q: %w", entry.Value.Line, cmdKey, err)
		}
	}
	if id.From == "" {
		return fmt.Errorf("line %d: command %q async.id requires from (a singular JSONPath in the create response)", node.Line, cmdKey)
	}
	if toNode == nil {
		return fmt.Errorf("line %d: command %q async.id requires to", node.Line, cmdKey)
	}
	return d.decodeAsyncIDTo(cmdKey, toNode, &id.To)
}

func (d *cliManifestDecoder) decodeAsyncParams(cmdKey string, node *yaml.Node, params map[string]any) error {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q async.params", cmdKey))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name, err := cliScalarString(entry.Key, "async.params parameter name")
		if err != nil || strings.TrimSpace(name) == "" {
			return fmt.Errorf("line %d: command %q async.params keys must be non-empty parameter names", entry.Key.Line, cmdKey)
		}
		if entry.Value.Kind != yaml.ScalarNode || entry.Value.Tag == "!!null" {
			return fmt.Errorf("line %d: command %q async.params.%s must be a string, integer, number, or boolean", entry.Value.Line, cmdKey, name)
		}
		value, err := cliDecodeValue(entry.Value)
		if err != nil {
			return err
		}
		params[name] = value
	}
	return nil
}

func (d *cliManifestDecoder) decodeAsyncIDTo(cmdKey string, node *yaml.Node, to *CLICommandAsyncParameter) error {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q async.id.to", cmdKey))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		switch entry.Key.Value {
		case "in":
			to.In, err = cliScalarString(entry.Value, "async.id.to.in")
		case "name":
			to.Name, err = cliScalarString(entry.Value, "async.id.to.name")
		default:
			return fmt.Errorf("line %d: command %q async.id.to has unknown key %q%s", entry.Key.Line, cmdKey, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliAsyncIDToKeys))
		}
		if err != nil {
			return fmt.Errorf("line %d: command %q: %w", entry.Value.Line, cmdKey, err)
		}
	}
	if to.In != "path" && to.In != "query" {
		return fmt.Errorf("line %d: command %q async.id.to.in must be path or query, got %q", node.Line, cmdKey, to.In)
	}
	if strings.TrimSpace(to.Name) == "" {
		return fmt.Errorf("line %d: command %q async.id.to requires name", node.Line, cmdKey)
	}
	return nil
}

func (d *cliManifestDecoder) decodeAsyncStates(cmdKey string, node *yaml.Node, states map[string]string) error {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q async.states", cmdKey))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		state, err := cliScalarString(entry.Key, "async state name")
		if err != nil || strings.TrimSpace(state) == "" {
			return fmt.Errorf("line %d: command %q async.states keys must be non-empty strings", entry.Key.Line, cmdKey)
		}
		class, err := cliScalarString(entry.Value, fmt.Sprintf("classification for async state %q", state))
		if err != nil {
			return err
		}
		if !cliContains(cliAsyncStateKind, class) {
			return fmt.Errorf("line %d: command %q async.states[%q] has unsupported classification %q (expected %s)", entry.Value.Line, cmdKey, state, class, cliCandidateList(cliAsyncStateKind))
		}
		states[state] = class
	}
	return nil
}

func cliDecodeAsyncPointer(node *yaml.Node, what string) (string, string, []CLICommandAsyncPathSegment, error) {
	raw, err := cliScalarString(node, what)
	if err != nil {
		return "", "", nil, err
	}
	segments, err := cliParseSingularPath(raw)
	if err != nil {
		return "", "", nil, fmt.Errorf("%s: %w", what, err)
	}
	if len(segments) == 0 {
		return "", "", nil, fmt.Errorf("%s must name a response property, not the response root", what)
	}
	lowered := make([]CLICommandAsyncPathSegment, 0, len(segments))
	for _, segment := range segments {
		lowered = append(lowered, CLICommandAsyncPathSegment{Field: segment.Name, Index: segment.Index, IsIndex: segment.IsIndex})
	}
	return raw, cliSegmentsToPointer(segments), lowered, nil
}

func cliDecodeDuration(node *yaml.Node, what string) (string, error) {
	raw, err := cliScalarString(node, what)
	if err != nil {
		return "", err
	}
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		return "", fmt.Errorf("%s must be a positive Go duration string (for example %q), got %q", what, "2s", raw)
	}
	return raw, nil
}

func cliDecodeBackoff(node *yaml.Node) (float64, error) {
	if node == nil || node.Kind != yaml.ScalarNode || (node.Tag != "!!float" && node.Tag != "!!int") {
		return 0, stderrors.New("async.backoff must be a number greater than or equal to 1")
	}
	value, err := strconv.ParseFloat(node.Value, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 1 {
		return 0, fmt.Errorf("async.backoff must be a finite number greater than or equal to 1, got %q", node.Value)
	}
	return value, nil
}

// decodeOutput decodes the output: block — the home for a command's output
// capabilities beyond the command-level jq sugar: artifact (the result is a
// written media file) and stream (the selected field of each streamed event
// is written raw to stdout as it arrives).
func (d *cliManifestDecoder) decodeOutput(cmdKey string, node *yaml.Node) (*CLICommandArtifact, *CLICommandStreamProjection, *CLICommandOutcome, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q output", cmdKey))
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		artifact *CLICommandArtifact
		stream   *CLICommandStreamProjection
		outcome  *CLICommandOutcome
	)
	for _, entry := range entries {
		switch entry.Key.Value {
		case "artifact":
			artifact, err = d.decodeArtifact(cmdKey, entry.Value)
			if err != nil {
				return nil, nil, nil, err
			}
		case "stream":
			stream, err = d.decodeStreamProjection(cmdKey, entry.Value)
			if err != nil {
				return nil, nil, nil, err
			}
		case "outcome":
			outcome, err = d.decodeOutcome(fmt.Sprintf("command %q", cmdKey), entry.Value)
			if err != nil {
				return nil, nil, nil, err
			}
		case "jq":
			return nil, nil, nil, fmt.Errorf("line %d: command %q: jq is declared at the command level (jq: <expression>), not under output", entry.Key.Line, cmdKey)
		default:
			return nil, nil, nil, fmt.Errorf("line %d: command %q output has unknown key %q%s", entry.Key.Line, cmdKey, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliOutputKeys))
		}
	}
	if artifact == nil && stream == nil && outcome == nil {
		return nil, nil, nil, fmt.Errorf("line %d: command %q: output must declare artifact, stream, or outcome (the output capabilities beyond the command-level jq key)", node.Line, cmdKey)
	}
	return artifact, stream, outcome, nil
}

// decodeOutcome decodes an output.outcome declaration. Pointer resolvability
// and enum coverage are checked during linking; this stage validates the
// local grammar: at least one state pointer, a states table whose values are
// the async classifications, and member-name error bindings.
func (d *cliManifestDecoder) decodeOutcome(owner string, node *yaml.Node) (*CLICommandOutcome, error) {
	entries, err := cliMapEntries(node, owner+" output.outcome")
	if err != nil {
		return nil, err
	}
	outcome := &CLICommandOutcome{States: map[string]string{}}
	for _, entry := range entries {
		switch entry.Key.Value {
		case "statePointer":
			outcome.StateFrom, outcome.StatePointer, _, err = cliDecodeAsyncPointer(entry.Value, "output.outcome.statePointer")
		case "streamStatePointer":
			outcome.StreamStateFrom, outcome.StreamStatePointer, _, err = cliDecodeAsyncPointer(entry.Value, "output.outcome.streamStatePointer")
		case "states":
			err = d.decodeOutcomeStates(owner, entry.Value, outcome.States)
		case "errorPointer":
			outcome.ErrorFrom, outcome.ErrorPointer, _, err = cliDecodeAsyncPointer(entry.Value, "output.outcome.errorPointer")
		case "streamErrorPointer":
			outcome.StreamErrorFrom, outcome.StreamErrorPointer, _, err = cliDecodeAsyncPointer(entry.Value, "output.outcome.streamErrorPointer")
		case "errorMessageField":
			outcome.ErrorMessageField, err = cliDecodeArtifactFieldName(entry.Value, "output.outcome.errorMessageField")
		default:
			return nil, fmt.Errorf("line %d: %s output.outcome has unknown key %q%s", entry.Key.Line, owner, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliOutcomeKeys))
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %s: %w", entry.Value.Line, owner, err)
		}
	}
	if outcome.StatePointer == "" && outcome.StreamStatePointer == "" {
		return nil, fmt.Errorf("line %d: %s output.outcome requires statePointer (the state in the JSON response) or streamStatePointer (the state inside streamed events)", node.Line, owner)
	}
	if len(outcome.States) == 0 {
		return nil, fmt.Errorf("line %d: %s output.outcome requires states (a map from every API state to pending, success, failure, or handoff)", node.Line, owner)
	}
	hasFailure := false
	for _, class := range outcome.States {
		hasFailure = hasFailure || class == "failure"
	}
	if !hasFailure {
		return nil, fmt.Errorf("line %d: %s output.outcome.states must classify at least one state as failure; the declaration exists to fail the command on that state", node.Line, owner)
	}
	if outcome.ErrorPointer != "" && outcome.StatePointer == "" {
		return nil, fmt.Errorf("line %d: %s output.outcome.errorPointer requires statePointer; use streamErrorPointer for streamed events", node.Line, owner)
	}
	if outcome.StreamErrorPointer != "" && outcome.StreamStatePointer == "" {
		return nil, fmt.Errorf("line %d: %s output.outcome.streamErrorPointer requires streamStatePointer", node.Line, owner)
	}
	if outcome.ErrorMessageField != "" && outcome.ErrorPointer == "" && outcome.StreamErrorPointer == "" {
		return nil, fmt.Errorf("line %d: %s output.outcome.errorMessageField requires errorPointer or streamErrorPointer", node.Line, owner)
	}
	if (outcome.ErrorPointer != "" || outcome.StreamErrorPointer != "") && outcome.ErrorMessageField == "" {
		outcome.ErrorMessageField = cliAsyncDefaultErrorMessageField
	}
	return outcome, nil
}

func (d *cliManifestDecoder) decodeOutcomeStates(owner string, node *yaml.Node, states map[string]string) error {
	entries, err := cliMapEntries(node, owner+" output.outcome.states")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		state, err := cliScalarString(entry.Key, "outcome state name")
		if err != nil || strings.TrimSpace(state) == "" {
			return fmt.Errorf("line %d: %s output.outcome.states keys must be non-empty strings", entry.Key.Line, owner)
		}
		class, err := cliScalarString(entry.Value, fmt.Sprintf("classification for outcome state %q", state))
		if err != nil {
			return err
		}
		if !cliContains(cliOutcomeStateKind, class) {
			return fmt.Errorf("line %d: %s output.outcome.states[%q] has unsupported classification %q (expected %s)", entry.Value.Line, owner, state, class, cliCandidateList(cliOutcomeStateKind))
		}
		states[state] = class
	}
	return nil
}

// decodeContinuation decodes a continuation recipe. Pointer resolvability,
// the request-body targets, and the required output.outcome continue state
// are checked during linking.
func (d *cliManifestDecoder) decodeContinuation(owner string, node *yaml.Node) (*CLICommandContinuation, error) {
	entries, err := cliMapEntries(node, owner+" continuation")
	if err != nil {
		return nil, err
	}
	recipe := &CLICommandContinuation{MaxHops: cliContinuationDefaultMaxHops, Default: true}
	var bindNode *yaml.Node
	for _, entry := range entries {
		switch entry.Key.Value {
		case "bind":
			bindNode = entry.Value
		case "merge":
			recipe.MergeFrom, recipe.Merge, err = cliDecodeContinuationPaths(entry.Value, "continuation.merge")
		case "sum":
			recipe.SumFrom, recipe.Sum, err = cliDecodeContinuationPaths(entry.Value, "continuation.sum")
		case "maxHops":
			if entry.Value.Kind != yaml.ScalarNode || entry.Value.Tag != "!!int" {
				err = stderrors.New("continuation.maxHops must be an integer of at least 1")
				break
			}
			recipe.MaxHops, err = strconv.Atoi(entry.Value.Value)
			if err == nil && recipe.MaxHops < 1 {
				err = fmt.Errorf("continuation.maxHops must be at least 1, got %d", recipe.MaxHops)
			}
		case "default":
			recipe.Default, err = cliScalarBool(entry.Value, "continuation.default")
		default:
			return nil, fmt.Errorf("line %d: %s continuation has unknown key %q%s", entry.Key.Line, owner, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliContinuationKeys))
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %s: %w", entry.Value.Line, owner, err)
		}
	}
	if bindNode == nil || bindNode.Kind != yaml.SequenceNode || len(bindNode.Content) == 0 {
		return nil, fmt.Errorf("line %d: %s continuation requires bind (a non-empty list of {from, to} resume values copied into the next request)", node.Line, owner)
	}
	targets := map[string]bool{}
	required := false
	for _, item := range bindNode.Content {
		binding, err := d.decodeContinuationBinding(owner, item)
		if err != nil {
			return nil, err
		}
		if targets[binding.ToPointer] {
			return nil, fmt.Errorf("line %d: %s continuation.bind writes %s more than once", item.Line, owner, binding.To)
		}
		targets[binding.ToPointer] = true
		required = required || !binding.Optional
		recipe.Bind = append(recipe.Bind, *binding)
	}
	if !required {
		return nil, fmt.Errorf("line %d: %s continuation.bind must include at least one binding that is not optional; the next request needs a resume value", bindNode.Line, owner)
	}
	return recipe, nil
}

func (d *cliManifestDecoder) decodeContinuationBinding(owner string, node *yaml.Node) (*CLICommandContinuationBinding, error) {
	entries, err := cliMapEntries(node, owner+" continuation.bind entry")
	if err != nil {
		return nil, err
	}
	binding := &CLICommandContinuationBinding{}
	for _, entry := range entries {
		switch entry.Key.Value {
		case "from":
			binding.From, binding.Pointer, _, err = cliDecodeAsyncPointer(entry.Value, "continuation.bind.from")
		case "streamFrom":
			binding.StreamFrom, binding.StreamPointers, err = cliDecodeContinuationPaths(entry.Value, "continuation.bind.streamFrom")
		case "to":
			var segments []CLICommandAsyncPathSegment
			binding.To, binding.ToPointer, segments, err = cliDecodeAsyncPointer(entry.Value, "continuation.bind.to")
			for _, segment := range segments {
				if segment.IsIndex {
					err = fmt.Errorf("continuation.bind.to %s must name object properties only", binding.To)
					break
				}
				binding.ToPath = append(binding.ToPath, segment.Field)
			}
		case "optional":
			binding.Optional, err = cliScalarBool(entry.Value, "continuation.bind.optional")
		default:
			return nil, fmt.Errorf("line %d: %s continuation.bind has unknown key %q%s", entry.Key.Line, owner, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliContinuationBindingKeys))
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %s: %w", entry.Value.Line, owner, err)
		}
	}
	if binding.To == "" {
		return nil, fmt.Errorf("line %d: %s continuation.bind entry requires to (the request-body path that receives the value)", node.Line, owner)
	}
	if binding.From == "" && len(binding.StreamFrom) == 0 {
		return nil, fmt.Errorf("line %d: %s continuation.bind entry for %s requires from (JSON response) or streamFrom (streamed events)", node.Line, owner, binding.To)
	}
	return binding, nil
}

// cliDecodeContinuationPaths accepts one singular JSONPath or a list of them
// and returns the authored paths with their RFC 6901 lowerings.
func cliDecodeContinuationPaths(node *yaml.Node, what string) ([]string, []string, error) {
	items := []*yaml.Node{node}
	if node.Kind == yaml.SequenceNode {
		if len(node.Content) == 0 {
			return nil, nil, fmt.Errorf("%s must list at least one path", what)
		}
		items = node.Content
	}
	var authored, pointers []string
	for _, item := range items {
		from, pointer, _, err := cliDecodeAsyncPointer(item, what)
		if err != nil {
			return nil, nil, err
		}
		if cliContains(pointers, pointer) {
			return nil, nil, fmt.Errorf("%s lists %s more than once", what, from)
		}
		authored = append(authored, from)
		pointers = append(pointers, pointer)
	}
	return authored, pointers, nil
}

// decodeArtifact decodes and validates an output.artifact declaration. The
// content pointer's resolvability against the response schema is checked
// during linking; this stage validates the local grammar.
func (d *cliManifestDecoder) decodeArtifact(cmdKey string, node *yaml.Node) (*CLICommandArtifact, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q output.artifact", cmdKey))
	if err != nil {
		return nil, err
	}

	artifact := &CLICommandArtifact{}
	for _, entry := range entries {
		switch entry.Key.Value {
		case "contentPointer":
			raw, err := cliScalarString(entry.Value, "contentPointer")
			if err != nil {
				return nil, err
			}
			segments, err := cliParseArtifactPath(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d: command %q output.artifact contentPointer: %w", entry.Value.Line, cmdKey, err)
			}
			artifact.ContentPointer = raw
			for _, seg := range segments {
				artifact.Segments = append(artifact.Segments, CLICommandArtifactSegment{Field: seg.Name, Wild: seg.IsWild})
			}
		case "kind":
			kind, err := cliScalarString(entry.Value, "kind")
			if err != nil {
				return nil, err
			}
			if !cliContains(cliArtifactKinds, kind) {
				return nil, fmt.Errorf("line %d: command %q output.artifact: unsupported kind %q (expected %s)", entry.Value.Line, cmdKey, kind, cliCandidateList(cliArtifactKinds))
			}
			artifact.Kind = kind
		case "defaultPath":
			pattern, err := cliScalarString(entry.Value, "defaultPath")
			if err != nil {
				return nil, err
			}
			if err := cliCheckArtifactDefaultPath(pattern); err != nil {
				return nil, fmt.Errorf("line %d: command %q output.artifact defaultPath: %w", entry.Value.Line, cmdKey, err)
			}
			artifact.DefaultPath = pattern
		case "block":
			if err := d.decodeArtifactBlock(cmdKey, entry.Value, artifact); err != nil {
				return nil, err
			}
		case "identity":
			if err := d.decodeArtifactIdentity(cmdKey, entry.Value, artifact); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("line %d: command %q output.artifact has unknown key %q%s", entry.Key.Line, cmdKey, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliArtifactKeys))
		}
	}

	if err := cliFillArtifactDefaults(cmdKey, artifact); err != nil {
		return nil, err
	}

	if artifact.ContentPointer == "" {
		return nil, fmt.Errorf("line %d: command %q output.artifact requires contentPointer (a restricted JSONPath such as $.steps[*].content[*] naming where the media content lives in the response)", node.Line, cmdKey)
	}
	if artifact.Kind == "" {
		return nil, fmt.Errorf("line %d: command %q output.artifact requires kind (%s)", node.Line, cmdKey, cliCandidateList(cliArtifactKinds))
	}
	if artifact.DefaultPath == "" {
		return nil, fmt.Errorf("line %d: command %q output.artifact requires defaultPath (a filename pattern such as %q)", node.Line, cmdKey, "asset-{timestamp}-{rand}.{ext}")
	}
	return artifact, nil
}

// decodeArtifactBlock decodes the block: field bindings — the exact JSON
// member names the runtime reads on each candidate content block. Declaring
// any binding opts the whole block into strict linking against the response
// schema (see linkArtifact); undeclared members keep the v1 defaults.
func (d *cliManifestDecoder) decodeArtifactBlock(cmdKey string, node *yaml.Node, artifact *CLICommandArtifact) error {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q output.artifact block", cmdKey))
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("line %d: command %q output.artifact block must declare at least one of %s", node.Line, cmdKey, cliCandidateList(cliArtifactBlockKeys))
	}
	for _, entry := range entries {
		var target *string
		switch entry.Key.Value {
		case "typeField":
			target = &artifact.TypeField
		case "dataField":
			target = &artifact.DataField
		case "mimeTypeField":
			target = &artifact.MimeTypeField
		case "uriField":
			target = &artifact.URIField
		default:
			return fmt.Errorf("line %d: command %q output.artifact block has unknown key %q%s", entry.Key.Line, cmdKey, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliArtifactBlockKeys))
		}
		value, err := cliDecodeArtifactFieldName(entry.Value, "block "+entry.Key.Value)
		if err != nil {
			return fmt.Errorf("line %d: command %q output.artifact: %w", entry.Value.Line, cmdKey, err)
		}
		*target = value
		artifact.markBindingExplicit(entry.Key.Value)
	}
	artifact.blockExplicit = true
	return nil
}

// markBindingExplicit records that the author wrote a specific binding key.
func (a *CLICommandArtifact) markBindingExplicit(key string) {
	if a.explicitBindings == nil {
		a.explicitBindings = map[string]bool{}
	}
	a.explicitBindings[key] = true
}

// decodeArtifactIdentity decodes the identity: bindings — the response-root
// member names carried into the reported envelope (response_id/status) and
// the not-ready diagnostic, plus the terminal status value content is
// expected at. Declaring any binding opts the whole group into strict
// linking; undeclared members keep the v1 defaults.
func (d *cliManifestDecoder) decodeArtifactIdentity(cmdKey string, node *yaml.Node, artifact *CLICommandArtifact) error {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q output.artifact identity", cmdKey))
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("line %d: command %q output.artifact identity must declare at least one of %s", node.Line, cmdKey, cliCandidateList(cliArtifactIdentityKeys))
	}
	for _, entry := range entries {
		switch entry.Key.Value {
		case "idField", "statusField":
			value, err := cliDecodeArtifactFieldName(entry.Value, "identity "+entry.Key.Value)
			if err != nil {
				return fmt.Errorf("line %d: command %q output.artifact: %w", entry.Value.Line, cmdKey, err)
			}
			if entry.Key.Value == "idField" {
				artifact.IDField = value
			} else {
				artifact.StatusField = value
			}
			artifact.markBindingExplicit(entry.Key.Value)
		case "terminalStatus":
			value, err := cliScalarString(entry.Value, "identity terminalStatus")
			if err != nil {
				return fmt.Errorf("line %d: command %q output.artifact: %w", entry.Value.Line, cmdKey, err)
			}
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("line %d: command %q output.artifact identity terminalStatus must be a non-empty status value", entry.Value.Line, cmdKey)
			}
			artifact.TerminalStatus = value
			artifact.markBindingExplicit(entry.Key.Value)
		default:
			return fmt.Errorf("line %d: command %q output.artifact identity has unknown key %q%s", entry.Key.Line, cmdKey, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliArtifactIdentityKeys))
		}
	}
	artifact.identityExplicit = true
	return nil
}

// cliDecodeArtifactFieldName decodes a block/identity binding: the exact JSON
// member name the runtime reads, not a path. Rejecting path-looking values
// keeps a JSONPath habit from silently binding a literal "$.data" member.
func cliDecodeArtifactFieldName(node *yaml.Node, what string) (string, error) {
	value, err := cliScalarString(node, what)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be a non-empty JSON member name", what)
	}
	if strings.HasPrefix(value, "$") || strings.HasPrefix(value, "/") || strings.ContainsAny(value, "* \t") {
		return "", fmt.Errorf("%s %q must be the exact JSON member name of one property (paths and wildcards are not supported)", what, value)
	}
	return value, nil
}

// cliFillArtifactDefaults applies the v1 shape to every undeclared binding
// and rejects bindings that collapse onto one member (a property cannot be
// both the payload and its MIME type, and an identity id cannot double as the
// status).
func cliFillArtifactDefaults(cmdKey string, artifact *CLICommandArtifact) error {
	if artifact.TypeField == "" {
		artifact.TypeField = cliArtifactDefaultTypeField
	}
	if artifact.DataField == "" {
		artifact.DataField = cliArtifactDefaultDataField
	}
	if artifact.MimeTypeField == "" {
		artifact.MimeTypeField = cliArtifactDefaultMimeTypeField
	}
	if artifact.URIField == "" {
		artifact.URIField = cliArtifactDefaultURIField
	}
	if artifact.IDField == "" {
		artifact.IDField = cliArtifactDefaultIDField
	}
	if artifact.StatusField == "" {
		artifact.StatusField = cliArtifactDefaultStatusField
	}
	if artifact.TerminalStatus == "" {
		artifact.TerminalStatus = cliArtifactDefaultTerminalStatus
	}

	bound := map[string]string{}
	for _, binding := range []struct{ key, field string }{
		{"typeField", artifact.TypeField},
		{"dataField", artifact.DataField},
		{"mimeTypeField", artifact.MimeTypeField},
		{"uriField", artifact.URIField},
	} {
		if previous, dup := bound[binding.field]; dup {
			return cliArtifactDuplicateBindingError(cmdKey, "block", previous, binding.key, binding.field, artifact.explicitBindings)
		}
		bound[binding.field] = binding.key
	}
	if artifact.IDField == artifact.StatusField {
		return cliArtifactDuplicateBindingError(cmdKey, "identity", "idField", "statusField", artifact.IDField, artifact.explicitBindings)
	}
	return nil
}

// cliArtifactDuplicateBindingError reports two bindings collapsing onto one
// member. When one side of the collision is a filled-in default the message
// says so instead of claiming the author declared it.
func cliArtifactDuplicateBindingError(cmdKey, group, first, second, member string, explicit map[string]bool) error {
	declared, defaulted := first, second
	if !explicit[first] {
		declared, defaulted = second, first
	}
	if explicit[first] && explicit[second] {
		return fmt.Errorf("command %q output.artifact %s: %s and %s both bind property %q; each %s field must bind a distinct property", cmdKey, group, first, second, member, group)
	}
	return fmt.Errorf("command %q output.artifact %s: %s binds property %q, which is also the default %s binding; bind %s to a different member or declare %s explicitly so each %s field binds a distinct property", cmdKey, group, declared, member, defaulted, declared, defaulted, group)
}

// cliCheckArtifactDefaultPath validates a defaultPath pattern: relative,
// traversal-free, known placeholders only, and {ext} present so the written
// file's extension always tracks the returned content type.
func cliCheckArtifactDefaultPath(pattern string) error {
	if strings.TrimSpace(pattern) == "" {
		return stderrors.New("pattern must be a non-empty filename pattern")
	}
	if strings.HasPrefix(pattern, "/") || strings.Contains(pattern, "\\") || (len(pattern) > 1 && pattern[1] == ':') {
		return fmt.Errorf("pattern %q must be a relative path using forward slashes (it resolves against the invocation directory)", pattern)
	}
	for _, part := range strings.Split(pattern, "/") {
		if part == ".." {
			return fmt.Errorf("pattern %q must not contain .. segments", pattern)
		}
	}
	rest := pattern
	sawExt := false
	for {
		open := strings.Index(rest, "{")
		if open == -1 {
			if strings.Contains(rest, "}") {
				return fmt.Errorf("pattern %q has an unmatched } brace", pattern)
			}
			break
		}
		closing := strings.Index(rest[open:], "}")
		if closing == -1 {
			return fmt.Errorf("pattern %q has an unterminated placeholder", pattern)
		}
		name := rest[open+1 : open+closing]
		if !cliArtifactPlaceholders[name] {
			return fmt.Errorf("pattern %q uses unknown placeholder {%s} (known: %s)", pattern, name, cliCandidateList(cliMapKeys(cliArtifactPlaceholders)))
		}
		if name == "ext" {
			sawExt = true
		}
		rest = rest[open+closing+1:]
	}
	if !sawExt {
		return fmt.Errorf("pattern %q must contain the {ext} placeholder so the extension tracks the returned content type", pattern)
	}
	return nil
}

func (d *cliManifestDecoder) decodeSource(cmd *CLICommand, key string, node, opNode, routesNode, plannedNode, customNode, groupNode, sourceNode *yaml.Node) error {
	present := 0
	for _, n := range []*yaml.Node{opNode, routesNode, plannedNode, customNode, groupNode, sourceNode} {
		if n != nil {
			present++
		}
	}
	if present == 0 {
		return fmt.Errorf("line %d: command %q must declare exactly one of op, routes, planned, custom, group, or source", node.Line, key)
	}
	if present > 1 {
		return fmt.Errorf("line %d: command %q declares more than one of op, routes, planned, custom, group, source; exactly one source is allowed", node.Line, key)
	}

	switch {
	case opNode != nil:
		routes, err := d.decodeOpSugar(key, opNode)
		if err != nil {
			return err
		}
		cmd.Source = CLICommandSource{Type: CLICommandSourceOperation, Routes: routes}
	case routesNode != nil:
		routes, err := d.decodeDispatchRoutes(key, routesNode)
		if err != nil {
			return err
		}
		cmd.Source = CLICommandSource{Type: CLICommandSourceOperation, Routes: routes}
	case plannedNode != nil:
		note, err := cliScalarString(plannedNode, "planned")
		if err != nil {
			return err
		}
		note = strings.TrimSpace(note)
		if note == "" {
			return fmt.Errorf("line %d: command %q: planned requires a non-empty note that teaches the escalation path", plannedNode.Line, key)
		}
		cmd.Source = CLICommandSource{Type: CLICommandSourcePlanned, Note: note}
	case customNode != nil:
		isTrue, err := cliScalarBool(customNode, "custom")
		if err != nil || !isTrue {
			line := customNode.Line
			return fmt.Errorf("line %d: command %q: custom must be the literal true (it reserves a name for a hand-written command supplied after generation)", line, key)
		}
		cmd.Source = CLICommandSource{Type: CLICommandSourceCustom}
	case groupNode != nil:
		isTrue, err := cliScalarBool(groupNode, "group")
		if err != nil || !isTrue {
			line := groupNode.Line
			return fmt.Errorf("line %d: command %q: group must be the literal true (it tags a generated command group into a category)", line, key)
		}
		cmd.Source = CLICommandSource{Type: CLICommandSourceGroup}
	case sourceNode != nil:
		source, err := d.decodeLongFormSource(key, sourceNode)
		if err != nil {
			return err
		}
		cmd.Source = *source
	}
	return nil
}

// decodeOpSugar parses the op: "OperationID#Variant" sugar (scalar or
// sequence form). A bare variant name expands to #/components/schemas/<name>;
// a variant starting with / is taken as an absolute document fragment.
func (d *cliManifestDecoder) decodeOpSugar(key string, node *yaml.Node) ([]CLICommandRoute, error) {
	var routeNodes []*yaml.Node
	switch node.Kind {
	case yaml.ScalarNode:
		routeNodes = []*yaml.Node{node}
	case yaml.SequenceNode:
		routeNodes = node.Content
	default:
		return nil, fmt.Errorf("line %d: command %q: op must be a string (\"OperationID#Variant\") or a sequence of them", node.Line, key)
	}
	if len(routeNodes) == 0 {
		return nil, fmt.Errorf("line %d: command %q: op sequence must not be empty", node.Line, key)
	}
	if len(routeNodes) > 1 {
		return nil, fmt.Errorf("line %d: command %q declares %d entries in op:; use a routes: map for same-operation request-variant dispatch (multi-operation dispatch is not part of v1)", node.Line, key, len(routeNodes))
	}

	var routes []CLICommandRoute
	seen := map[string]bool{}
	for _, routeNode := range routeNodes {
		route, raw, err := decodeCLIOpReference(key, routeNode)
		if err != nil {
			return nil, err
		}
		dedupeKey := route.OperationID + "#" + route.RequestVariant
		if seen[dedupeKey] {
			return nil, fmt.Errorf("line %d: command %q declares route %q twice", routeNode.Line, key, raw)
		}
		seen[dedupeKey] = true
		routes = append(routes, *route)
	}
	return routes, nil
}

func decodeCLIOpReference(key string, node *yaml.Node) (*CLICommandRoute, string, error) {
	raw, err := cliScalarString(node, "op")
	if err != nil {
		return nil, "", err
	}
	opID, variant, hasVariant := strings.Cut(raw, "#")
	opID = strings.TrimSpace(opID)
	if opID == "" {
		return nil, raw, fmt.Errorf("line %d: command %q: op %q is missing the operation id", node.Line, key, raw)
	}
	route := &CLICommandRoute{OperationID: opID}
	if hasVariant {
		variant = strings.TrimSpace(variant)
		switch {
		case variant == "":
			return nil, raw, fmt.Errorf("line %d: command %q: op %q has an empty variant after #", node.Line, key, raw)
		case strings.HasPrefix(variant, "/"):
			route.RequestVariant = "#" + variant
		default:
			route.RequestVariant = "#/components/schemas/" + variant
		}
	}
	return route, raw, nil
}

func (d *cliManifestDecoder) decodeDispatchRoutes(key string, node *yaml.Node) ([]CLICommandRoute, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q routes", key))
	if err != nil {
		return nil, err
	}
	if len(entries) < 2 {
		return nil, fmt.Errorf("line %d: command %q: routes must declare at least two route IDs", node.Line, key)
	}

	routes := make([]CLICommandRoute, 0, len(entries))
	seenIDs := map[string]bool{}
	seenVariants := map[string]string{}
	var operationID, defaultID string
	for _, routeEntry := range entries {
		id := routeEntry.Key.Value
		if !cliInputNamePattern.MatchString(id) {
			return nil, fmt.Errorf("line %d: command %q route ID %q is invalid (expected lowercase [a-z0-9] words starting with a letter, joined by single hyphens)", routeEntry.Key.Line, key, id)
		}
		if seenIDs[id] {
			return nil, fmt.Errorf("line %d: command %q declares route ID %q twice", routeEntry.Key.Line, key, id)
		}
		seenIDs[id] = true
		routeEntries, err := cliMapEntries(routeEntry.Value, fmt.Sprintf("command %q route %q", key, id))
		if err != nil {
			return nil, err
		}
		route := CLICommandRoute{ID: id, Label: cliHumanizeRouteID(id)}
		var opNode, presetNode *yaml.Node
		for _, entry := range routeEntries {
			switch entry.Key.Value {
			case "label":
				route.Label, err = cliScalarString(entry.Value, "route label")
				if err != nil {
					return nil, err
				}
				if strings.TrimSpace(route.Label) == "" {
					return nil, fmt.Errorf("line %d: command %q route %q label must be non-empty", entry.Value.Line, key, id)
				}
			case "op":
				opNode = entry.Value
			case "default":
				route.Default, err = cliScalarBool(entry.Value, "route default")
				if err != nil {
					return nil, err
				}
			case "selector":
				route.Selector, err = cliScalarString(entry.Value, "route selector")
				if err != nil {
					return nil, err
				}
				if !cliInputNamePattern.MatchString(route.Selector) {
					return nil, fmt.Errorf("line %d: command %q route %q selector %q is invalid (expected a flag name of lowercase [a-z0-9] words starting with a letter, joined by single hyphens)", entry.Value.Line, key, id, route.Selector)
				}
			case "preset":
				presetNode = entry.Value
			default:
				return nil, fmt.Errorf("line %d: command %q route %q has unknown key %q%s", entry.Key.Line, key, id, entry.Key.Value, cliDidYouMean(entry.Key.Value, []string{"label", "op", "default", "selector", "preset"}))
			}
		}
		if opNode == nil {
			return nil, fmt.Errorf("line %d: command %q route %q requires op", routeEntry.Value.Line, key, id)
		}
		parsed, _, err := decodeCLIOpReference(key, opNode)
		if err != nil {
			return nil, err
		}
		if parsed.RequestVariant == "" {
			return nil, fmt.Errorf("line %d: command %q route %q op must pin a request variant (OperationID#Variant)", opNode.Line, key, id)
		}
		route.OperationID = parsed.OperationID
		route.RequestVariant = parsed.RequestVariant
		if operationID == "" {
			operationID = route.OperationID
		} else if operationID != route.OperationID {
			return nil, fmt.Errorf("command %q: routes must all use one operation; route %q uses %q and route %q uses %q (multi-operation dispatch is not part of v1)", key, routes[0].ID, operationID, id, route.OperationID)
		}
		if firstID := seenVariants[route.RequestVariant]; firstID != "" {
			return nil, fmt.Errorf("command %q: routes %q and %q both pin request variant %q; each dispatch route must pin a distinct variant", key, firstID, id, cliRequestVariantName(route.RequestVariant))
		}
		seenVariants[route.RequestVariant] = id
		if route.Default {
			if defaultID != "" {
				return nil, fmt.Errorf("command %q: routes %q and %q both declare default: true; at most one default route is allowed", key, defaultID, id)
			}
			defaultID = id
		}
		if presetNode != nil {
			route.Presets, err = d.decodePresets(key, presetNode)
			if err != nil {
				return nil, err
			}
		}
		routes = append(routes, route)
	}
	return routes, nil
}

// cliHumanizeRouteID derives a display label from a route ID by title-casing
// its hyphen-separated words. IDs are validated against cliInputNamePattern
// before reaching here, but the split is guarded so an empty segment can
// never index out of range.
func cliHumanizeRouteID(id string) string {
	var words []string
	for _, part := range strings.Split(id, "-") {
		if part == "" {
			continue
		}
		words = append(words, strings.ToUpper(part[:1])+part[1:])
	}
	return strings.Join(words, " ")
}

func cliRequestVariantName(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[i+1:]
	}
	return strings.TrimPrefix(ref, "#")
}

// decodeLongFormSource decodes the source: escape hatch. It expresses exactly
// what the sugar forms expand to and is validated identically downstream.
func (d *cliManifestDecoder) decodeLongFormSource(key string, node *yaml.Node) (*CLICommandSource, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q source", key))
	if err != nil {
		return nil, err
	}

	source := &CLICommandSource{}
	var routesNode *yaml.Node
	for _, entry := range entries {
		switch entry.Key.Value {
		case "type":
			typ, err := cliScalarString(entry.Value, "source type")
			if err != nil {
				return nil, err
			}
			source.Type = CLICommandSourceType(typ)
		case "routes":
			routesNode = entry.Value
		case "note":
			if source.Note, err = cliScalarString(entry.Value, "source note"); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("line %d: command %q source has unknown key %q%s", entry.Key.Line, key, entry.Key.Value, cliDidYouMean(entry.Key.Value, []string{"type", "routes", "note"}))
		}
	}

	switch source.Type {
	case CLICommandSourceOperation:
		if routesNode == nil || routesNode.Kind != yaml.SequenceNode || len(routesNode.Content) == 0 {
			return nil, fmt.Errorf("line %d: command %q: source type operation requires a non-empty routes sequence", node.Line, key)
		}
		if len(routesNode.Content) > 1 {
			return nil, fmt.Errorf("line %d: command %q declares %d entries in source.routes; use the top-level routes: map for same-operation request-variant dispatch (multi-operation dispatch is not part of v1)", routesNode.Line, key, len(routesNode.Content))
		}
		seen := map[string]bool{}
		for _, routeNode := range routesNode.Content {
			routeEntries, err := cliMapEntries(routeNode, fmt.Sprintf("command %q route", key))
			if err != nil {
				return nil, err
			}
			route := CLICommandRoute{}
			for _, entry := range routeEntries {
				switch entry.Key.Value {
				case "operationId":
					if route.OperationID, err = cliScalarString(entry.Value, "operationId"); err != nil {
						return nil, err
					}
				case "requestVariant":
					if route.RequestVariant, err = cliScalarString(entry.Value, "requestVariant"); err != nil {
						return nil, err
					}
				default:
					return nil, fmt.Errorf("line %d: command %q route has unknown key %q%s", entry.Key.Line, key, entry.Key.Value, cliDidYouMean(entry.Key.Value, []string{"operationId", "requestVariant"}))
				}
			}
			if route.OperationID == "" {
				return nil, fmt.Errorf("line %d: command %q: route requires operationId", routeNode.Line, key)
			}
			if route.RequestVariant != "" && !strings.HasPrefix(route.RequestVariant, "#/") {
				return nil, fmt.Errorf("line %d: command %q: requestVariant %q must be a document fragment reference (e.g. #/components/schemas/Name)", routeNode.Line, key, route.RequestVariant)
			}
			dedupeKey := route.OperationID + "#" + route.RequestVariant
			if seen[dedupeKey] {
				return nil, fmt.Errorf("line %d: command %q declares the same route twice", routeNode.Line, key)
			}
			seen[dedupeKey] = true
			source.Routes = append(source.Routes, route)
		}
		if source.Note != "" {
			return nil, fmt.Errorf("line %d: command %q: note is only valid on planned sources", node.Line, key)
		}
	case CLICommandSourcePlanned:
		source.Note = strings.TrimSpace(source.Note)
		if source.Note == "" {
			return nil, fmt.Errorf("line %d: command %q: source type planned requires a non-empty note", node.Line, key)
		}
		if routesNode != nil {
			return nil, fmt.Errorf("line %d: command %q: planned sources do not take routes", node.Line, key)
		}
	case CLICommandSourceGroup, CLICommandSourceCustom:
		if routesNode != nil || source.Note != "" {
			return nil, fmt.Errorf("line %d: command %q: %s sources take no routes or note", node.Line, key, source.Type)
		}
	case "":
		return nil, fmt.Errorf("line %d: command %q: source requires a type (operation, planned, custom, or group)", node.Line, key)
	default:
		return nil, fmt.Errorf("line %d: command %q: unsupported source type %q (expected operation, planned, custom, or group)", node.Line, key, source.Type)
	}
	return source, nil
}

func (d *cliManifestDecoder) decodeInputs(cmdKey string, node *yaml.Node, isArg bool) ([]CLICommandInput, error) {
	kind := "flags"
	if isArg {
		kind = "args"
	}
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q %s", cmdKey, kind))
	if err != nil {
		return nil, err
	}

	var inputs []CLICommandInput
	for _, entry := range entries {
		name := entry.Key.Value
		if !cliInputNamePattern.MatchString(name) {
			return nil, fmt.Errorf("line %d: command %q: %s name %q is invalid (expected lowercase [a-z0-9] words starting with a letter, joined by single hyphens)", entry.Key.Line, cmdKey, strings.TrimSuffix(kind, "s"), name)
		}
		if !isArg && cliReservedFlagNames[name] {
			return nil, fmt.Errorf("line %d: command %q: flag name %q is reserved by the generated runtime (reserved: %s)", entry.Key.Line, cmdKey, name, cliCandidateList(cliMapKeys(cliReservedFlagNames)))
		}
		input, err := d.decodeInput(cmdKey, name, entry.Value, isArg)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, *input)
	}
	return inputs, nil
}

func (d *cliManifestDecoder) decodeInput(cmdKey, name string, node *yaml.Node, isArg bool) (*CLICommandInput, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q input %q", cmdKey, name))
	if err != nil {
		return nil, err
	}

	input := &CLICommandInput{ID: name, Name: name}
	var toNode *yaml.Node
	for _, entry := range entries {
		switch entry.Key.Value {
		case "to":
			toNode = entry.Value
		case "type":
			t, err := cliScalarString(entry.Value, "type")
			if err != nil {
				return nil, err
			}
			switch t {
			case "string", "int", "float", "bool":
				input.Type = t
			default:
				return nil, fmt.Errorf("line %d: command %q input %q: unsupported type %q (expected string, int, float, or bool)", entry.Value.Line, cmdKey, name, t)
			}
		case "required":
			b, err := cliScalarBool(entry.Value, "required")
			if err != nil {
				return nil, err
			}
			input.Required = b
			input.requiredExplicit = true
		case "variadic":
			b, err := cliScalarBool(entry.Value, "variadic")
			if err != nil {
				return nil, err
			}
			if !isArg && b {
				return nil, fmt.Errorf("line %d: command %q flag %q: variadic is only valid on the positional argument", entry.Value.Line, cmdKey, name)
			}
			input.Variadic = b
		case "shorthand":
			if isArg {
				return nil, fmt.Errorf("line %d: command %q arg %q: shorthand is only valid on flags", entry.Value.Line, cmdKey, name)
			}
			s, err := cliScalarString(entry.Value, "shorthand")
			if err != nil {
				return nil, err
			}
			if !cliShorthandPattern.MatchString(s) {
				return nil, fmt.Errorf("line %d: command %q flag %q: shorthand must be a single alphanumeric character", entry.Value.Line, cmdKey, name)
			}
			if owner, reserved := cliReservedFlagShorthands[s]; reserved {
				return nil, fmt.Errorf("line %d: command %q flag %q: shorthand -%s is reserved by persistent flag %q", entry.Value.Line, cmdKey, name, s, owner)
			}
			input.Shorthand = s
		case "summary":
			if input.Summary, err = cliScalarString(entry.Value, "summary"); err != nil {
				return nil, err
			}
		case "default":
			if isArg {
				return nil, fmt.Errorf("line %d: command %q arg %q: default is only valid on flags (a positional default would never be exercised)", entry.Value.Line, cmdKey, name)
			}
			v, err := cliDecodeValue(entry.Value)
			if err != nil {
				return nil, err
			}
			input.Default = v
		case "defaultFrom":
			if isArg {
				return nil, fmt.Errorf("line %d: command %q arg %q: defaultFrom is only valid on flags", entry.Value.Line, cmdKey, name)
			}
			v, err := cliScalarString(entry.Value, "defaultFrom")
			if err != nil {
				return nil, err
			}
			if v != "schema" {
				return nil, fmt.Errorf("line %d: command %q flag %q: defaultFrom only supports \"schema\"", entry.Value.Line, cmdKey, name)
			}
			input.DefaultFrom = v
		case "suggestions":
			values, err := cliDecodeInputSuggestions(cmdKey, name, entry.Value)
			if err != nil {
				return nil, err
			}
			input.Enum = values
			input.SuggestionsDeclared = true
		case "mode":
			return nil, fmt.Errorf("line %d: command %q input %q: mode lives inside the to: object and only \"set\" is part of v1 (append/merge are reserved capabilities)", entry.Key.Line, cmdKey, name)
		default:
			return nil, fmt.Errorf("line %d: command %q input %q has unknown key %q%s", entry.Key.Line, cmdKey, name, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliInputKeys))
		}
	}

	if input.Default != nil && input.DefaultFrom != "" {
		return nil, fmt.Errorf("command %q flag %q declares both default and defaultFrom; pick one", cmdKey, name)
	}

	if toNode == nil {
		return nil, fmt.Errorf("command %q input %q must declare to: (inputs without a binding require the request-plan.payload-holes capability, which is not part of v1)", cmdKey, name)
	}
	bind, err := d.decodeBind(cmdKey, name, toNode)
	if err != nil {
		return nil, err
	}
	input.Bind = bind
	return input, nil
}

// cliDecodeInputSuggestions parses an authored suggestions: list. The values
// replace the schema-derived suggestions in help (an empty list shows none);
// the linker checks them against the bound property's type.
func cliDecodeInputSuggestions(cmdKey, name string, node *yaml.Node) ([]any, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("line %d: command %q input %q: suggestions must be a list of scalar values", node.Line, cmdKey, name)
	}
	values := make([]any, 0, len(node.Content))
	for _, child := range node.Content {
		if child.Kind != yaml.ScalarNode {
			return nil, fmt.Errorf("line %d: command %q input %q: suggestions entries must be scalar values", child.Line, cmdKey, name)
		}
		value, err := cliDecodeValue(child)
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, fmt.Errorf("line %d: command %q input %q: suggestions entries cannot be null", child.Line, cmdKey, name)
		}
		if cliContainsValue(values, value) {
			return nil, fmt.Errorf("line %d: command %q input %q: suggestions lists %v more than once", child.Line, cmdKey, name, value)
		}
		values = append(values, value)
	}
	return values, nil
}

// decodeBind parses a to: value. The scalar form is a singular JSONPath body
// bind; the object form addresses non-body parameter targets (a reserved
// renderer capability in v1, validated then gated during linking).
func (d *cliManifestDecoder) decodeBind(cmdKey, name string, node *yaml.Node) (*CLICommandBind, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		raw, err := cliScalarString(node, "to")
		if err != nil {
			return nil, err
		}
		segments, err := cliParseSingularPath(raw)
		if err != nil {
			return nil, fmt.Errorf("line %d: command %q input %q: %w", node.Line, cmdKey, name, err)
		}
		if len(segments) > 2 {
			return nil, fmt.Errorf("line %d: command %q input %q: to: %s is a multi-segment body path; a flag may bind one level beneath an object preset ($.<preset>.<field>), deeper construction requires the request-plan.payload-holes capability and stays a hard error until it ships", node.Line, cmdKey, name, raw)
		}
		if segments[0].IsIndex {
			return nil, fmt.Errorf("line %d: command %q input %q: to: %s addresses an array index at the body root, which cannot be a JSON object field", node.Line, cmdKey, name, raw)
		}
		if len(segments) == 2 && segments[1].IsIndex {
			return nil, fmt.Errorf("line %d: command %q input %q: to: %s addresses an array index; a nested bind targets a field of an object preset", node.Line, cmdKey, name, raw)
		}
		return &CLICommandBind{In: "body", Pointer: cliSegmentsToPointer(segments), Mode: "set"}, nil
	case yaml.MappingNode:
		entries, err := cliMapEntries(node, fmt.Sprintf("command %q input %q to", cmdKey, name))
		if err != nil {
			return nil, err
		}
		bind := &CLICommandBind{Mode: "set"}
		var paramName string
		for _, entry := range entries {
			switch entry.Key.Value {
			case "in":
				if bind.In, err = cliScalarString(entry.Value, "to.in"); err != nil {
					return nil, err
				}
			case "name":
				if paramName, err = cliScalarString(entry.Value, "to.name"); err != nil {
					return nil, err
				}
			case "mode":
				mode, err := cliScalarString(entry.Value, "to.mode")
				if err != nil {
					return nil, err
				}
				if mode != "set" {
					return nil, fmt.Errorf("line %d: command %q input %q: bind mode %q is reserved for a future capability; v1 only executes set", entry.Value.Line, cmdKey, name, mode)
				}
			default:
				return nil, fmt.Errorf("line %d: command %q input %q to has unknown key %q%s", entry.Key.Line, cmdKey, name, entry.Key.Value, cliDidYouMean(entry.Key.Value, []string{"in", "name", "mode"}))
			}
		}
		switch bind.In {
		case "path", "query", "header":
			if paramName == "" {
				return nil, fmt.Errorf("line %d: command %q input %q: to: {in: %s} requires name: naming the operation parameter", node.Line, cmdKey, name, bind.In)
			}
			bind.Pointer = "/" + strings.ReplaceAll(strings.ReplaceAll(paramName, "~", "~0"), "/", "~1")
			return bind, nil
		case "body":
			return nil, fmt.Errorf("line %d: command %q input %q: body binds use the scalar form (to: $.field)", node.Line, cmdKey, name)
		case "":
			return nil, fmt.Errorf("line %d: command %q input %q: to: object requires in: (path, query, or header)", node.Line, cmdKey, name)
		default:
			return nil, fmt.Errorf("line %d: command %q input %q: unsupported bind location %q (expected path, query, or header)", node.Line, cmdKey, name, bind.In)
		}
	default:
		return nil, fmt.Errorf("line %d: command %q input %q: to must be a singular JSONPath string or a {in, name} object", node.Line, cmdKey, name)
	}
}

// decodeStreamProjection decodes output.stream. select is a singular
// JSONPath from the streamed event root; the linker checks it against the
// operation's stream schema.
func (d *cliManifestDecoder) decodeStreamProjection(cmdKey string, node *yaml.Node) (*CLICommandStreamProjection, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q output.stream", cmdKey))
	if err != nil {
		return nil, err
	}
	stream := &CLICommandStreamProjection{}
	for _, entry := range entries {
		switch entry.Key.Value {
		case "select":
			raw, err := cliScalarString(entry.Value, "output.stream.select")
			if err != nil {
				return nil, err
			}
			segments, err := cliParseSingularPath(raw)
			if err != nil {
				return nil, fmt.Errorf("line %d: command %q output.stream.select: %w", entry.Value.Line, cmdKey, err)
			}
			stream.Select = raw
			stream.Pointer = cliSegmentsToPointer(segments)
		case "metadata":
			metadata, err := d.decodeStreamMetadata(entry.Value)
			if err != nil {
				return nil, fmt.Errorf("command %q: %w", cmdKey, err)
			}
			stream.Metadata = metadata

		default:
			return nil, fmt.Errorf("line %d: command %q output.stream has unknown key %q%s", entry.Key.Line, cmdKey, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliOutputStreamKeys))
		}
	}
	if stream.Select == "" {
		return nil, fmt.Errorf("line %d: command %q: output.stream requires select: naming the event field to write (a singular JSONPath such as $.data.text)", node.Line, cmdKey)
	}
	return stream, nil
}

func (d *cliManifestDecoder) decodePresets(cmdKey string, node *yaml.Node) ([]CLICommandPreset, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q preset", cmdKey))
	if err != nil {
		return nil, err
	}

	var presets []CLICommandPreset
	seenPointers := map[string]string{}
	for _, entry := range entries {
		segments, err := cliParseSingularPath(entry.Key.Value)
		if err != nil {
			return nil, fmt.Errorf("line %d: command %q preset key: %w", entry.Key.Line, cmdKey, err)
		}
		if len(segments) > 1 {
			return nil, fmt.Errorf("line %d: command %q preset key %q: presets are root-level in v1 (nested construction requires the request-plan.payload-holes capability)", entry.Key.Line, cmdKey, entry.Key.Value)
		}
		if segments[0].IsIndex {
			return nil, fmt.Errorf("line %d: command %q preset key %q addresses an array index at the body root, which cannot be a JSON object field", entry.Key.Line, cmdKey, entry.Key.Value)
		}
		pointer := cliSegmentsToPointer(segments)
		if first, duplicate := seenPointers[pointer]; duplicate {
			return nil, fmt.Errorf("line %d: command %q preset key %q duplicates %q after both normalize to pointer %q", entry.Key.Line, cmdKey, entry.Key.Value, first, pointer)
		}
		seenPointers[pointer] = entry.Key.Value
		value, err := cliDecodeValue(entry.Value)
		if err != nil {
			return nil, err
		}
		if value == nil {
			return nil, fmt.Errorf("line %d: command %q preset %q: null presets are not supported (remove the key instead)", entry.Value.Line, cmdKey, entry.Key.Value)
		}
		presets = append(presets, CLICommandPreset{
			Bind:  CLICommandBind{In: "body", Pointer: pointer, Mode: "set"},
			Value: value,
		})
	}
	return presets, nil
}

func (d *cliManifestDecoder) decodeHints(cmdKey string, node *yaml.Node) (map[string][]string, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q hints", cmdKey))
	if err != nil {
		return nil, err
	}

	hints := map[string][]string{}
	for _, entry := range entries {
		reason := entry.Key.Value
		if !cliHintReasonPattern.MatchString(reason) {
			return nil, fmt.Errorf("line %d: command %q hint reason %q is invalid (expected an UPPER_SNAKE_CASE reason such as RESOURCE_EXHAUSTED)", entry.Key.Line, cmdKey, reason)
		}
		if strings.HasPrefix(reason, "CLI_") && !cliContains(CLIRuntimeHintReasons, reason) {
			return nil, fmt.Errorf("line %d: command %q hint reason %q is not a reason the generated runtime emits (CLI_* namespace: %s)", entry.Key.Line, cmdKey, reason, cliCandidateList(CLIRuntimeHintReasons))
		}
		if !strings.HasPrefix(reason, "CLI_") {
			d.warnf("command %q declares a hint for external reason %q; it will fire only when the server reports that reason", cmdKey, reason)
		}

		var lines []string
		switch entry.Value.Kind {
		case yaml.ScalarNode:
			line, err := cliScalarString(entry.Value, "hint")
			if err != nil {
				return nil, err
			}
			lines = []string{line}
		case yaml.SequenceNode:
			for _, item := range entry.Value.Content {
				line, err := cliScalarString(item, "hint")
				if err != nil {
					return nil, err
				}
				lines = append(lines, line)
			}
		default:
			return nil, fmt.Errorf("line %d: command %q hint %q must be a string or a sequence of strings", entry.Value.Line, cmdKey, reason)
		}
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				return nil, fmt.Errorf("line %d: command %q hint %q contains an empty line", entry.Value.Line, cmdKey, reason)
			}
		}
		if len(lines) == 0 {
			return nil, fmt.Errorf("line %d: command %q hint %q must contain at least one line", entry.Value.Line, cmdKey, reason)
		}
		hints[reason] = lines
	}
	return hints, nil
}

func (d *cliManifestDecoder) decodeExamples(cmdKey string, node *yaml.Node) ([]CLICommandExample, error) {
	entries, err := cliMapEntries(node, fmt.Sprintf("command %q examples", cmdKey))
	if err != nil {
		return nil, err
	}

	var examples []CLICommandExample
	for _, entry := range entries {
		slug := entry.Key.Value
		if !cliExampleSlugRegexp.MatchString(slug) {
			return nil, fmt.Errorf("line %d: command %q example slug %q is invalid (expected [a-z0-9-]+)", entry.Key.Line, cmdKey, slug)
		}
		example := CLICommandExample{}
		switch entry.Value.Kind {
		case yaml.ScalarNode:
			if example.Command, err = cliScalarString(entry.Value, "example"); err != nil {
				return nil, err
			}
		case yaml.MappingNode:
			exampleEntries, err := cliMapEntries(entry.Value, fmt.Sprintf("command %q example %q", cmdKey, slug))
			if err != nil {
				return nil, err
			}
			for _, exampleEntry := range exampleEntries {
				switch exampleEntry.Key.Value {
				case "summary":
					if example.Summary, err = cliScalarString(exampleEntry.Value, "example summary"); err != nil {
						return nil, err
					}
				case "command":
					if example.Command, err = cliScalarString(exampleEntry.Value, "example command"); err != nil {
						return nil, err
					}
				default:
					return nil, fmt.Errorf("line %d: command %q example %q has unknown key %q%s", exampleEntry.Key.Line, cmdKey, slug, exampleEntry.Key.Value, cliDidYouMean(exampleEntry.Key.Value, []string{"summary", "command"}))
				}
			}
		default:
			return nil, fmt.Errorf("line %d: command %q example %q must be a command string or a {summary, command} object", entry.Value.Line, cmdKey, slug)
		}
		if strings.TrimSpace(example.Command) == "" {
			return nil, fmt.Errorf("line %d: command %q example %q must declare a non-empty command", entry.Value.Line, cmdKey, slug)
		}
		examples = append(examples, example)
	}
	if len(examples) > 3 {
		d.warnf("command %q declares %d examples; the help renderer shows at most 3", cmdKey, len(examples))
	}
	return examples, nil
}

// checkCommandInputs enforces the per-command input contract that does not
// need schema access: positional arity, duplicate names, duplicate bind
// pointers, and shorthand collisions.
func (d *cliManifestDecoder) checkCommandInputs(cmd *CLICommand, key string) error {
	if len(cmd.Args) > 1 {
		return fmt.Errorf("command %q declares %d positional arguments; v1 permits exactly one, and it must be variadic", key, len(cmd.Args))
	}
	if len(cmd.Args) == 1 && !cmd.Args[0].Variadic {
		return fmt.Errorf("command %q arg %q must declare variadic: true (the CLI joins argv into a single value; declaring otherwise would lie)", key, cmd.Args[0].Name)
	}

	names := map[string]bool{}
	shorthands := map[string]string{}
	pointers := map[string]string{}
	for _, list := range [][]CLICommandInput{cmd.Args, cmd.Flags} {
		for _, input := range list {
			if names[input.Name] {
				return fmt.Errorf("command %q declares input %q more than once across args and flags", key, input.Name)
			}
			names[input.Name] = true
			if input.Shorthand != "" {
				if prev, ok := shorthands[input.Shorthand]; ok {
					return fmt.Errorf("command %q: flags %q and %q both use shorthand -%s", key, prev, input.Name, input.Shorthand)
				}
				shorthands[input.Shorthand] = input.Name
			}
			if input.Bind != nil && input.Bind.In == "body" {
				if prev, ok := pointers[input.Bind.Pointer]; ok {
					return fmt.Errorf("command %q: inputs %q and %q both bind %s", key, prev, input.Name, input.Bind.Pointer)
				}
				pointers[input.Bind.Pointer] = input.Name
			}
		}
	}
	for _, list := range [][]CLICommandInput{cmd.Args, cmd.Flags} {
		for _, input := range list {
			if input.Bind == nil || input.Bind.In != "body" {
				continue
			}
			parent, _, nested := cliSplitNestedPointer(input.Bind.Pointer)
			if !nested {
				continue
			}
			if prev, ok := pointers[parent]; ok {
				return fmt.Errorf("command %q: input %q binds %s and input %q binds %s beneath it; bind the object or its field, not both", key, prev, parent, input.Name, input.Bind.Pointer)
			}
		}
	}
	return nil
}

func cliContains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

func cliMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (d *cliManifestDecoder) decodeStreamMetadata(node *yaml.Node) ([]CLICommandStreamMetadata, error) {
	items := []*yaml.Node{node}
	if node.Kind == yaml.SequenceNode {
		if len(node.Content) == 0 {
			return nil, fmt.Errorf("line %d: output.stream.metadata must list at least one field", node.Line)
		}
		items = node.Content
	}

	fields := make([]CLICommandStreamMetadata, 0, len(items))
	labels := map[string]bool{}
	statusLine := 0
	incompleteLine := 0
	for _, item := range items {
		field, err := d.decodeStreamMetadataField(item)
		if err != nil {
			return nil, err
		}
		if labels[field.Label] {
			return nil, fmt.Errorf("line %d: output.stream.metadata label %q is declared more than once", item.Line, field.Label)
		}
		labels[field.Label] = true
		if field.Complete != nil {
			if statusLine != 0 {
				return nil, fmt.Errorf("line %d: output.stream.metadata declares complete on more than one field; only the stream's status field lists its complete values", item.Line)
			}
			statusLine = item.Line
		}
		if field.When != "" && incompleteLine == 0 {
			incompleteLine = item.Line
		}
		fields = append(fields, field)
	}
	if incompleteLine != 0 && statusLine == 0 {
		return nil, fmt.Errorf("line %d: output.stream.metadata when: incomplete requires a status field that lists its complete values", incompleteLine)
	}
	return fields, nil
}

func (d *cliManifestDecoder) decodeStreamMetadataField(node *yaml.Node) (CLICommandStreamMetadata, error) {
	var field CLICommandStreamMetadata
	entries, err := cliMapEntries(node, "output.stream.metadata")
	if err != nil {
		return field, err
	}
	for _, entry := range entries {
		switch entry.Key.Value {
		case "select":
			paths, err := cliScalarOrStringList(entry.Value, "output.stream.metadata.select")
			if err != nil {
				return field, err
			}
			for _, path := range paths {
				segments, err := cliParseSingularPath(path)
				if err != nil {
					return field, fmt.Errorf("line %d: output.stream.metadata.select: %w", entry.Value.Line, err)
				}
				field.Select = append(field.Select, path)
				field.Pointers = append(field.Pointers, cliSegmentsToPointer(segments))
			}
		case "label":
			value, err := cliScalarString(entry.Value, "output.stream.metadata.label")
			if err != nil {
				return field, err
			}
			for _, r := range value {
				if !unicode.IsPrint(r) {
					return field, fmt.Errorf("line %d: output.stream.metadata.label must contain only printable characters", entry.Value.Line)
				}
			}
			field.Label = value
		case "complete":
			values, err := cliScalarOrStringList(entry.Value, "output.stream.metadata.complete")
			if err != nil {
				return field, err
			}
			field.Complete = values
		case "when":
			value, err := cliScalarString(entry.Value, "output.stream.metadata.when")
			if err != nil {
				return field, err
			}
			if value != "incomplete" {
				return field, fmt.Errorf("line %d: output.stream.metadata.when must be incomplete, got %q", entry.Value.Line, value)
			}
			field.When = value
		default:
			return field, fmt.Errorf("line %d: output.stream.metadata has unknown key %q%s", entry.Key.Line, entry.Key.Value, cliDidYouMean(entry.Key.Value, cliOutputStreamMetadataKeys))
		}
	}
	if len(field.Select) == 0 || strings.TrimSpace(field.Label) == "" {
		return field, fmt.Errorf("line %d: output.stream.metadata requires nonempty select and label", node.Line)
	}
	if field.Complete != nil && field.When != "" {
		return field, fmt.Errorf("line %d: output.stream.metadata field %q lists complete values, so it is already reported only when incomplete; remove when", node.Line, field.Label)
	}
	return field, nil
}
