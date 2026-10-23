package changes

import (
	"fmt"
	"strings"

	"github.com/speakeasy-api/openapi-generation/v2/internal/ast"
	"github.com/speakeasy-api/openapi/sequencedmap"
)

// DiffTypeDefsParams provides parameters for the diff operation
type DiffTypeDefsParams struct {
	A         *ast.TypeDef  // First typedef to compare
	B         *ast.TypeDef  // Second typedef to compare
	Path      PathSegments  // Current path in the tree
	IsRequest bool          // Whether this is comparing request types (affects breaking change detection)
	seen      map[pair]bool // Internal tracking of visited nodes for circular reference detection
}

type pair struct{ x, y *ast.TypeDef }

// DiffTypeDefs does a DFS over two ast.TypeDef roots, detects circular refs,
// and returns the highest‐ancestor path where they first diverge with proper reason categorization.
func DiffTypeDefs(params DiffTypeDefsParams) TypeDefDiffResult {
	params.seen = make(map[pair]bool)
	params.Path = PathSegments{}
	return diffTypeDefs(params)
}

// diffTypeDefs performs DFS traversal for typedef comparison
func diffTypeDefs(params DiffTypeDefsParams) TypeDefDiffResult {
	if params.A == nil || params.B == nil {
		return TypeDefDiffResult{Equal: true}
	}

	key := pair{params.A, params.B}
	if params.seen[key] {
		return TypeDefDiffResult{Equal: true}
	}
	params.seen[key] = true

	ret := diffKind(&params)
	if !ret.Equal {
		return ret
	}

	for _, child := range diffTypeDefsShallow(&params) {
		if child.params != nil {
			// We recurse into the child if the field exists on both a and b
			childDeepDiff := diffTypeDefs(*child.params)
			// Merge the shallow diff with the deep diff
			childType := ast.DataTypeAny
			if child.params.B != nil {
				childType = child.params.B.Type
			}
			child.diff = mergeTypeDefDiffResult(child.diff, childDeepDiff, childType)
		}
		parentType := ast.DataTypeAny
		if params.B != nil {
			parentType = params.B.Type
		}
		ret = mergeTypeDefDiffResult(ret, child.diff, parentType)
	}

	return ret
}

// shallowDiffResult diffs shallow things like added, removed, nullability and `params`
// dictates whether we should recurse if the field exists on both a and b
type shallowDiffResult struct {
	diff   TypeDefDiffResult
	params *DiffTypeDefsParams
}

// diffTypeDefsShallow returns shallow diffs and child parameters for recursion
func diffTypeDefsShallow(params *DiffTypeDefsParams) []shallowDiffResult {
	switch {
	case params.A.AssociatedTypes != nil:
		return diffUnionsShallow(params)
	case params.A.ItemType != nil:
		return diffContainerShallow(params)
	case params.A.Enum != nil && params.B.Enum != nil:
		return diffEnumsShallow(params)
	case params.A.IsTypeWithFields():
		return diffFieldsShallow(params)
	}
	return nil
}

