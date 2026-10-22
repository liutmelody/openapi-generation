package schemas

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/ettle/strcase"
	"github.com/speakeasy-api/openapi-generation/v2/internal/ast"
	"github.com/speakeasy-api/openapi-generation/v2/internal/configuration"
	"github.com/speakeasy-api/openapi-generation/v2/internal/contenttypes"
	"github.com/speakeasy-api/openapi-generation/v2/internal/document"
	"github.com/speakeasy-api/openapi-generation/v2/internal/extensions"
	"github.com/speakeasy-api/openapi-generation/v2/internal/features"
	"github.com/speakeasy-api/openapi-generation/v2/internal/namer"
	"github.com/speakeasy-api/openapi-generation/v2/internal/resolution"
	"github.com/speakeasy-api/openapi-generation/v2/internal/subsystem"
	"github.com/speakeasy-api/openapi-generation/v2/internal/types"
	"github.com/speakeasy-api/openapi-generation/v2/internal/utils"
	"github.com/speakeasy-api/openapi-generation/v2/pkg/errors"
	"github.com/speakeasy-api/openapi-generation/v2/pkg/logging"
	oaextensions "github.com/speakeasy-api/openapi/extensions"
	"github.com/speakeasy-api/openapi/hashing"
	"github.com/speakeasy-api/openapi/jsonschema/oas3"
	"github.com/speakeasy-api/openapi/jsonschema/oas3/core"
	"github.com/speakeasy-api/openapi/marshaller"
	oas "github.com/speakeasy-api/openapi/openapi"
	"github.com/speakeasy-api/openapi/pointer"
	"github.com/speakeasy-api/openapi/references"
	"github.com/speakeasy-api/openapi/sequencedmap"
	"github.com/speakeasy-api/openapi/values"
	config "github.com/speakeasy-api/sdk-gen-config"
	"gopkg.in/yaml.v3"

	pluralizeModule "github.com/gertd/go-pluralize"

	"github.com/speakeasy-api/openapi-generation/v2/internal/openapi"
)

type ParamType string

const (
	ParamTypePath   ParamType = "path"
	ParamTypeQuery  ParamType = "query"
	ParamTypeHeader ParamType = "header"
)

const maxSelfRecursiveAdditionalPropertiesDepth = 3

type LoopFrame struct {
	Type       string
	Restricted bool
}

type Params struct {
	ContextStack               ast.ContextStack
	SerializationMethod        ast.SerializationMethod
	ParamType                  ParamType
	ParentOneOfSchema          *oas3.JSONSchema[oas3.Referenceable]
	Schema                     *oas3.JSONSchema[oas3.Referenceable]
	Encoding                   *sequencedmap.Map[string, *oas.Encoding]
	Scope                      ast.Scope
	IsRequest                  bool
	Depth                      int
	MaxDepth                   int
	Parents                    []string
	TypeDefCache               map[string]*ast.TypeDef // Avoid re-traversing the same subtree if we've already processed it
	LoopContext                []LoopFrame
	Nullable                   bool
	CircularReference          bool
	ParentComponentRef         references.Reference // If this schema is a inline component of a request/response component we will try to use the parent component ref to resolve the name
	ParentComponentDescription string
	UsedInUnion                bool
	DocInfo                    *document.DocumentInfo
	SkipNestedRefTracking      bool // Set to true for array items and additionalProperties (libopenapi bug doesn't affect these)
}

type Features interface {
	IsFeatureSupported(ctx context.Context, feature features.Feature) bool
	RecordFeatureUsage(ctx context.Context, feature features.Feature)
}

type Schemas struct {
	Config    *configuration.Config
	Target    types.Target
	Subsystem *subsystem.Subsystem
	Namer     *namer.Namer
}

// Checks schema for circular reference (adding it to the visited context stack)
// and then based on resolved type, creates FieldDef.
func (s *Schemas) HandleSchema(ctx context.Context, params Params) (*ast.FieldDef, error) {
	if err := s.checkCircularReference(ctx, &params, params.Schema); err != nil {
		return nil, err
	}

	if params.TypeDefCache == nil {
		params.TypeDefCache = map[string]*ast.TypeDef{}
	}

	return s.handleSchema(ctx, params)
}

// Based on resolved type, creates FieldDef for schema. Always prefer using
// HandleSchema instead of this method directly as it handles circular
// references.
func (s *Schemas) handleSchema(ctx context.Context, params Params) (*ast.FieldDef, error) {
	js, err := resolution.Resolve(ctx, params.Schema, params.DocInfo)
	if err != nil {
		return nil, err
	}

	if js.IsBool() {
		return s.handleAnyType(ctx, params, false)
	}

	schema := js.GetSchema()

	typ, subTypes, nullable := openapi.GetResolvedType(ctx, js)
	overrideFieldDef, err := s.handleTypeOverride(ctx, params.Schema, params, nullable)
	if err != nil {
		return nil, err
	}
	if overrideFieldDef != nil {
		return overrideFieldDef, nil
	}

	if err := s.handleMaxDepthCheck(ctx, js, params, typ); err != nil {
		return nil, err
	}

	// nullable may be overridden when flattening a nullable union with a single item
	if params.Nullable {
		nullable = true
		params.Nullable = false
	}

	if schema.GetNot() != nil {
		return nil, errors.NewUnsupportedError("not schemas unsupported", schema.GetCore().Not.GetKeyNodeOrRoot(schema.GetRootNode()))
	}

	if len(schema.Enum) > 0 {
		return s.handleEnum(ctx, params, nullable)
	}

	switch typ {
	case "allOf":
		return s.handleAllOf(ctx, params, nullable)
	case "anyOf":
		fallthrough
	case "oneOf":
		return s.handleAnyOfOneOf(ctx, params, nullable, subTypes)
	case "any":
		return s.handleAnyType(ctx, params, nullable)
	case "array":
		return s.handleArray(ctx, params, nullable)
	case "object":
		return s.handleObject(ctx, params, nullable)
	case "string":
		return s.handleString(ctx, params, nullable)
	case "number":
		return s.handleNumber(ctx, params, nullable)
	case "integer":
		return s.handleInteger(ctx, params, nullable)
	default:
		fieldName, err := s.getFieldName(ctx, params.Schema, params, typ, "")
		if err != nil {
			return nil, err
		}

		typeDef := &ast.TypeDef{
			Examples: s.getExamples(js),
			Type:     ast.DataType(typ),
		}

		// Set TypeDef.Name when explicitly specified via title or x-speakeasy-name-override
		if s.respectTitlesForPrimitiveUnionMembers() && fieldName != typ {
			typeDef.Name = fieldName
		}

		extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
		if err != nil {
			return nil, fmt.Errorf("error handling type extensions: %w", err)
		}

		typeDef.Extensions = extensions

		return s.handleFieldDefMetaData(ctx, &ast.FieldDef{
			Name: fieldName,
			Type: ast.NewType(typeDef, nil),
		}, nullable, schema), nil
	}
}

func (s *Schemas) checkCircularReference(ctx context.Context, params *Params, schema *oas3.JSONSchema[oas3.Referenceable]) error {
	if params.Parents == nil {
		params.Parents = []string{}
	}

	circularReference := params.CircularReference

	if openapi.IsReferenceForCircularReferences(schema) {
		ref := openapi.RefForCircularReferences(schema)

		if slices.Contains(params.Parents, ref) {
			isRestricted := true
			containsObject := false

			for _, v := range params.LoopContext {
				if v.Type == "object" {
					containsObject = true
				}

				if v.Type == "array" && !v.Restricted {
					isRestricted = false
				} else if !v.Restricted {
					isRestricted = false
				}
			}

			if !containsObject {
				isRestricted = true
			}

			if isRestricted {
				return errors.NewValidationError("circular reference", schema.GetRootNode(), errors.NewCircularReferenceError(append(params.Parents, ref)))
			}
			if s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureDisallowCircularRefs) {
				logging.LogWarning(ctx, "validation warning", errors.NewValidationWarning("circular reference", schema.GetRootNode(), fmt.Errorf("%s, but no circular references are allowed in %s types. Consider using x-speakeasy-type-override: any to declare %s as an arbitrary JSON blob", errors.GetCircularReferenceString(append(params.Parents, ref)), s.Target.Target, ref)))
			}
			circularReference = true
		}

		params.Parents = append(params.Parents, ref)
	}

	params.CircularReference = circularReference
	return nil
}

func (s *Schemas) handleMaxDepthCheck(ctx context.Context, schema *oas3.JSONSchema[oas3.Concrete], params Params, typ string) error {
	// TODO: this wont catch too deep a depth for arrays in query params for example
	if params.MaxDepth > 0 && params.Depth > params.MaxDepth && typ == "object" {
		switch {
		case params.SerializationMethod == ast.SerializationMethodForm && params.Encoding.Len() > 0:
			for enc := range params.Encoding.Values() {
				contentType := enc.GetContentType(schema)
				if !contenttypes.IsJSON(contentType) {
					return errors.NewUnsupportedError(fmt.Sprintf("Invalid encoding %s for www-form-urlencoded object", contentType), openapi.GetTypePropertyNode(schema))
				}
			}
			params.SerializationMethod = ast.SerializationMethodJSON
		case params.SerializationMethod != "":
			return errors.NewUnsupportedError(fmt.Sprintf("only simple objects supported for %s", params.SerializationMethod), openapi.GetTypePropertyNode(schema))
		case s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureDeepObjectParams):
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureDeepObjectParams)

			return nil
		default:
			return errors.NewUnsupportedError(fmt.Sprintf("only simple objects supported for %s", params.ParamType), openapi.GetTypePropertyNode(schema))
		}
	}

	return nil
}

func (s *Schemas) handleTypeOverride(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], params Params, nullable bool) (*ast.FieldDef, error) {
	js := schema.MustGetResolvedSchema()

	typeOverride, err := s.Subsystem.Extensions.TypeOverride(js.GetExtensions())
	if typeOverride == "" {
		return nil, err
	}

	s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureTypeOverrides)

	fieldName, err := s.getFieldName(ctx, schema, params, typeOverride, "")
	if err != nil {
		return nil, err
	}

	typeDef := &ast.TypeDef{
		Location: getTypeLocation(schema.GetSchema()),
		Type:     ast.DataTypeAny,
	}

	extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, js.GetExtensions())
	if err != nil {
		return nil, fmt.Errorf("error handling type extensions: %w", err)
	}

	typeDef.Extensions = extensions

	return &ast.FieldDef{
		Name:     fieldName,
		Type:     ast.NewType(typeDef, nil),
		Optional: nullable,
	}, nil
}

func (s *Schemas) handleNumber(ctx context.Context, params Params, nullable bool) (*ast.FieldDef, error) {
	js := params.Schema.MustGetResolvedSchema()
	schema := js.GetSchema()

	dataType := ast.DataTypeNumber

	// double handled as number which we treat as float64
	switch schema.GetFormat() {
	case "float":
		dataType = ast.DataTypeFloat32
	case "decimal":
		if s.Subsystem.Features.IsFeatureSupported(context.Background(), features.FeatureDecimal) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureDecimal)
			dataType = ast.DataTypeDecimal
		}
	}

	fieldName, err := s.getFieldName(ctx, params.Schema, params, string(dataType), "")
	if err != nil {
		return nil, err
	}

	typeDef := &ast.TypeDef{
		Examples: s.getExamples(js),
		Location: getTypeLocation(schema),
		Type:     dataType,
		Validations: &ast.Validations{
			Maximum: schema.Maximum,
			Minimum: schema.Minimum,
		},
	}

	// Set TypeDef.Name when explicitly specified via title or x-speakeasy-name-override
	if s.respectTitlesForPrimitiveUnionMembers() && fieldName != string(dataType) {
		typeDef.Name = fieldName
	}

	extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
	if err != nil {
		return nil, fmt.Errorf("error handling type extensions: %w", err)
	}

	typeDef.Extensions = extensions

	return s.handleFieldDefMetaData(ctx, &ast.FieldDef{
		Name: fieldName,
		Type: ast.NewType(typeDef, nil),
	}, nullable, schema), nil
}

func (s *Schemas) handleInteger(ctx context.Context, params Params, nullable bool) (*ast.FieldDef, error) {
	js := params.Schema.MustGetResolvedSchema()
	schema := js.GetSchema()

	dataType := ast.DataTypeInteger

	// int64 handled as integer which we treat as int64
	switch schema.GetFormat() {
	case "int32":
		dataType = ast.DataTypeInt32
	case "bigint":
		if s.Subsystem.Features.IsFeatureSupported(context.Background(), features.FeatureBigInt) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureBigInt)
			dataType = ast.DataTypeBigInt
		}
	}

	fieldName, err := s.getFieldName(ctx, params.Schema, params, string(dataType), "")
	if err != nil {
		return nil, err
	}

	typeDef := &ast.TypeDef{
		Examples: s.getExamples(js),
		Location: getTypeLocation(schema),
		Type:     dataType,
		Validations: &ast.Validations{
			Maximum: schema.Maximum,
			Minimum: schema.Minimum,
		},
	}

	// Set TypeDef.Name when explicitly specified via title or x-speakeasy-name-override
	if s.respectTitlesForPrimitiveUnionMembers() && fieldName != string(dataType) {
		typeDef.Name = fieldName
	}

	extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
	if err != nil {
		return nil, fmt.Errorf("error handling type extensions: %w", err)
	}

	typeDef.Extensions = extensions

	return s.handleFieldDefMetaData(ctx, &ast.FieldDef{
		Name: fieldName,
		Type: ast.NewType(typeDef, nil),
	}, nullable, schema), nil
}

var pluralizeClient = pluralizeModule.NewClient()

func cachedPluralize() func(string) string {
	mu := sync.Mutex{}
	cache := map[string]string{}
	return func(s string) string {
		mu.Lock()
		defer mu.Unlock()
		if cached, ok := cache[s]; ok {
			return cached
		}
		pluralized := pluralizeClient.Plural(s)
		cache[s] = pluralized
		return pluralized
	}
}

var pluralize = cachedPluralize()

func cachedSingularize() func(string) string {
	mu := sync.Mutex{}
	cache := map[string]string{}
	return func(s string) string {
		if s == "data" {
			return "data" // avoid "datum"
		}
		mu.Lock()
		defer mu.Unlock()
		if cached, ok := cache[s]; ok {
			return cached
		}
		singularized := pluralizeClient.Singular(s)
		cache[s] = singularized
		return singularized
	}
}

var singularize = cachedSingularize()

