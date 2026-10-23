module github.com/speakeasy-api/openapi-generation/v2/cmd/wasm/ast

go 1.26.10

require (
	github.com/speakeasy-api/openapi v1.25.7-0.20261016002657-683982d5baaf
	github.com/speakeasy-api/openapi-generation/v2 v2.0.0-00010101000000-000000000000
)

require (
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/apparentlymart/go-textseg/v17 v17.0.1 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/dop251/goja v0.0.0 // indirect
	github.com/dop251/goja_nodejs v0.0.0-20240728170619-29b559befffc // indirect
	github.com/ettle/strcase v0.2.0 // indirect
	github.com/evanw/esbuild v0.28.2 // indirect
	github.com/gammban/numtow v0.0.4 // indirect
	github.com/gertd/go-pluralize v0.2.1 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/hashicorp/hcl/v2 v2.25.0 // indirect
	github.com/itchyny/gojq v0.12.19 // indirect
	github.com/itchyny/timefmt-go v0.1.8 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	github.com/speakeasy-api/generation-context v1.1.0 // indirect
	github.com/speakeasy-api/jq v0.13.0 // indirect
	github.com/speakeasy-api/jsonpath v0.7.0 // indirect
	github.com/speakeasy-api/sdk-gen-config v1.60.0 // indirect
	github.com/speakeasy-api/speakeasy-client-sdk-go/v3 v3.28.1 // indirect
	github.com/swaggest/jsonschema-go v0.3.79 // indirect
	github.com/swaggest/refl v1.4.0 // indirect
	github.com/zclconf/go-cty v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/tools v0.51.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/speakeasy-api/openapi-generation/v2 => ../../..

// replace github.com/dop251/goja => github.com/speakeasy-api/goja feat/debugger
replace github.com/dop251/goja => github.com/speakeasy-api/goja v0.0.0-20260223084236-ed0328a0a462

// replace github.com/dop251/goja/debugger => github.com/speakeasy-api/goja/debugger feat/debugger
replace github.com/dop251/goja/debugger => github.com/speakeasy-api/goja/debugger v0.0.0-20260223084236-ed0328a0a462
