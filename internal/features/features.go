//go:generate go run ../../cmd/features/generate
package features

import (
	"context"
	"fmt"
	"slices"

	"github.com/speakeasy-api/openapi-generation/v2/internal/executor"
	"github.com/speakeasy-api/openapi-generation/v2/internal/extensions"
	"github.com/speakeasy-api/openapi-generation/v2/internal/types"
	generatorErrors "github.com/speakeasy-api/openapi-generation/v2/pkg/errors"
	"github.com/speakeasy-api/openapi-generation/v2/pkg/logging"
)

const (
	errUnsupportedFeature = generatorErrors.Error("unsupported feature")

	// ErrUnsupportedFeature is the warning cause for features, or feature
	// combinations, a target does not support.
	ErrUnsupportedFeature = errUnsupportedFeature
)

type Feature int

const (
	// FeatureCore represents the minimum required implementation for all clients to make simple HTTP requests
	// ReadmeSections: usage,operations,http-client
	// Tier: Core
	FeatureCore Feature = iota
	// FeatureGetRequestBodies relates to the ability to have request bodies for GET operations
	// Tier: Core
	FeatureGetRequestBodies
	// FeatureFlattening relates to the ability to flatten the operation parameters into the method signature instead of using a request wrapper object
	// Tier: Beta
	FeatureFlattening
	// FeatureGlobalSecurity relates to the ability to accept global security configuration at the SDK level
	// ReadmeSections: security
	// Tier: Core
	FeatureGlobalSecurity
	// FeatureMethodSecurity relates to the ability to accept security configuration at the operation level
	// Tier: Beta
	FeatureMethodSecurity
	// FeatureGlobalServerURLs relates to the ability to accept global server URLs at the SDK level including templating of those URLs
	// ReadmeSections: server
	// Tier: Core
	FeatureGlobalServerURLs
	// FeatureMethodServerURLs relates to the ability to accept server URLs at the operation level including templating of those URLs
	// Tier: Beta
	FeatureMethodServerURLs
	// FeatureGlobals relates to the ability to accept values for parameters from globals provide at SDK instantiation time
	// ReadmeSections: global-parameters
	// Tier: GA
	FeatureGlobals
	// FeatureHiddenGlobals relates to the ability to hide any parameters at the operation level that are provided by globals
	// Tier: Gold
	FeatureHiddenGlobals
	// FeatureEnums relates to the ability to handle enum types
	// Tier: Core
	FeatureEnums
	// FeatureOpenEnums relates to the handling of unrecognized values with enums
	// Tier: Experimental
	FeatureOpenEnums
	// FeatureServerIDs relates to the ability to handle the x-speakeasy-server-id extension to allow selection of servers by ID
	// Tier: GA
	FeatureServerIDs
	// FeatureNameOverrides relates to the ability to override the names of generated types, methods, and parameters using the x-speakeasy-name-override extension
	// Tier: Core
	FeatureNameOverrides
	// FeatureIncludes relates to the ability to mark unused components to be included in the generated code
	// This is handled at the AST level so a target just needs to mark support for it and it will be handled automatically
	// Tier: Core
	FeatureIncludes
	// FeatureDocs relates to the ability to include custom documentation via the x-speakeasy-docs extension
	// This is handled at the AST level so a target just needs to mark support for it and it will be handled automatically
	// Tier: Core
	FeatureDocs
	// FeatureExamples relates to the ability to mark usage examples via the x-speakeasy-usage-example and provide security examples via the x-speakeasy-example extension
	// This is handled at the AST level so a target just needs to mark support for it and it will be handled automatically
	// Tier: Core
	FeatureExamples
	// FeatureGroups relates to the ability to group operations together using the x-speakeasy-group extension
	// This is handled at the AST level so a target just needs to mark support for it and it will be handled automatically
	// Tier: Core
	FeatureGroups
	// FeatureDeprecations relates to the ability to mark operations as deprecated using the `deprecated` property and using the x-speakeasy-deprecation-message extension and provide a replacement using the x-speakeasy-deprecation-replacement extension
	// Tier: GA
	FeatureDeprecations
	// FeatureRetries relates to the ability to enable retries for operations using the x-speakeasy-retries extension
	// ReadmeSections: retries
	// Tier: GA
	FeatureRetries
	// FeaturePagination relates to the ability to enable pagination for operations using the x-speakeasy-pagination extension
	// ReadmeSections: pagination
	// Tier: GA
	FeaturePagination
	// FeatureInputOutputModels relates to the ability to generate input/output models depending on the `writeOnly` and `readOnly` properties
	// This is handled at the AST level so a target just needs to mark support for it and it will be handled automatically
	// Tier: Core
	FeatureInputOutputModels
	// FeatureIgnores relates to the ability to ignore operations, parameters, and components using the x-speakeasy-ignore extension
	// This is handled at the AST level so a target just needs to mark support for it and it will be handled automatically
	// Tier: Core
	FeatureIgnores
	// FeatureTypeOverrides relates to the ability to override the types of generated types, methods, and parameters using the x-speakeasy-type-override extension
	// This is handled at the AST level so a target just needs to mark support for it and it will be handled automatically
	// Tier: Core
	FeatureTypeOverrides
	// FeatureErrors relates to the ability to handle error responses using the x-speakeasy-errors extension
	// ReadmeSections: errors
	// Tier: Beta
	FeatureErrors
	// FeatureErrorUnions relates to the ability to handle error responses using the x-speakeasy-errors extension as a union type
	// Tier: Gold
	FeatureErrorUnions
	// FeatureAcceptHeaders relates to the ability to pass accept headers to the server if an operation has multiple response types
	// Tier: Gold
	FeatureAcceptHeaders
	// FeatureUnions relates to the ability to handle unions of types
	// Tier: Beta
	FeatureUnions
	// FeatureSliceUnions relates to the ability to handle unions of arrays
	// Tier: GA
	FeatureSliceUnions
	// FeatureMultiLevelTagging relates to the ability to handle nested sub sdks driven by dot separated tags or groups
	// Tier: Beta
	FeatureMultiLevelTagging
	// FeatureDownloadStreams relates to the ability to download streams from the server
	// Tier: Gold
	FeatureDownloadStreams
	// FeatureBigInt relates to the ability to handle the bigint format with a specific BigInt type including serialization and deserialization to/from strings and numbers
	// Tier: GA
	FeatureBigInt
	// FeatureDecimal relates to the ability to handle the decimal format with a specific Decimal type including serialization and deserialization to/from strings and numbers
	// Tier: GA
	FeatureDecimal
	// FeatureSets relates to the ability to handle the set format with a specific Set type including serialization and deserialization to/from arrays
	// Tier: GA
	FeatureSets
	// FeatureDevContainers relates to the ability to generate dev container configurations for the SDK
	// Tier: Gold
	FeatureDevContainers
	// FeatureConstsAndDefaults relates to the ability to handle constants and default values for schemas
	// Tier: Beta
	FeatureConstsAndDefaults
	// FeatureDefaultArrays relates to the ability to handle default values for arrays
	// Tier: Experimental
	FeatureDefaultArrays
	// FeatureDefaultObjects relates to the ability to handle default values for objects
	// Tier: Experimental
	FeatureDefaultObjects
	// FeatureConstArrays relates to the ability to handle const values for arrays
	// Tier: Experimental
	FeatureConstArrays
	// FeatureConstObjects relates to the ability to handle const values for objects
	// Tier: Experimental
	FeatureConstObjects
	// FeatureAdditionalProperties relates to the ability to deserialize any unknown properties into a map available as an extra property on the generated models for both request bodies and parameters
	// Tier: Gold
	FeatureAdditionalProperties
	// FeatureTests relates to the ability to generate tests for the SDK
	// Tier: Gold
	FeatureTests
	// FeatureTagBasedOrdering relates to the ability to order operations based on tags (docs target only)
	// Tier: Core
	FeatureTagBasedOrdering
	// FeatureDisallowCircularRefs relates to the ability to disallow circular references in the generated models
	// Tier: Core
	FeatureDisallowCircularRefs
	// FeatureServerEvents relates to the ability to handle server-sent events
	// ReadmeSections: eventstream
	// Tier: Gold
	FeatureServerEvents
	// FeatureServerEventsSentinels relates to the ability to handle the sentinel/terminal event in an SSE stream
	// ReadmeSections: eventstream
	// Tier: Gold
	FeatureServerEventsSentinels
	// FeatureURLBasedPagination relates to the ability to handle pagination using URL based pagination
	// Tier: Experimental
	FeatureURLBasedPagination
	// FeatureResponseFormat relates to the ability to handle different response formats for operations
	// Tier: Gold
	FeatureResponseFormat
	// FeatureOAuth2ClientCredentials relates to the ability to handle OAuth2 client credentials via SDK hooks
	// Tier: Gold
	FeatureOAuth2ClientCredentials
	// FeatureOAuth2Password relates to the ability to handle OAuth2 resource owner password flow via SDK hooks
	// Tier: Gold
	FeatureOAuth2Password
	// FeatureWebhooks relates to the ability to handle generating models for webhooks defined in the spec
	// This is handled at the AST level so a target just needs to mark support for it and it will be handled automatically
	// Tier: Core
	FeatureWebhooks
	// FeatureWebhookHandlers relates to the inclusion of webhook handlers in the generated SDK
	// Tier: Gold
	FeatureWebhookHandlers
	// FeatureCallbacks relates to the ability to handle generating models for callbacks defined in the spec
	// This is handled at the AST level so a target just needs to mark support for it and it will be handled automatically
	// Tier: Core
	FeatureCallbacks
	// FeatureSDKHooks relates to the ability to handle SDK hooks for various parts of the SDK
	// Tier: Gold
	FeatureSDKHooks
	// FeatureAdditionalDependencies relates to the ability to add additional dependencies to the SDK
	// Tier: GA
	FeatureAdditionalDependencies
	// FeatureNullables relates to the ability to handle nullable values natively in target language
	// Tier: Beta
	FeatureNullables
	// FeatureUploadStreams relates to the ability to upload streams to the server
	// Tier: Experimental
	FeatureUploadStreams
	// FeatureGlobalSecurityFlattening relates to the ability to flatten simple global security configurations to avoid the need to wrap them in a security object
	// Tier: GA
	FeatureGlobalSecurityFlattening
	// FeatureGlobalSecurityCallbacks relates to the ability to pass security via callbacks that are call on each request
	// Tier: Beta
	FeatureGlobalSecurityCallbacks
	// FeatureIntellisenseMarkdownSupport relates to the ability to correctly render descriptions using markdown natively for the target language so things like links etc show correctly in intellisense
	// Tier: Experimental
	FeatureIntellisenseMarkdownSupport
	// FeatureStringNumberFormats relates to the ability to serialize 64 bit numbers as strings in JSON
	// Tier: Gold
	FeatureStringNumberFormats
	// FeatureMethodArguments TODO: description
	// Tier: Experimental
	FeatureMethodArguments
	// FeatureMultipartFileContentType relates to the ability to set the content type of multipart file uploads
	// Tier: Experimental
	FeatureMultipartFileContentType
	// FeatureFlatRequests relates to the ability to flatten the operation parameters into the method signature instead of using a request wrapper object
	// Tier: Experimental
	FeatureFlatRequests
	// FeatureOperationTimeout relates to the ability to provide an extension to set a default operation timeout
	// Tier: Experimental
	FeatureOperationTimeout
	// FeatureDefaultEnabledRetries makes retries on by default for all SDK operations
	// Tier: Experimental
	FeatureDefaultEnabledRetries
	// FeatureDeepObjectParams relates to the ability to use deepObject style parameters
	// Tier: GA
	FeatureDeepObjectParams
	// FeatureEnvVarSecurityUsage relates to referencing security schemas as env variable where relevant in usage snippets
	// Tier: Experimental
	FeatureEnvVarSecurityUsage
	// FeatureEnvVarGlobals relates to referencing global parameters as env variable where relevant in usage snippets
	// Tier: Experimental
	FeatureEnvVarGlobals
	// FeatureCustomSecuritySchemes relates to the ability to define and use custom security schemes in hooks
	// Tier: Experimental
	FeatureCustomSecuritySchemes
	// FeatureMockServer relates to the ability to generate an embedded mock server
	// Tier: Experimental
	FeatureMockServer
	// FeatureEnumUnions relates to the ability to handle enum types as unions of literals
	// Tier: Gold
	FeatureEnumUnions
	// FeatureAllowReserved relates to not encoding reserved characters in query and path parameters in URIs
	// Tier: Gold
	FeatureAllowReserved
	// FeatureReactQueryHooks relates to the ability to generate an React hooks using TanStack Query in TypeScript
	// Tier: Experimental
	FeatureReactQueryHooks
	// FeatureCustomCodeRegions relates to the ability to add custom code to generated files
	// Tier: Experimental
	FeatureCustomCodeRegions
	// FeatureMCPServer relates to the ability to run a Model Context Protocol server using the SDK
	// Tier: Experimental
	FeatureMCPServer
	// FeatureJsonlResponses relates to the ability to handle JSONL (JSON Lines) responses and x-ndjson (Newline delimited JSON) responses. Both formats are the same.
	// Tier: Experimental
	FeatureJsonlResponses
	// FeatureConfigurableModuleName relates to the ability to configure the module name of the generated SDK in a target language
	// Tier: Experimental
	FeatureConfigurableModuleName
	// FeatureExamplesDirectory relates to the ability to configure the directory path where SDK usage examples are generated
	// Tier: Experimental
	FeatureExamplesDirectory
	// FeatureTransformJQ relates to the ability to have divergent API and SDK signatures through applying inline JQ transformers
	// Tier: Experimental
	FeatureTransformJQ
	// FeatureOperationPolling relates to the ability to configure opt-in operation polling for long-running operations
	// Tier: Experimental
	FeatureOperationPolling
	// FeatureModelNamespaces relates to the ability to organize component schemas into custom namespace folders using the x-speakeasy-model-namespace extension
	// This is handled at the AST level so a target just needs to mark support for it and it will be handled automatically
	// Tier: Experimental
	FeatureModelNamespaces
	// FeaturePublicExports relates to the ability to expose configured public aliases using the x-speakeasy-exports extension
	// Tier: Experimental
	FeaturePublicExports
	// FeatureMethodSignatures relates to the ability to configure the generated SDK class method shape (e.g. params-object signatures)
	// Tier: Experimental
	FeatureMethodSignatures
	// FeatureFormatBinary relates to the ability to handle the binary format for string types in targets that require special handling
	// Tier: Experimental
	FeatureFormatBinary
	// FeatureContentMediaTypeApplicationJSON relates to supporting JSON Schema content vocabulary contentMediaType of application/json
	// Tier: Experimental
	FeatureContentMediaTypeApplicationJSON
	// FeatureUUID relates to the ability to handle the uuid format with a native UUID type instead of a plain string
	// Tier: Experimental
	FeatureUUID
	// FeatureDuration relates to the ability to handle the duration format with a native duration type instead of a plain string
	// Tier: Experimental
	FeatureDuration
	// FeatureCLICommands relates to the ability to declare a curated CLI command surface in the OpenAPI document using the x-speakeasy-cli-commands extension
	// Tier: Experimental
	FeatureCLICommands
)