func (s *Schemas) handleArray(ctx context.Context, params Params, nullable bool) (*ast.FieldDef, error) {
	js := params.Schema.MustGetResolvedSchema()
	schema := js.GetSchema()

	params.Depth++

	var itemField *ast.FieldDef
	var itemType *ast.TypeDef
	containsNull := false
	dataType := ast.DataTypeArray
	if schema.GetFormat() == "set" {
		if s.Subsystem.Features.IsFeatureSupported(context.Background(), features.FeatureSets) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureSets)
			dataType = ast.DataTypeSet
		}
	}

	fieldName, err := s.getFieldName(ctx, params.Schema, params, "", "")
	if err != nil {
		return nil, err
	}

	if schema.Items == nil {
		itemType = ast.NewType(&ast.TypeDef{Type: ast.DataTypeAny, Location: getTypeLocation(schema)}, nil)

		if fieldName == "" {
			fieldName = "any"
		}
	} else {
		itemParams := params
		itemParams.Schema = schema.Items
		itemParams.SkipNestedRefTracking = true // Array items are not affected by libopenapi nested ref bug

		if params.Schema.IsReference() && !schema.Items.IsReference() {
			itemParams.Schema = oas3.NewReferencedScheme(ctx, params.Schema.GetRef(), schema.Items.GetResolvedSchema())
			// Using parent ref so reduce visited depth to avoid it being double counted
			itemParams.Parents = itemParams.Parents[:len(itemParams.Parents)-1]
		}

		// Singularize the property name
		lastFrame := itemParams.ContextStack.LastFrame()
		if lastFrame != nil && lastFrame.Type == ast.ContextTypeProperty {
			singularized := singularize(lastFrame.Identifier)
			lastFrame.IdentifierForNaming = &singularized
		}

		itemParams.LoopContext = append(itemParams.LoopContext, LoopFrame{Type: "array", Restricted: schema.MinItems != nil && *schema.MinItems > 0})
		itemField, err = s.HandleSchema(ctx, itemParams)
		if err != nil {
			return nil, err
		}
		itemType = itemField.Type
		containsNull = itemField.Nullable

		if fieldName == "" {
			defaultName := string(itemType.Type)
			if itemType.Name != "" && s.Config.Generation.UseClassNamesForArrayFields {
				defaultName = itemType.Name
			}

			itemFieldName, err := s.getFieldName(ctx, params.Schema, params, defaultName, "")
			if err != nil {
				return nil, err
			}

			fieldName = pluralize(itemFieldName)
		}
	}

	// We don't natively handle schemas with either `PrefixItems` or `Contains` so we try and treat the subType as a oneOf which should at least let the data be accessible
	if len(schema.PrefixItems) > 0 || schema.Contains != nil {
		p := params

		itemSchema := &oas3.Schema{
			OneOf: schema.PrefixItems,
		}

		if schema.GetContains() != nil {
			itemSchema.OneOf = append(itemSchema.OneOf, schema.Contains)
		}
		p.Schema = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](itemSchema)

		p.LoopContext = append(p.LoopContext, LoopFrame{Type: "array", Restricted: schema.MinItems != nil && *schema.MinItems > 0})
		itemField, err := s.handleAnyOfOneOf(ctx, p, nullable, []string{})
		if err != nil {
			return nil, err
		}
		itemType = itemField.Type
		containsNull = itemField.Nullable
	}

	if nullable {
		s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureNullables)
	}

	typeDef := &ast.TypeDef{
		ContainsNull: containsNull,
		Examples:     s.getExamples(js),
		ItemType:     itemType,
		Location:     getTypeLocation(schema),
		Type:         dataType,
		Validations: &ast.Validations{
			MinItems:    schema.MinItems,
			MaxItems:    schema.MaxItems,
			UniqueItems: schema.UniqueItems,
		},
	}

	extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
	if err != nil {
		return nil, fmt.Errorf("error handling type extensions: %w", err)
	}

	typeDef.Extensions = extensions

	fieldDef := &ast.FieldDef{
		Name:     fieldName,
		Type:     ast.NewType(typeDef, nil),
		Optional: nullable,
		Nullable: nullable,
	}
	if s.containerMetaDataApplies(ctx, schema, features.FeatureDefaultArrays, features.FeatureConstArrays) {
		fieldDef = s.handleFieldDefMetaData(ctx, fieldDef, nullable, schema)
		s.recordContainerConstUsage(ctx, fieldDef, features.FeatureConstArrays)
	}

	// We need to inherit the multipart annotations
	if itemField != nil {
		anno := itemField.Annotations.Get(ast.AnnotationTypeMultipartForm)
		if anno != nil {
			fieldDef.Annotations.Append(anno)
		}
	}

	return fieldDef, nil
}

func (s *Schemas) handleMultipartFormDataBinary(ctx context.Context, params Params) (*ast.FieldDef, error) {
	js := params.Schema.MustGetResolvedSchema()
	schema := js.GetSchema()
	// When the multipart schema is a property reference, e.g.
	//    content:
	//      multipart/form-data:
	//        schema:
	//          type: object
	//          properties:
	//            file:
	//              $ref: "#/components/schemas/File"
	// We need to preserve the original context stack to extract the parent
	// property name as handleReferencedType replaces the stack for references.
	originalContextStack := params.ContextStack

	typeName, refType, contextStack, parentComponentRef, _, err := s.handleReferencedType(ctx, &params, params.Schema, schema)
	if err != nil {
		return nil, err
	}

	typeDef := &ast.TypeDef{
		Examples:        s.getExamples(js),
		IsComponent:     refType != "",
		IsMultipartFile: true, // The annotation associated with this type gets lost when its an item of an array or map and this helps the templating code know how to render this
		Location:        getTypeLocation(schema),
		Name:            typeName,
		Scope:           params.Scope,
		Type:            ast.DataTypeClass,
	}

	extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
	if err != nil {
		return nil, fmt.Errorf("error handling type extensions: %w", err)
	}

	typeDef.Extensions = extensions

	formType := ast.NewType(typeDef, contextStack)

	formType.Fields = formType.Fields.MustAddField(&ast.FieldDef{
		Name: "fileName",
		Type: ast.NewType(&ast.TypeDef{
			Type:     ast.DataTypeString,
			Examples: s.getExamples(js),
			Location: getTypeLocation(schema),
		}, nil),
		Annotations: []ast.Annotation{
			&ast.MultipartFormAnnotation{Name: "fileName"},
		},
	}, s.Config.Generation.MaintainOpenAPIOrder)

	contentDataType := ast.DataTypeBytes

	if params.IsRequest && s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureUploadStreams) {
		s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureUploadStreams)
		contentDataType = ast.DataTypeRequestStream
	}

	formType.Fields = formType.Fields.MustAddField(&ast.FieldDef{
		Name: "content",
		Type: ast.NewType(&ast.TypeDef{
			Type:     contentDataType,
			Location: getTypeLocation(schema),
			Examples: s.getExamples(js),
		}, nil),
		Annotations: []ast.Annotation{
			&ast.MultipartFormAnnotation{Content: true},
		},
	}, s.Config.Generation.MaintainOpenAPIOrder)

	if s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureMultipartFileContentType) {
		s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureMultipartFileContentType)

		formType.Fields = formType.Fields.MustAddField(&ast.FieldDef{
			Name: "contentType",
			Type: ast.NewType(&ast.TypeDef{Type: ast.DataTypeString, Location: getTypeLocation(schema)}, nil),
			Annotations: []ast.Annotation{
				&ast.MultipartFormAnnotation{Name: "Content-Type"},
			},
			Optional: true,
		}, s.Config.Generation.MaintainOpenAPIOrder)
	}

	fieldName, err := s.getFieldName(ctx, params.Schema, params, "file", parentComponentRef)
	if err != nil {
		return nil, err
	}

	propName := "fileName"
	propertyFrame := params.ContextStack.FindLastFrameOfType(ast.ContextTypeProperty)
	originalPropertyFrame := originalContextStack.FindLastFrameOfType(ast.ContextTypeProperty)

	if propertyFrame != nil {
		propName = propertyFrame.Identifier
	} else if originalPropertyFrame != nil {
		propName = originalPropertyFrame.Identifier
	}

	return &ast.FieldDef{
		Name: fieldName,
		Type: s.Subsystem.Register.RegisterType(ctx, formType, params.IsRequest),
		Annotations: []ast.Annotation{
			&ast.MultipartFormAnnotation{
				File: true,
				Name: propName,
			},
		},
	}, nil
}

