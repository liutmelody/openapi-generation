package changes

import (
	"github.com/speakeasy-api/openapi-generation/v2/internal/ast"
	"github.com/speakeasy-api/openapi-generation/v2/internal/subsystem"
)

// ChangeType represents the type of change detected
type ChangeType string

const (
	MethodAdded                 ChangeType = "method_added"
	MethodDeleted               ChangeType = "method_deleted"
	MethodDeprecated            ChangeType = "method_deprecated"
	ArgumentsChanged            ChangeType = "arguments_changed"
	ResponseChanged             ChangeType = "response_changed"
	ArgumentsAndResponseChanged ChangeType = "arguments_and_response_changed"
)

// MethodDiff represents a single change detected between ASTs
type MethodDiff struct {
	MethodParts         []string          `json:"method_parts"`          // e.g., ["stripe", "foo", "bar", "baz"]
	Operation           *ast.Operation    `json:"-"`                     // Reference to the operation (not serialized)
	MethodKey           string            `json:"method_key"`            // Unique key for the method
	Type                ChangeType        `json:"type"`                  // added, removed, deprecated, modified
	ArgumentsDiff       TypeDefDiffResult `json:"arguments_diff"`        // Result of arguments comparison
	SuccessResponseDiff TypeDefDiffResult `json:"success_response_diff"` // Result of success response comparison
	ErrorResponseDiff   TypeDefDiffResult `json:"error_response_diff"`   // Result of error response comparison
}

// DiffOptions contains the configuration for AST comparison
type DiffOptions struct {
	OldAST       *ast.AST
	NewAST       *ast.AST
	OldSubsystem *subsystem.Subsystem
	NewSubsystem *subsystem.Subsystem
}

// DiffReason represents why two TypeDefs differ
type DiffReason string

const (
	DiffReasonKind                      DiffReason = "kind"                        // Different type kinds (e.g., class vs array)
	DiffReasonFieldAdded                DiffReason = "field_added"                 // Field was added
	DiffReasonFieldRemoved              DiffReason = "field_removed"               // Field was removed
	DiffReasonFieldChanged              DiffReason = "field_changed"               // Single field changed
	DiffReasonFieldDeprecated           DiffReason = "field_deprecated"            // Field was marked deprecated
	DiffReasonFieldsChanged             DiffReason = "fields_changed"              // Multiple fields changed
	DiffReasonUnionOptionAdded          DiffReason = "union_option_added"          // Union option was added
	DiffReasonUnionOptionRemoved        DiffReason = "union_option_removed"        // Union option was removed
	DiffReasonUnionOptionChanged        DiffReason = "union_option_changed"        // Single union option changed
	DiffReasonUnionChanged              DiffReason = "union_changed"               // Union changed
	DiffReasonUnionDiscriminatorAdded   DiffReason = "union_discriminator_added"   // Type became part of a discriminated union
	DiffReasonUnionDiscriminatorRemoved DiffReason = "union_discriminator_removed" // Union lost its discriminator
	DiffReasonEnumValueAdded            DiffReason = "enum_value_added"            // Enum value was added
	DiffReasonEnumValueRemoved          DiffReason = "enum_value_removed"          // Enum value was removed
)

// PathSegmentType represents the type of a path segment
type PathSegmentType string

const (
	PathSegmentTypeField           PathSegmentType = "field"
	PathSegmentTypeUnionOption     PathSegmentType = "union_option"
	PathSegmentTypeArrayItem       PathSegmentType = "array_item"
	PathSegmentTypeMapItem         PathSegmentType = "map_item"
	PathSegmentTypeEnumValue       PathSegmentType = "enum_value"
	PathSegmentResponseStatus      PathSegmentType = "response_status"
	PathSegmentResponseContentType PathSegmentType = "response_content_type"
)

// DiffChangeType represents the type of change detected
type DiffChangeType string

const (
	DiffChangeTypeAdded        DiffChangeType = "added"
	DiffChangeTypeRemoved      DiffChangeType = "removed"
	DiffChangeTypeChanged      DiffChangeType = "changed"
	DiffChangeTypeTypeMismatch DiffChangeType = "type_mismatch"
)

// PathSegment represents a single segment in a path
type PathSegment struct {
	Type PathSegmentType `json:"type"` // The type of segment
	Name string          `json:"name"` // The name of the field, union option, status code, etc.
}

// Represents a collection of PathSegment
type PathSegments []PathSegment

// Returns a copy of PathSegments with the given segment appended.
func (p PathSegments) Append(segment PathSegment) PathSegments {
	result := make(PathSegments, len(p), len(p)+1)
	copy(result, p)
	result = append(result, segment)

	return result
}

// TypeDefDiffResult holds the outcome of diffing two AST TypeDef trees.
type TypeDefDiffResult struct {
	Equal      bool                `json:"equal"`       // true if completely identical
	Path       []PathSegment       `json:"path"`        // the highest‐ancestor path at which they differ
	Reason     DiffReason          `json:"reason"`      // why they differ (only set when Equal is false)
	IsBreaking bool                `json:"is_breaking"` // true if the change is breaking
	Children   []TypeDefDiffResult `json:"children"`    // all child differences found during traversal
}

// DetailLevel controls how much detail is shown in changelog output
type DetailLevel string

const (
	DetailLevelCompact DetailLevel = "compact" // Just parent: "request **Changed**"
	DetailLevelFull    DetailLevel = "full"    // Parent + all leaf children as sub-bullets
)
