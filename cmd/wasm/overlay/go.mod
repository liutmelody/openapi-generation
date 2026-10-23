module github.com/speakeasy-api/openapi-generation/v2/cmd/wasm/overlay

go 1.26.10

require (
	github.com/speakeasy-api/jsonpath v0.7.0
	github.com/speakeasy-api/openapi v1.25.7-0.20261016002657-683982d5baaf
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/dprotaso/go-yit v0.0.0-20250804204205-7afe4252a1dd // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/speakeasy-api/yaml-jsonpath v0.0.0-20261016002013-07e5f48e6d17 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

replace github.com/dop251/goja => github.com/speakeasy-api/goja v0.0.0-20260223084236-ed0328a0a462

replace github.com/dop251/goja/debugger => github.com/speakeasy-api/goja/debugger v0.0.0-20260223084236-ed0328a0a462

replace github.com/speakeasy-api/openapi-generation/v2 => ../../..