func (s *Schemas) handleObject(ctx context.Context, params Params, nullable bool) (*ast.FieldDef, error) {
	js := params.Schema.MustGetResolvedSchema()
	schema := js.GetSchema()

	typeName, refType, contextStack, parentComponentRef, parentComponentDescription, err := s.handleReferencedType(ctx, &params, params.Schema, schema)
	if err != nil {
		return nil, err
	}

	if s.Config.Generation.Fixes.NameResolutionFeb2025 {
		// Certain properties can aid in name resolution, so we will add them to the context stack
		identifiers := s.extractIdentifiersFromProperties(ctx, params.ParentOneOfSchema, params.Schema, params.DocInfo)
		for _, v := range identifiers.All() {
			contextStack.Append(ast.ContextTypeConstProperty, strcase.ToGoPascal(v))
		}
	}

	fieldName, err := s.getFieldName(ctx, params.Schema, params, "object", parentComponentRef)
	if err != nil {
		return nil, err
	}

	cachedType := s.getCachedType(&params, contextStack, params.Schema, typeName)

	if cachedType != nil {
		return &ast.FieldDef{
			Name:     fieldName,
			Type:     cachedType,
			Optional: nullable,
			Nullable: nullable,
		}, nil
	}

	var additionalProperties *ast.FieldDef = nil

	var additionalPropertiesSchema *oas3.JSONSchema[oas3.Concrete]

	if schema.GetAdditionalProperties() != nil {
		additionalPropertiesSchema, err = resolution.Resolve(ctx, schema.GetAdditionalProperties(), params.DocInfo)
		if err != nil {
			return nil, err
		}
	}

	recurseAdditionalProperties := !params.CircularReference
	if params.CircularReference && schema.GetProperties().Len() == 0 {
		if !openapi.IsReferenceForCircularReferences(params.Schema) {
			recurseAdditionalProperties = true
		} else {
			ref := openapi.RefForCircularReferences(params.Schema)
			occurrences := 0
			for _, parent := range params.Parents {
				if parent == ref {
					occurrences++
				}
			}
			recurseAdditionalProperties = occurrences < maxSelfRecursiveAdditionalPropertiesDepth
		}
	}

	if additionalPropertiesSchema != nil && (additionalPropertiesSchema.IsSchema() || *additionalPropertiesSchema.GetBool()) {
		switch {
		case params.SerializationMethod == ast.SerializationMethodEventStream:
			// Ignore additional properties when parsing event stream payloads.
			// They're only allowed when parsing the "data" field if its type is
			// "object" but by that point we'd have switched the serialization
			// method to JSON.
			logging.LogWarning(
				ctx,
				"validation warning",
				errors.NewValidationWarning("server-sent events cannot have additional properties", schema.GetCore().AdditionalProperties.GetKeyNodeOrRoot(schema.GetRootNode()), nil),
			)
		case additionalPropertiesSchema.IsBool():
			if s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureAdditionalProperties) && schema.GetProperties().Len() > 0 && params.SerializationMethod == ast.SerializationMethodJSON {
				s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureAdditionalProperties)

				typeDef := &ast.TypeDef{
					Examples: s.getExamples(js),
					ItemType: ast.NewType(&ast.TypeDef{
						Type: ast.DataTypeAny,
					}, nil),
					Location: getTypeLocation(schema),
					Type:     ast.DataTypeMap,
				}

				if s.isTerraformProvider() {
					typeDef = &ast.TypeDef{
						Examples: s.getExamples(js),
						Location: getTypeLocation(schema),
						Type:     ast.DataTypeAny,
					}
				}

				extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
				if err != nil {
					return nil, fmt.Errorf("error handling type extensions: %w", err)
				}

				typeDef.Extensions = extensions

				additionalProperties = &ast.FieldDef{
					Type: ast.NewType(typeDef, nil),
					Annotations: []ast.Annotation{
						&ast.JSONAnnotation{Ignore: true, FieldName: "-"},
						&ast.NeedsCasingAnnotation{},
					},
					Optional:               true,
					Nullable:               false,
					IsAdditionalProperties: true,
				}
			} else {
				if nullable {
					s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureNullables)
				}

				typeDef := &ast.TypeDef{
					Examples: s.getExamples(js),
					ItemType: ast.NewType(&ast.TypeDef{
						Type: ast.DataTypeAny,
					}, nil),
					Location: getTypeLocation(schema),
					Type:     ast.DataTypeMap,
				}

				extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
				if err != nil {
					return nil, fmt.Errorf("error handling type extensions: %w", err)
				}

				typeDef.Extensions = extensions

				mapField := &ast.FieldDef{
					Name:     fieldName,
					Type:     ast.NewType(typeDef, nil),
					Optional: nullable,
					Nullable: nullable,
				}
				if schema.Const != nil && s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureConstObjects) {
					mapField = s.handleFieldDefMetaData(ctx, mapField, nullable, schema)
					s.recordContainerConstUsage(ctx, mapField, features.FeatureConstObjects)
				}

				return mapField, nil
			}
		case recurseAdditionalProperties:
			additionalPropertiesParams := params
			additionalPropertiesParams.Schema = schema.GetAdditionalProperties()
			additionalPropertiesParams.SkipNestedRefTracking = true // AdditionalProperties are not affected by libopenapi nested ref bug
			additionalPropertiesParams.LoopContext = append(additionalPropertiesParams.LoopContext, LoopFrame{Type: "object", Restricted: false})

			if params.Schema.IsReference() && !additionalPropertiesParams.Schema.IsReference() {
				// Setting with parent ref to ensure dependencies are handled correctly
				additionalPropertiesParams.Parents = additionalPropertiesParams.Parents[:len(additionalPropertiesParams.Parents)-1]
				additionalPropertiesParams.Schema = oas3.NewReferencedScheme(ctx, params.Schema.GetRef(), schema.GetAdditionalProperties().GetResolvedSchema())
			}

			additionalPropertiesParams.Depth++
			valueType, err := s.HandleSchema(ctx, additionalPropertiesParams)
			if err != nil {
				return nil, err
			}

			if s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureAdditionalProperties) && schema.GetProperties().Len() > 0 && additionalPropertiesParams.SerializationMethod == ast.SerializationMethodJSON {
				s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureAdditionalProperties)

				typeDef := &ast.TypeDef{
					ContainsNull: valueType.Nullable,
					Examples:     s.getExamples(js),
					ItemType:     valueType.Type,
					Location:     getTypeLocation(schema),
					Type:         ast.DataTypeMap,
				}

				extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
				if err != nil {
					return nil, fmt.Errorf("error handling type extensions: %w", err)
				}

				typeDef.Extensions = extensions

				additionalProperties = &ast.FieldDef{
					Type: ast.NewType(typeDef, nil),
					Annotations: []ast.Annotation{
						&ast.JSONAnnotation{Ignore: true, FieldName: "-"},
						&ast.NeedsCasingAnnotation{},
					},
					Optional:               true,
					Nullable:               false,
					IsAdditionalProperties: true,
				}
			} else {
				if nullable {
					s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureNullables)
				}

				typeDef := &ast.TypeDef{
					ContainsNull: valueType.Nullable,
					Examples:     s.getExamples(js),
					ItemType:     valueType.Type,
					Location:     getTypeLocation(schema),
					Type:         ast.DataTypeMap,
				}

				extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
				if err != nil {
					return nil, fmt.Errorf("error handling type extensions: %w", err)
				}

				typeDef.Extensions = extensions

				mapField := &ast.FieldDef{
					Name:     fieldName,
					Type:     ast.NewType(typeDef, nil),
					Optional: nullable,
					Nullable: nullable,
				}
				if schema.Const != nil && s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureConstObjects) {
					mapField = s.handleFieldDefMetaData(ctx, mapField, nullable, schema)
					s.recordContainerConstUsage(ctx, mapField, features.FeatureConstObjects)
				}

				return mapField, nil
			}
		}
	}

	comments, err := s.handleSchemaComments(ctx, schema, parentComponentDescription)
	if err != nil {
		return nil, err
	}

	var hash string
	if s.Config.Generation.DeduplicateErrors {
		hash = hashing.Hash(schema)
	}

	objType := ast.NewType(&ast.TypeDef{
		Name:        typeName,
		Hash:        hash,
		Type:        ast.DataTypeClass,
		Location:    getTypeLocation(schema),
		Scope:       params.Scope,
		Fields:      make(ast.Fields, 0),
		IsComponent: refType != "",
		Comments:    comments,
		Examples:    s.getExamples(js),
		UsedInUnion: params.UsedInUnion,
	}, contextStack)

	extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, objType, schema.GetExtensions())
	if err != nil {
		return nil, fmt.Errorf("error handling type extensions: %w", err)
	}

	objType.Extensions = extensions

	if schema.GetProperties().Len() == 0 {
		if nullable {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureNullables)
		}

		return &ast.FieldDef{
			Name:     fieldName,
			Type:     s.Subsystem.Register.RegisterType(ctx, objType, params.IsRequest),
			Optional: nullable,
			Nullable: nullable,
		}, nil
	}

	inputModel := false
	fullInput := true

	outputModel := false
	fullOutput := true

	for prop := range schema.GetProperties().Values() {
		propSchema, err := resolution.Resolve(ctx, prop, params.DocInfo)
		if err != nil {
			return nil, err
		}
		if propSchema.IsBool() {
			continue
		}

		ignore, err := s.Subsystem.Extensions.Ignore(propSchema.GetExtensions())
		if err != nil {
			return nil, err
		}
		if ignore {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureIgnores)
			continue
		}

		fullInput = fullInput && propSchema.GetSchema().GetReadOnly()
		fullOutput = fullOutput && propSchema.GetSchema().GetWriteOnly()

		if propSchema.GetSchema().GetReadOnly() && params.IsRequest {
			inputModel = true
		}
		if propSchema.GetSchema().GetWriteOnly() && !params.IsRequest {
			outputModel = true
		}
	}

	usedNames := make(map[string]bool)

	if !params.CircularReference {
		resolvedFullInput := true
		resolvedFullOutput := true

		for origPropName, prop := range schema.GetProperties().All() {
			propSchema := prop.MustGetResolvedSchema()

			// Special case where the extensions for allOfs need to be hoisted before we handle the schema so any extensions that impact the properties resolved name are handled correctly
			if propSchema.IsSchema() {
				exts := openapi.HoistAllOfExtensions(ctx, prop, params.DocInfo, nil, hoistChildExtensionPredicate(s.Config.Generation.Fixes, s.Subsystem.Extensions))
				propExts := propSchema.GetExtensions()
				if exts != nil {
					for k, v := range exts.All() {
						propExts.Set(k, v)
					}
				}
				propSchema.GetSchema().Extensions = propExts
			}

			ignore, err := s.Subsystem.Extensions.Ignore(propSchema.GetExtensions())
			if err != nil {
				return nil, err
			}
			if ignore {
				s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureIgnores)
				continue
			}

			propName, err := s.Subsystem.Extensions.GetPropertyName(prop, propSchema, origPropName, s.Config.Generation.Fixes.NameOverrideFeb2026)
			if err != nil {
				return nil, err
			}

			if propSchema.IsSchema() {
				if propSchema.GetSchema().GetReadOnly() && params.IsRequest {
					continue
				}
				if propSchema.GetSchema().GetWriteOnly() && !params.IsRequest {
					continue
				}
			}

			propParams := params
			required := slices.Contains(schema.Required, origPropName)
			propParams.ContextStack = append(propParams.ContextStack, ast.ContextFrame{
				Type:       ast.ContextTypeProperty,
				Identifier: propName,
			})
			propParams.Schema = prop
			propParams.Depth++
			propParams.SkipNestedRefTracking = false // Reset for object properties - nested ref tracking should apply to property refs
			propParams.LoopContext = append(params.LoopContext, LoopFrame{Type: "object", Restricted: required})

			switch params.SerializationMethod {
			case ast.SerializationMethodMultipart:
				// Check if there's an explicit encoding for this property
				if params.Encoding.Len() > 0 {
					encoding, ok := params.Encoding.Get(origPropName)
					// If a contentType is specified, we treat JSON as a special case and handle it accordingly
					if ok && encoding.ContentType != nil && contenttypes.IsJSON(pointer.Value(encoding.ContentType)) {
						propParams.SerializationMethod = ast.SerializationMethodJSON
					}
					// otherwise, we continue with the default multipart serialization
				}

				// If no encoding specified or contentType not handled, check if complex
				if propParams.SerializationMethod == params.SerializationMethod {
					isComplex, err := openapi.IsComplex(ctx, propSchema, params.DocInfo)
					if err != nil {
						return nil, err
					}

					if isComplex {
						propParams.SerializationMethod = ast.SerializationMethodJSON
					}
				}
			case ast.SerializationMethodForm:
				complexEncoding := false

				checkEncoding := true

				if params.Encoding.Len() > 0 {
					encoding, ok := params.Encoding.Get(origPropName)
					if ok {
						checkEncoding = false

						if encoding.GetAllowReserved() {
							return nil, errors.NewUnsupportedError("allowReserved is not currently supported", encoding.GetCore().AllowReserved.GetKeyNodeOrRoot(encoding.GetRootNode()))
						}

						contentType := encoding.GetContentType(propSchema)
						if contentType != "" {
							switch {
							case contenttypes.IsJSON(contentType):
								propParams.SerializationMethod = ast.SerializationMethodJSON
								complexEncoding = true
							default:
								return nil, errors.NewUnsupportedError("unsupported content type for application/x-www-form-urlencoded encoding: "+contentType, encoding.GetRootNode())
							}
						}
					}
				}

				if checkEncoding {
					isComplex, err := openapi.IsComplex(ctx, propSchema, params.DocInfo)
					if err != nil {
						return nil, err
					}

					if isComplex {
						if params.Encoding == nil {
							params.Encoding = sequencedmap.New[string, *oas.Encoding]()
						}

						params.Encoding.Set(origPropName, &oas.Encoding{
							ContentType: pointer.From("application/json"),
						})
						propParams.SerializationMethod = ast.SerializationMethodJSON
						complexEncoding = true
					}
				}

				if !complexEncoding {
					propParams.MaxDepth = params.Depth + 1
				}

			case ast.SerializationMethodEventStream:
				method, err := selectEventStreamFieldSerialization(ctx, origPropName, propSchema, params.DocInfo)
				if err != nil {
					return nil, err
				}

				propParams.SerializationMethod = method
			case ast.SerializationMethodJsonL:
				isComplex, err := openapi.IsComplex(ctx, propSchema, params.DocInfo)
				if err != nil {
					return nil, err
				}

				if isComplex {
					propParams.SerializationMethod = ast.SerializationMethodJSON
				}
			}

			f, err := s.HandleSchema(ctx, propParams)
			if err != nil {
				return nil, err
			}

			f.Name = propName
			f.OriginalName = origPropName

			if f.Default == nil {
				f.Optional = !required
			}

			switch {
			case f.Type.IsInput(s.Config.Generation.Fixes.NameResolutionDec2023):
				inputModel = true
				resolvedFullInput = resolvedFullInput && true
				resolvedFullOutput = false
			case f.Type.IsOutput(s.Config.Generation.Fixes.NameResolutionDec2023):
				outputModel = true
				resolvedFullInput = false
				resolvedFullOutput = resolvedFullOutput && true
			default:
				resolvedFullInput = false
				resolvedFullOutput = false
			}

			switch params.SerializationMethod {
			case ast.SerializationMethodJSON, ast.SerializationMethodJsonL:
				f.Annotations = append(f.Annotations, &ast.JSONAnnotation{FieldName: origPropName})
			case ast.SerializationMethodMultipart:
				if !f.Annotations.Has(ast.AnnotationTypeMultipartForm) {
					multipartAnno := &ast.MultipartFormAnnotation{Name: origPropName, JSON: propParams.SerializationMethod == ast.SerializationMethodJSON}

					fieldType := f.Type

					switch fieldType.Type {
					case ast.DataTypeArray:
					case ast.DataTypeMap:
						fieldType = f.Type.ItemType
					}

					multipartAnno.FieldType = fieldType

					f.Annotations = append(f.Annotations, multipartAnno)
				}
			case ast.SerializationMethodForm:
				var style oas.SerializationStyle
				explode := true

				encoding, ok := params.Encoding.Get(origPropName)
				if ok {
					style = encoding.GetStyle()

					if style != "" && style != oas.SerializationStyleForm {
						return nil, errors.NewUnsupportedError("unsupported style for application/x-www-form-urlencoded encoding: "+style.String(), encoding.GetCore().Style.GetKeyNodeOrRoot(encoding.GetRootNode()))
					}

					if encoding.Explode != nil {
						explode = *encoding.Explode
					}
				}

				formAnno := &ast.FormAnnotation{Name: origPropName, JSON: propParams.SerializationMethod == ast.SerializationMethodJSON, Style: string(style), Explode: explode}

				fieldType := f.Type

				switch fieldType.Type {
				case ast.DataTypeArray:
				case ast.DataTypeMap:
					fieldType = f.Type.ItemType
				}

				formAnno.FieldType = fieldType

				f.Annotations = append(f.Annotations, formAnno)
			}

			fieldType := f.Type

			switch fieldType.Type {
			case ast.DataTypeArray:
			case ast.DataTypeMap:
				fieldType = f.Type.ItemType
			}

			switch params.ParamType {
			case ParamTypeQuery:
				f.Annotations = append(f.Annotations, &ast.ParamAnnotation{ParamType: ast.ParamTypeQueryParam, Name: origPropName, FieldType: fieldType, AllowReserved: false})
			case ParamTypePath:
				f.Annotations = append(f.Annotations, &ast.ParamAnnotation{ParamType: ast.ParamTypePathParam, Name: origPropName, FieldType: fieldType, AllowReserved: false})
			case ParamTypeHeader:
				f.Annotations = append(f.Annotations, &ast.ParamAnnotation{ParamType: ast.ParamTypeHeader, Name: origPropName, FieldType: fieldType, AllowReserved: false})
			}

			if propSchema.IsSchema() {
				f.Comments, err = s.handleSchemaComments(ctx, propSchema.GetSchema(), "")
				if err != nil {
					return nil, err
				}
			}

			isErrorMessage, err := s.Subsystem.Extensions.IsErrorMessage(propSchema.GetExtensions())
			if err != nil {
				return nil, err
			}
			if isErrorMessage {
				s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureErrors)
			}

			f.ErrorMessage = isErrorMessage

			objType.Fields = objType.Fields.MustAddField(f, s.Config.Generation.MaintainOpenAPIOrder)

			usedNames[f.Name] = true
		}

		if additionalProperties != nil {
			additionalProperties.Name, err = pickAdditionalPropertiesNameHandleConflict(fieldName, objType, usedNames)
			if err != nil {
				return nil, err
			}

			objType.Fields = objType.Fields.MustAddField(additionalProperties, s.Config.Generation.MaintainOpenAPIOrder)
		}

		objType.Input = inputModel
		objType.Output = outputModel

		if inputModel || outputModel {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureInputOutputModels)
		}

		fullInput = fullInput || resolvedFullInput
		fullOutput = fullOutput || resolvedFullOutput

		// This logic is confusing so I will try to explain it:
		//  - `inputModel` and `outputModel` specifies whether the object is used as the input of a method or the output
		//    in the case the schema has readOnly/writeOnly properties. If the schema doesn't have readOnly/writeOnly properties
		//    then both `inputModel` and `outputModel` will be false.
		//  - `fullInput` and `fullOutput` specifies whether the object is fully made up of readOnly/writeOnly properties.
		//
		// If the schema has readOnly/writeOnly properties we attempt to generate multiple models to hide the readOnly/writeOnly properties in the input/output models.
		// But we don't need to generate multiple models if the object is fully readOnly/writeOnly, except if for example a fully readOnly object is used as the input of a method.
		if inputModel && (!fullInput || params.IsRequest) {
			objType.ContextStack = s.addInputOutputContext(objType.ContextStack, inputSuffix)
			objType.Input = true
		} else if outputModel && (!fullOutput || !params.IsRequest) {
			objType.ContextStack = s.addInputOutputContext(objType.ContextStack, outputSuffix)
			objType.Output = true
		}

		if params.SerializationMethod == ast.SerializationMethodEventStream {
			if err := postProcessEventStreamEnvelope(objType); err != nil {
				return nil, err
			}
		}
	}

	objType.Truncated = params.CircularReference

	if nullable {
		s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureNullables)
	}

	fieldType := s.Subsystem.Register.RegisterType(ctx, objType, params.IsRequest)

	fieldDef := &ast.FieldDef{
		Name:     fieldName,
		Type:     fieldType,
		Optional: nullable,
		Nullable: nullable,
	}

	if s.containerMetaDataApplies(ctx, schema, features.FeatureDefaultObjects, features.FeatureConstObjects) {
		fieldDef = s.handleFieldDefMetaData(ctx, fieldDef, nullable, schema)
		s.recordContainerConstUsage(ctx, fieldDef, features.FeatureConstObjects)
	}

	s.setCachedType(fieldType, &params)

	return fieldDef, nil
}

func getTypeLocation(schema *oas3.Schema) *ast.OpenAPILocation {
	if schema == nil || schema.GetRootNode() == nil {
		return nil
	}

	// Create a minimal node with just line and column for location tracking
	rootNode := schema.GetRootNode()
	return &ast.OpenAPILocation{
		Node: &yaml.Node{
			Line:   rootNode.Line,
			Column: rootNode.Column,
		},
	}
}

func getUnionLocation(schema *oas3.Schema, originallyAnyOf bool) *ast.OpenAPILocation {
	if schema == nil {
		return nil
	}

	var keyNode *yaml.Node
	if originallyAnyOf {
		keyNode = schema.GetCore().AnyOf.KeyNode
	} else {
		keyNode = schema.GetCore().OneOf.KeyNode
	}

	if keyNode == nil {
		return nil
	}
	// Create a minimal node with just line and column for location tracking
	return &ast.OpenAPILocation{
		Node: &yaml.Node{
			Line:   keyNode.Line,
			Column: keyNode.Column,
		},
	}
}