// diffFieldsShallow handles shallow field comparison and returns diffs/children
func diffFieldsShallow(params *DiffTypeDefsParams) []shallowDiffResult {
	aMap := makeFieldMap(params.A.Fields)
	bMap := makeFieldMap(params.B.Fields)
	results := make([]shallowDiffResult, 0, len(params.A.Fields)+len(params.B.Fields))

	// Create one shallowDiffResult for each added field
	for name, bField := range bMap.All() {
		if _, exists := aMap.Get(name); !exists {
			isBreaking := params.IsRequest && !bField.Optional
			diff := TypeDefDiffResult{
				Equal:      false,
				Path:       params.Path.Append(PathSegment{Type: PathSegmentTypeField, Name: name}),
				IsBreaking: isBreaking,
				Reason:     DiffReasonFieldAdded,
			}
			results = append(results, shallowDiffResult{diff: diff})
		}
	}

	// Create one shallowDiffResult for each removed field
	for name := range aMap.All() {
		if _, exists := bMap.Get(name); !exists {
			diff := TypeDefDiffResult{
				Equal:      false,
				Path:       params.Path.Append(PathSegment{Type: PathSegmentTypeField, Name: name}),
				IsBreaking: true, // Removing fields is always breaking
				Reason:     DiffReasonFieldRemoved,
			}
			results = append(results, shallowDiffResult{diff: diff})
		}
	}

	// Process fields that exist in both
	for name, aField := range aMap.All() {
		bField, exists := bMap.Get(name)
		if !exists {
			continue
		}

		childPath := params.Path.Append(PathSegment{Type: PathSegmentTypeField, Name: name})

		// Check field metadata changes
		equal, breaking := diffFieldDefs(aField, bField, params)
		diff := TypeDefDiffResult{Equal: true}
		switch {
		case !equal:
			diff = TypeDefDiffResult{
				Equal:      false,
				Path:       childPath,
				Reason:     DiffReasonFieldChanged,
				IsBreaking: breaking,
			}
		case !isFieldDeprecated(aField) && isFieldDeprecated(bField):
			diff = TypeDefDiffResult{
				Equal:  false,
				Path:   childPath,
				Reason: DiffReasonFieldDeprecated,
			}
		}

		// Always create child params for recursion
		childParams := &DiffTypeDefsParams{
			A:         aField.Type,
			B:         bField.Type,
			Path:      childPath,
			IsRequest: params.IsRequest,
			seen:      params.seen,
		}

		results = append(results, shallowDiffResult{diff: diff, params: childParams})
	}

	return results
}

// diffUnionsShallow handles shallow union comparison and returns diffs/children
func diffUnionsShallow(params *DiffTypeDefsParams) []shallowDiffResult {
	// Build maps: name -> index
	aMap := make(map[string]int)
	bMap := make(map[string]int)
	results := make([]shallowDiffResult, 0, len(params.A.AssociatedTypes)+len(params.B.AssociatedTypes))

	// Check if union gained a discriminator
	if params.A.Discriminator == nil && params.B.Discriminator != nil {
		diff := TypeDefDiffResult{
			Equal: false,
			Path:  params.Path,
			// Unclear if this is actually going to be breaking in the end, depends on target implementation
			IsBreaking: true,
			Reason:     DiffReasonUnionDiscriminatorAdded,
		}
		results = append(results, shallowDiffResult{diff: diff})
	}

	// Check if union lost a discriminator (breaking for responses: SDK may lose type narrowing capability)
	if params.A.Discriminator != nil && params.B.Discriminator == nil {
		diff := TypeDefDiffResult{
			Equal: false,
			Path:  params.Path,
			// Unclear if this is actually going to be breaking in the end, depends on target implementation
			IsBreaking: true,
			Reason:     DiffReasonUnionDiscriminatorRemoved,
		}
		results = append(results, shallowDiffResult{diff: diff})
	}

	for i := range params.A.AssociatedTypes {
		id := getUnionOptionId(params.A, i)
		aMap[id] = i
	}
	for i := range params.B.AssociatedTypes {
		id := getUnionOptionId(params.B, i)
		bMap[id] = i
	}

	// Create one shallowDiffResult for each added union option
	for id := range bMap {
		if _, exists := aMap[id]; !exists {
			// Adding union option to request: NOT breaking (server accepts more options)
			// Adding union option to response: BREAKING (old SDKs can't deserialize new variants)
			// EXCEPT: if the old union was open (allows unknown options), then adding to response is NOT breaking
			isBreaking := !params.IsRequest && !params.A.IsUnionOpen
			diff := TypeDefDiffResult{
				Equal:      false,
				Path:       params.Path.Append(PathSegment{Type: PathSegmentTypeUnionOption, Name: id}),
				IsBreaking: isBreaking,
				Reason:     DiffReasonUnionOptionAdded,
			}
			results = append(results, shallowDiffResult{diff: diff})
		}
	}

	// Create one shallowDiffResult for each removed union option
	for id := range aMap {
		if _, exists := bMap[id]; !exists {
			// Removing union option is always breaking (SDK code using that option breaks)
			diff := TypeDefDiffResult{
				Equal:      false,
				Path:       params.Path.Append(PathSegment{Type: PathSegmentTypeUnionOption, Name: id}),
				IsBreaking: true,
				Reason:     DiffReasonUnionOptionRemoved,
			}
			results = append(results, shallowDiffResult{diff: diff})
		}
	}

	// Process options that exist in both
	for name, aIdx := range aMap {
		bIdx, exists := bMap[name]
		if !exists {
			continue
		}

		childPath := params.Path.Append(PathSegment{Type: PathSegmentTypeUnionOption, Name: name})

		childParams := &DiffTypeDefsParams{
			A:         params.A.AssociatedTypes[aIdx],
			B:         params.B.AssociatedTypes[bIdx],
			Path:      childPath,
			IsRequest: params.IsRequest,
			seen:      params.seen,
		}

		results = append(results, shallowDiffResult{
			diff:   TypeDefDiffResult{Equal: true},
			params: childParams,
		})
	}

	return results
}

