package schemas

import (
	"context"
	"testing"

	"github.com/speakeasy-api/openapi-generation/v2/internal/ast"
	"github.com/speakeasy-api/openapi-generation/v2/internal/features"
	"github.com/speakeasy-api/openapi-generation/v2/internal/testutils"
	"github.com/speakeasy-api/openapi-generation/v2/internal/types"
	"github.com/speakeasy-api/openapi/pointer"
	config "github.com/speakeasy-api/sdk-gen-config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestHandleArray tests the handleArray method with various array scenarios
func TestHandleArray(t *testing.T) {
	tests := []struct {
		name         string
		schemasYAML  string
		schemaName   string
		fixes        *config.Fixes                        // nil means use default
		mockFeatures func() *testutils.MockFeaturesConfig // nil means use default (all features supported)
		isRequest    *bool                                // nil means use default (false), otherwise use specified value
		expected     *ast.FieldDef
	}{
		{
			name: "StringArray",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				Build(),
		},
		{
			name: "IntegerArray",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: integer`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("integers",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeInteger).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				Build(),
		},
		{
			name: "ArrayWithoutItems",
			schemasYAML: `TestSchema:
  type: array`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("any",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeAny).WithLocation(&yaml.Node{Line: 9, Column: 7}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				Build(),
		},
		{
			name: "ArrayWithConst",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string
  const:
    - read
    - write`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureConstArrays:       true,
					features.FeatureConstsAndDefaults: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				WithConst(&ast.AnyValue{Value: []interface{}{"read", "write"}}).
				Build(),
		},
		{
			name: "ArrayWithEmptyConst",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string
  const: []`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureConstArrays:       true,
					features.FeatureConstsAndDefaults: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				WithConst(&ast.AnyValue{Value: []interface{}{}}).
				Build(),
		},
		{
			name: "ArrayWithConstFeatureUnsupported",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string
  const: []`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureConstsAndDefaults: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				Build(),
		},
		{
			name: "ArrayWithConstItemTypeMismatch",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string
  const:
    - 1`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig()
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureConstArrays:       true,
					features.FeatureConstsAndDefaults: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				Build(),
		},
		{
			name: "ArrayWithMinMaxItems",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string
  minItems: 1
  maxItems: 10`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{
						MinItems: pointer.From[int64](1),
						MaxItems: pointer.From[int64](10),
					}).
					Build()).
				Build(),
		},
		{
			name: "ArrayWithUniqueItems",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string
  uniqueItems: true`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{
						UniqueItems: pointer.From(true),
					}).
					Build()).
				Build(),
		},
		{
			name: "SetFormat",
			schemasYAML: `TestSchema:
  type: array
  format: set
  items:
    type: string`,
			schemaName: "TestSchema",
			mockFeatures: func() *testutils.MockFeaturesConfig {
				config := testutils.NewMockFeaturesConfig().WithTarget(types.Target{Target: "go"})
				config.SupportedFeatures = map[features.Feature]bool{
					features.FeatureSets: true,
				}
				return config
			},
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeSet).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 12, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				Build(),
		},
		{
			name: "ArrayWithNullable",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string
  nullable: true`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				WithNullable(true).
				WithOptional(true).
				Build(),
		},
		{
			name: "ArrayWithNullableItems",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string
    nullable: true`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithContainsNull(true).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				Build(),
		},
		{
			name: "ArrayWithExample",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string
  example: ["item1", "item2"]`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					WithExamples(
						&ast.Example{
							Value: &yaml.Node{
								Kind:  yaml.SequenceNode,
								Style: 32, // Flow style
								Tag:   "!!seq",
								Content: []*yaml.Node{
									{
										Kind:   yaml.ScalarNode,
										Style:  yaml.DoubleQuotedStyle,
										Tag:    "!!str",
										Value:  "item1",
										Line:   12,
										Column: 17,
									},
									{
										Kind:   yaml.ScalarNode,
										Style:  yaml.DoubleQuotedStyle,
										Tag:    "!!str",
										Value:  "item2",
										Line:   12,
										Column: 26,
									},
								},
								Line:   12,
								Column: 16,
							},
						},
					).
					Build()).
				Build(),
		},
		{
			name: "ArrayWithExtensions",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: string
  x-custom-extension: "custom-value"`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("strings",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					WithExtensions(map[string]interface{}{
						"x-custom-extension": "custom-value",
					}).
					Build()).
				Build(),
		},
		{
			name: "ArrayReference",
			schemasYAML: `StringArray:
  type: array
  items:
    type: string
TestSchema:
  $ref: "#/components/schemas/StringArray"`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("StringArray",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithName("StringArray").WithOriginalName("").WithLocation(&yaml.Node{Line: 11, Column: 9}).WithValidations(&ast.Validations{}).Build()).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
					Build()).
				Build(),
		},
		{
			name: "NestedArray",
			schemasYAML: `TestSchema:
  type: array
  items:
    type: array
    items:
      type: string`,
			schemaName: "TestSchema",
			expected: testutils.NewFieldDef("arrays",
				testutils.NewTypeDef(ast.DataTypeArray).
					WithItemType(
						testutils.NewTypeDef(ast.DataTypeArray).
							WithItemType(testutils.NewTypeDef(ast.DataTypeString).WithLocation(&yaml.Node{Line: 13, Column: 11}).WithValidations(&ast.Validations{}).Build()).
							WithLocation(&yaml.Node{Line: 11, Column: 9}).
							WithValidations(&ast.Validations{}).
							Build(),
					).
					WithLocation(&yaml.Node{Line: 9, Column: 7}).
					WithValidations(&ast.Validations{}).
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
			params := CreateTestParams(testSchema, common.DocInfo)
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