func pickAdditionalPropertiesNameHandleConflict(name string, typedef *ast.TypeDef, usedNames map[string]bool) (string, error) {
	if typedef.Extensions.All["x-speakeasy-additional-properties-name"] != nil {
		return typedef.Extensions.All["x-speakeasy-additional-properties-name"].(string), nil
	}
	if _, ok := usedNames["AdditionalProperties"]; !ok {
		return "AdditionalProperties", nil
	}
	if _, ok := usedNames["AdditionalPropertiesT"]; !ok {
		return "AdditionalPropertiesT", nil
	}

	for i := 2; i < 100; i++ {
		suffix := strconv.Itoa(i)
		if _, ok := usedNames["AdditionalPropertiesT"+suffix]; !ok {
			return "AdditionalPropertiesT" + suffix, nil
		}
	}
	return "", fmt.Errorf("couldn't pick a name without a conflict for %s", name)
}

func (s *Schemas) handleEnum(ctx context.Context, params Params, nullable bool) (*ast.FieldDef, error) {
	js := params.Schema.MustGetResolvedSchema()
	schema := js.GetSchema()

	typ, subTypes, _ := openapi.GetResolvedType(ctx, js)
	switch typ {
	case "string":
	case "integer":
	case "anyOf", "oneOf":
		// Special case as a union may be merged with an enum type
		return s.handleAnyOfOneOf(ctx, params, nullable, subTypes)
	case "boolean":
		// Boolean enums (e.g. enum: [true] or enum: [false]) are used as discriminators
		// in oneOf schemas. Treat them as plain booleans — if single-valued, set as const.
		enumValues := schema.GetEnum()

		if len(enumValues) == 1 {
			schema.Const = enumValues[0]
		}

		schema.Enum = nil
		params.Schema = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](schema)
		return s.handleSchema(ctx, params)
	default:
		logging.LogWarning(ctx, "validation warning", errors.NewUnsupportedError("currently only string/integer enums supported, treating as base type", openapi.GetTypePropertyNode(js)))

		schema.Enum = nil
		params.Schema = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](schema)

		// Prevent circular reference error by skipping check.
		return s.handleSchema(ctx, params)
	}

	typeName, refType, contextStack, parentComponentRef, parentComponentDescription, err := s.handleReferencedType(ctx, &params, params.Schema, schema)
	if err != nil {
		return nil, err
	}

	fieldName, err := s.getFieldName(ctx, params.Schema, params, "enum", parentComponentRef)
	if err != nil {
		return nil, err
	}

	dataType := ast.DataTypeString
	if typ == "integer" {
		dataType = ast.DataTypeInteger

		if schema.GetFormat() == "int32" {
			dataType = ast.DataTypeInt32
		}
	}

	comments, err := s.handleSchemaComments(ctx, schema, parentComponentDescription)
	if err != nil {
		return nil, err
	}

	groupList, groupMap, err := s.Subsystem.Extensions.GetEnumGroups(schema)
	if err != nil {
		return nil, err
	}
	enumGroups := groupMap
	if len(groupList) > 0 {
		enumGroups = make(map[string]string)
		for i, group := range groupList {
			if i < len(schema.Enum) && schema.Enum[i] != nil && schema.Enum[i].Tag != "!!null" && group != "" {
				if _, exists := enumGroups[schema.Enum[i].Value]; !exists {
					enumGroups[schema.Enum[i].Value] = group
				}
			}
		}
	}

	if len(schema.GetEnum()) == 1 && schema.GetEnum()[0].Tag == "!!null" {
		schema.Nullable = pointer.From(true)
		schema.Const = schema.GetEnum()[0]
		schema.Enum = nil

		params.Schema = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](schema)
		return s.HandleSchema(ctx, params)
	}

	var isOpen bool
	if s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureOpenEnums) {
		open, err := s.Subsystem.Extensions.IsOpenEnum(schema)
		if err != nil {
			return nil, err
		}

		isOpen = open
	}

	if isOpen {
		s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureOpenEnums)
	}

	enumValues := []string{}

	enums := map[string]string{}

	for _, enumValue := range schema.Enum {
		if enumValue == nil || enumValue.Tag == "!!null" {
			// Skip nil enum values (full validation will warn about this if not nullable)
			continue
		}

		enumVal := enumValue.Value

		if _, ok := enums[enumVal]; ok {
			// Duplicate enum value skip (full validation will warn about this)
			continue
		}
		enums[enumVal] = enumVal

		enumValues = append(enumValues, enumVal)
	}

	var names []string
	enumDescriptions := map[string]string{}

	extNames, extNamesMap, err := s.Subsystem.Extensions.GetEnumNames(schema)
	if err != nil {
		return nil, err
	}
	if len(extNames) > 0 {
		names = extNames
		s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureEnums)
	} else if len(extNamesMap) > 0 {
		namesMap := map[string]string{}

		// Convert the map keys to string
		for k, v := range extNamesMap {
			enumValue := fmt.Sprintf("%v", k)
			namesMap[enumValue] = v
		}

		// Find any name overrides otherwise use the original enum value
		names = []string{}
		for _, enumValue := range enumValues {
			if name, ok := namesMap[enumValue]; ok {
				names = append(names, name)
			} else {
				names = append(names, enumValue)
			}
		}
	}

	if enumDescList, enumDescMap, err := s.Subsystem.Extensions.GetEnumDescriptions(schema); err != nil {
		return nil, err
	} else if len(enumDescList) > 0 {
		enumDescriptions = make(map[string]string, len(enumDescList))
		for i, description := range enumDescList {
			if i >= len(enumValues) {
				break
			}
			enumDescriptions[enumValues[i]] = description
		}
	} else if len(enumDescMap) > 0 {
		enumDescriptions = make(map[string]string, len(enumDescMap))
		for k, v := range enumDescMap {
			enumValue := fmt.Sprintf("%v", k)
			enumDescriptions[enumValue] = v
		}
	}

	if len(enumValues) == 1 && !isOpen && s.Subsystem.Config.Generation.Fixes.NameResolutionFeb2025 {
		// The enum value, will be available for naming disambiguation if this is in a oneOf/anyOf
		contextStack.AppendWithHumanized(ast.ContextTypeConstProperty, enumValues[0], strcase.ToGoPascal(enumValues[0]))
	}

	enumFormat := ""
	if s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureEnumUnions) {
		s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureEnumUnions)

		enumFormatVal := s.Config.GetLanguageConfigValue("enumFormat")
		if enumFormatVal != nil {
			enumFormat = enumFormatVal.(string)
		}
		enumFormatOverride, err := s.Subsystem.Extensions.GetEnumFormat(schema)
		if err != nil {
			return nil, err
		}
		if enumFormatOverride != "" {
			enumFormat = enumFormatOverride
		}
		if enumFormat != "" && enumFormat != "enum" && enumFormat != "union" {
			return nil, errors.NewUnsupportedError("unsupported enum format: "+enumFormat, nil)
		}
	}

	var hash string
	if s.Config.Generation.DeduplicateErrors {
		hash = hashing.Hash(schema)
	}

	typeDef := &ast.TypeDef{
		Comments: comments,
		Enum: &ast.Enum{
			Type:         ast.NewType(&ast.TypeDef{Type: dataType, Location: getTypeLocation(schema)}, nil),
			Values:       enumValues,
			Names:        names,
			Descriptions: enumDescriptions,
			Groups:       enumGroups,
			Open:         isOpen,
			Format:       enumFormat,
		},
		Examples:    s.getExamples(js),
		Hash:        hash,
		IsComponent: refType != "",
		Name:        typeName,
		Scope:       params.Scope,
		Type:        ast.DataTypeEnum,
	}

	extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
	if err != nil {
		return nil, fmt.Errorf("error handling type extensions: %w", err)
	}

	typeDef.Extensions = extensions

	enumType := ast.NewType(typeDef, contextStack)

	field := s.handleFieldDefMetaData(ctx, &ast.FieldDef{
		Name: fieldName,
		Type: s.Subsystem.Register.RegisterType(ctx, enumType, params.IsRequest),
	}, nullable, schema)

	s.checkDefaultConformsToEnum(ctx, field, schema)

	return field, nil
}

func (s *Schemas) checkDefaultConformsToEnum(ctx context.Context, field *ast.FieldDef, schema *oas3.Schema) {
	defaultValue := field.Default
	if field.Type.Enum.Type.Type == ast.DataTypeString && defaultValue != nil {
		// null is a valid default for nullable fields
		if defaultValue.Value == nil && field.Nullable {
			return
		}

		node := schema.GetPropertyNode("Default")

		if defaultValueStr, ok := defaultValue.Value.(string); ok {
			if !slices.Contains(field.Type.Enum.Values, defaultValueStr) {
				enumValuesList := s.formatEnumValuesForLogging(field.Type.Enum.Values)
				logging.LogWarning(ctx, "validation warning", errors.NewValidationWarning(fmt.Sprintf("enum value %q is not a valid value, expected one of: %s", defaultValue.Value, enumValuesList), node, nil))
				// Omit invalid default to prevent type errors in generated code
				field.Default = nil
			}
		} else {
			// Warn if default value is not a string for a string enum
			enumValuesList := s.formatEnumValuesForLogging(field.Type.Enum.Values)
			logging.LogWarning(ctx, "validation warning", errors.NewValidationWarning(fmt.Sprintf("enum default value must be a string, got %T for enum %s", defaultValue.Value, enumValuesList), node, nil))
			// Omit invalid default to prevent type errors in generated code
			field.Default = nil
		}
	}
}

// checkDefaultConformsToType clears default/const values that don't match the field's declared
// primitive type. This mirrors checkDefaultConformsToEnum but for non-enum primitives.
func (s *Schemas) checkDefaultConformsToType(ctx context.Context, field *ast.FieldDef, schema *oas3.Schema) {
	if field.Type == nil {
		return
	}

	// When the AST type was remapped from a string-typed schema (e.g. type: string, format: bigint →
	// DataTypeBigInt with Format "string"), the default value in the spec is still a string.
	// Validate against string in that case.
	dataType := field.Type.Type
	if field.Type.Format == "string" {
		dataType = ast.DataTypeString
	}

	for _, pair := range []struct {
		label string
		val   **ast.AnyValue
	}{
		{"default", &field.Default},
		{"const", &field.Const},
	} {
		v := *pair.val
		if v == nil {
			continue
		}
		// null is a valid default for nullable fields
		if v.Value == nil && field.Nullable {
			continue
		}
		if v.Value == nil {
			continue
		}

		if pair.label == "const" && (field.Type.IsContainer() || field.Type.Type == ast.DataTypeClass) {
			if !constContainerValueSupported(v.Value, field.Type) {
				logging.LogWarning(ctx, "validation warning", errors.NewValidationWarning(
					fmt.Sprintf("const value %v is not supported for type %q, only arrays and maps of scalar values are supported, omitting const", v.Value, dataType),
					schema.GetPropertyNode("Const"), nil,
				))
				*pair.val = nil
			}
			continue
		}

		if !defaultValueMatchesType(v.Value, dataType) {
			// Before clearing, try to coerce string values to the target type.
			// Many real-world specs have string defaults on numeric fields (e.g. default: "0.00").
			if coerced, ok := coerceDefaultValue(v.Value, dataType); ok {
				v.Value = coerced
			} else {
				nodeKey := "Default"
				if pair.label == "const" {
					nodeKey = "Const"
				}
				node := schema.GetPropertyNode(nodeKey)
				logging.LogWarning(ctx, "validation warning", errors.NewValidationWarning(
					fmt.Sprintf("%s value %v does not match type %q, omitting invalid %s", pair.label, v.Value, dataType, pair.label),
					node, nil,
				))
				*pair.val = nil
			}
		}
	}
}