const (
	ReadmeSectionInstallation     = "installation"
	ReadmeSectionUsage            = "usage"
	ReadmeSectionOperations       = "operations"
	ReadmeSectionHttpClient       = "http-client"
	ReadmeSectionSecurity         = "security"
	ReadmeSectionServer           = "server"
	ReadmeSectionGlobalParameters = "global-parameters"
	ReadmeSectionRetries          = "retries"
	ReadmeSectionPagination       = "pagination"
	ReadmeSectionErrors           = "errors"
	ReadmeSectionDebug            = "debug"
)

var readmeSections = []string{
	ReadmeSectionInstallation,
	ReadmeSectionUsage,
	ReadmeSectionOperations,
	ReadmeSectionHttpClient,
	ReadmeSectionSecurity,
	ReadmeSectionServer,
	ReadmeSectionGlobalParameters,
	ReadmeSectionRetries,
	ReadmeSectionPagination,
	ReadmeSectionErrors,
	ReadmeSectionDebug,
}

const (
	TierCore         = "Core"
	TierBeta         = "Beta"
	TierGA           = "GA"
	TierGold         = "Gold"
	TierExperimental = "Experimental"
)

func GetAllFeatures() []Feature {
	return featureList
}

func GetAllReadmeSections() []string {
	return readmeSections
}

