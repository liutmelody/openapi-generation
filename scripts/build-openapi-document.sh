#!/usr/bin/env bash

set -euo pipefail
shopt -s nullglob

TARGET="$1"
VARIANT="$2"

TARGET_DIR="sdk-${TARGET}-${VARIANT}"

if [[ "${TARGET}" == "mcp-"* ]]; then
	TARGET_DIR="${TARGET}-${VARIANT}"
elif [[ "${TARGET}" == "terraform" ]]; then
	TARGET_DIR="terraform-provider-${VARIANT}"
fi

template_spec="./tests/specs/uber.yaml"
component_spec="./tests/specs/components.yaml"
if [ "${VARIANT}" == "oauth2-password" ] || [ "${VARIANT}" == "custom-http" ]; then
    template_spec="./tests/specs/ecommerce.yaml"
    component_spec="./tests/specs/components.ecommerce.yaml"
elif [ "${VARIANT}" == "no-servers" ] || [ "${VARIANT}" == "relative-servers" ]; then
    template_spec="./tests/specs/no-servers.yaml"
    component_spec="./tests/specs/components.yaml"
elif [ "${VARIANT}" == "basic-http" ]; then
    template_spec="./tests/specs/basic-http.yaml"
elif [ "${VARIANT}" == "security-options" ]; then
    template_spec="./tests/specs/security-options.yaml"
fi

build_supporting_spec() {
	local source_spec="$1"
	local destination_spec="$2"
	shift 2
	local join_args=("$@")

	if [[ "${#join_args[@]}" -eq 0 ]]; then
		cp "${source_spec}" "${destination_spec}"
		return
	fi

	local cmd=(go run ./cmd/overlay -s "${source_spec}")
	if [[ ${#join_args[@]} -gt 0 ]]; then
		cmd+=("${join_args[@]}")
	fi
	cmd+=(-out "${destination_spec}")
	"${cmd[@]}"
}

list_fragment_files() {
	local dir
	for dir in "$@"; do
		if [[ ! -d "${dir}" ]]; then
			continue
		fi

		local fragment
		for fragment in "${dir}"/*.yaml "${dir}"/*.yml; do
			printf '%s\n' "${fragment}"
		done
	done
}

template_join_args=()
template_spec_name="$(basename "${template_spec}" .yaml)"

# Each target gets their own fragment directory
template_fragment_dirs=("./tests/specs/fragments/${VARIANT}")

# Additionally, if the base spec is named differently than the target, also
# include fragments named after the base spec. For example the `primary` target
# (based on `uber.yaml`) will receive both `uber/` and `primary/` fragments.
# Conversely the `no-servers/` fragment will apply to both `no-servers` and
# `relative-servers` targets since the base spec is `no-servers.yaml`.
if [[ "${template_spec_name}" != "${VARIANT}" ]]; then
	template_fragment_dirs+=("./tests/specs/fragments/${template_spec_name}")
fi

while IFS= read -r fragment; do
	[[ -n "${fragment}" ]] || continue
	template_join_args+=(-join "${fragment}")
done < <(list_fragment_files "${template_fragment_dirs[@]}")

component_join_args=()
while IFS= read -r fragment; do
	[[ -n "${fragment}" ]] || continue
	component_join_args+=(-join "${fragment}")
done < <(list_fragment_files "${component_spec%.yaml}.d")

sed -e "s/\$TARGET/${TARGET}/g" ./tests/overlays/base.yaml > ./testSDKs/${TARGET_DIR}/exclude-target-overlay.yaml

if test -f ./tests/overlays/${VARIANT}/overlay.yaml; then
	if test -f ./tests/overlays/${VARIANT}/${TARGET}/overlay.yaml; then
		cmd=(go run ./cmd/overlay -s "$template_spec")
		if [[ ${#template_join_args[@]} -gt 0 ]]; then
			cmd+=("${template_join_args[@]}")
		fi
		cmd+=(
			-overlay ./testSDKs/${TARGET_DIR}/exclude-target-overlay.yaml
			-overlay ./tests/overlays/${VARIANT}/overlay.yaml
			-overlay ./tests/overlays/${VARIANT}/${TARGET}/overlay.yaml
			-disable-strict
			-out ./testSDKs/${TARGET_DIR}/openapi.yaml
		)
		"${cmd[@]}";
	else
		cmd=(go run ./cmd/overlay -s "$template_spec")
		if [[ ${#template_join_args[@]} -gt 0 ]]; then
			cmd+=("${template_join_args[@]}")
		fi
		cmd+=(
			-overlay ./testSDKs/${TARGET_DIR}/exclude-target-overlay.yaml
			-overlay ./tests/overlays/${VARIANT}/overlay.yaml
			-disable-strict
			-out ./testSDKs/${TARGET_DIR}/openapi.yaml
		)
		"${cmd[@]}";
	fi;
else
	if test -f ./tests/overlays/${VARIANT}/${TARGET}/overlay.yaml; then
		cmd=(go run ./cmd/overlay -s "$template_spec")
		if [[ ${#template_join_args[@]} -gt 0 ]]; then
			cmd+=("${template_join_args[@]}")
		fi
		cmd+=(
			-overlay ./testSDKs/${TARGET_DIR}/exclude-target-overlay.yaml
			-overlay ./tests/overlays/${VARIANT}/${TARGET}/overlay.yaml
			-disable-strict
			-out ./testSDKs/${TARGET_DIR}/openapi.yaml
		)
		"${cmd[@]}";
	else
		cmd=(go run ./cmd/overlay -s "$template_spec")
		if [[ ${#template_join_args[@]} -gt 0 ]]; then
			cmd+=("${template_join_args[@]}")
		fi
		cmd+=(
			-overlay ./testSDKs/${TARGET_DIR}/exclude-target-overlay.yaml
			-disable-strict
			-out ./testSDKs/${TARGET_DIR}/openapi.yaml
		)
		"${cmd[@]}";
	fi;
fi ;
if [[ ${#component_join_args[@]} -gt 0 ]]; then
	build_supporting_spec "${component_spec}" "./testSDKs/${TARGET_DIR}/$(basename "${component_spec}")" "${component_join_args[@]}"
else
	build_supporting_spec "${component_spec}" "./testSDKs/${TARGET_DIR}/$(basename "${component_spec}")"
fi