func constContainerValueSupported(val any, typ *ast.TypeDef) bool {
	switch typ.Type {
	case ast.DataTypeArray, ast.DataTypeSet:
		items, ok := val.([]any)
		if !ok {
			return false
		}
		for _, item := range items {
			if !constScalarMatchesType(item, typ.ItemType) {
				return false
			}
		}
		return true
	case ast.DataTypeMap:
		entries, ok := val.(map[string]any)
		if !ok {
			return false
		}
		for _, item := range entries {
			if !constScalarMatchesType(item, typ.ItemType) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func constScalarMatchesType(val any, itemType *ast.TypeDef) bool {
	if val == nil || itemType == nil {
		return false
	}

	switch itemType.Type {
	case ast.DataTypeEnum:
		return itemType.Enum != nil && slices.Contains(itemType.Enum.Values, fmt.Sprint(val))
	case ast.DataTypeBoolean, ast.DataTypeInteger, ast.DataTypeInt32, ast.DataTypeBigInt,
		ast.DataTypeNumber, ast.DataTypeFloat32, ast.DataTypeDecimal,
		ast.DataTypeString, ast.DataTypeDate, ast.DataTypeDateTime:
		dataType := itemType.Type
		if itemType.Format == "string" {
			dataType = ast.DataTypeString
		}
		return defaultValueMatchesType(val, dataType)
	default:
		return false
	}
}

// defaultValueMatchesType checks whether a decoded default/const value is compatible
// with the given AST data type.
func defaultValueMatchesType(val any, dataType ast.DataType) bool {
	switch dataType {
	case ast.DataTypeBoolean:
		_, ok := val.(bool)
		return ok
	case ast.DataTypeInteger, ast.DataTypeInt32, ast.DataTypeBigInt:
		switch val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		default:
			return false
		}
	case ast.DataTypeNumber, ast.DataTypeFloat32, ast.DataTypeDecimal:
		switch val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64,
			float32, float64:
			return true
		default:
			return false
		}
	case ast.DataTypeString, ast.DataTypeDate, ast.DataTypeDateTime, ast.DataTypeUUID, ast.DataTypeDuration:
		_, ok := val.(string)
		return ok
	default:
		// For complex types (object, array, union, etc.) we don't validate here
		return true
	}
}

// coerceDefaultValue attempts to convert a mismatched default value to the expected type.
// For example, a string "0.00" on a number field is coerced to float64(0).
// Returns the coerced value and true if coercion succeeded, or (nil, false) if not.
func coerceDefaultValue(val any, dataType ast.DataType) (any, bool) {
	str, isString := val.(string)
	if !isString {
		// Coerce numeric types to integers when the field expects an integer.
		// For example, float64(0) from JSON parsing on an integer field.
		switch dataType {
		case ast.DataTypeInteger, ast.DataTypeInt32, ast.DataTypeBigInt:
			switch v := val.(type) {
			case float32:
				if v == float32(int64(v)) {
					return int(int64(v)), true
				}
			case float64:
				if v == float64(int64(v)) {
					return int(int64(v)), true
				}
			}
		}
		return nil, false
	}

	switch dataType {
	case ast.DataTypeBoolean:
		switch strings.ToLower(str) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	case ast.DataTypeInteger, ast.DataTypeInt32, ast.DataTypeBigInt:
		if i, err := strconv.ParseInt(str, 10, 64); err == nil {
			return int(i), true
		}
		// Also accept float strings that represent whole numbers (e.g. "1.0")
		if f, err := strconv.ParseFloat(str, 64); err == nil && f == float64(int64(f)) {
			return int(int64(f)), true
		}
	case ast.DataTypeNumber, ast.DataTypeFloat32, ast.DataTypeDecimal:
		if f, err := strconv.ParseFloat(str, 64); err == nil {
			return f, true
		}
	}

	return nil, false
}

func (s *Schemas) formatEnumValuesForLogging(values []string) string {
	if len(values) == 0 {
		return ""
	}

	if len(values) <= 3 {
		return strings.Join(values, ", ")
	}

	// Show first 3 values and count of remaining
	first3 := values[:3]
	remaining := len(values) - 3
	return fmt.Sprintf("%s... %d more", strings.Join(first3, ", "), remaining)
}

type typeMap struct {
	Type   *ast.TypeDef
	Schema *oas3.JSONSchema[oas3.Referenceable]
	Ref    references.Reference
	node   *yaml.Node
}

type subSchema struct {
	schema *oas3.JSONSchema[oas3.Referenceable]
	node   *yaml.Node
}

// TODO: feature checks should be used instead of this
func (s *Schemas) isTerraformProvider() bool {
	return s.Target.Target == "terraform"
}

func (s *Schemas) handleAnyOfOneOf(ctx context.Context, params Params, nullable bool, subTypes []string) (*ast.FieldDef, error) {
	// TODO: we should document the types that anyOf or oneOf can be in the comments for the field
	js := params.Schema.MustGetResolvedSchema()

	originalSchemaRef := params.Schema.GetSchema().Ref
	originalSchema := js.GetSchema()
	schema := *originalSchema

	originallyAnyOf := false
	if len(schema.GetAnyOf()) > 0 {
		schema.OneOf = schema.AnyOf
		schema.AnyOf = nil
		originallyAnyOf = true
	}

	if len(subTypes) > 0 {
		// If we had a type array of multiple types treat that as a oneOf

		// create empty base schema to merge with
		schema = oas3.Schema{}

		for _, subType := range subTypes {
			subTypeSchema := &oas3.Schema{
				Type: oas3.NewTypeFromString(oas3.SchemaType(subType)),
			}

			// TODO: we don't consider all other properties of the schema and maybe we should
			switch subType {
			case "string", "integer":
				// If we have enums carry them over, but be smart about which subtype gets them
				if len(originalSchema.GetEnum()) > 0 {
					enumsNodes := s.distributeEnumsToSubtype(originalSchema.Enum, subType, subTypes)
					if len(enumsNodes) > 0 {
						subTypeSchema.Enum = enumsNodes
					}
				}
			case "object":
				// if we have an object type and the schema also has object properties make sure to carry them over
				// TODO: probably more we should associate
				subTypeSchema.Properties = originalSchema.Properties
				subTypeSchema.Required = originalSchema.Required
				subTypeSchema.AdditionalProperties = originalSchema.AdditionalProperties
			}

			schema.OneOf = append(schema.OneOf, oas3.NewJSONSchemaFromSchema[oas3.Referenceable](subTypeSchema))
		}
	}

	schemas := make([]subSchema, 0, len(schema.OneOf))

	allSamePrimitiveType := true
	lastType := ""
	var firstSchema *oas3.JSONSchema[oas3.Referenceable]
	var firstResolvedSchema *oas3.Schema
	// First member that declares a type, used when the member the flattened
	// schema is copied from only implied its type (for example a bare const)
	var firstTypedSchema *oas3.Schema
	nullableTypeFound := false
	// Members collapsed into the flattened primitive that each described a
	// closed set of values (an enum or a const), and the values they allowed
	numCollapsedClosedSetMembers := 0
	collapsedClosedSetValues := []values.Value{}
	memberFormats := []string{}
	// Values allowed by untyped const or enum members that collapsed into the
	// flattened primitive; a sibling's format only holds if they all fit it
	untypedMemberValues := []values.Value{}

	for i, o := range schema.GetOneOf() {
		ss, err := resolution.Resolve(ctx, o, params.DocInfo)
		if err != nil {
			return nil, err
		}

		typ, _, subTypeNullable := openapi.GetResolvedType(ctx, ss)
		if subTypeNullable {
			nullable = true
		}

		if ss.IsSchema() {
			// If its a nullable type then we can ignore the null type
			isNullType := len(ss.GetSchema().GetType()) == 1 && ss.GetSchema().GetType()[0] == "null"
			isNullEnum := len(ss.GetSchema().GetEnum()) == 1 && ss.GetSchema().GetEnum()[0] == nil
			if isNullType || isNullEnum {
				nullableTypeFound = true
				continue
			}
		}

		if firstSchema == nil {
			firstSchema = o
			firstResolvedSchema = ss.GetSchema()
		}

		node := originalSchema.GetRootNode()
		if len(subTypes) > 0 {
			node = openapi.GetTypePropertyNode(js)
		} else {
			var anyOfOneOfNode marshaller.Node[[]core.JSONSchema]

			if schema.GetCore() != nil {
				if originallyAnyOf {
					anyOfOneOfNode = schema.GetCore().AnyOf
				} else {
					anyOfOneOfNode = schema.GetCore().OneOf
				}
			}

			// Shouldn't need to do this check here but found a case where the low wasn't populated as I expect
			if i < len(anyOfOneOfNode.Value) {
				node = anyOfOneOfNode.Value[i].GetRootNode()
			}
		}

		schemas = append(schemas, subSchema{
			schema: o,
			node:   node,
		})
		if len(ss.GetSchema().GetType()) > 0 {
			memberFormats = append(memberFormats, ss.GetSchema().GetFormat())
		}

		if firstTypedSchema == nil && len(ss.GetSchema().GetType()) > 0 {
			firstTypedSchema = ss.GetSchema()
		}

		isEnumMember := len(ss.GetSchema().GetEnum()) > 0
		constValue := ss.GetSchema().GetConst()

		// A path parameter can't serialize a union, so the whole operation would
		// otherwise be skipped. Treat an enum member as its underlying primitive
		// so a same-primitive union flattens to it. Anywhere else an enum member
		// keeps its own type and the union stands.
		isPathParam := params.ParamType == ParamTypePath

		// If its a complex type then don't try to flatten
		switch {
		case slices.Contains([]string{"object", "oneOf", "allOf", "anyOf", "any"}, typ) || (isEnumMember && !isPathParam):
			allSamePrimitiveType = false
		case s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureSliceUnions) && typ == "array":
			allSamePrimitiveType = false
		case lastType == "" || lastType == typ:
			lastType = typ
			if len(ss.GetSchema().GetType()) == 0 {
				untypedMemberValues = append(untypedMemberValues, ss.GetSchema().GetEnum()...)
				if constValue != nil {
					untypedMemberValues = append(untypedMemberValues, constValue)
				}
			}
			// Enums and consts each describe a closed set of values, so track
			// what they allowed: the collapsed parameter can only stay
			// constrained if every member was one of them.
			if isPathParam {
				switch {
				case isEnumMember:
					numCollapsedClosedSetMembers++
					collapsedClosedSetValues = append(collapsedClosedSetValues, ss.GetSchema().GetEnum()...)
				case constValue != nil:
					numCollapsedClosedSetMembers++
					collapsedClosedSetValues = append(collapsedClosedSetValues, constValue)
				}
			}
		default:
			allSamePrimitiveType = false
		}
	}

	typeName, refType, contextStack, parentComponentRef, parentComponentDescription, err := s.handleReferencedType(ctx, &params, params.Schema, originalSchema)
	if err != nil {
		return nil, err
	}

	childContextStack := contextStack
	if originalSchemaRef == nil {
		childContextStack = append(contextStack, ast.ContextFrame{
			Type:       ast.ContextTypeOneOf,
			Identifier: typeName,
		})
	}

	if len(schemas) == 1 && nullableTypeFound {
		resolvedSchema, err := resolution.Resolve(ctx, firstSchema, params.DocInfo)
		if err != nil {
			return nil, err
		}

		merged, emptyBaseSchema, err := s.mergeOneOfSchemas(ctx, &schema, resolvedSchema.GetSchema(), params.DocInfo)
		if err != nil {
			return nil, err
		}

		var mergedRefJS *oas3.JSONSchema[oas3.Referenceable]
		if firstSchema.IsReference() && emptyBaseSchema {
			mergedRefJS = oas3.NewReferencedScheme(ctx, firstSchema.GetRef(), oas3.NewJSONSchemaFromSchema[oas3.Concrete](merged))
		} else {
			mergedRefJS = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](merged)
		}

		// If it was merged with the parent schema and changed then it is no longer the referenced type so we need to add the namespace
		if !emptyBaseSchema && firstSchema.IsReference() && !resolvedSchema.GetSchema().IsEqual(merged) {
			if s.Config.Generation.Fixes.NameResolutionDec2023 {
				refName, _ := namer.GetRefName(firstSchema.GetRef())

				params.ContextStack = append(childContextStack, ast.ContextFrame{
					Type:       ast.ContextTypeRefName,
					Identifier: refName,
				})
			} else {
				typeName, refType, _, err := s.Namer.GetTypeName(ctx, firstSchema, ast.ContextStack{}, "", false)
				if err != nil {
					return nil, err
				}
				addComponentContext(&params, typeName, refType)
			}

			openapi.TrackRefForCircularReferences(merged, firstSchema.GetRef().String())
		}

		params.Schema = mergedRefJS
		params.Nullable = nullable
		fieldDef, err := s.HandleSchema(ctx, params)
		if err != nil {
			return nil, err
		}
		// Public exports declared on the wrapper alias the resolved type.
		// Only the exports extension is applied — and additively — because
		// the resolved TypeDef can be the shared cached instance for a
		// referenced component; merging arbitrary wrapper extensions onto it
		// would leak them into every other use of the component.
		if fieldDef.Type != nil {
			wrapperExports, err := s.Subsystem.Extensions.HandlePublicExportsExtension(originalSchema.GetExtensions())
			if err != nil {
				return nil, fmt.Errorf("error handling nullable wrapper exports: %w", err)
			}
			if len(wrapperExports) > 0 {
				fieldDef.Type.EnsureExtensions()
				existing := fieldDef.Type.Extensions.PublicExports
				existingKeys := map[string]bool{}
				for _, export := range existing {
					existingKeys[export.Key()] = true
				}
				for _, export := range wrapperExports {
					if !existingKeys[export.Key()] {
						existing = append(existing, export)
						existingKeys[export.Key()] = true
					}
				}
				fieldDef.Type.Extensions.PublicExports = existing
			}
		}
		return fieldDef, nil
	}

	// All the sub schema types were the same primitive we can flatten
	if allSamePrimitiveType {
		descriptions := make([]*string, 0, len(schemas))
		examples := make([]subSchemaExamples, 0, len(schemas))
		maximums := make([]*float64, 0, len(schemas))
		maxLengths := make([]*int64, 0, len(schemas))
		maxItems := make([]*int64, 0, len(schemas))
		minimums := make([]*float64, 0, len(schemas))
		minLengths := make([]*int64, 0, len(schemas))
		minItems := make([]*int64, 0, len(schemas))
		patterns := make([]*string, 0, len(schemas))

		for _, subSchema := range schemas {
			oasSchema := subSchema.schema.GetSchema()

			descriptions = append(descriptions, oasSchema.Description)
			examples = append(examples, subSchemaExamples{
				Example:  oasSchema.Example,
				Examples: oasSchema.Examples,
			})
			maximums = append(maximums, oasSchema.Maximum)
			maxLengths = append(maxLengths, oasSchema.MaxLength)
			maxItems = append(maxItems, oasSchema.MaxItems)
			minimums = append(minimums, oasSchema.Minimum)
			minLengths = append(minLengths, oasSchema.MinLength)
			minItems = append(minItems, oasSchema.MinItems)
			patterns = append(patterns, oasSchema.Pattern)
		}

		mergedSchema := firstSchema.GetSchema().ShallowCopy()

		// The flattened schema inherits the first member's enum or const, which
		// no longer describes the collapsed union. If every member described a
		// closed set merge their values, otherwise the union is open and the
		// constraint must be dropped. A single member is left alone: its
		// constraint still describes the whole union.
		if numCollapsedClosedSetMembers > 0 && len(schemas) > 1 {
			// A referenced first member carries its enum or const on the
			// component it points at, so inline the resolved schema before
			// rewriting it. Left as a reference the rewrite is silently
			// discarded and the component's own constraint wins.
			if firstSchema.IsReference() && (len(firstResolvedSchema.GetEnum()) > 0 || firstResolvedSchema.GetConst() != nil) {
				mergedSchema = firstResolvedSchema.ShallowCopy()
			}

			mergedSchema.Const = nil

			if numCollapsedClosedSetMembers == len(schemas) {
				mergedSchema.Enum = mergeEnumValues(collapsedClosedSetValues)
			} else {
				mergedSchema.Enum = nil
			}
		}

		if len(memberFormats) > 0 {
			sharedFormat := memberFormats[0]
			switch {
			case slices.ContainsFunc(memberFormats, func(f string) bool { return f != sharedFormat }):
				if mergedSchema.IsReference() {
					mergedSchema = firstResolvedSchema.ShallowCopy()
				}
				mergedSchema.Format = widestFormat(lastType, memberFormats)
			case sharedFormat != "" && !valuesFitFormat(lastType, sharedFormat, untypedMemberValues):
				if mergedSchema.IsReference() {
					mergedSchema = firstResolvedSchema.ShallowCopy()
				}
				mergedSchema.Format = nil
			case sharedFormat != "" && mergedSchema.Format == nil && !mergedSchema.IsReference():
				mergedSchema.Format = &sharedFormat
			}
		}

		if mergedSchema.Type == nil && firstTypedSchema != nil && !mergedSchema.IsReference() {
			mergedSchema.Type = firstTypedSchema.Type
		}

		// Preserve the parent schema's type and format when sub-schemas don't
		// specify them. For example, `type: integer, format: int32` with
		// `oneOf: [{const: 1}, {const: 2}]` inherits both from the parent.
		if mergedSchema.Type == nil && originalSchema.Type != nil {
			mergedSchema.Type = originalSchema.Type
		}
		if mergedSchema.Format == nil && originalSchema.Format != nil {
			mergedSchema.Format = originalSchema.Format
		}

		// For description, prefer the parent oneOf schema's description if present,
		// otherwise use the first non-nil description from sub-schemas.
		mergedSchema.Description = mergeDescriptions(originalSchema.Description, descriptions)

		// For examples, prefer the parent oneOf schema's example/examples if present,
		// otherwise use the first non-empty example or examples from sub-schemas.
		mergedSchema.Example, mergedSchema.Examples = mergeExamples(originalSchema.Example, originalSchema.Examples, examples)

		// Merge validations from all oneOf members into the flattened schema.
		// For unions, a value only needs to match ONE of the schemas, so use
		// the least restrictive values:
		//
		// - Minimums: take the smallest (least restrictive lower bound)
		// - Maximums: take the largest (least restrictive upper bound)
		// - Patterns: combined with alternation (pattern1|pattern2|pattern3)
		// - If any schema has nil for a validation, the merged result is nil
		//   (no restriction), since that schema accepts any value for that field.
		//
		// Without this merging, only the first schema validations are applied,
		// which is invalid for the overall union.
		mergedSchema.Maximum = leastRestrictiveMax(maximums)
		mergedSchema.MaxItems = leastRestrictiveMax(maxItems)
		mergedSchema.MaxLength = leastRestrictiveMax(maxLengths)
		mergedSchema.Minimum = leastRestrictiveMin(minimums)
		mergedSchema.MinItems = leastRestrictiveMin(minItems)
		mergedSchema.MinLength = leastRestrictiveMin(minLengths)
		pattern := mergePatterns(patterns)
		if mergedSchema.Pattern != nil && (pattern == nil || *mergedSchema.Pattern != *pattern) {
			messageName := s.Subsystem.Extensions.GetResolvedName(extensions.ExtPatternErrorMessage)
			if mergedSchema.GetExtensions().Has(messageName) {
				mergedSchema.Extensions = &oaextensions.Extensions{Map: sequencedmap.From(mergedSchema.GetExtensions().All())}
				mergedSchema.Extensions.Delete(messageName)
			}
		}
		mergedSchema.Pattern = pattern

		// When all oneOf sub-schemas are consts, preserve the parent title so a
		// variant title does not leak into naming. If there are multiple const
		// branches, flatten them into an enum. A single const branch should remain
		// a typed const.
		enumValues := make([]values.Value, 0, len(schemas))
		allConst := true
		for _, sub := range schemas {
			c := sub.schema.GetSchema().GetConst()
			if c == nil {
				allConst = false
				break
			}
			enumValues = append(enumValues, c)
		}
		if allConst {
			mergedSchema.Title = originalSchema.Title
			if len(schemas) > 1 {
				mergedSchema.Enum = enumValues
				mergedSchema.Const = nil
			}
		}

		params.Schema = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](mergedSchema)
		params.Nullable = nullable

		fieldDef, err := s.HandleSchema(ctx, params)

		if err != nil {
			return nil, err
		}

		// Primitive type handlers don't set Comments on TypeDef, so we need to
		// handle it here for flattened oneOf schemas with descriptions.
		if mergedSchema.Description != nil && fieldDef.Type != nil && fieldDef.Type.Comments == nil {
			fieldDef.Type.Comments = &ast.Comment{
				Description: *mergedSchema.Description,
			}
		}

		return fieldDef, nil
	}

	fieldName, err := s.getFieldName(ctx, params.Schema, params, "oneOf", parentComponentRef)
	if err != nil {
		return nil, err
	}

	if nullable {
		s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureNullables)
	}

	field := &ast.FieldDef{
		Name:     fieldName,
		Optional: nullable,
		Nullable: nullable,
	}

	handleAsUnion := true

	if !s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureUnions) {
		logging.LogWarning(ctx, "validation warning", errors.NewValidationWarning(fmt.Sprintf("oneOf/anyOf or multiple types in type property not supported by '%s' - a generic any type will be generated", s.Target.Target), params.Schema.GetRootNode(), nil))
		handleAsUnion = false
	}

	handleAsUnion = handleAsUnion && len(schema.OneOf) > 0

	if !handleAsUnion {
		typeDef := &ast.TypeDef{
			ComplexAny: true,
			Examples:   s.getExamples(js),
			Location:   getTypeLocation(&schema),
			Type:       ast.DataTypeAny,
		}

		extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
		if err != nil {
			return nil, fmt.Errorf("error handling type extensions: %w", err)
		}

		typeDef.Extensions = extensions

		field.Type = ast.NewType(typeDef, nil)

		return field, nil
	}

	comments, err := s.handleSchemaComments(ctx, originalSchema, parentComponentDescription)
	if err != nil {
		return nil, err
	}

	typeDef := &ast.TypeDef{
		Comments:        comments,
		Examples:        s.getExamples(js),
		IsComponent:     refType != "",
		IsNullableUnion: nullable,
		Location:        getUnionLocation(&schema, originallyAnyOf),
		Name:            typeName,
		Scope:           params.Scope,
		Type:            ast.DataTypeUnion,
	}

	extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
	if err != nil {
		return nil, fmt.Errorf("error handling type extensions: %w", err)
	}

	typeDef.Extensions = extensions

	typ := ast.NewType(typeDef, contextStack)

	if params.CircularReference {
		typeDef.Truncated = params.CircularReference
		field.Type = s.Subsystem.Register.RegisterType(ctx, typ, params.IsRequest)
		return field, nil
	}

	associatedTypesMap := []typeMap{}

	// TODO: need to validate discriminator and make sure oneOf is only using references within it
	typeMapping := map[references.Reference]typeMap{}

	if len(schemas) > 0 {
		for i, subSchema := range schemas {
			resolvedSubSchema, err := resolution.Resolve(ctx, subSchema.schema, params.DocInfo)
			if err != nil {
				return nil, err
			}

			merged, emptyBaseSchema, err := s.mergeOneOfSchemas(ctx, &schema, resolvedSubSchema.GetSchema(), params.DocInfo)
			if err != nil {
				return nil, err
			}

			originalRef := subSchema.schema.GetRef()

			refParams := params
			refParams.ParentOneOfSchema = params.Schema
			refParams.ContextStack = childContextStack
			refParams.LoopContext = append(refParams.LoopContext, LoopFrame{Type: "oneOf", Restricted: len(schema.OneOf) == 1})
			refParams.UsedInUnion = true

			var mergedSchema *oas3.JSONSchema[oas3.Referenceable]
			if subSchema.schema.IsReference() {
				if !emptyBaseSchema && !resolvedSubSchema.GetSchema().IsEqual(merged) {
					if s.Config.Generation.Fixes.NameResolutionDec2023 {
						refName, _ := namer.GetRefName(originalRef)

						refParams.ContextStack = append(childContextStack, ast.ContextFrame{
							Type:       ast.ContextTypeRefName,
							Identifier: refName,
						})
					} else {
						typeName, refType, _, err := s.Namer.GetTypeName(ctx, oas3.NewReferencedScheme(ctx, originalRef, oas3.NewJSONSchemaFromSchema[oas3.Concrete](merged)), ast.ContextStack{}, "", false)
						if err != nil {
							return nil, err
						}
						addComponentContext(&refParams, typeName, refType)
					}

					openapi.TrackRefForCircularReferences(merged, originalRef.String())
					// Schema was changed by merge, treat as non-reference
					mergedSchema = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](merged)
				} else {
					// Schema unchanged by merge, preserve reference
					mergedSchema = oas3.NewReferencedScheme(ctx, originalRef, oas3.NewJSONSchemaFromSchema[oas3.Concrete](merged))
				}
			} else {
				// If we have a inline oneOf schema and it doesn't have a title it will just get named based on its position in the array
				refParams.ContextStack.AppendWithHumanized(ast.ContextTypeOneOfPosition, strconv.Itoa(i+1), "")
				// Schema is inline, treat as non-reference
				mergedSchema = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](merged)
			}

			refParams.Schema = mergedSchema
			f, err := s.HandleSchema(ctx, refParams)
			if err != nil {
				return nil, err
			}

			tm := typeMap{
				Type:   f.Type,
				Schema: mergedSchema,
				Ref:    originalRef,
				node:   subSchema.node,
			}

			associatedTypesMap = append(associatedTypesMap, tm)

			if originalRef != "" {
				typeMapping[utils.GetSimplifiedRef(originalRef.String())] = tm
			}
		}
	}

	var discriminator *ast.Discriminator

	if schema.Discriminator != nil {
		updatedMapping := []*ast.DiscriminatorMapping{}
		var updatedMappingErr error

		propertyName := schema.Discriminator.PropertyName

		if schema.Discriminator.Mapping.Len() > 0 {
			for typ, ref := range schema.Discriminator.Mapping.AllOrdered(s.Config.GetSequencedMapIterationOrder()) {
				// use the same function used previously to construct the key to typeMapping
				typeMap, ok := typeMapping[utils.GetSimplifiedRef(ref)]
				if !ok {
					updatedMappingErr = fmt.Errorf("discriminator mapping ref %q not found in oneOf", ref)
					break
				}

				typeMapSchema := typeMap.Schema.GetResolvedSchema().GetSchema()

				if err := s.validateDiscriminatorForType(typeMap.Type, typ, ref, propertyName, typeMapSchema, typeMap.node, &schema); err != nil {
					updatedMappingErr = fmt.Errorf("discriminator validation failed: %w", err)
					break
				}

				updatedMapping = append(updatedMapping, &ast.DiscriminatorMapping{
					Name: typ,
					Type: typeMap.Type,
				})
			}
		} else {
			for i, typeMap := range associatedTypesMap {
				if _, ok := typeMapping[typeMap.Ref]; !ok {
					updatedMappingErr = errors.NewValidationError("object in oneOf array must be a reference when using discriminator", schema.OneOf[i].GetRootNode(), nil)
					break
				}

				typ, _ := namer.GetRefName(typeMap.Ref)

				typeMapSchema := typeMap.Schema.GetResolvedSchema().GetSchema()

				if err := s.validateDiscriminatorForTypeImplicit(typeMap.Type, typ, typeMap.Ref.String(), propertyName, typeMapSchema, schema.OneOf[i].GetRootNode()); err != nil {
					updatedMappingErr = fmt.Errorf("discriminator validation failed: %w", err)
					break
				}

				updatedMapping = append(updatedMapping, &ast.DiscriminatorMapping{
					Name: typ,
					Type: typeMap.Type,
				})
			}
		}

		if updatedMappingErr == nil {
			discriminator = &ast.Discriminator{
				TypePropertyName: propertyName,
				Mapping:          updatedMapping,
			}

			nameOverrides, err := s.Subsystem.Extensions.GetDiscriminatorNameOverrides(&schema)
			if err != nil {
				logging.LogWarning(ctx, "validation warning", errors.NewValidationWarning("failed to parse x-speakeasy-discriminator", schema.GetDiscriminator().GetRootNode(), err))
			} else if len(nameOverrides) > 0 {
				for _, m := range discriminator.Mapping {
					if displayName, ok := nameOverrides[m.Name]; ok {
						m.DisplayName = displayName
					}
				}
			}
		} else {
			var node *yaml.Node
			if schema.GetDiscriminator().GetCore().Mapping.Present {
				node = schema.GetDiscriminator().GetCore().Mapping.GetKeyNodeOrRoot(schema.GetDiscriminator().GetRootNode())
			} else if schema.GetDiscriminator().GetCore().PropertyName.Present {
				node = schema.GetDiscriminator().GetCore().PropertyName.GetKeyNodeOrRoot(schema.GetDiscriminator().GetRootNode())
			}
			logging.LogWarning(ctx, "validation warning", errors.NewValidationWarning(fmt.Sprintf("%v, ignoring discriminator", updatedMappingErr), node, nil))
		}
	}

	inputTypes := false

	outputTypes := false

	associatedTypes := ast.TypeDefs{}

	for _, typeMap := range associatedTypesMap {
		associatedTypes = append(associatedTypes, typeMap.Type)

		var input bool
		if s.Config.Generation.Fixes.NameResolutionDec2023 {
			input = typeMap.Type.IsInput(true)
		} else {
			input = typeMap.Type.Input
		}

		var output bool
		if s.Config.Generation.Fixes.NameResolutionDec2023 {
			output = typeMap.Type.IsOutput(true)
		} else {
			output = typeMap.Type.Output
		}

		if input {
			inputTypes = true
		}
		if output {
			outputTypes = true
		}
	}

	typ.AssociatedTypes = associatedTypes

	typ.Discriminator = discriminator

	canBeUnionType := true
	if discriminator == nil {
		uniqueTypes := map[string]bool{}

		for _, associatedType := range associatedTypes {
			fullyQualifiedType := associatedType.GetFullyQualifiedName()
			if _, ok := uniqueTypes[fullyQualifiedType]; ok {
				logging.LogWarning(ctx, "validation warning", errors.NewValidationWarning("oneOf schemas must be unique or have a discriminator, treating as any type", params.Schema.GetRootNode(), nil))
				canBeUnionType = false
				break
			}
			uniqueTypes[fullyQualifiedType] = true
		}
	}

	if canBeUnionType {
		if inputTypes && params.IsRequest {
			typ.ContextStack = s.addInputOutputContext(typ.ContextStack, inputSuffix)
		}
		if outputTypes && !params.IsRequest {
			typ.ContextStack = s.addInputOutputContext(typ.ContextStack, outputSuffix)
		}

		typ = s.Subsystem.Register.RegisterType(ctx, typ, params.IsRequest)

		for _, typ := range associatedTypes {
			if discriminator == nil {
				s.Subsystem.Register.RegisterTypeUsedInWeakUnion(typ)
			}
		}
	} else {
		typ = ast.NewType(&ast.TypeDef{
			Type:            ast.DataTypeAny,
			Extensions:      extensions,
			Location:        getTypeLocation(&schema),
			AssociatedTypes: associatedTypes,
		}, nil)
	}

	if typ.Type == ast.DataTypeUnion {
		s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureUnions)
	}

	field.Type = typ
	return field, nil
}