// diffContainerShallow handles shallow container comparison and returns diffs/children
func diffContainerShallow(params *DiffTypeDefsParams) []shallowDiffResult {
	var results []shallowDiffResult

	// Check if container structures differ
	if (params.A.ItemType == nil) != (params.B.ItemType == nil) {
		// One has an item type and the other doesn't - this is a type mismatch
		diff := TypeDefDiffResult{
			Equal:      false,
			Path:       params.Path,
			Reason:     DiffReasonKind,
			IsBreaking: true,
		}
		results = append(results, shallowDiffResult{diff: diff})
		return results // Don't recurse if structure differs
	}

	// Both have item types - queue up for recursion
	if params.A.ItemType != nil && params.B.ItemType != nil {
		seg := PathSegment{PathSegmentTypeArrayItem, getTypeDisplayName(params.B.ItemType)}
		if params.A.Type == ast.DataTypeMap {
			// Include the value type name for maps
			valueTypeName := getTypeDisplayName(params.B.ItemType)
			seg = PathSegment{PathSegmentTypeMapItem, valueTypeName}
		}

		childPath := params.Path.Append(seg)
		childParams := &DiffTypeDefsParams{
			A:         params.A.ItemType,
			B:         params.B.ItemType,
			Path:      childPath,
			IsRequest: params.IsRequest,
			seen:      params.seen,
		}
		results = append(results, shallowDiffResult{
			diff:   TypeDefDiffResult{Equal: true},
			params: childParams,
		})
	}

	return results
}

// diffEnumsShallow handles shallow enum comparison and returns diffs for added/removed values
func diffEnumsShallow(params *DiffTypeDefsParams) []shallowDiffResult {
	if params.A.Enum == nil || params.B.Enum == nil {
		return nil
	}

	aValues := make(map[string]bool)
	bValues := make(map[string]bool)
	results := make([]shallowDiffResult, 0)

	for _, v := range params.A.Enum.Values {
		aValues[v] = true
	}
	for _, v := range params.B.Enum.Values {
		bValues[v] = true
	}

	// Check for added enum values
	for v := range bValues {
		if !aValues[v] {
			// Adding enum value to request is NOT breaking (server accepts more values)
			// Adding enum value to response IS breaking (old SDK can't handle new values)
			// EXCEPT: if the old enum was Open (allows unknown values), then adding to response is NOT breaking
			isBreaking := !params.IsRequest && !params.A.Enum.Open
			diff := TypeDefDiffResult{
				Equal:      false,
				Path:       params.Path.Append(PathSegment{Type: PathSegmentTypeEnumValue, Name: v}),
				IsBreaking: isBreaking,
				Reason:     DiffReasonEnumValueAdded,
			}
			results = append(results, shallowDiffResult{diff: diff})
		}
	}

	// Check for removed enum values
	for v := range aValues {
		if !bValues[v] {
			// Removing enum value is always breaking (SDK code using that value will break)
			diff := TypeDefDiffResult{
				Equal:      false,
				Path:       params.Path.Append(PathSegment{Type: PathSegmentTypeEnumValue, Name: v}),
				IsBreaking: true,
				Reason:     DiffReasonEnumValueRemoved,
			}
			results = append(results, shallowDiffResult{diff: diff})
		}
	}

	return results
}