func GetReadmeSectionsByFeature(feature Feature) []string {
	return readmeSectionsByFeature[feature]
}

func GetFeaturesByTier(tier string) []Feature {
	switch tier {
	case TierCore:
		return tierCoreFeatures
	case TierBeta:
		return tierBetaFeatures
	case TierGA:
		return tierGAFeatures
	case TierGold:
		return tierGoldFeatures
	case TierExperimental:
		return tierExperimentalFeatures
	default:
		return nil
	}
}

func GetFeatureTier(feature Feature) string {
	return featureTiers[feature]
}

type Features struct {
	target             types.Target
	executor           *FeatureExecutor
	featureUsage       map[Feature]bool
	baseTemplateConfig map[string]any
	tooLate            bool
	mockConfig         *MockConfig
}

// Mock implementation of features that are retrieved from the executor, which
// typically reads the template config.ts. Intended for testing.
type MockConfig struct {
	// Features that will return true for IsFeatureEnabled method calls.
	EnabledFeatures []Feature

	// Features that will return true for IsFeatureIgnored method calls.
	IgnoredFeatures []Feature

	// Features that will return for getStaticallyUsedFeatures method calls.
	StaticallyUsedFeatures []Feature

	// Features that will return true for IsFeatureSupported method calls.
	SupportedFeatures []Feature
}