func (s *Schemas) handleAnyType(ctx context.Context, params Params, nullable bool) (*ast.FieldDef, error) {
	schema := params.Schema.MustGetResolvedSchema()

	var parentComponentRef references.Reference
	if s.Config.Generation.Fixes.RequestResponseComponentNamesFeb2024 {
		parentComponentRef = params.ParentComponentRef
	}

	fieldName, err := s.getFieldName(ctx, params.Schema, params, "any", parentComponentRef)
	if err != nil {
		return nil, err
	}

	typeDef := &ast.TypeDef{
		Examples: s.getExamples(schema),
		Location: getTypeLocation(schema.GetSchema()),
		Type:     ast.DataTypeAny,
	}

	extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
	if err != nil {
		return nil, fmt.Errorf("error handling type extensions: %w", err)
	}

	typeDef.Extensions = extensions

	return s.handleFieldDefMetaData(ctx, &ast.FieldDef{
		Name: fieldName,
		Type: ast.NewType(typeDef, nil),
	}, nullable, schema.GetSchema()), nil
}

func (s *Schemas) handleString(ctx context.Context, params Params, nullable bool) (*ast.FieldDef, error) {
	js := params.Schema.MustGetResolvedSchema()
	schema := js.GetSchema()

	dataType := ast.DataTypeString
	outputType := false

	format := ""

	switch schema.GetFormat() {
	case "date":
		dataType = ast.DataTypeDate
	case "date-time":
		dataType = ast.DataTypeDateTime
	case "uuid":
		if s.Subsystem.Features.IsFeatureEnabled(ctx, features.FeatureUUID) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureUUID)
			dataType = ast.DataTypeUUID
		} else {
			format = schema.GetFormat()
		}
	case "duration":
		if s.Subsystem.Features.IsFeatureEnabled(ctx, features.FeatureDuration) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureDuration)
			dataType = ast.DataTypeDuration
		} else {
			format = schema.GetFormat()
		}
	case "binary":
		if s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureFormatBinary) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureFormatBinary)
		}

		switch params.SerializationMethod {
		case ast.SerializationMethodMultipart:
			return s.handleMultipartFormDataBinary(ctx, params)
		case ast.SerializationMethodForm:
			return nil, errors.NewValidationError("binary data not supported in application/x-www-form-urlencoded data without encoding", params.Schema.GetRootNode(), nil)
		}

		dataType = ast.DataTypeBytes

		if params.IsRequest && params.Depth == 1 && s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureUploadStreams) {
			dataType = ast.DataTypeRequestStream
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureUploadStreams)
		} else if !params.IsRequest && params.Depth == 1 && s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureDownloadStreams) {
			dataType = ast.DataTypeResponseStream
			outputType = true
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureDownloadStreams)
		}
	case "byte":
		// format:byte denotes a base64-encoded string. The string surface stays — opt-in
		// to richer input ergonomics (file/PathLike/IO) is signaled via the
		// x-speakeasy-base64-input-mode extension, handled after typeDef construction.
		format = schema.GetFormat()
	case "int64":
		if s.Subsystem.Features.IsFeatureSupported(context.Background(), features.FeatureStringNumberFormats) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureStringNumberFormats)
			dataType = ast.DataTypeInteger
			format = "string"
		} else {
			format = schema.GetFormat()
		}
	case "float64":
		if s.Subsystem.Features.IsFeatureSupported(context.Background(), features.FeatureStringNumberFormats) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureStringNumberFormats)
			dataType = ast.DataTypeNumber
			format = "string"
		} else {
			format = schema.GetFormat()
		}
	case "bigint":
		if s.Subsystem.Features.IsFeatureSupported(context.Background(), features.FeatureBigInt) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureBigInt)
			dataType = ast.DataTypeBigInt
			format = "string"
		} else {
			format = schema.GetFormat()
		}
	case "decimal":
		if s.Subsystem.Features.IsFeatureSupported(context.Background(), features.FeatureDecimal) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureDecimal)
			dataType = ast.DataTypeDecimal
			format = "string"
		} else {
			format = schema.GetFormat()
		}
	default:
		format = schema.GetFormat()
	}

	name, err := s.getFieldName(ctx, params.Schema, params, string(dataType), "")
	if err != nil {
		return nil, err
	}

	typeDef := &ast.TypeDef{
		Examples: s.getExamples(js),
		Format:   format,
		Location: getTypeLocation(schema),
		Output:   outputType,
		Type:     dataType,
		Validations: &ast.Validations{
			MaxLength: schema.MaxLength,
			MinLength: schema.MinLength,
		},
	}

	// Set TypeDef.Name when explicitly specified via title or x-speakeasy-name-override
	// This allows union member field names to use custom names instead of hardcoded defaults
	if s.respectTitlesForPrimitiveUnionMembers() && name != string(dataType) {
		typeDef.Name = name
	}

	extensions, err := ast.NewTypeDefExtensions(s.Subsystem.Extensions, typeDef, schema.GetExtensions())
	if err != nil {
		return nil, fmt.Errorf("error handling type extensions: %w", err)
	}

	typeDef.Extensions = extensions

	s.applyBase64InputMode(ctx, typeDef, schema, params)

	if contentMediaType := schema.GetContentMediaType(); contentMediaType == "application/json" {
		if dataType != ast.DataTypeString {
			return nil, errors.NewValidationError(
				fmt.Sprintf("contentMediaType application/json requires string type, got %s", dataType),
				schema.GetPropertyNode("ContentMediaType"),
				nil,
			)
		}

		if s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureContentMediaTypeApplicationJSON) {
			s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureContentMediaTypeApplicationJSON)
			typeDef.ContentMediaType = contentMediaType
		}
	}

	if schema.GetPattern() != "" {
		if s.isTerraformProvider() {
			_, err := regexp.Compile(schema.GetPattern())
			if err != nil {
				logging.LogWarning(
					ctx,
					"validation warning",
					errors.NewValidationWarning("pattern must be valid RE2 engine expression for Go-based targets", schema.GetPropertyNode("Pattern"), err),
				)
			}
		}
		typeDef.Validations.Pattern = schema.Pattern
	}

	return s.handleFieldDefMetaData(ctx, &ast.FieldDef{
		Name: name,
		Type: ast.NewType(typeDef, nil),
	}, nullable, schema), nil
}

