package schemas

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi-generation/v2/internal/ast"
	"github.com/speakeasy-api/openapi-generation/v2/internal/features"
	"github.com/speakeasy-api/openapi-generation/v2/internal/testutils"
	config "github.com/speakeasy-api/sdk-gen-config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestHandleObject tests the handleObject method with various object scenarios
func TestHandleObject(t *testing.T) {
	tests := []struct {
		name                string
		schemasYAML         string
		schemaName          string
		fixes               *config.Fixes                        // nil means use default
		mockFeatures        func() *testutils.MockFeaturesConfig // nil means use default (all features supported)
		isRequest           *bool                                // nil means use default (false), otherwise use specified value
		serializationMethod *ast.SerializationMethod             // nil means use default (JSON)
		expected            *ast.FieldDef
	}{
		{
			name: "BasicObject",
			schemasYAML: `TestSchema:
  type: object
  properties:
    name:
      type: string
    age:
      type: integer
  required:
    - name`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("age",
							testutils.NewTypeDef(ast.DataTypeInteger).
								WithLocation(&yaml.Node{Line: 14, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("age").
							WithJSONAnnotation("age").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "EmptyObject",
			schemasYAML: `TestSchema:
  type: object`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithEmptyFields().
					Build()).
				Build(),
		},
		{
			name: "ObjectWithNullable",
			schemasYAML: `TestSchema:
  type: object
  nullable: true
  properties:
    name:
      type: string`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 13, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				WithNullable(true).
				WithOptional(true).
				Build(),
		},
		{
			name: "ObjectWithAdditionalPropertiesTrue",
			schemasYAML: `TestSchema:
  type: object
  properties:
    name:
      type: string
  additionalProperties: true`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureAdditionalProperties: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("AdditionalProperties",
							testutils.NewTypeDef(ast.DataTypeMap).
								WithLocation(&yaml.Node{Line: 9, Column: 7}).
								WithItemType(
									testutils.NewTypeDef(ast.DataTypeAny).Build(),
								).
								Build()).
							WithAnnotations(&ast.JSONAnnotation{Ignore: true, FieldName: "-"}, &ast.NeedsCasingAnnotation{}).
							WithOptional(true).
							WithIsAdditionalProperties(true).
							Build(),
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithAdditionalPropertiesFalse",
			schemasYAML: `TestSchema:
  type: object
  properties:
    name:
      type: string
  additionalProperties: false`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithAdditionalPropertiesSchema",
			schemasYAML: `TestSchema:
  type: object
  properties:
    name:
      type: string
  additionalProperties:
    type: string`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureAdditionalProperties: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						// Additional properties field comes first alphabetically
						testutils.NewFieldDef("AdditionalProperties",
							testutils.NewTypeDef(ast.DataTypeMap).
								WithLocation(&yaml.Node{Line: 9, Column: 7}).
								WithItemType(
									testutils.NewTypeDef(ast.DataTypeString).
										WithLocation(&yaml.Node{Line: 14, Column: 9}).
										WithValidations(&ast.Validations{}).
										Build(),
								).
								Build()).
							WithAnnotations(&ast.JSONAnnotation{Ignore: true, FieldName: "-"}, &ast.NeedsCasingAnnotation{}).
							WithOptional(true).
							WithIsAdditionalProperties(true).
							Build(),
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "MapWithConst",
			schemasYAML: `TestSchema:
  type: object
  additionalProperties:
    type: string
  const:
    key: value`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureConstObjects:      true,
					features.FeatureConstsAndDefaults: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeMap).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithItemType(
						testutils.NewTypeDef(ast.DataTypeString).
							WithLocation(&yaml.Node{Line: 11, Column: 9}).
							WithValidations(&ast.Validations{}).
							Build(),
					).
					Build()).
				WithConst(&ast.AnyValue{Value: map[string]interface{}{"key": "value"}}).
				Build(),
		},
		{
			name: "MapWithConstFeatureUnsupported",
			schemasYAML: `TestSchema:
  type: object
  additionalProperties:
    type: string
  const: {}`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureConstsAndDefaults: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeMap).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithItemType(
						testutils.NewTypeDef(ast.DataTypeString).
							WithLocation(&yaml.Node{Line: 11, Column: 9}).
							WithValidations(&ast.Validations{}).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithUnsupportedConst",
			schemasYAML: `TestSchema:
  type: object
  properties:
    name:
      type: string
  const:
    name: "fixed"`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureConstObjects:      true,
					features.FeatureConstsAndDefaults: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithExample",
			schemasYAML: `TestSchema:
  type: object
  properties:
    name:
      type: string
  example:
    name: "John Doe"`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithExamples(&ast.Example{
						Value: &yaml.Node{
							Kind:   yaml.MappingNode,
							Style:  0, // Plain style, not FlowStyle
							Tag:    "!!map",
							Line:   14,
							Column: 9,
							Content: []*yaml.Node{
								{
									Kind:   yaml.ScalarNode,
									Style:  0, // Plain style
									Tag:    "!!str",
									Value:  "name",
									Line:   14,
									Column: 9,
								},
								{
									Kind:   yaml.ScalarNode,
									Style:  yaml.DoubleQuotedStyle,
									Tag:    "!!str",
									Value:  "John Doe",
									Line:   14,
									Column: 15,
								},
							},
						},
					}).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithExtensions",
			schemasYAML: `TestSchema:
  type: object
  properties:
    name:
      type: string
  x-custom: "value"`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithExtensions(map[string]interface{}{
						"x-custom": "value",
					}).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithDefault",
			schemasYAML: `TestSchema:
  type: object
  properties:
    name:
      type: string
  default:
    name: "default"`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureDefaultObjects:    true,
					features.FeatureConstsAndDefaults: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				WithOptional(true).
				WithDefault(&ast.AnyValue{
					Value: map[string]interface{}{
						"name": "default",
					},
				}).
				Build(),
		},
		{
			name: "ObjectReference",
			schemasYAML: `ObjectType:
  type: object
  properties:
    name:
      type: string
TestSchema:
  $ref: "#/components/schemas/ObjectType"`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("ObjectType",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithName("ObjectType").
					WithOriginalName("ObjectType").
					WithIsComponent(true).
					WithContextStack(ast.ContextStack{
						{Type: "refType", Identifier: "Schemas", IdentifierForNaming: &[]string{""}[0]},
						{Type: "refName", Identifier: "ObjectType"},
						{Type: "component", Identifier: "true"},
					}).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithNestedObject",
			schemasYAML: `TestSchema:
  type: object
  properties:
    user:
      type: object
      properties:
        name:
          type: string
        age:
          type: integer`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("user",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithName("user").
								WithOriginalName("user").
								WithContextStack(ast.ContextStack{}).
								WithFields(
									// Fields in alphabetical order
									testutils.NewFieldDef("age",
										testutils.NewTypeDef(ast.DataTypeInteger).
											WithLocation(&yaml.Node{Line: 17, Column: 15}).
											WithValidations(&ast.Validations{}).
											Build()).
										WithOriginalName("age").
										WithJSONAnnotation("age").
										WithOptional(true).
										Build(),
									testutils.NewFieldDef("name",
										testutils.NewTypeDef(ast.DataTypeString).
											WithLocation(&yaml.Node{Line: 15, Column: 15}).
											WithValidations(&ast.Validations{}).
											Build()).
										WithOriginalName("name").
										WithJSONAnnotation("name").
										WithOptional(true).
										Build(),
								).
								Build()).
							WithOriginalName("user").
							WithJSONAnnotation("user").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithAllRequiredFields",
			schemasYAML: `TestSchema:
  type: object
  properties:
    name:
      type: string
    age:
      type: integer
    email:
      type: string
  required:
    - name
    - age
    - email`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						// Fields in alphabetical order
						testutils.NewFieldDef("age",
							testutils.NewTypeDef(ast.DataTypeInteger).
								WithLocation(&yaml.Node{Line: 14, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("age").
							WithJSONAnnotation("age").
							Build(),
						testutils.NewFieldDef("email",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 16, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("email").
							WithJSONAnnotation("email").
							Build(),
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithReadOnlyProperties",
			schemasYAML: `TestSchema:
 type: object
 properties:
   id:
     type: string
     readOnly: true
   name:
     type: string
   email:
     type: string
     readOnly: true`,
			schemaName: "TestSchema",
			isRequest:  &[]bool{true}[0], // Test as request to trigger readOnly filtering
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{
						{Type: "inputOutput", Identifier: "Input"},
					}).
					WithInput(true).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 15, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithWriteOnlyProperties",
			schemasYAML: `TestSchema:
 type: object
 properties:
   password:
     type: string
     writeOnly: true
   name:
     type: string
   secret:
     type: string
     writeOnly: true`,
			schemaName: "TestSchema",
			isRequest:  &[]bool{false}[0], // Test as response to trigger writeOnly filtering
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{
						{Type: "inputOutput", Identifier: "Output"},
					}).
					WithOutput(true).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 15, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithComplexAdditionalProperties",
			schemasYAML: `TestSchema:
 type: object
 properties:
   name:
     type: string
 additionalProperties:
   type: object
   properties:
     value:
       type: string
     count:
       type: integer`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureAdditionalProperties: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{
						{Type: "registerDuplicate", Identifier: "2", IdentifierForNaming: &[]string{""}[0]},
					}).
					WithFields(
						testutils.NewFieldDef("AdditionalProperties",
							testutils.NewTypeDef(ast.DataTypeMap).
								WithLocation(&yaml.Node{Line: 9, Column: 6}).
								WithItemType(
									testutils.NewTypeDef(ast.DataTypeClass).
										WithLocation(&yaml.Node{Line: 14, Column: 8}).
										WithScope(ast.ScopeShared).
										WithRegistered().
										WithOriginalNameFrozen().
										WithContextStack(ast.ContextStack{}).
										WithFields(
											testutils.NewFieldDef("count",
												testutils.NewTypeDef(ast.DataTypeInteger).
													WithLocation(&yaml.Node{Line: 19, Column: 12}).
													WithValidations(&ast.Validations{}).
													Build()).
												WithOriginalName("count").
												WithJSONAnnotation("count").
												WithOptional(true).
												Build(),
											testutils.NewFieldDef("value",
												testutils.NewTypeDef(ast.DataTypeString).
													WithLocation(&yaml.Node{Line: 17, Column: 12}).
													WithValidations(&ast.Validations{}).
													Build()).
												WithOriginalName("value").
												WithJSONAnnotation("value").
												WithOptional(true).
												Build(),
										).
										Build(),
								).
								Build()).
							WithAnnotations(&ast.JSONAnnotation{Ignore: true, FieldName: "-"}, &ast.NeedsCasingAnnotation{}).
							WithOptional(true).
							WithIsAdditionalProperties(true).
							Build(),
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithAdditionalPropertiesArray",
			schemasYAML: `TestSchema:
 type: object
 properties:
   name:
     type: string
 additionalProperties:
   type: array
   items:
     type: string`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureAdditionalProperties: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("AdditionalProperties",
							testutils.NewTypeDef(ast.DataTypeMap).
								WithLocation(&yaml.Node{Line: 9, Column: 6}).
								WithItemType(
									testutils.NewTypeDef(ast.DataTypeArray).
										WithLocation(&yaml.Node{Line: 14, Column: 8}).
										WithItemType(
											testutils.NewTypeDef(ast.DataTypeString).
												WithLocation(&yaml.Node{Line: 16, Column: 10}).
												WithValidations(&ast.Validations{}).
												Build(),
										).
										WithValidations(&ast.Validations{}).
										Build(),
								).
								Build()).
							WithAnnotations(&ast.JSONAnnotation{Ignore: true, FieldName: "-"}, &ast.NeedsCasingAnnotation{}).
							WithOptional(true).
							WithIsAdditionalProperties(true).
							Build(),
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithDescription",
			schemasYAML: `TestSchema:
 type: object
 description: "A test schema with description"
 properties:
   name:
     type: string`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithComments(&ast.Comment{
						Description:      "A test schema with description",
						ExtendedComments: map[string]*ast.ExtendedComment{},
					}).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 13, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithPropertyDescription",
			schemasYAML: `TestSchema:
 type: object
 properties:
   name:
     type: string
     description: "The user's name"
   age:
     type: integer
     description: "The user's age in years"`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("age",
							testutils.NewTypeDef(ast.DataTypeInteger).
								WithLocation(&yaml.Node{Line: 15, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("age").
							WithJSONAnnotation("age").
							WithComments(&ast.Comment{
								Description:      "The user's age in years",
								ExtendedComments: map[string]*ast.ExtendedComment{},
							}).
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithComments(&ast.Comment{
								Description:      "The user's name",
								ExtendedComments: map[string]*ast.ExtendedComment{},
							}).
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithMixedRequiredOptional",
			schemasYAML: `TestSchema:
 type: object
 properties:
   id:
     type: string
   name:
     type: string
   email:
     type: string
   phone:
     type: string
 required:
   - id
   - name`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("email",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 16, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("email").
							WithJSONAnnotation("email").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("id",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("id").
							WithJSONAnnotation("id").
							Build(),
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 14, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							Build(),
						testutils.NewFieldDef("phone",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 18, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("phone").
							WithJSONAnnotation("phone").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithArrayProperty",
			schemasYAML: `TestSchema:
 type: object
 properties:
   name:
     type: string
   tags:
     type: array
     items:
       type: string
   scores:
     type: array
     items:
       type: integer`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("scores",
							testutils.NewTypeDef(ast.DataTypeArray).
								WithLocation(&yaml.Node{Line: 18, Column: 10}).
								WithItemType(
									testutils.NewTypeDef(ast.DataTypeInteger).
										WithLocation(&yaml.Node{Line: 20, Column: 12}).
										WithValidations(&ast.Validations{}).
										Build(),
								).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("scores").
							WithJSONAnnotation("scores").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("tags",
							testutils.NewTypeDef(ast.DataTypeArray).
								WithLocation(&yaml.Node{Line: 14, Column: 10}).
								WithItemType(
									testutils.NewTypeDef(ast.DataTypeString).
										WithLocation(&yaml.Node{Line: 16, Column: 12}).
										WithValidations(&ast.Validations{}).
										Build(),
								).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("tags").
							WithJSONAnnotation("tags").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithEnumProperty",
			schemasYAML: `TestSchema:
 type: object
 properties:
   name:
     type: string
   status:
     type: string
     enum:
       - active
       - inactive
       - pending`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("status",
							testutils.NewEnumTypeDef(ast.DataTypeString, []string{"active", "inactive", "pending"}, "enum").
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithName("status").
								WithOriginalName("status").
								WithContextStack(ast.ContextStack{}).
								WithEnumUnderlyingLocation(&yaml.Node{Line: 14, Column: 10}).
								Build()).
							WithOriginalName("status").
							WithJSONAnnotation("status").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithMultipleExamples",
			schemasYAML: `TestSchema:
 type: object
 properties:
   name:
     type: string
 examples:
   - name: "John"
   - name: "Jane"`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithExamples(
						&ast.Example{
							Value: &yaml.Node{
								Kind:   yaml.MappingNode,
								Style:  0,
								Tag:    "!!map",
								Line:   14,
								Column: 10,
								Content: []*yaml.Node{
									{
										Kind:   yaml.ScalarNode,
										Style:  0,
										Tag:    "!!str",
										Value:  "name",
										Line:   14,
										Column: 10,
									},
									{
										Kind:   yaml.ScalarNode,
										Style:  yaml.DoubleQuotedStyle,
										Tag:    "!!str",
										Value:  "John",
										Line:   14,
										Column: 16,
									},
								},
							},
						},
						&ast.Example{
							Value: &yaml.Node{
								Kind:   yaml.MappingNode,
								Style:  0,
								Tag:    "!!map",
								Line:   15,
								Column: 10,
								Content: []*yaml.Node{
									{
										Kind:   yaml.ScalarNode,
										Style:  0,
										Tag:    "!!str",
										Value:  "name",
										Line:   15,
										Column: 10,
									},
									{
										Kind:   yaml.ScalarNode,
										Style:  yaml.DoubleQuotedStyle,
										Tag:    "!!str",
										Value:  "Jane",
										Line:   15,
										Column: 16,
									},
								},
							},
						},
					).
					WithFields(
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 10}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithAdditionalPropertiesNoFeatureSupport",
			schemasYAML: `TestSchema:
 type: object
 properties:
   name:
     type: string
 additionalProperties: true`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureAdditionalProperties: false, // Not supported
				}
				return config
			},
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeMap).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithItemType(
						testutils.NewTypeDef(ast.DataTypeAny).Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithComplexNestedStructure",
			schemasYAML: `TestSchema:
 type: object
 properties:
   user:
     type: object
     properties:
       profile:
         type: object
         properties:
           name:
             type: string
           settings:
             type: object
             properties:
               theme:
                 type: string
               notifications:
                 type: boolean
       contacts:
         type: array
         items:
           type: object
           properties:
             email:
               type: string
             phone:
               type: string`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 6}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("user",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithLocation(&yaml.Node{Line: 12, Column: 10}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithName("user").
								WithOriginalName("user").
								WithContextStack(ast.ContextStack{}).
								WithFields(
									testutils.NewFieldDef("contacts",
										testutils.NewTypeDef(ast.DataTypeArray).
											WithLocation(&yaml.Node{Line: 27, Column: 14}).
											WithItemType(
												testutils.NewTypeDef(ast.DataTypeClass).
													WithLocation(&yaml.Node{Line: 29, Column: 16}).
													WithScope(ast.ScopeShared).
													WithRegistered().
													WithOriginalNameFrozen().
													WithName("contact").
													WithOriginalName("contact").
													WithContextStack(ast.ContextStack{
														{Type: "property", Identifier: "user"},
													}).
													WithFields(
														testutils.NewFieldDef("email",
															testutils.NewTypeDef(ast.DataTypeString).
																WithLocation(&yaml.Node{Line: 32, Column: 20}).
																WithValidations(&ast.Validations{}).
																Build()).
															WithOriginalName("email").
															WithJSONAnnotation("email").
															WithOptional(true).
															Build(),
														testutils.NewFieldDef("phone",
															testutils.NewTypeDef(ast.DataTypeString).
																WithLocation(&yaml.Node{Line: 34, Column: 20}).
																WithValidations(&ast.Validations{}).
																Build()).
															WithOriginalName("phone").
															WithJSONAnnotation("phone").
															WithOptional(true).
															Build(),
													).
													Build(),
											).
											WithValidations(&ast.Validations{}).
											Build()).
										WithOriginalName("contacts").
										WithJSONAnnotation("contacts").
										WithOptional(true).
										Build(),
									testutils.NewFieldDef("profile",
										testutils.NewTypeDef(ast.DataTypeClass).
											WithLocation(&yaml.Node{Line: 15, Column: 14}).
											WithScope(ast.ScopeShared).
											WithRegistered().
											WithOriginalNameFrozen().
											WithName("profile").
											WithOriginalName("profile").
											WithContextStack(ast.ContextStack{
												{Type: "property", Identifier: "user"},
											}).
											WithFields(
												testutils.NewFieldDef("name",
													testutils.NewTypeDef(ast.DataTypeString).
														WithLocation(&yaml.Node{Line: 18, Column: 18}).
														WithValidations(&ast.Validations{}).
														Build()).
													WithOriginalName("name").
													WithJSONAnnotation("name").
													WithOptional(true).
													Build(),
												testutils.NewFieldDef("settings",
													testutils.NewTypeDef(ast.DataTypeClass).
														WithLocation(&yaml.Node{Line: 20, Column: 18}).
														WithScope(ast.ScopeShared).
														WithRegistered().
														WithOriginalNameFrozen().
														WithName("settings").
														WithOriginalName("settings").
														WithContextStack(ast.ContextStack{
															{Type: "property", Identifier: "user"},
															{Type: "property", Identifier: "profile"},
														}).
														WithFields(
															testutils.NewFieldDef("notifications",
																testutils.NewTypeDef(ast.DataTypeBoolean).
																	Build()).
																WithOriginalName("notifications").
																WithJSONAnnotation("notifications").
																WithOptional(true).
																Build(),
															testutils.NewFieldDef("theme",
																testutils.NewTypeDef(ast.DataTypeString).
																	WithLocation(&yaml.Node{Line: 23, Column: 22}).
																	WithValidations(&ast.Validations{}).
																	Build()).
																WithOriginalName("theme").
																WithJSONAnnotation("theme").
																WithOptional(true).
																Build(),
														).
														Build()).
													WithOriginalName("settings").
													WithJSONAnnotation("settings").
													WithOptional(true).
													Build(),
											).
											Build()).
										WithOriginalName("profile").
										WithJSONAnnotation("profile").
										WithOptional(true).
										Build(),
								).
								Build()).
							WithOriginalName("user").
							WithJSONAnnotation("user").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithSerializationMethodRAW",
			schemasYAML: `TestSchema:
  type: object
  properties:
    name:
      type: string
    data:
      type: string`,
			schemaName:          "TestSchema",
			serializationMethod: &[]ast.SerializationMethod{ast.SerializationMethodRAW}[0],
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("data",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 14, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("data").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithSerializationMethodForm",
			schemasYAML: `TestSchema:
  type: object
  properties:
    username:
      type: string
    password:
      type: string`,
			schemaName:          "TestSchema",
			serializationMethod: &[]ast.SerializationMethod{ast.SerializationMethodForm}[0],
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("password",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 14, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("password").
							WithAnnotations(&ast.FormAnnotation{
								Name:      "password",
								JSON:      false,
								Style:     "",
								Explode:   true,
								FieldType: testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 14, Column: 11}).WithValidations(&ast.Validations{}).Build(),
							}).
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("username",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("username").
							WithAnnotations(&ast.FormAnnotation{
								Name:      "username",
								JSON:      false,
								Style:     "",
								Explode:   true,
								FieldType: testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 12, Column: 11}).WithValidations(&ast.Validations{}).Build(),
							}).
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithSerializationMethodMultipart",
			schemasYAML: `TestSchema:
  type: object
  properties:
    file:
      type: string
      format: binary
    description:
      type: string`,
			schemaName:          "TestSchema",
			serializationMethod: &[]ast.SerializationMethod{ast.SerializationMethodMultipart}[0],
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("description",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 15, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("description").
							WithAnnotations(&ast.MultipartFormAnnotation{
								Name:      "description",
								JSON:      false,
								FieldType: testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 15, Column: 11}).WithValidations(&ast.Validations{}).Build(),
							}).
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("file",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithName("file").
								WithOriginalName("file").
								WithContextStack(ast.ContextStack{}).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithFields(
									testutils.NewFieldDef("content",
										testutils.NewTypeDef(ast.DataTypeBytes).
											WithLocation(&yaml.Node{Line: 12, Column: 11}).
											Build()).
										WithAnnotations(&ast.MultipartFormAnnotation{Content: true}).
										Build(),
									testutils.NewFieldDef("contentType",
										testutils.NewTypeDef(ast.DataTypeString).
											WithLocation(&yaml.Node{Line: 12, Column: 11}).
											WithValidations(nil).
											Build()).
										WithAnnotations(&ast.MultipartFormAnnotation{Name: "Content-Type"}).
										WithOptional(true).
										Build(),
									testutils.NewFieldDef("fileName",
										testutils.NewTypeDef(ast.DataTypeString).
											WithLocation(&yaml.Node{Line: 12, Column: 11}).
											WithValidations(nil).
											Build()).
										WithAnnotations(&ast.MultipartFormAnnotation{Name: "fileName"}).
										Build(),
								).
								WithExtensions(map[string]interface{}{}).
								WithIsMultipartFile(true).
								Build()).
							WithOriginalName("file").
							WithAnnotations(&ast.MultipartFormAnnotation{
								File: true,
								Name: "file",
							}).
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithSerializationMethodString",
			schemasYAML: `TestSchema:
  type: object
  properties:
    content:
      type: string
    encoding:
      type: string`,
			schemaName:          "TestSchema",
			serializationMethod: &[]ast.SerializationMethod{ast.SerializationMethodString}[0],
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("content",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("content").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("encoding",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 14, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("encoding").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithSerializationMethodEventStream",
			schemasYAML: `TestSchema:
  type: object
  properties:
    event:
      type: string
    data:
      type: string`,
			schemaName:          "TestSchema",
			serializationMethod: &[]ast.SerializationMethod{ast.SerializationMethodEventStream}[0],
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("data",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 14, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("data").
							WithJSONAnnotation("data").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("event",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("event").
							WithJSONAnnotation("event").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithSerializationMethodJsonL",
			schemasYAML: `TestSchema:
  type: object
  properties:
    record:
      type: string
    timestamp:
      type: string`,
			schemaName:          "TestSchema",
			serializationMethod: &[]ast.SerializationMethod{ast.SerializationMethodJsonL}[0],
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("record",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("record").
							WithJSONAnnotation("record").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("timestamp",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 14, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("timestamp").
							WithJSONAnnotation("timestamp").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "TopLevelComponentReferenceWithObject",
			schemasYAML: `User:
  type: object
  properties:
    name:
      type: string
    email:
      type: string
  required:
    - name
TestSchema:
  $ref: "#/components/schemas/User"`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("User",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithName("User").
					WithOriginalName("User").
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithIsComponent(true).
					WithContextStack(
						testutils.NewContextStack().WithRefType("Schemas").WithRefName("User").WithIdentifierForNaming(nil).WithComponent().Build(),
					).
					WithFields(
						testutils.NewFieldDef("email",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 14, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("email").
							WithJSONAnnotation("email").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("name",
							testutils.NewTypeDef(ast.DataTypeString).
								WithLocation(&yaml.Node{Line: 12, Column: 11}).
								WithValidations(&ast.Validations{}).
								Build()).
							WithOriginalName("name").
							WithJSONAnnotation("name").
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name:  "ObjectWithNameOverrideAndComplexAllOfStructure_NameOverrideBleeds",
			fixes: testutils.WithNameOverrideFeb2026(false),
			schemasYAML: `TestSchema:
  type: object
  x-speakeasy-name-override: overriddenResponse
  properties:
    json:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
    normal:
      type: object
      x-speakeasy-name-override: overriddenNormal
      properties:
        id:
          type: string
        name:
          type: string
    allOf:
      allOf:
        - type: object
          properties:
            id:
              type: string
        - x-speakeasy-name-override: overriddenAllOf
    deepAllOf:
      allOf:
        - allOf:
            - x-speakeasy-name-override: deepOverriddenAllOf
            - type: object
              properties:
                value:
                  type: string`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("overriddenResponse",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithName("overriddenResponse").
					WithOriginalName("overriddenResponse").
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithExtensions(map[string]interface{}{
						"x-speakeasy-name-override": "overriddenResponse",
					}).
					WithFields(
						// Fields are sorted alphabetically by field name
						testutils.NewFieldDef("deepOverriddenAllOf",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithName("deepOverriddenAllOf").
								WithOriginalName("deepOverriddenAllOf").
								WithLocation(&yaml.Node{Line: 35, Column: 11}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithContextStack(ast.ContextStack{
									{Type: "property", Identifier: "deepOverriddenAllOf"},
								}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override":      "deepOverriddenAllOf",
									"x-speakeasy-reference-override": "e0976835ac163e67-allOf[b772744d7b8106d7,cbf29ce484222325]",
								}).
								WithFields(
									testutils.NewFieldDef("value", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 41, Column: 23}).Build()).
										WithOriginalName("value").
										WithOptional(true).
										WithJSONAnnotation("value").
										Build(),
								).
								Build()).
							WithOriginalName("deepAllOf").
							WithOptional(true).
							WithJSONAnnotation("deepAllOf").
							Build(),
						testutils.NewFieldDef("json",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithLocation(&yaml.Node{Line: 13, Column: 11}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithName("json").
								WithOriginalName("json").
								WithContextStack(ast.ContextStack{}).
								WithFields(
									testutils.NewFieldDef("id", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 16, Column: 15}).Build()).
										WithOriginalName("id").
										WithOptional(true).
										WithJSONAnnotation("id").
										Build(),
									testutils.NewFieldDef("name", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 18, Column: 15}).Build()).
										WithOriginalName("name").
										WithOptional(true).
										WithJSONAnnotation("name").
										Build(),
								).
								Build()).
							WithOriginalName("json").
							WithOptional(true).
							WithJSONAnnotation("json").
							Build(),
						testutils.NewFieldDef("overriddenAllOf",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithName("overriddenAllOf").
								WithOriginalName("overriddenAllOf").
								WithLocation(&yaml.Node{Line: 28, Column: 11}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithContextStack(ast.ContextStack{
									{Type: "property", Identifier: "overriddenAllOf"},
								}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override":      "overriddenAllOf",
									"x-speakeasy-reference-override": "082ee8fa7c9864e3-allOf[340d40e0d55e80f3,cbf29ce484222325]",
								}).
								WithFields(
									testutils.NewFieldDef("id", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 32, Column: 19}).Build()).
										WithOriginalName("id").
										WithOptional(true).
										WithJSONAnnotation("id").
										Build(),
								).
								Build()).
							WithOriginalName("allOf").
							WithOptional(true).
							WithJSONAnnotation("allOf").
							Build(),
						testutils.NewFieldDef("overriddenNormal",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithName("overriddenNormal").
								WithOriginalName("overriddenNormal").
								WithLocation(&yaml.Node{Line: 20, Column: 11}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithContextStack(ast.ContextStack{
									{Type: "property", Identifier: "overriddenNormal"},
								}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override": "overriddenNormal",
								}).
								WithFields(
									testutils.NewFieldDef("id", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 24, Column: 15}).Build()).
										WithOriginalName("id").
										WithOptional(true).
										WithJSONAnnotation("id").
										Build(),
									testutils.NewFieldDef("name", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 26, Column: 15}).Build()).
										WithOriginalName("name").
										WithOptional(true).
										WithJSONAnnotation("name").
										Build(),
								).
								Build()).
							WithOriginalName("normal").
							WithOptional(true).
							WithJSONAnnotation("normal").
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name:  "ObjectWithNameOverrideAndComplexAllOfStructure_NameOverrideIgnored",
			fixes: testutils.WithNameOverrideFeb2026(true),
			schemasYAML: `TestSchema:
  type: object
  x-speakeasy-name-override: overriddenResponse
  properties:
    json:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
    normal:
      type: object
      x-speakeasy-name-override: overriddenNormal
      properties:
        id:
          type: string
        name:
          type: string
    allOf:
      allOf:
        - type: object
          properties:
            id:
              type: string
        - x-speakeasy-name-override: ignoredOverride
    deepAllOf:
      allOf:
        - allOf:
            - x-speakeasy-name-override: deeplyIgnoredOverride
            - type: object
              properties:
                value:
                  type: string`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("overriddenResponse",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithName("overriddenResponse").
					WithOriginalName("overriddenResponse").
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithExtensions(map[string]interface{}{
						"x-speakeasy-name-override": "overriddenResponse",
					}).
					WithFields(
						testutils.NewFieldDef("allOf",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithName("allOf").
								WithOriginalName("allOf").
								WithLocation(&yaml.Node{Line: 28, Column: 11}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithContextStack(ast.ContextStack{}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-reference-override": "allOf[340d40e0d55e80f3,cbf29ce484222325]",
								}).
								WithFields(
									testutils.NewFieldDef("id", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 32, Column: 19}).Build()).
										WithOriginalName("id").
										WithOptional(true).
										WithJSONAnnotation("id").
										Build(),
								).
								Build()).
							WithOriginalName("allOf").
							WithOptional(true).
							WithJSONAnnotation("allOf").
							Build(),
						testutils.NewFieldDef("deepAllOf",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithName("deepAllOf").
								WithOriginalName("deepAllOf").
								WithLocation(&yaml.Node{Line: 36, Column: 15}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithContextStack(ast.ContextStack{}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-reference-override": "allOf[b772744d7b8106d7,cbf29ce484222325]",
								}).
								WithFields(
									testutils.NewFieldDef("value", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 41, Column: 23}).Build()).
										WithOriginalName("value").
										WithOptional(true).
										WithJSONAnnotation("value").
										Build(),
								).
								Build()).
							WithOriginalName("deepAllOf").
							WithOptional(true).
							WithJSONAnnotation("deepAllOf").
							Build(),
						testutils.NewFieldDef("json",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithLocation(&yaml.Node{Line: 13, Column: 11}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithName("json").
								WithOriginalName("json").
								WithContextStack(ast.ContextStack{}).
								WithFields(
									testutils.NewFieldDef("id", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 16, Column: 15}).Build()).
										WithOriginalName("id").
										WithOptional(true).
										WithJSONAnnotation("id").
										Build(),
									testutils.NewFieldDef("name", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 18, Column: 15}).Build()).
										WithOriginalName("name").
										WithOptional(true).
										WithJSONAnnotation("name").
										Build(),
								).
								Build()).
							WithOriginalName("json").
							WithOptional(true).
							WithJSONAnnotation("json").
							Build(),
						testutils.NewFieldDef("overriddenNormal",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithName("overriddenNormal").
								WithOriginalName("overriddenNormal").
								WithLocation(&yaml.Node{Line: 20, Column: 11}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithContextStack(ast.ContextStack{
									{Type: "property", Identifier: "overriddenNormal"},
								}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override": "overriddenNormal",
								}).
								WithFields(
									testutils.NewFieldDef("id", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 24, Column: 15}).Build()).
										WithOriginalName("id").
										WithOptional(true).
										WithJSONAnnotation("id").
										Build(),
									testutils.NewFieldDef("name", testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 26, Column: 15}).Build()).
										WithOriginalName("name").
										WithOptional(true).
										WithJSONAnnotation("name").
										Build(),
								).
								Build()).
							WithOriginalName("normal").
							WithOptional(true).
							WithJSONAnnotation("normal").
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithRefToRenamedComponent_NameOverrideBleeds",
			schemasYAML: `RenamedSchema:
  x-speakeasy-name-override: renamed
  type: boolean
TestSchema:
  type: object
  properties:
    overridden:
      $ref: '#/components/schemas/RenamedSchema'`,
			schemaName: "TestSchema",
			fixes:      testutils.WithNameOverrideFeb2026(false),
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 12, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("renamed",
							testutils.NewTypeDef(ast.DataTypeBoolean).
								WithName("renamed").
								WithOriginalName("").
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override": "renamed",
								}).
								Build()).
							WithOriginalName("overridden").
							WithJSONAnnotation("overridden").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithRefToRenamedComponent_NameOverrideIgnored",
			schemasYAML: `RenamedSchema:
  x-speakeasy-name-override: renamed
  type: boolean
TestSchema:
  type: object
  properties:
    preserved:
      $ref: '#/components/schemas/RenamedSchema'`,
			schemaName: "TestSchema",
			fixes:      testutils.WithNameOverrideFeb2026(true),
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 12, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("preserved",
							testutils.NewTypeDef(ast.DataTypeBoolean).
								WithName("renamed").
								WithOriginalName("").
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override": "renamed",
								}).
								Build()).
							WithOriginalName("preserved").
							WithJSONAnnotation("preserved").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithSiblingNameOverride",
			schemasYAML: `SomeSchema:
  type: boolean
TestSchema:
  type: object
  properties:
    overridden:
      x-speakeasy-name-override: renamed
      $ref: '#/components/schemas/SomeSchema'`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 11, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("renamed",
							testutils.NewTypeDef(ast.DataTypeBoolean).
								WithName("SomeSchema").
								WithOriginalName("").
								Build()).
							WithOriginalName("overridden").
							WithJSONAnnotation("overridden").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithNameOverrideOnRefProperties",
			schemasYAML: `Leaf:
  type: boolean
Nested:
  type: object
  properties:
    nestedInline:
      x-speakeasy-name-override: renamedNestedInline
      type: string
    nestedRef:
      x-speakeasy-name-override: renamedNestedRef
      $ref: '#/components/schemas/Leaf'
TestSchema:
  type: object
  properties:
    inline:
      x-speakeasy-name-override: renamedInline
      type: string
    reference:
      x-speakeasy-name-override: leaf
      $ref: '#/components/schemas/Leaf'
    nested:
      x-speakeasy-name-override: renamedNested
      $ref: '#/components/schemas/Nested'`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 20, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("leaf",
							testutils.NewTypeDef(ast.DataTypeBoolean).
								WithName("Leaf").
								WithOriginalName("").
								Build()).
							WithOriginalName("reference").
							WithJSONAnnotation("reference").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("renamedInline",
							testutils.NewTypeDef(ast.DataTypeString).
								WithName("renamedInline").
								WithOriginalName("").
								WithLocation(&yaml.Node{Line: 23, Column: 11}).
								WithValidations(&ast.Validations{}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override": "renamedInline",
								}).
								Build()).
							WithOriginalName("inline").
							WithJSONAnnotation("inline").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("renamedNested",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithName("Nested").
								WithOriginalName("Nested").
								WithLocation(&yaml.Node{Line: 11, Column: 7}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithOriginalNameFrozen().
								WithIsComponent(true).
								WithContextStack(
									testutils.NewContextStack().WithRefType("Schemas").WithRefName("Nested").WithIdentifierForNaming(nil).WithComponent().Build(),
								).
								WithFields(
									testutils.NewFieldDef("renamedNestedInline",
										testutils.NewTypeDef(ast.DataTypeString).
											WithName("renamedNestedInline").
											WithOriginalName("").
											WithLocation(&yaml.Node{Line: 14, Column: 11}).
											WithValidations(&ast.Validations{}).
											WithExtensions(map[string]interface{}{
												"x-speakeasy-name-override": "renamedNestedInline",
											}).
											Build()).
										WithOriginalName("nestedInline").
										WithJSONAnnotation("nestedInline").
										WithOptional(true).
										Build(),
									testutils.NewFieldDef("renamedNestedRef",
										testutils.NewTypeDef(ast.DataTypeBoolean).
											WithName("Leaf").
											WithOriginalName("").
											Build()).
										WithOriginalName("nestedRef").
										WithJSONAnnotation("nestedRef").
										WithOptional(true).
										Build(),
								).
								Build()).
							WithOriginalName("nested").
							WithJSONAnnotation("nested").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithNameOverrideShadowing",
			schemasYAML: `RenamedSchema:
  x-speakeasy-name-override: renamed
  type: boolean
TestSchema:
  type: object
  properties:
    overridden:
      x-speakeasy-name-override: override
      $ref: '#/components/schemas/RenamedSchema'`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 12, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("override",
							testutils.NewTypeDef(ast.DataTypeBoolean).
								WithName("renamed").
								WithOriginalName("").
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override": "renamed",
								}).
								Build()).
							WithOriginalName("overridden").
							WithJSONAnnotation("overridden").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithAllOfHoistingMutatesSharedComponent_NameOverrideBleeds",
			schemasYAML: `Shared:
  allOf:
    - x-speakeasy-name-override: "custom"
    - type: boolean
TestSchema:
  type: object
  properties:
    first:
      $ref: '#/components/schemas/Shared'
    second:
      $ref: '#/components/schemas/Shared'`,
			schemaName: "TestSchema",
			fixes:      testutils.WithNameOverrideFeb2026(false),
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 13, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("custom",
							testutils.NewTypeDef(ast.DataTypeBoolean).
								WithName("custom").
								WithOriginalName("").
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override":      "custom",
									"x-speakeasy-reference-override": "#/components/schemas/Shared-allOf[17e4f22d3a3f4872,cbf29ce484222325]",
								}).
								Build()).
							WithOriginalName("first").
							WithJSONAnnotation("first").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("custom1",
							testutils.NewTypeDef(ast.DataTypeBoolean).
								WithName("custom").
								WithOriginalName("").
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override":      "custom",
									"x-speakeasy-reference-override": "#/components/schemas/Shared-allOf[17e4f22d3a3f4872,cbf29ce484222325]",
								}).
								Build()).
							WithOriginalName("second").
							WithJSONAnnotation("second").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithAllOfHoistingMutatesSharedComponent_NameOverrideIgnored",
			schemasYAML: `Shared:
  allOf:
    - x-speakeasy-name-override: "custom"
    - type: boolean
TestSchema:
  type: object
  properties:
    first:
      $ref: '#/components/schemas/Shared'
    second:
      $ref: '#/components/schemas/Shared'`,
			schemaName: "TestSchema",
			fixes:      testutils.WithNameOverrideFeb2026(true),
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 13, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("first",
							testutils.NewTypeDef(ast.DataTypeBoolean).
								WithName("Shared").
								WithOriginalName("").
								WithExtensions(map[string]interface{}{
									"x-speakeasy-reference-override": "#/components/schemas/Shared-allOf[17e4f22d3a3f4872,cbf29ce484222325]",
								}).
								Build()).
							WithOriginalName("first").
							WithJSONAnnotation("first").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("second",
							testutils.NewTypeDef(ast.DataTypeBoolean).
								WithName("Shared").
								WithOriginalName("").
								WithExtensions(map[string]interface{}{
									"x-speakeasy-reference-override": "#/components/schemas/Shared-allOf[17e4f22d3a3f4872,cbf29ce484222325]",
								}).
								Build()).
							WithOriginalName("second").
							WithJSONAnnotation("second").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name:  "ObjectWithAllOfChildShadowing_NameOverrideBleeds",
			fixes: testutils.WithNameOverrideFeb2026(false),
			schemasYAML: `Number:
  type: number
  format: float
Int:
  x-speakeasy-name-override: int32
  type: number
  format: int32
TestSchema:
  type: object
  properties:
    foo:
      allOf:
        - x-speakeasy-name-override: float
          $ref: '#/components/schemas/Number'
        - $ref: '#/components/schemas/Int'`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 16, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("int32",
							testutils.NewTypeDef(ast.DataTypeNumber).
								WithName("int32").
								WithOriginalName("").
								WithLocation(&yaml.Node{Line: 19, Column: 11}).
								WithValidations(&ast.Validations{}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override":      "int32",
									"x-speakeasy-reference-override": "c29859421f83c6b9-allOf[#/components/schemas/Int,#/components/schemas/Number]",
								}).
								Build()).
							WithOriginalName("foo").
							WithJSONAnnotation("foo").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name:  "ObjectWithAllOfChildShadowing_NameOverrideIgnored",
			fixes: testutils.WithNameOverrideFeb2026(true),
			schemasYAML: `Number:
  type: number
  format: float
Int:
  x-speakeasy-name-override: int32
  type: number
  format: int32
TestSchema:
  type: object
  properties:
    foo:
      allOf:
        - x-speakeasy-name-override: float
          $ref: '#/components/schemas/Number'
        - $ref: '#/components/schemas/Int'`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 16, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("foo",
							testutils.NewTypeDef(ast.DataTypeNumber).
								WithLocation(&yaml.Node{Line: 19, Column: 11}).
								WithValidations(&ast.Validations{}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-reference-override": "allOf[#/components/schemas/Int,#/components/schemas/Number]",
								}).
								Build()).
							WithOriginalName("foo").
							WithJSONAnnotation("foo").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			// Demonstrates allOf hoisting bleed: x-speakeasy-name-override on an allOf child
			// gets hoisted into the resolved property schema, causing the property name
			// "overridden" to be renamed to "child"
			name: "ObjectWithAllOfChildNameOverrideBleed",
			schemasYAML: `Shared:
  type: object
  properties:
    sharedProp:
      type: boolean
TestSchema:
  type: object
  properties:
    overridden:
      allOf:
        - $ref: '#/components/schemas/Shared'
        - x-speakeasy-name-override: "child"
          type: object
          properties:
            inlineProp:
              type: string`,
			schemaName: "TestSchema",
			fixes:      testutils.WithNameOverrideFeb2026(false),
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 14, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("child",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithName("child").
								WithOriginalName("child").
								WithOriginalNameFrozen().
								WithLocation(&yaml.Node{Line: 17, Column: 11}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithContextStack(ast.ContextStack{
									{Type: ast.ContextTypeProperty, Identifier: "child"},
								}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override":      "child",
									"x-speakeasy-reference-override": "c73e3319b9bbe861-allOf[#/components/schemas/Shared,6c49c48dda75dfc4]",
								}).
								WithFields(
									testutils.NewFieldDef("inlineProp",
										testutils.NewTypeDef(ast.DataTypeString).
											WithLocation(&yaml.Node{Line: 23, Column: 19}).
											WithValidations(&ast.Validations{}).
											WithExtensions(map[string]interface{}{}).
											Build()).
										WithOriginalName("inlineProp").
										WithJSONAnnotation("inlineProp").
										WithOptional(true).
										Build(),
									testutils.NewFieldDef("sharedProp",
										testutils.NewTypeDef(ast.DataTypeBoolean).
											WithExtensions(map[string]interface{}{}).
											Build()).
										WithOriginalName("sharedProp").
										WithJSONAnnotation("sharedProp").
										WithOptional(true).
										Build(),
								).
								Build()).
							WithOriginalName("overridden").
							WithJSONAnnotation("overridden").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithAllOfChildNameOverrideBleed_NameOverrideIgnored",
			schemasYAML: `Shared:
  type: object
  properties:
    sharedProp:
      type: boolean
TestSchema:
  type: object
  properties:
    overridden:
      allOf:
        - $ref: '#/components/schemas/Shared'
        - x-speakeasy-name-override: "child"
          type: object
          properties:
            inlineProp:
              type: string`,
			schemaName: "TestSchema",
			fixes:      testutils.WithNameOverrideFeb2026(true),
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 14, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("overridden",
							testutils.NewTypeDef(ast.DataTypeClass).
								WithName("overridden").
								WithOriginalName("overridden").
								WithOriginalNameFrozen().
								WithLocation(&yaml.Node{Line: 17, Column: 11}).
								WithScope(ast.ScopeShared).
								WithRegistered().
								WithContextStack(ast.ContextStack{}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-reference-override": "allOf[#/components/schemas/Shared,6c49c48dda75dfc4]",
								}).
								WithFields(
									testutils.NewFieldDef("inlineProp",
										testutils.NewTypeDef(ast.DataTypeString).
											WithLocation(&yaml.Node{Line: 23, Column: 19}).
											WithValidations(&ast.Validations{}).
											WithExtensions(map[string]interface{}{}).
											Build()).
										WithOriginalName("inlineProp").
										WithJSONAnnotation("inlineProp").
										WithOptional(true).
										Build(),
									testutils.NewFieldDef("sharedProp",
										testutils.NewTypeDef(ast.DataTypeBoolean).
											WithExtensions(map[string]interface{}{}).
											Build()).
										WithOriginalName("sharedProp").
										WithJSONAnnotation("sharedProp").
										WithOptional(true).
										Build(),
								).
								Build()).
							WithOriginalName("overridden").
							WithJSONAnnotation("overridden").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithNamespacedRefProperties_NameOverrideBleeds",
			schemasYAML: `Foo_pet:
  x-speakeasy-name-override: pet
  x-speakeasy-model-namespace: Foo
  type: string
Bar_pet:
  x-speakeasy-model-namespace: Bar
  type: string
TestSchema:
  type: object
  properties:
    fooPet:
      $ref: '#/components/schemas/Foo_pet'
    barPet:
      x-speakeasy-name-override: pet
      $ref: '#/components/schemas/Bar_pet'`,
			schemaName: "TestSchema",
			fixes:      testutils.WithNameOverrideFeb2026(false),
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 16, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("pet",
							testutils.NewTypeDef(ast.DataTypeString).
								WithName("pet").
								WithOriginalName("").
								WithLocation(&yaml.Node{Line: 9, Column: 7}).
								WithValidations(&ast.Validations{}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override":   "pet",
									"x-speakeasy-model-namespace": "Foo",
								}).
								WithModelNamespace("Foo").
								Build()).
							WithOriginalName("fooPet").
							WithJSONAnnotation("fooPet").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("pet1",
							testutils.NewTypeDef(ast.DataTypeString).
								WithName("Bar_pet").
								WithOriginalName("").
								WithLocation(&yaml.Node{Line: 13, Column: 7}).
								WithValidations(&ast.Validations{}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-model-namespace": "Bar",
								}).
								WithModelNamespace("Bar").
								Build()).
							WithOriginalName("barPet").
							WithJSONAnnotation("barPet").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
		{
			name: "ObjectWithNamespacedRefProperties_NameOverrideIgnored",
			schemasYAML: `Foo_pet:
  x-speakeasy-name-override: pet
  x-speakeasy-model-namespace: Foo
  type: string
Bar_pet:
  x-speakeasy-model-namespace: Bar
  type: string
TestSchema:
  type: object
  properties:
    fooPet:
      $ref: '#/components/schemas/Foo_pet'
    barPet:
      x-speakeasy-name-override: pet
      $ref: '#/components/schemas/Bar_pet'`,
			schemaName: "TestSchema",
			fixes:      testutils.WithNameOverrideFeb2026(true),
			expected: testutils.NewFieldDef("object",
				testutils.NewTypeDef(ast.DataTypeClass).
					WithLocation(&yaml.Node{Line: 16, Column: 7}).
					WithScope(ast.ScopeShared).
					WithRegistered().
					WithOriginalNameFrozen().
					WithContextStack(ast.ContextStack{}).
					WithFields(
						testutils.NewFieldDef("fooPet",
							testutils.NewTypeDef(ast.DataTypeString).
								WithName("pet").
								WithOriginalName("").
								WithLocation(&yaml.Node{Line: 9, Column: 7}).
								WithValidations(&ast.Validations{}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-name-override":   "pet",
									"x-speakeasy-model-namespace": "Foo",
								}).
								WithModelNamespace("Foo").
								Build()).
							WithOriginalName("fooPet").
							WithJSONAnnotation("fooPet").
							WithOptional(true).
							Build(),
						testutils.NewFieldDef("pet",
							testutils.NewTypeDef(ast.DataTypeString).
								WithName("Bar_pet").
								WithOriginalName("").
								WithLocation(&yaml.Node{Line: 13, Column: 7}).
								WithValidations(&ast.Validations{}).
								WithExtensions(map[string]interface{}{
									"x-speakeasy-model-namespace": "Bar",
								}).
								WithModelNamespace("Bar").
								Build()).
							WithOriginalName("barPet").
							WithJSONAnnotation("barPet").
							WithOptional(true).
							Build(),
					).
					Build()).
				Build(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test environment using common testutils
			fullYAML := testutils.CreateOpenAPIDoc(tt.schemasYAML)

			var mockFeatures *testutils.MockFeaturesConfig
			if tt.mockFeatures != nil {
				mockFeatures = tt.mockFeatures()
			} else {
				mockFeatures = testutils.NewMockFeaturesConfig().WithSupportAllFeatures()
			}

			opts := testutils.TestEnvironmentOptions{
				OpenAPIYAML:  fullYAML,
				Fixes:        tt.fixes,
				MockFeatures: mockFeatures,
			}

			common, err := testutils.SetupTestEnvironment(opts)
			require.NoError(t, err)

			// Create the Schemas instance
			schemas := &Schemas{
				Config:    common.Config,
				Target:    common.Target,
				Subsystem: common.Subsystem,
				Namer:     common.Namer,
			}

			// Get the test schema
			testSchema, exists := common.DocInfo.Doc.GetComponents().GetSchemas().Get(tt.schemaName)
			require.True(t, exists)
			require.NotNil(t, testSchema)

			// Create test parameters
			var params Params
			if tt.serializationMethod != nil {
				params = CreateTestParamsWithSerialization(testSchema, *tt.serializationMethod, common.DocInfo)
			} else {
				params = CreateTestParams(testSchema, common.DocInfo)
			}
			if tt.isRequest != nil {
				params.IsRequest = *tt.isRequest
			}

			// Call HandleSchema - let it handle nullable extraction and routing
			result, err := schemas.HandleSchema(context.Background(), params)
			require.NoError(t, err)

			// Use deep comparison from the working test
			isEqual := testutils.DeepCompare(t, tt.expected, result)
			if !isEqual {
				t.Errorf("Deep comparison failed for test case: %s", tt.name)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}