func New(target types.Target, baseTemplateConfig map[string]any) (*Features, error) {
	features, err := executor.New(target, "features.ts")
	if err != nil {
		return nil, err
	}

	return &Features{
		target:             target,
		featureUsage:       map[Feature]bool{},
		executor:           NewFeatureExecutor(features, baseTemplateConfig, target),
		baseTemplateConfig: baseTemplateConfig,
	}, nil
}

// Returns a Features without any execution abilities. Intended for testing,
// such as verifying feature recording.
func NewMock(mockConfig *MockConfig) *Features {
	return &Features{
		featureUsage: make(map[Feature]bool),
		mockConfig:   mockConfig,
	}
}

func (f *Features) Target() types.Target {
	return f.target
}

func (f *Features) GetImplementedFeaturesList(ctx context.Context) []Feature {
	languageFeatureList := []Feature{}

	for _, feature := range featureList {
		if f.IsFeatureSupported(ctx, feature) && !f.IsFeatureIgnored(ctx, feature) {
			languageFeatureList = append(languageFeatureList, feature)
		}
	}

	return languageFeatureList
}

func (f *Features) IsFeatureSupported(ctx context.Context, feature Feature) bool {
	if f.mockConfig != nil {
		return slices.Contains(f.mockConfig.SupportedFeatures, feature)
	}

	return f.executor.IsFeatureSupported(ctx, feature)
}