// applyBase64InputMode promotes a valid x-speakeasy-base64-input-mode extension onto a request-side TypeDef.
func (s *Schemas) applyBase64InputMode(_ context.Context, typeDef *ast.TypeDef, schema *oas3.Schema, params Params) {
	if !params.IsRequest {
		// Response/inbound fields stay plain base64 strings.
		return
	}
	mode, err := s.Subsystem.Extensions.Base64InputMode(schema)
	if err != nil || mode == "" {
		return
	}
	if schema.GetFormat() != "byte" && schema.GetContentEncoding() != "base64" {
		return
	}
	typeDef.EnsureExtensions()
	typeDef.Extensions.Base64InputMode = mode
}

func (s *Schemas) handleFieldDefMetaData(ctx context.Context, fieldDef *ast.FieldDef, nullable bool, schema *oas3.Schema) *ast.FieldDef {
	constValue := s.sanitizeDefaultConstValue(ctx, schema.GetConst())
	defaultValue := s.sanitizeDefaultConstValue(ctx, schema.GetDefault())

	fieldDef.Const = constValue
	fieldDef.Default = defaultValue
	fieldDef.Nullable = nullable

	// Clear default/const values that don't match the field's declared primitive type.
	// This prevents invalid literals (e.g. string "variant" for a boolean field) from
	// reaching templates and causing compile errors in generated SDKs.
	s.checkDefaultConformsToType(ctx, fieldDef, schema)

	fieldDef.Optional = (fieldDef.Default != nil || nullable)

	if fieldDef.Nullable {
		s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureNullables)
	}

	return fieldDef
}

func (s *Schemas) containerMetaDataApplies(ctx context.Context, schema *oas3.Schema, defaultFeature, constFeature features.Feature) bool {
	if schema.Default != nil && s.Subsystem.Features.IsFeatureSupported(ctx, defaultFeature) {
		return true
	}

	return schema.Const != nil && s.Subsystem.Features.IsFeatureSupported(ctx, constFeature)
}

func (s *Schemas) recordContainerConstUsage(ctx context.Context, fieldDef *ast.FieldDef, constFeature features.Feature) {
	if fieldDef.Const == nil {
		return
	}

	s.Subsystem.Features.RecordFeatureUsage(ctx, constFeature)
}

func (s *Schemas) sanitizeDefaultConstValue(ctx context.Context, value values.Value) *ast.AnyValue {
	if value == nil {
		return nil
	}

	if !s.Subsystem.Features.IsFeatureSupported(ctx, features.FeatureConstsAndDefaults) {
		return nil
	}

	s.Subsystem.Features.RecordFeatureUsage(ctx, features.FeatureConstsAndDefaults)

	var v any
	if err := value.Decode(&v); err != nil {
		return nil
	}

	return &ast.AnyValue{
		Value: v,
	}
}

func (s *Schemas) handleAllOf(ctx context.Context, params Params, nullable bool) (*ast.FieldDef, error) {
	originalSchema := params.Schema
	refJS := originalSchema
	js := refJS.MustGetResolvedSchema()
	schema := js.GetSchema()

	exts := openapi.HoistAllOfExtensions(ctx, refJS, params.DocInfo, nil, hoistChildExtensionPredicate(s.Config.Generation.Fixes, s.Subsystem.Extensions))
	schemaExtensions := schema.GetExtensions()
	if exts != nil {
		for k, v := range exts.All() {
			schemaExtensions.Set(k, v)
		}
	}
	schema.Extensions = schemaExtensions

	emptySchema := isSchemaEmpty(schema, s.Subsystem.Extensions)

	handled := false

	// Short Circuit if we only have one allOf
	if emptySchema && len(schema.GetAllOf()) == 1 {
		refJS = schema.GetAllOf()[0]
		handled = true
	} else if emptySchema && len(schema.GetAllOf()) == 2 {
		additionalAllOfSchema, err := resolution.Resolve(ctx, schema.GetAllOf()[1], params.DocInfo)
		if err != nil {
			return nil, err
		}

		// Handling a special case where allOf is used to override the description but nothing else about the type
		if additionalAllOfSchema.IsSchema() && isSchemaEmpty(additionalAllOfSchema.GetSchema(), s.Subsystem.Extensions) && additionalAllOfSchema.GetSchema().GetDescription() != "" {
			allOfSchema := schema.GetAllOf()[0]
			resolvedSchema, err := resolution.Resolve(ctx, allOfSchema, params.DocInfo)
			if err != nil {
				return nil, err
			}

			schemaCopy := openapi.Copy(resolvedSchema.GetSchema())
			schemaCopy.Description = additionalAllOfSchema.GetSchema().Description

			if allOfSchema.IsReference() {
				refJS = oas3.NewReferencedScheme(ctx, allOfSchema.GetRef(), oas3.NewJSONSchemaFromSchema[oas3.Concrete](&schemaCopy))
			} else {
				refJS = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](&schemaCopy)
			}
			handled = true
		}
	}

	if !handled {
		toMerge := schema.AllOf

		allOfRef := buildAllOfRef(refJS, s.Subsystem.Extensions)

		subReferences := []string{}

		for _, s := range toMerge {
			ref := utils.GetSimplifiedRef(s.GetRef().String())
			if ref != "" {
				subReferences = append(subReferences, ref.String())
			} else {
				hash := hashing.Hash(s)
				subReferences = append(subReferences, hash)
			}
		}

		slices.Sort(subReferences)

		merged, err := s.mergeAllOfSchemas(ctx, schema, toMerge, params.DocInfo)
		if err != nil {
			return nil, err
		}

		// Re-apply hoisted extensions that may have been filtered during merge
		// Only re-apply extensions that are identifying but not mergeable
		// (e.g., x-speakeasy-name-override is not mergeable but should be preserved)
		if exts != nil {
			mergedExtensions := merged.GetExtensions()
			for k, v := range exts.All() {
				if s.Subsystem.Extensions.IsExtensionIdentifying(k) && !s.Subsystem.Extensions.IsExtensionMergable(k) {
					mergedExtensions.Set(k, v)
				}
			}
			merged.Extensions = mergedExtensions
		}

		overridenRefParts := []string{}
		if allOfRef != "" {
			overridenRefParts = append(overridenRefParts, allOfRef)
		}
		if len(subReferences) > 0 {
			overridenRefParts = append(overridenRefParts, "allOf["+strings.Join(subReferences, ",")+"]")
		}
		openapi.TrackRefForCircularReferences(merged, strings.Join(overridenRefParts, "-"))

		if originalSchema.IsReference() {
			refJS = oas3.NewReferencedScheme(ctx, originalSchema.GetRef(), oas3.NewJSONSchemaFromSchema[oas3.Concrete](merged))
		} else {
			refJS = oas3.NewJSONSchemaFromSchema[oas3.Referenceable](merged)
		}
	}

	params.Schema = refJS
	params.Nullable = nullable

	if params.Schema.IsReference() {
		_, _, _, _, _, err := s.handleReferencedType(ctx, &params, params.Schema, schema)
		if err != nil {
			return nil, err
		}
	}

	return s.HandleSchema(ctx, params)
}

// mergeOneOfSchemas will merge the subSchema with the factored out properties from the top level schema.
// Examples from oneOfSchema are only propagated if they match the subSchema's declared type(s).
func (s *Schemas) mergeOneOfSchemas(ctx context.Context, oneOfSchema *oas3.Schema, subSchema *oas3.Schema, docInfo *document.DocumentInfo) (*oas3.Schema, bool, error) {
	factoredSchema := openapi.Copy(oneOfSchema)
	factoredSchema.AnyOf = nil
	factoredSchema.OneOf = nil

	if openapi.IsEmpty(&factoredSchema, true, s.Subsystem.Extensions) {
		subSchemaCopy := openapi.Copy(subSchema)

		if oneOfSchema.Example != nil && exampleMatchesSchemaType(oneOfSchema.Example, subSchemaCopy.GetType()) {
			subSchemaCopy.Example = oneOfSchema.Example
		}

		if len(oneOfSchema.Examples) > 0 {
			filtered := filterExamplesBySchemaType(oneOfSchema.Examples, subSchemaCopy.GetType())
			if len(filtered) > 0 {
				subSchemaCopy.Examples = filtered
			}
		}

		return &subSchemaCopy, true, nil
	}

	baseSchema := openapi.Copy(subSchema)

	if err := openapi.Merge(ctx, &baseSchema, &factoredSchema, false, false, s.Subsystem.Extensions, config.AllOfMergeStrategyShallowMerge, docInfo); err != nil {
		return nil, false, err
	}

	if oneOfSchema.Example != nil && exampleMatchesSchemaType(oneOfSchema.Example, baseSchema.GetType()) {
		baseSchema.Example = oneOfSchema.Example
	}

	if len(oneOfSchema.Examples) > 0 {
		filtered := filterExamplesBySchemaType(oneOfSchema.Examples, baseSchema.GetType())
		if len(filtered) > 0 {
			baseSchema.Examples = filtered
		}
	}

	return &baseSchema, false, nil
}

// mergeAllOfSchemas merges all the schemas in the allOf array into one schema with later schemas overriding earlier ones
// and with any factored out properties from the top level schema merged over the top of the sub schemas
func (s *Schemas) mergeAllOfSchemas(ctx context.Context, allOfSchema *oas3.Schema, toMerge []*oas3.JSONSchema[oas3.Referenceable], docInfo *document.DocumentInfo) (*oas3.Schema, error) {
	factoredSchema := openapi.Copy(allOfSchema)
	factoredSchema.AllOf = nil

	if len(toMerge) == 0 {
		return &factoredSchema, nil
	}

	baseSchema := &oas3.Schema{}
	baseSchema.SetCore(pointer.From(*allOfSchema.GetCore()))

	for _, sch := range toMerge {
		resolvedSchema, err := resolution.Resolve(ctx, sch, docInfo)
		if err != nil {
			return nil, err
		}
		if err := openapi.Merge(ctx, baseSchema, resolvedSchema.GetSchema(), true, true, s.Subsystem.Extensions, s.Subsystem.Config.Generation.Schemas.AllOfMergeStrategy, docInfo); err != nil {
			return nil, err
		}
	}

	if err := openapi.Merge(ctx, baseSchema, &factoredSchema, true, true, s.Subsystem.Extensions, s.Subsystem.Config.Generation.Schemas.AllOfMergeStrategy, docInfo); err != nil {
		return nil, err
	}

	return baseSchema, nil
}

