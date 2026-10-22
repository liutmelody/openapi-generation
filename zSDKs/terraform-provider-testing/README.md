# testing

Terraform Provider for the *testing* API.

[![Built by Speakeasy](https://img.shields.io/badge/Built_by-SPEAKEASY-374151?style=for-the-badge&labelColor=f3f4f6)](https://www.speakeasy.com/?utm_source=testing&utm_campaign=terraform)
[![License: MIT](https://img.shields.io/badge/LICENSE_//_MIT-3b5bdb?style=for-the-badge&labelColor=eff6ff)](https://opensource.org/licenses/MIT)


## 🏗 **Welcome to your new Terraform Provider!** 🏗

It has been generated successfully based on your OpenAPI spec. However, it is not yet ready for production use. Here are some next steps:
- [ ] 🛠 Add resources and datasources to your SDK by [annotating your OAS](https://www.speakeasy.com/docs/customize-terraform/terraform-extensions#map-api-entities-to-terraform-resources)
- [ ] ♻️ Refine your terraform provider quickly by iterating locally with the [Speakeasy CLI](https://github.com/speakeasy-api/speakeasy)
- [ ] 🎁 Publish your terraform provider to hashicorp registry by [configuring automatic publishing](https://www.speakeasy.com/docs/terraform-publishing)
- [ ] ✨ When ready to productionize, delete this section from the README

<!-- Start Summary [summary] -->
## Summary

Terraform Provider Review: A test document for reviewing Terraform Provider generation.

This document covers generator mappings to and from the Terraform type
system, OAS properties, and x-speakeasy-* annotations. Terraform Provider
generation and testing requires stateful request and response handling,
which is why this document is separate from the standard language SDK
document.

Each concept (whether Terraform, OAS, or Speakeasy) should be implemented in
an individual Terraform resource while test cases should cover request only,
response only, and both request and response handling since the generator
must merge multiple and/or manipulate parts of the AST to support Terraform
properly.
<!-- End Summary [summary] -->

<!-- Start Table of Contents [toc] -->
## Table of Contents
<!-- $toc-max-depth=2 -->
* [testing](#testing)
  * [🏗 **Welcome to your new Terraform Provider!** 🏗](#welcome-to-your-new-terraform-provider)
  * [Installation](#installation)
  * [Authentication](#authentication)
  * [Available Resources and Data Sources](#available-resources-and-data-sources)
  * [Testing the provider locally](#testing-the-provider-locally)
* [Development](#development)
  * [Contributions](#contributions)

<!-- End Table of Contents [toc] -->

<!-- Start Installation [installation] -->
## Installation

To install this provider, copy and paste this code into your Terraform configuration. Then, run `terraform init`.

```hcl
terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.0.1"
    }
  }
}

provider "testing" {
  hostname    = "..." # Optional - can use TESTING_HOSTNAME environment variable
  server_port = "..." # Optional - can use TESTING_PORT environment variable
  server_url  = "..." # Optional
}
```
<!-- End Installation [installation] -->

<!-- Start Authentication [security] -->
## Authentication

This provider supports authentication configuration via environment variables and provider configuration.

The configuration precedence is:

- Provider configuration
- Environment variables

Available configuration:

| Provider Attribute | Description |
|---|---|
| `api_key` | Custom global API Key security scheme description. Configurable via environment variable `TESTING_API_KEY`. |
| `audience` | Custom global OAuth Client Credentials flow description audience. |
| `bearer` | Custom global HTTP Bearer token description. Configurable via environment variable `TESTING_BEARER`. |
| `client_id` | Custom global OAuth Client Credentials flow description client identifier. Configurable via environment variable `TESTING_CLIENT_ID`. |
| `client_secret` | Custom global OAuth Client Credentials flow description client secret. Configurable via environment variable `TESTING_CLIENT_SECRET`. |
| `custom_key` | Custom security scheme key. Configurable via environment variable `TESTING_CUSTOM_KEY`. |
| `custom_secret` | Custom security scheme secret. Configurable via environment variable `TESTING_CUSTOM_SECRET`. |
| `password` | Custom global HTTP Basic security scheme description password. Configurable via environment variable `TESTING_PASSWORD`. |
| `token_url` | Custom global OAuth Client Credentials flow description token URL. |
| `username` | Custom global HTTP Basic security scheme description username. Configurable via environment variable `TESTING_USERNAME`. |
<!-- End Authentication [security] -->

<!-- Start Available Resources and Data Sources [operations] -->
## Available Resources and Data Sources

### Managed Resources

* [testing_api_create_and_update](docs/resources/api_create_and_update.md)
* [testing_basic](docs/resources/basic.md)
* [testing_const_container](docs/resources/const_container.md)
* [testing_discriminated_union](docs/resources/discriminated_union.md)
* [testing_discriminated_union_array](docs/resources/discriminated_union_array.md)
* [testing_framework_type](docs/resources/framework_type.md)
* [testing_import_defaulted_id](docs/resources/import_defaulted_id.md)
* [testing_import_id_enum_string](docs/resources/import_id_enum_string.md)
* [testing_import_id_int32](docs/resources/import_id_int32.md)
* [testing_import_id_int64](docs/resources/import_id_int64.md)
* [testing_import_id_integer](docs/resources/import_id_integer.md)
* [testing_import_id_string](docs/resources/import_id_string.md)
* [testing_import_id_string_acronym](docs/resources/import_id_string_acronym.md)
* [testing_import_matched_id](docs/resources/import_matched_id.md)
* [testing_import_multiple_id](docs/resources/import_multiple_id.md)
* [testing_import_multiple_id_acronym](docs/resources/import_multiple_id_acronym.md)
* [testing_mixed_request_and_response_type](docs/resources/mixed_request_and_response_type.md)
* [testing_name_shadowing](docs/resources/name_shadowing.md)
* [testing_oas_content_media_type](docs/resources/oas_content_media_type.md)
* [testing_oas_default](docs/resources/oas_default.md)
* [testing_oas_deprecated](docs/resources/oas_deprecated.md)
* [testing_oas_enum](docs/resources/oas_enum.md)
* [testing_oas_example](docs/resources/oas_example.md)
* [testing_oas_format_binary_json](docs/resources/oas_format_binary_json.md)
* [testing_oas_maximum](docs/resources/oas_maximum.md)
* [testing_oas_maxitems](docs/resources/oas_maxitems.md)
* [testing_oas_maxlength](docs/resources/oas_maxlength.md)
* [testing_oas_minimum](docs/resources/oas_minimum.md)
* [testing_oas_minitems](docs/resources/oas_minitems.md)
* [testing_oas_minlength](docs/resources/oas_minlength.md)
* [testing_oas_one_of](docs/resources/oas_one_of.md)
* [testing_oas_pattern](docs/resources/oas_pattern.md)
* [testing_oas_read_only](docs/resources/oas_read_only.md)
* [testing_oas_required](docs/resources/oas_required.md)
* [testing_oas_security_operation](docs/resources/oas_security_operation.md)
* [testing_oas_servers_operation](docs/resources/oas_servers_operation.md)
* [testing_oas_servers_path](docs/resources/oas_servers_path.md)
* [testing_oas_uniqueitems](docs/resources/oas_uniqueitems.md)
* [testing_oas_write_only](docs/resources/oas_write_only.md)
* [testing_oas_write_only_nested](docs/resources/oas_write_only_nested.md)
* [testing_patch](docs/resources/patch.md)
* [testing_root_union_write_only](docs/resources/root_union_write_only.md)
* [testing_status_code_2xx](docs/resources/status_code_2xx.md)
* [testing_status_code_409](docs/resources/status_code_409.md)
* [testing_transform](docs/resources/transform.md)
* [testing_undiscriminated_union_array](docs/resources/undiscriminated_union_array.md)
* [testing_unsound_readonly_matched_op](docs/resources/unsound_readonly_matched_op.md)
* [testing_unsound_readonly_single_op](docs/resources/unsound_readonly_single_op.md)
* [testing_x_additional_properties_name](docs/resources/x_additional_properties_name.md)
* [testing_x_client_filter](docs/resources/x_client_filter.md)
* [testing_x_deprecation_message](docs/resources/x_deprecation_message.md)
* [testing_x_entity_description](docs/resources/x_entity_description.md)
* [testing_x_entity_missing_codes](docs/resources/x_entity_missing_codes.md)
* [testing_x_entity_object_nested_optional](docs/resources/x_entity_object_nested_optional.md)
* [testing_x_entity_object_nested_required](docs/resources/x_entity_object_nested_required.md)
* [testing_x_entity_one_of](docs/resources/x_entity_one_of.md)
* [testing_x_entity_version](docs/resources/x_entity_version.md)
* [testing_x_globals](docs/resources/x_globals.md)
* [testing_x_match](docs/resources/x_match.md)
* [testing_x_match_nested_readonly](docs/resources/x_match_nested_readonly.md)
* [testing_x_match_prior_state](docs/resources/x_match_prior_state.md)
* [testing_x_pagination_offset_limits](docs/resources/x_pagination_offset_limits.md)
* [testing_x_pagination_with_request_array](docs/resources/x_pagination_with_request_array.md)
* [testing_x_param_sensitive](docs/resources/x_param_sensitive.md)
* [testing_x_param_suppress_computed_diff](docs/resources/x_param_suppress_computed_diff.md)
* [testing_x_plan_modifiers](docs/resources/x_plan_modifiers.md)
* [testing_x_plan_validators](docs/resources/x_plan_validators.md)
* [testing_x_polling](docs/resources/x_polling.md)
* [testing_x_soft_delete_property](docs/resources/x_soft_delete_property.md)
* [testing_x_terraform_custom_default](docs/resources/x_terraform_custom_default.md)
* [testing_x_terraform_custom_type](docs/resources/x_terraform_custom_type.md)
* [testing_x_terraform_ignore](docs/resources/x_terraform_ignore.md)
* [testing_x_terraform_write_only](docs/resources/x_terraform_write_only.md)
* [testing_x_unknown_values](docs/resources/x_unknown_values.md)
* [testing_x_wrapped_attribute](docs/resources/x_wrapped_attribute.md)

### Data Sources

* [testing_api_create_and_update](docs/data-sources/api_create_and_update.md)
* [testing_basic](docs/data-sources/basic.md)
* [testing_const_container](docs/data-sources/const_container.md)
* [testing_discriminated_union](docs/data-sources/discriminated_union.md)
* [testing_framework_type](docs/data-sources/framework_type.md)
* [testing_import_defaulted_id](docs/data-sources/import_defaulted_id.md)
* [testing_import_id_enum_string](docs/data-sources/import_id_enum_string.md)
* [testing_import_id_int32](docs/data-sources/import_id_int32.md)
* [testing_import_id_int64](docs/data-sources/import_id_int64.md)
* [testing_import_id_integer](docs/data-sources/import_id_integer.md)
* [testing_import_id_string](docs/data-sources/import_id_string.md)
* [testing_import_id_string_acronym](docs/data-sources/import_id_string_acronym.md)
* [testing_import_matched_id](docs/data-sources/import_matched_id.md)
* [testing_import_multiple_id](docs/data-sources/import_multiple_id.md)
* [testing_import_multiple_id_acronym](docs/data-sources/import_multiple_id_acronym.md)
* [testing_mixed_request_and_response_type](docs/data-sources/mixed_request_and_response_type.md)
* [testing_name_shadowing](docs/data-sources/name_shadowing.md)
* [testing_oas_content_media_type](docs/data-sources/oas_content_media_type.md)
* [testing_oas_default](docs/data-sources/oas_default.md)
* [testing_oas_deprecated](docs/data-sources/oas_deprecated.md)
* [testing_oas_enum](docs/data-sources/oas_enum.md)
* [testing_oas_example](docs/data-sources/oas_example.md)
* [testing_oas_format_binary_json](docs/data-sources/oas_format_binary_json.md)
* [testing_oas_maximum](docs/data-sources/oas_maximum.md)
* [testing_oas_maxitems](docs/data-sources/oas_maxitems.md)
* [testing_oas_maxlength](docs/data-sources/oas_maxlength.md)
* [testing_oas_minimum](docs/data-sources/oas_minimum.md)
* [testing_oas_minitems](docs/data-sources/oas_minitems.md)
* [testing_oas_minlength](docs/data-sources/oas_minlength.md)
* [testing_oas_one_of](docs/data-sources/oas_one_of.md)
* [testing_oas_pattern](docs/data-sources/oas_pattern.md)
* [testing_oas_read_only](docs/data-sources/oas_read_only.md)
* [testing_oas_required](docs/data-sources/oas_required.md)
* [testing_oas_security_operation](docs/data-sources/oas_security_operation.md)
* [testing_oas_servers_operation](docs/data-sources/oas_servers_operation.md)
* [testing_oas_servers_path](docs/data-sources/oas_servers_path.md)
* [testing_oas_uniqueitems](docs/data-sources/oas_uniqueitems.md)
* [testing_oas_write_only](docs/data-sources/oas_write_only.md)
* [testing_oas_write_only_nested](docs/data-sources/oas_write_only_nested.md)
* [testing_parameters_deep_object](docs/data-sources/parameters_deep_object.md)
* [testing_request_body_optional_inline](docs/data-sources/request_body_optional_inline.md)
* [testing_request_body_optional_inline_with_parameter](docs/data-sources/request_body_optional_inline_with_parameter.md)
* [testing_request_body_required_inline](docs/data-sources/request_body_required_inline.md)
* [testing_request_body_required_inline_with_parameter](docs/data-sources/request_body_required_inline_with_parameter.md)
* [testing_request_body_required_inline_x_speakeasy_entity_level1](docs/data-sources/request_body_required_inline_x_speakeasy_entity_level1.md)
* [testing_request_body_required_inline_x_speakeasy_entity_level1_with_parameter](docs/data-sources/request_body_required_inline_x_speakeasy_entity_level1_with_parameter.md)
* [testing_request_body_required_inline_x_speakeasy_entity_level2](docs/data-sources/request_body_required_inline_x_speakeasy_entity_level2.md)
* [testing_request_body_required_inline_x_speakeasy_entity_level2_with_parameter](docs/data-sources/request_body_required_inline_x_speakeasy_entity_level2_with_parameter.md)
* [testing_request_body_required_ref](docs/data-sources/request_body_required_ref.md)
* [testing_request_body_required_ref_nullable](docs/data-sources/request_body_required_ref_nullable.md)
* [testing_request_body_required_ref_nullable_with_parameter](docs/data-sources/request_body_required_ref_nullable_with_parameter.md)
* [testing_request_body_required_ref_with_parameter](docs/data-sources/request_body_required_ref_with_parameter.md)
* [testing_request_body_required_ref_x_speakeasy_entity_level1](docs/data-sources/request_body_required_ref_x_speakeasy_entity_level1.md)
* [testing_request_body_required_ref_x_speakeasy_entity_level1_array](docs/data-sources/request_body_required_ref_x_speakeasy_entity_level1_array.md)
* [testing_request_body_required_ref_x_speakeasy_entity_level1_array_with_parameter](docs/data-sources/request_body_required_ref_x_speakeasy_entity_level1_array_with_parameter.md)
* [testing_request_body_required_ref_x_speakeasy_entity_level1_with_parameter](docs/data-sources/request_body_required_ref_x_speakeasy_entity_level1_with_parameter.md)
* [testing_request_body_required_ref_x_speakeasy_entity_level2](docs/data-sources/request_body_required_ref_x_speakeasy_entity_level2.md)
* [testing_request_body_required_ref_x_speakeasy_entity_level2_array](docs/data-sources/request_body_required_ref_x_speakeasy_entity_level2_array.md)
* [testing_request_body_required_ref_x_speakeasy_entity_level2_array_with_parameter](docs/data-sources/request_body_required_ref_x_speakeasy_entity_level2_array_with_parameter.md)
* [testing_request_body_required_ref_x_speakeasy_entity_level2_with_parameter](docs/data-sources/request_body_required_ref_x_speakeasy_entity_level2_with_parameter.md)
* [testing_root_union_write_only](docs/data-sources/root_union_write_only.md)
* [testing_status_code_2xx](docs/data-sources/status_code_2xx.md)
* [testing_status_code_409](docs/data-sources/status_code_409.md)
* [testing_unsound_readonly_matched_op](docs/data-sources/unsound_readonly_matched_op.md)
* [testing_unsound_readonly_single_op](docs/data-sources/unsound_readonly_single_op.md)
* [testing_x_additional_properties_name](docs/data-sources/x_additional_properties_name.md)
* [testing_x_client_filter](docs/data-sources/x_client_filter.md)
* [testing_x_client_filter_array](docs/data-sources/x_client_filter_array.md)
* [testing_x_deprecation_message](docs/data-sources/x_deprecation_message.md)
* [testing_x_entity_array_items](docs/data-sources/x_entity_array_items.md)
* [testing_x_entity_array_nested_optionals](docs/data-sources/x_entity_array_nested_optionals.md)
* [testing_x_entity_array_nested_requireds](docs/data-sources/x_entity_array_nested_requireds.md)
* [testing_x_entity_arrays](docs/data-sources/x_entity_arrays.md)
* [testing_x_entity_description](docs/data-sources/x_entity_description.md)
* [testing_x_entity_missing_codes](docs/data-sources/x_entity_missing_codes.md)
* [testing_x_entity_object](docs/data-sources/x_entity_object.md)
* [testing_x_entity_object_nested_optional](docs/data-sources/x_entity_object_nested_optional.md)
* [testing_x_entity_object_nested_required](docs/data-sources/x_entity_object_nested_required.md)
* [testing_x_entity_object_twice_nested](docs/data-sources/x_entity_object_twice_nested.md)
* [testing_x_entity_one_of](docs/data-sources/x_entity_one_of.md)
* [testing_x_entity_version](docs/data-sources/x_entity_version.md)
* [testing_x_globals](docs/data-sources/x_globals.md)
* [testing_x_match](docs/data-sources/x_match.md)
* [testing_x_match_nested_readonly](docs/data-sources/x_match_nested_readonly.md)
* [testing_x_match_prior_state](docs/data-sources/x_match_prior_state.md)
* [testing_x_pagination_const_page_size](docs/data-sources/x_pagination_const_page_size.md)
* [testing_x_pagination_cursors](docs/data-sources/x_pagination_cursors.md)
* [testing_x_pagination_offset_limits](docs/data-sources/x_pagination_offset_limits.md)
* [testing_x_pagination_offset_limits_nested](docs/data-sources/x_pagination_offset_limits_nested.md)
* [testing_x_pagination_singleton](docs/data-sources/x_pagination_singleton.md)
* [testing_x_pagination_singleton_list](docs/data-sources/x_pagination_singleton_list.md)
* [testing_x_pagination_urls](docs/data-sources/x_pagination_urls.md)
* [testing_x_pagination_with_request_array](docs/data-sources/x_pagination_with_request_array.md)
* [testing_x_param_sensitive](docs/data-sources/x_param_sensitive.md)
* [testing_x_param_suppress_computed_diff_property](docs/data-sources/x_param_suppress_computed_diff_property.md)
* [testing_x_plan_modifiers](docs/data-sources/x_plan_modifiers.md)
* [testing_x_plan_validators](docs/data-sources/x_plan_validators.md)
* [testing_x_polling](docs/data-sources/x_polling.md)
* [testing_x_soft_delete_property](docs/data-sources/x_soft_delete_property.md)
* [testing_x_terraform_custom_default](docs/data-sources/x_terraform_custom_default.md)
* [testing_x_terraform_custom_type](docs/data-sources/x_terraform_custom_type.md)
* [testing_x_terraform_ignore](docs/data-sources/x_terraform_ignore.md)
* [testing_x_terraform_write_only](docs/data-sources/x_terraform_write_only.md)
* [testing_x_unknown_values](docs/data-sources/x_unknown_values.md)
* [testing_x_wrapped_attribute](docs/data-sources/x_wrapped_attribute.md)

### Ephemeral Resources

* [testing_x_entity_description](docs/ephemeral-resources/x_entity_description.md)
* [testing_x_entity_operation_close_multiple_op](docs/ephemeral-resources/x_entity_operation_close_multiple_op.md)
* [testing_x_entity_operation_close_single_op](docs/ephemeral-resources/x_entity_operation_close_single_op.md)
* [testing_x_entity_operation_open_multiple_op](docs/ephemeral-resources/x_entity_operation_open_multiple_op.md)
* [testing_x_entity_operation_open_single_op](docs/ephemeral-resources/x_entity_operation_open_single_op.md)

### Actions

* [testing_x_entity_description](docs/actions/x_entity_description.md)
* [testing_x_entity_operation_invoke_multiple_op](docs/actions/x_entity_operation_invoke_multiple_op.md)
* [testing_x_entity_operation_invoke_single_op](docs/actions/x_entity_operation_invoke_single_op.md)
<!-- End Available Resources and Data Sources [operations] -->

<!-- Start Testing the provider locally [usage] -->
## Testing the provider locally

#### Local Provider

Should you want to validate a change locally, the `--debug` flag allows you to execute the provider against a terraform instance locally.

This also allows for debuggers (e.g. delve) to be attached to the provider.

```sh
go run main.go --debug
# Copy the TF_REATTACH_PROVIDERS env var
# In a new terminal
cd examples/your-example
TF_REATTACH_PROVIDERS=... terraform init
TF_REATTACH_PROVIDERS=... terraform apply
```

#### Compiled Provider

Terraform allows you to use local provider builds by setting a `dev_overrides` block in a configuration file called `.terraformrc`. This block overrides all other configured installation methods.

1. Execute `go build` to construct a binary called `terraform-provider-testing`
2. Ensure that the `.terraformrc` file is configured with a `dev_overrides` section such that your local copy of terraform can see the provider binary

Terraform searches for the `.terraformrc` file in your home directory and applies any configuration settings you set.

```
provider_installation {

  dev_overrides {
      "registry.terraform.io/hashicorp/testing" = "<PATH>"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```
<!-- End Testing the provider locally [usage] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Contributions

While we value open-source contributions to this terraform provider, this library is generated programmatically. Any manual changes added to internal files will be overwritten on the next generation.
We look forward to hearing your feedback. Feel free to open a PR or an issue with a proof of concept and we'll do our best to include it in a future release. 

### SDK Created by [Speakeasy](https://www.speakeasy.com/?utm_source=testing&utm_campaign=terraform)