func isFieldDeprecated(field *ast.FieldDef) bool {
	return field.Comments != nil && field.Comments.Deprecated
}

// diffFieldDefs compares two field definitions and returns whether they're equal and if any change is breaking
func diffFieldDefs(a, b *ast.FieldDef, params *DiffTypeDefsParams) (equal bool, isBreaking bool) {
	if a == nil || b == nil {
		return a == b, false
	}

	// Check basic equality
	if a.Name != b.Name {
		return false, true
	}

	// Track if fields are equal
	equal = true
	isBreaking = false

	// Check nullable differences
	if a.Nullable != b.Nullable {
		equal = false
		// Making a field non-nullable is breaking for requests
		// Making a field nullable is breaking for responses
		isBreaking = !params.IsRequest || a.Nullable || !b.Nullable
	}

	// Check optional differences
	if a.Optional != b.Optional {
		equal = false
		// Making a field required is breaking for requests
		if params.IsRequest && a.Optional && !b.Optional {
			isBreaking = true
		}
	}

	// Check other properties
	if (a.Default == nil) != (b.Default == nil) {
		equal = false
	}
	if (a.Const == nil) != (b.Const == nil) {
		equal = false
	}
	if len(a.Annotations) != len(b.Annotations) {
		equal = false
	} else {
		for i := range a.Annotations {
			if !a.Annotations[i].IsEqual(b.Annotations[i]) {
				equal = false
				break
			}
		}
	}

	return equal, isBreaking
}

// diffKind checks if two TypeDefs have the same kind (nil check and type check)
func diffKind(params *DiffTypeDefsParams) TypeDefDiffResult {
	if params.A == nil && params.B == nil {
		return TypeDefDiffResult{Equal: true}
	}

	if params.A == nil || params.B == nil || params.A.Type != params.B.Type {
		ret := TypeDefDiffResult{
			Equal:      false,
			Path:       params.Path,
			Reason:     DiffReasonKind,
			IsBreaking: true,
		}
		return ret
	}

	return TypeDefDiffResult{Equal: true}
}

// mergeTypeDefDiffResult merges two TypeDefDiffResult values
func mergeTypeDefDiffResult(a, b TypeDefDiffResult, dataType ast.DataType) TypeDefDiffResult {
	if a.Equal {
		return b
	}
	if b.Equal {
		return a
	}

	// If both have the same path but different reasons, merge them
	// This can happen if there was a shallow field difference
	// and a deep diff eg shallowDiff=optional, deepDiff=....
	if pathsEqual(a.Path, b.Path) {
		isUnion := dataType == ast.DataTypeUnion
		if a.Reason != b.Reason {
			b.Reason = DiffReasonFieldChanged
			if isUnion {
				b.Reason = DiffReasonUnionChanged
			}
		}

		if a.IsBreaking || b.IsBreaking {
			b.IsBreaking = true
		}

		if len(a.Children) > len(b.Children) {
			b.Children = a.Children
		}

		return b
	}

	// Calculate common prefix first
	commonPath := commonPathPrefix(a.Path, b.Path)

	// Ensure a is always the shallower diff
	if len(a.Path) > len(b.Path) {
		a, b = b, a
	}

	// Check if both diffs are deeper than the common prefix (siblings at the same level)
	aIsDeeper := len(a.Path) > len(commonPath)
	bIsDeeper := len(b.Path) > len(commonPath)

	result := TypeDefDiffResult{
		Equal:      a.Equal && b.Equal,
		Path:       commonPath,
		IsBreaking: a.IsBreaking || b.IsBreaking,
	}

	// Collect children
	switch {
	case aIsDeeper && bIsDeeper:
		// Both are siblings - add both as children
		result.Children = append(result.Children, a)
		result.Children = append(result.Children, b)
	case bIsDeeper:
		// Only b is deeper - a is at or above common prefix
		result.Children = append(result.Children, a.Children...)
		result.Children = append(result.Children, b)
	default:
		// Neither is deeper, just merge children
		result.Children = append(result.Children, a.Children...)
		result.Children = append(result.Children, b.Children...)
	}

	isUnion := a.Reason == DiffReasonUnionChanged || a.Reason == DiffReasonUnionOptionAdded || a.Reason == DiffReasonUnionOptionRemoved

	// Determine the reason
	result.Reason = DiffReasonFieldsChanged
	if isUnion {
		result.Reason = DiffReasonUnionChanged
	}

	return result
}