func hoistChildExtensionPredicate(fixes *config.Fixes, exts *extensions.Extensions) func(string) bool {
	nameOverrideExt := exts.GetResolvedName(extensions.ExtNameOverride)
	patternMessageExt := exts.GetResolvedName(extensions.ExtPatternErrorMessage)
	return func(extName string) bool {
		if extName == patternMessageExt {
			return false
		}
		return fixes == nil || !fixes.NameOverrideFeb2026 || extName != nameOverrideExt
	}
}

func isSchemaEmpty(s *oas3.Schema, exts *extensions.Extensions) bool {
	schema := *s

	// Resetting some fields to their zero value as they are not relevant for the comparison
	schema.AllOf = nil
	schema.Description = nil

	return openapi.IsEmpty(&schema, false, exts)
}

func buildAllOfRef(js *oas3.JSONSchema[oas3.Referenceable], exts *extensions.Extensions) string {
	if js.IsReference() {
		return utils.GetSimplifiedRef(js.GetRef().String()).String()
	}
	schema := js.MustGetResolvedSchema().GetSchema()

	if isSchemaEmpty(schema, exts) {
		return ""
	}

	return hashing.Hash(schema)
}

func (s *Schemas) handleReferencedType(ctx context.Context, params *Params, schema *oas3.JSONSchema[oas3.Referenceable], js *oas3.Schema) (string, string, ast.ContextStack, references.Reference, string, error) {
	contextStack := params.ContextStack

	var parentComponentRef references.Reference
	if s.Config.Generation.Fixes.RequestResponseComponentNamesFeb2024 {
		parentComponentRef = params.ParentComponentRef
	}

	parentComponentDescription := params.ParentComponentDescription

	typeName, refType, contextStack, err := s.Namer.GetTypeName(ctx, schema, contextStack, parentComponentRef, params.SkipNestedRefTracking)
	if err != nil {
		return "", "", nil, "", "", err
	}
	params.ParentComponentRef = "" // Reset the parent component ref as it is only used for the first level of the schema
	params.ParentComponentDescription = ""

	// It's okay to have an empty typeName, if we don't have a decent name - let the namer package work out the best name
	if typeName == "" && !s.Config.Generation.Fixes.NameResolutionFeb2025 {
		panic("typeName should never be empty")
	}

	if schema.IsReference() || parentComponentRef != "" {
		addComponentContext(params, refType, typeName)
		params.Scope = ast.ScopeShared
		oldContextStack := contextStack
		contextStack = params.ContextStack
		// remember if we're in an error or not.. this impacts if we should duplicate the type
		if oldContextStack.HasFrameOfType(ast.ContextTypeResponseError) {
			contextStack = append(contextStack, *oldContextStack.FindLastFrameOfType(ast.ContextTypeResponseError))
		}
		contextStack = append(contextStack, ast.ContextFrame{
			Type:       ast.ContextTypeComponent,
			Identifier: "true",
			Used:       s.Config.Generation.Fixes.NameResolutionDec2023,
		})

		// Add model namespace to context stack for type differentiation
		// This allows types with the same name but different namespaces to coexist
		modelNamespace, _ := s.Subsystem.Extensions.GetModelNamespace(js.GetExtensions())
		if modelNamespace != "" {
			contextStack.AppendModelNamespace(modelNamespace)
			// Also update params.ContextStack so inline schemas (properties) inherit the namespace
			params.ContextStack = contextStack
		}
	}

	return typeName, refType, contextStack, parentComponentRef, parentComponentDescription, nil
}

// Fetches a cached TypeDef if schema is a reference and registration ID is in cache
func (s *Schemas) getCachedType(params *Params, contextStack ast.ContextStack, schema *oas3.JSONSchema[oas3.Referenceable], typeName string) *ast.TypeDef {
	// To keep the blast radius small we only cache referenced types
	if !schema.IsReference() {
		return nil
	}

	// Assert that the cache is initialized
	if params.TypeDefCache == nil {
		panic("params.TypeDefCache is nil")
	}

	// Build the registration ID from the current context stack
	registrationID := ast.GetRegistrationID(params.Scope, contextStack, typeName)

	t, ok := params.TypeDefCache[registrationID]
	if !ok {
		return nil
	}

	return t
}

// Caches TypeDef by registration ID if not truncated
func (s *Schemas) setCachedType(t *ast.TypeDef, params *Params) {
	if params.TypeDefCache == nil {
		params.TypeDefCache = map[string]*ast.TypeDef{}
	}

	if t.Truncated {
		return
	}

	params.TypeDefCache[t.GetRegistrationID()] = t
}

func addComponentContext(params *Params, refType, typeName string) {
	// Reset the context stack for components
	params.ContextStack = ast.ContextStack{}
	params.ContextStack.AppendRefType(refType)
	params.ContextStack.Append(ast.ContextTypeRefName, typeName)
}

type inputOutputSuffixType string

const (
	inputSuffix  inputOutputSuffixType = "Input"
	outputSuffix inputOutputSuffixType = "Output"
)

func (s *Schemas) addInputOutputContext(contextStack ast.ContextStack, typ inputOutputSuffixType) ast.ContextStack {
	identifier := ""

	switch typ {
	case inputSuffix:
		if suf, ok := s.Config.GetLanguageConfigValue("inputModelSuffix").(string); ok && suf != "" {
			identifier = suf
		}
	case outputSuffix:
		if suf, ok := s.Config.GetLanguageConfigValue("outputModelSuffix").(string); ok && suf != "" {
			identifier = suf
		}
	}

	return append(contextStack, ast.ContextFrame{
		Type:       ast.ContextTypeInputOutput,
		Identifier: identifier,
	})
}

// respectTitlesForPrimitiveUnionMembers returns true if titles (via title or x-speakeasy-name-override)
// should be used for primitive union member field names instead of type-based defaults. Defaults to true for new SDKs.
func (s *Schemas) respectTitlesForPrimitiveUnionMembers() bool {
	if val, ok := s.Config.GetLanguageConfigValue("respectTitlesForPrimitiveUnionMembers").(bool); ok {
		return val
	}
	return true // default to true for backwards compatibility with SDKs that don't have this config
}

func (s *Schemas) getExamples(schema *oas3.JSONSchema[oas3.Concrete]) []*ast.Example {
	if schema.IsBool() || (len(schema.GetSchema().GetExamples()) == 0 && schema.GetSchema().GetExample() == nil) {
		return nil
	}

	examples := []*ast.Example{}
	if schema.GetSchema().GetExample() != nil {
		examples = append(examples, ast.NewExample("", "", schema.GetSchema().GetExample()))
	} else {
		for _, example := range schema.GetSchema().GetExamples() {
			examples = append(examples, ast.NewExample("", "", example))
		}
	}

	return examples
}

// validateDiscriminatorProperty validates that a discriminator property exists and has the correct type in a class
func (s *Schemas) validateDiscriminatorProperty(typeField *ast.FieldDef, propertyName, ref, typ string, propertyNode *yaml.Node) error {
	if typeField.Type.Type != ast.DataTypeString && typeField.Type.Type != ast.DataTypeEnum {
		return errors.NewValidationError(fmt.Sprintf("discriminator mapping ref %s must have a property named %s of type string or enum", ref, propertyName), propertyNode, nil)
	} else if typeField.Type.Type == ast.DataTypeEnum && (typeField.Type.Enum == nil || !slices.Contains(typeField.Type.Enum.Values, typ)) {
		return errors.NewValidationError(fmt.Sprintf("discriminator mapping ref %s must have a property named %s with an enum value of type %s", ref, propertyName, typ), propertyNode, nil)
	}
	return nil
}

// validateDiscriminatorForType validates discriminator for both class and union types (explicit mapping)
func (s *Schemas) validateDiscriminatorForType(astType *ast.TypeDef, typ, ref, propertyName string, schema *oas3.Schema, _ *yaml.Node, parentSchema *oas3.Schema) error {
	switch astType.Type {
	case ast.DataTypeClass:
		if parentSchema.Properties != nil {
			var err error

			propertyName, err = s.Subsystem.Extensions.GetResolvedSchemaName(parentSchema.GetExtensions(), propertyName)
			if err != nil {
				return err
			}
		}

		if !astType.Truncated {
			typeField := astType.Fields.GetField(propertyName)
			if typeField == nil {
				return errors.NewValidationError(fmt.Sprintf("discriminator mapping ref %s must have a property named %s", ref, propertyName), schema.GetCore().Properties.GetKeyNodeOrRoot(schema.GetRootNode()), nil)
			}

			var propertyNode *yaml.Node
			if schemaProperty, ok := schema.GetProperties().Get(propertyName); ok && schemaProperty != nil {
				propertyNode = schemaProperty.GetRootNode()
			}

			return s.validateDiscriminatorProperty(typeField, propertyName, ref, typ, propertyNode)
		}
	case ast.DataTypeUnion:
		// For union types, validate that the discriminator property is present in all associated types
		for _, associatedType := range astType.AssociatedTypes {
			if associatedType.Type == ast.DataTypeClass && !associatedType.Truncated {
				typeField := associatedType.Fields.GetField(propertyName)
				if typeField == nil {
					return errors.NewValidationError(fmt.Sprintf("discriminator mapping ref %s must have a property named %s in all union associated types", ref, propertyName), schema.GetCore().Properties.GetKeyNodeOrRoot(schema.GetRootNode()), nil)
				}

				var propertyNode *yaml.Node
				// Note: For union associated types, we use the same schema for property line lookup
				if schemaProperty, ok := schema.GetProperties().Get(propertyName); ok && schemaProperty != nil {
					propertyNode = schemaProperty.GetRootNode()
				}

				if err := s.validateDiscriminatorProperty(typeField, propertyName, ref, typ, propertyNode); err != nil {
					return err
				}
			}
		}
	default:
		var key string
		var node *yaml.Node

		for k, v := range parentSchema.GetDiscriminator().GetCore().Mapping.Value.All() {
			if v.Value == ref {
				key = k
				if v.ValueNode != nil {
					node = v.ValueNode
				}
			}
		}

		return errors.NewValidationError(fmt.Sprintf("discriminator mapping ref from key %s to %s must be of type object or union", key, ref), node, nil)
	}

	return nil
}

// validateDiscriminatorPropertyImplicit validates discriminator property for implicit mapping
func (s *Schemas) validateDiscriminatorPropertyImplicit(typeField *ast.FieldDef, propertyName, ref, typ string, schema *oas3.Schema) error {
	if typeField.Type.Type != ast.DataTypeString && typeField.Type.Type != ast.DataTypeEnum {
		return errors.NewValidationError(fmt.Sprintf("discriminator property named %s must be of type string or enum", propertyName), schema.GetCore().Properties.GetKeyNodeOrRoot(schema.GetRootNode()), nil)
	} else if typeField.Type.Type == ast.DataTypeEnum && (typeField.Type.Enum == nil || !slices.Contains(typeField.Type.Enum.Values, typ)) {
		return errors.NewValidationError(fmt.Sprintf("discriminator property named %s must either have all enum values referencing %s as explicit discriminator mappings or this property must have an enum value of %s if not using explicit discriminator mappings", propertyName, ref, typ), schema.GetCore().Properties.GetMapKeyNodeOrRoot(propertyName, schema.GetRootNode()), nil)
	}
	return nil
}

// validateDiscriminatorForTypeImplicit validates discriminator for both class and union types (implicit mapping)
func (s *Schemas) validateDiscriminatorForTypeImplicit(astType *ast.TypeDef, typ, ref, propertyName string, schema *oas3.Schema, node *yaml.Node) error {
	switch astType.Type {
	case ast.DataTypeClass:
		typeField := astType.Fields.GetField(propertyName)
		if typeField == nil {
			return errors.NewValidationError(fmt.Sprintf("object must have a property named %s to match discriminator propertyName", propertyName), schema.GetCore().Properties.GetKeyNodeOrRoot(schema.GetRootNode()), nil)
		}

		return s.validateDiscriminatorPropertyImplicit(typeField, propertyName, ref, typ, schema)
	case ast.DataTypeUnion:
		// For union types, validate that the discriminator property is present in all associated types
		for _, associatedType := range astType.AssociatedTypes {
			if associatedType.Type == ast.DataTypeClass && !associatedType.Truncated {
				typeField := associatedType.Fields.GetField(propertyName)
				if typeField == nil {
					return errors.NewValidationError(fmt.Sprintf("union associated type must have a property named %s to match discriminator propertyName", propertyName), schema.GetCore().Properties.GetKeyNodeOrRoot(schema.GetRootNode()), nil)
				}

				if err := s.validateDiscriminatorPropertyImplicit(typeField, propertyName, ref, typ, schema); err != nil {
					return err
				}
			}
		}
	default:
		return errors.NewValidationError("oneOf variant must be an object or union when using discriminator", node, nil)
	}

	return nil
}

func (s *Schemas) getFieldName(ctx context.Context, schema *oas3.JSONSchema[oas3.Referenceable], params Params, defaultName string, parentComponentRef references.Reference) (string, error) {
	if parentComponentRef == "" && s.Config.Generation.Fixes.RequestResponseComponentNamesFeb2024 {
		parentComponentRef = params.ParentComponentRef
	}

	return s.Namer.GetFieldName(ctx, schema, defaultName, parentComponentRef)
}

// distributeEnumsToSubtype intelligently distributes enum values to the appropriate subtype.
// If we have multiple subtypes including both string and integer, and all enum values are
// valid integers, they should go to the integer subtype, not the string subtype.
func (s *Schemas) distributeEnumsToSubtype(enumNodes []*yaml.Node, currentSubType string, allSubTypes []string) []*yaml.Node {
	if len(enumNodes) == 0 {
		return nil
	}

	// If we only have one subtype or don't have both string and integer, use all enums
	hasString := slices.Contains(allSubTypes, "string")
	hasInteger := slices.Contains(allSubTypes, "integer")

	if !hasString || !hasInteger {
		// No conflict, return all enums
		enumsNodes := make([]*yaml.Node, len(enumNodes))
		copy(enumsNodes, enumNodes)
		return enumsNodes
	}

	// We have both string and integer subtypes, need to be smart about distribution
	allEnumsAreIntegers := true

	// Check if all enum values are valid integers
	for _, enumNode := range enumNodes {
		if enumNode == nil {
			continue
		}

		enumValue := enumNode.Value
		if enumValue == "" {
			allEnumsAreIntegers = false
			break
		}

		// Try to parse as integer
		if _, err := strconv.ParseInt(enumValue, 10, 64); err != nil {
			allEnumsAreIntegers = false
			break
		}
	}

	// Distribute enums based on their nature
	if allEnumsAreIntegers {
		// All enum values are integers, they should go to the integer subtype
		if currentSubType == "integer" {
			enumsNodes := make([]*yaml.Node, len(enumNodes))
			copy(enumsNodes, enumNodes)
			return enumsNodes
		}
		// Don't give enums to string subtype if they're all integers
		return nil
	} else {
		// Mixed or non-integer enums, they should go to the string subtype
		if currentSubType == "string" {
			enumsNodes := make([]*yaml.Node, len(enumNodes))
			copy(enumsNodes, enumNodes)
			return enumsNodes
		}
		// Don't give non-integer enums to integer subtype
		return nil
	}
}