func (f *Features) IsFeatureEnabled(ctx context.Context, feature Feature) bool {
	if f.mockConfig != nil {
		return slices.Contains(f.mockConfig.EnabledFeatures, feature)
	}

	return f.executor.IsFeatureEnabled(ctx, feature)
}

func (f *Features) IsFeatureIgnored(ctx context.Context, feature Feature) bool {
	if f.mockConfig != nil {
		return slices.Contains(f.mockConfig.IgnoredFeatures, feature)
	}

	return f.executor.IsFeatureIgnored(ctx, feature)
}

func (f *Features) IsUnionWrapper(ctx context.Context) bool {
	return f.executor.IsUnionWrapper(ctx)
}

func (f *Features) GetFeatureVersion(ctx context.Context, feature Feature) string {
	return f.executor.GetFeatureVersion(ctx, feature)
}

func (f *Features) IsReadmeSectionImplemented(ctx context.Context, readmeSectionID string) bool {
	return f.executor.IsReadmeSectionImplemented(ctx, readmeSectionID)
}

func (f *Features) IsReadmeSectionIgnored(ctx context.Context, readmeSectionID string) bool {
	return f.executor.IsReadmeSectionIgnored(ctx, readmeSectionID)
}

func (f *Features) RecordFeatureUsage(ctx context.Context, feature Feature) {
	if f.tooLate {
		panic("cannot record feature usage after GetUsedFeatures has been called")
	}

	if !f.IsFeatureSupported(ctx, feature) && !f.IsFeatureIgnored(ctx, feature) {
		switch feature {
		case FeatureCore:
			logging.LogWarning(ctx, f.target.Target+" is not supported", errUnsupportedFeature)
		case FeatureGetRequestBodies:
			logging.LogWarning(ctx, fmt.Sprintf("request bodies are not supported for GET operations in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureGlobalSecurity:
			logging.LogWarning(ctx, fmt.Sprintf("global security is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureMethodSecurity:
			logging.LogWarning(ctx, fmt.Sprintf("per operation security is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureGlobalServerURLs:
			logging.LogWarning(ctx, fmt.Sprintf("global server urls are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureMethodServerURLs:
			logging.LogWarning(ctx, fmt.Sprintf("per operation/path server urls are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureGlobals:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtGlobals.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureHiddenGlobals:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtGlobalsHidden.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureEnums:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtEnums.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureOpenEnums:
			logging.LogWarning(ctx, fmt.Sprintf("open enums are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureServerIDs:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtServerID.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureNameOverrides:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtNameOverride.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureIncludes:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtInclude.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureDocs:
			logging.LogWarning(ctx, fmt.Sprintf("%s extensions not supported in %s clients", extensions.ExtDocs.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureExamples:
			logging.LogWarning(ctx, fmt.Sprintf("%s and %s extensions not supported in %s clients", extensions.ExtUsageExample.Name(), extensions.ExtExample.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureGroups:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtGroup.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureDeprecations:
			logging.LogWarning(ctx, fmt.Sprintf("%s and %s extensions not supported in %s clients", extensions.ExtDeprecationMessage.Name(), extensions.ExtDeprecationReplacement.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureRetries:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtRetries.Name(), f.target.Target), errUnsupportedFeature)
		case FeaturePagination:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtPagination.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureInputOutputModels:
			logging.LogWarning(ctx, fmt.Sprintf("input/output models are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureIgnores:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtIgnore.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureTypeOverrides:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtTypeOverride.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureDevContainers:
			logging.LogWarning(ctx, fmt.Sprintf("dev containers is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureErrors:
			logging.LogWarning(ctx, fmt.Sprintf("%s and %s extensions not supported in %s clients", extensions.ExtErrors.Name(), extensions.ExtErrorMessage.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureBigInt:
			logging.LogWarning(ctx, fmt.Sprintf("format: bigint is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureDecimal:
			logging.LogWarning(ctx, fmt.Sprintf("format: decimal is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureSets:
			logging.LogWarning(ctx, fmt.Sprintf("format: set is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureTests:
			logging.LogWarning(ctx, fmt.Sprintf("tests are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureTagBasedOrdering:
			logging.LogWarning(ctx, fmt.Sprintf("tag based ordering is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureServerEvents:
			logging.LogWarning(ctx, fmt.Sprintf("server-sent events are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureServerEventsSentinels:
			logging.LogWarning(ctx, fmt.Sprintf("server-sent event sentinels are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureURLBasedPagination:
			logging.LogWarning(ctx, fmt.Sprintf("url based pagination is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureResponseFormat:
			logging.LogWarning(ctx, fmt.Sprintf("response format is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureNullables:
			logging.LogWarning(ctx, fmt.Sprintf("nullable values are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureStringNumberFormats:
			logging.LogWarning(ctx, fmt.Sprintf("strings with format:int64 or format:float64 are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureMethodArguments:
			logging.LogWarning(ctx, fmt.Sprintf("method argument customization is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureFlatRequests:
			logging.LogWarning(ctx, fmt.Sprintf("request body flattening is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureOperationTimeout:
			logging.LogWarning(ctx, fmt.Sprintf("operation timeouts are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureDefaultEnabledRetries:
			logging.LogWarning(ctx, fmt.Sprintf("default enabled retries is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureDeepObjectParams:
			logging.LogWarning(ctx, fmt.Sprintf("deepObject style parameters is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureEnvVarSecurityUsage:
			logging.LogWarning(ctx, fmt.Sprintf("env var security usage is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureReactQueryHooks:
			logging.LogWarning(ctx, fmt.Sprintf("react hooks are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureCustomCodeRegions:
			logging.LogWarning(ctx, fmt.Sprintf("custom code regions are not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureMCPServer:
			logging.LogWarning(ctx, fmt.Sprintf("mcp servers are not supported in %s sdks", f.target.Target), errUnsupportedFeature)
		case FeatureConfigurableModuleName:
			logging.LogWarning(ctx, fmt.Sprintf("Configurable Module name is not supported in %s sdks", f.target.Target), errUnsupportedFeature)
		case FeatureExamplesDirectory:
			logging.LogWarning(ctx, fmt.Sprintf("examples directory configuration is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureModelNamespaces:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtModelNamespace.Name(), f.target.Target), errUnsupportedFeature)
		case FeaturePublicExports:
			logging.LogWarning(ctx, fmt.Sprintf("%s extension not supported in %s clients", extensions.ExtPublicExports.Name(), f.target.Target), errUnsupportedFeature)
		case FeatureMethodSignatures:
			logging.LogWarning(ctx, fmt.Sprintf("method signature customization is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureFormatBinary:
			logging.LogWarning(ctx, fmt.Sprintf("format: binary is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureContentMediaTypeApplicationJSON:
			logging.LogWarning(ctx, fmt.Sprintf("contentMediaType application/json is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureUUID:
			logging.LogWarning(ctx, fmt.Sprintf("format: uuid is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		case FeatureDuration:
			logging.LogWarning(ctx, fmt.Sprintf("format: duration is not supported in %s clients", f.target.Target), errUnsupportedFeature)
		default:
			panic(fmt.Sprintf("feature \"%v\" recorded but not handled likely not documented in features.ts", feature))
		}
	} else {
		f.featureUsage[feature] = true
	}
}

func (f *Features) RegisterReadmeSectionTemplated(readmeSectionID string, feature Feature) {
	// TODO: implement though not useful until we can have an uber spec the exercises all features
}

func (f *Features) ReinstateFeatureUsage(_ context.Context, featureUsage map[string]bool) {
	for feature := range featureUsage {
		f.featureUsage[FeatureFromString(feature)] = true
	}
}

func (f *Features) GetFeatureUsage(_ context.Context) map[string]bool {
	featureUsage := map[string]bool{}

	for feature := range f.featureUsage {
		featureUsage[feature.String()] = true
	}

	return featureUsage
}

func (f *Features) GetUsedFeatures(ctx context.Context) map[string]string {
	f.tooLate = true

	features := map[string]string{}
	for feature := range f.featureUsage {
		features[feature.String()] = f.GetFeatureVersion(ctx, feature)
	}

	staticallyUsedFeatures := f.getStaticallyUsedFeatures(ctx)
	for _, feature := range staticallyUsedFeatures {
		features[feature.String()] = f.GetFeatureVersion(ctx, feature)
	}

	return features
}

func (f *Features) GetAvailableFeatures(ctx context.Context) map[string]bool {
	if f.mockConfig != nil {
		availableFeatures := make(map[string]bool, len(f.mockConfig.SupportedFeatures))

		for _, feature := range f.mockConfig.SupportedFeatures {
			availableFeatures[feature.String()] = true
		}

		return availableFeatures
	}

	return f.executor.GetAvailableFeatures(ctx)
}

func (f *Features) IsFeatureUsed(feature Feature) bool {
	return f.featureUsage[feature]
}

func (f *Features) GetAutoGeneratedHeader(ctx context.Context, generatedID string) string {
	return f.executor.GetAutoGeneratedHeader(ctx, generatedID)
}

// Will return a list of features that are used by the template on all generations
func (f *Features) getStaticallyUsedFeatures(ctx context.Context) []Feature {
	if f.mockConfig != nil {
		return f.mockConfig.StaticallyUsedFeatures
	}

	return f.executor.GetStaticallyUsedFeatures(ctx)
}

func (f *Features) CollectGlobalsAndServersModels(ctx context.Context) bool {
	return f.executor.CollectGlobalsAndServersModels(ctx)
}