// Helper functions

func makeFieldMap(fields ast.Fields) *sequencedmap.Map[string, *ast.FieldDef] {
	m := sequencedmap.New[string, *ast.FieldDef]()
	for _, f := range fields {
		m.Set(f.Name, f)
	}
	return m
}

func commonPathPrefix(a, b []PathSegment) []PathSegment {
	n := minInt(len(a), len(b))
	i := 0
	for i < n && a[i].Type == b[i].Type && a[i].Name == b[i].Name {
		i++
	}
	return a[:i]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func pathsEqual(a, b []PathSegment) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type || a[i].Name != b[i].Name {
			return false
		}
	}
	return true
}

// getUnionOptionId returns the id for a union option (for diffing)
func getUnionOptionId(union *ast.TypeDef, index int) string {
	if index >= len(union.AssociatedTypes) {
		return ""
	}

	t := union.AssociatedTypes[index]

	// If there's a discriminator, find the mapping that matches this associated type
	if union.Discriminator != nil {
		for _, mapping := range union.Discriminator.Mapping {
			// Match by comparing the type pointer or name
			if mapping.Type == t ||
				(mapping.Type != nil && mapping.Type.Name == t.Name && t.Name != "") {
				return mapping.Name
			}
		}
	}

	// For non-discriminated unions
	name := getTypeDisplayName(t)
	if name != "" {
		return name
	}

	return fmt.Sprintf("%s (%d)", t.Type, index)
}

// getTypeDisplayName returns the user-facing name for a type (for path display)
func getTypeDisplayName(t *ast.TypeDef) string {
	if t == nil {
		return ""
	}

	if t.Name != "" {
		return t.Name
	}

	if t.OriginalName != "" {
		return t.OriginalName
	}

	switch t.Type {
	case ast.DataTypeString, ast.DataTypeInteger, ast.DataTypeInt32, ast.DataTypeBigInt,
		ast.DataTypeNumber, ast.DataTypeFloat32, ast.DataTypeDecimal,
		ast.DataTypeBoolean, ast.DataTypeDate, ast.DataTypeDateTime, ast.DataTypeAny:
		return string(t.Type)
	case ast.DataTypeEnum:
		if t.Enum != nil && len(t.Enum.Values) > 0 {
			values := t.Enum.Values
			isString := t.Enum.Type != nil && t.Enum.Type.Type == ast.DataTypeString

			formatValue := func(v string) string {
				if isString {
					return fmt.Sprintf("%q", v)
				}
				return v
			}

			if len(values) == 1 {
				return formatValue(values[0])
			}
			if len(values) <= 2 {
				formatted := make([]string, len(values))
				for i, v := range values {
					formatted[i] = formatValue(v)
				}
				return strings.Join(formatted, ",")
			}
			formatted := make([]string, 2)
			for i := 0; i < 2; i++ {
				formatted[i] = formatValue(values[i])
			}
			return fmt.Sprintf("%s and %d more", strings.Join(formatted, ","), len(values)-2)
		}
		return "enum"
	case ast.DataTypeSet:
		return fmt.Sprintf("Set<%s>", getTypeDisplayName(t.ItemType))
	case ast.DataTypeArray:
		return fmt.Sprintf("Array<%s>", getTypeDisplayName(t.ItemType))
	case ast.DataTypeMap:
		return fmt.Sprintf("Map<%s>", getTypeDisplayName(t.ItemType))
	case ast.DataTypeEventStream:
		return fmt.Sprintf("EventStream<%s>", getTypeDisplayName(t.ItemType))
	case ast.DataTypeJsonL:
		return fmt.Sprintf("JsonL<%s>", getTypeDisplayName(t.ItemType))
	}

	return ""
}
