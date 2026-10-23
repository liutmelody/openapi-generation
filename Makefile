.PHONY: *
SHELL := bash

override EXTRA_ARGS :=

ifdef USAGE_GROUP
# To compile all usage snippets inside a testproject (csharp, go, pythonv2, php, typescritpv2, ruby):
# $ USAGE_GROUP="all" make build-<lang>
# You can also specify a subSDK (tag) instead:
# $ TARGET="primary" USAGE_GROUP="unions" make build-<lang>
override EXTRA_ARGS += -u $(USAGE_GROUP)

# To generate standalone usage snippets for a given subSDK:
# $ TARGET="<target>" USAGE_GROUP="<tag>" LANG="<lang>" make standalone-snippet
standalone-snippet:
	$(call generate-standalone-snippet,$(LANG),$(TARGET),-g $(USAGE_GROUP))
endif

ifdef OPERATION
# To generate a standalone usage snippet for a given operation:
# $ TARGET="<target>" OPERATION="<opID>" LANG="<lang>" make standalone-snippet
standalone-snippet:
	$(call generate-standalone-snippet,$(LANG),$(TARGET),-op $(OPERATION))
endif

ifdef SKIP_COMPILE
override EXTRA_ARGS += --skip-compile
endif

ifdef TRACE
override EXTRA_ARGS += "--trace=grpc"
endif

ALL_TARGETS := cli csharp go javav2 mcp-typescript php postman pythonv2 ruby terraform typescriptv2 unity
# TODO: Rename TARGET to VARIANT. Not changing it so it doesn't break github and developer workflows.
ifndef TARGET
VARIANTS := primary \
secondary \
tertiary \
quaternary \
no-zod \
client-credentials \
client-credentials-basic \
oauth2-password \
custom-http \
basic-http \
security-options \
no-servers \
relative-servers \
review
else
VARIANTS := $(TARGET)
endif

# Redirect stdout to a log file
ifdef QUIET
override LOG_OUTPUT := true
endif

# $(1): language
# $(2): test target
# $(3): any additional parameters
define generate-standalone-snippet
if [ "$(2)" == "review" ]; then \
go run cmd/generateusage/main.go -s tests/specs/review.yaml -l $(patsubst %v2,%,$(1)) $(3) ; \
else \
./scripts/build-test-spec.sh $(1) $(2); \
go run cmd/generateusage/main.go -s ./testSDKs/sdk-$(1)-$(2)/openapi.yaml -l $(patsubst %v2,%,$(1)) $(3) ; \
fi
endef

define test-primary-usage-snippets
go run ./tests -lang $(1) -mode usage -group primary || exit 1
endef

define test-standalone-usage-snippets
./scripts/test-standalone-usage.sh $(1)
endef


# $(1): target name
# $(2): build variants
define build-target-rule
build-$(1):
	@EXTRA_ARGS="$(EXTRA_ARGS)" LOG_OUTPUT="$(LOG_OUTPUT)" ./scripts/build-target.sh $(1) $(2)
endef

# $(1): target name
define profile-target-rule
profile-$(1):
	@if [ -n "$(GROUP)" ]; then \
		./scripts/profile-target.sh $(1) $(GROUP); \
	else \
		echo "Usage: GROUP=<group> make profile-$(1)"; \
		echo "Example: GROUP=primary make profile-$(1)"; \
		exit 1; \
	fi
endef

# $(1): target name 
# $(2): test variants
# $(3): optional test flags (--skip-primary-usage and/or --skip-standalone-usage)
define test-target-rule
test-$(1): test-boot
	@source ./.test-ports && EXTRA_ARGS="$(EXTRA_ARGS)" LOG_OUTPUT="$(LOG_OUTPUT)" ./scripts/build-target.sh $(1) $(BUILD_VARIANTS)
	@if [ ! -z "$(2)" ]; then \
		LOG_OUTPUT="$(LOG_OUTPUT)" ./scripts/test-target.sh $(1) $(2) $(3); \
	fi
endef

# $(1): target name
define check-template-rule
check-template-$(1):
	@npx tsgo --project templates/templates/$(1)
endef

# Define build, test, and check rules for a target.
#
# $(1): target name
# $(2): build variants
# $(3): test variants
# $(4): optional test flags (--skip-primary-usage and/or --skip-standalone-usage)
define target-rules
$(eval $(call build-target-rule,$(1),$(2)))
$(eval $(call test-target-rule,$(1),$(3),$(4)))
$(eval $(call check-template-rule,$(1)))
$(eval $(call profile-target-rule,$(1)))
endef

BUILD_VARIANTS := $(VARIANTS)
TEST_VARIANTS := $(VARIANTS)

# Rules for standard SDK targets. Ideally we'd like all targets to be in this list.
TARGETS := go pythonv2 typescriptv2 javav2 csharp ruby cli
$(foreach TARGET,$(TARGETS),\
	$(eval $(call target-rules,$(TARGET),\
		$(BUILD_VARIANTS),\
		$(TEST_VARIANTS))))

# Rules for standard MCP targets. Ideally we'd like all targets to be in this list.
TARGETS := mcp-typescript
MCP_TEST_VARIANTS := $(filter basic-http client-credentials security-options primary,$(TEST_VARIANTS))
$(foreach TARGET,$(TARGETS),\
	$(eval $(call target-rules,$(TARGET),\
		$(BUILD_VARIANTS),\
		$(MCP_TEST_VARIANTS),\
		--skip-primary-usage --skip-standalone-usage)))

# Rules for targets that exclude testing their review variant.
TARGETS := php
FILTERED_VARIANTS := $(filter-out no-servers review,$(TEST_VARIANTS))
$(foreach TARGET,$(TARGETS),\
	$(eval $(call target-rules,$(TARGET),\
		$(BUILD_VARIANTS),\
		$(FILTERED_VARIANTS))))

# We don't run target tests altogether for postman and terraform.
TARGETS := postman terraform
$(foreach TARGET,$(TARGETS),\
	$(eval $(call target-rules,$(TARGET),$(BUILD_VARIANTS))))

include unity.mk # defines build-unity and test-unity targets
$(eval $(call check-template-rule,unity))

# Ensure mockserver TypeScript can be checked
$(eval $(call check-template-rule,mockserver))

build-sdks: $(foreach TARGET,$(ALL_TARGETS),build-$(TARGET))
	@:

test-sdks: $(foreach TARGET,$(ALL_TARGETS),test-$(TARGET))
	@:

check-templates: $(foreach TARGET,$(ALL_TARGETS),check-template-$(TARGET))
	@:

default: build-sdks
	@:

clean:
	rm -rf testSDKs/sdk-* || true
	rm -rf testSDKs/terraform-provider-* || true
	rm -rf testSDKs/mcp-* || true
	rm -rf testprojects/* || true
	rm -rf testusages/* || true

build-review-sdks:
# Ensures we have the latest deps otherwise each review build will attempt to
# download them again
	go mod tidy
	mise exec -- mprocs --config ./mprocs.review.yaml

build-review-sdks-without-mprocs:
	@source ./scripts/utils.sh && print_divider "Building Review SDKs"
	@./scripts/build-review-mcp.sh mcp-typescript &
	@./scripts/build-review-sdk.sh cli --skip-compile &
	@./scripts/build-review-sdk.sh csharp --skip-compile &
	@./scripts/build-review-sdk.sh go &
	@./scripts/build-review-sdk.sh javav2 --skip-compile &
	@./scripts/build-review-sdk.sh php --skip-compile &
	@./scripts/build-review-sdk.sh postman --skip-compile &
	@./scripts/build-review-sdk.sh pythonv2 &
	@./scripts/build-review-sdk.sh ruby --skip-compile &
	@./scripts/build-review-sdk.sh typescriptv2 &
	@./scripts/build-review-sdk.sh unity --skip-compile &
	@./scripts/build-review-terraform.sh &
	wait

test-cleanup:
	@find ./testSDKs -type f -name 'test-*-record.txt' -delete

test: test-generator test-validate test-sdks test-readme
	@:

LOGFILE = /tmp/openapi-generation-$(shell date --iso=seconds)

testp: test-boot
	$(MAKE) test-validate
	$(MAKE) -j test-generator test-cli test-csharp test-go test-javav2 test-mcp-typescript test-php test-pythonv2 test-ruby test-terraform test-typescriptv2 test-unity 2> >(tee $(LOGFILE))
	test $? -ne 0 && echo "build errors:" && cat $(LOGFILE)

test-validate:
	@go run cmd/validate/main.go -s ./tests/specs/uber.yaml
	@go run cmd/validate/main.go -s ./tests/specs/ecommerce.yaml

test-validation: test-validate
	@go test -v -json ./internal/validation 2>&1 | go tool gotestfmt -hide all

test-generator:
	@./scripts/test-generator.sh

test-readme:
	@source ./scripts/utils.sh && load_local_env && ./scripts/test-readme-sections.sh

test-review-terraform:
	./scripts/test-review-terraform.sh

./bin/validate: build-validate
	@:

./bin/generate: build-generate
	@:

build-generate:
	@mkdir -p ./bin
	@go build -o ./bin/generate ./cmd/generate/main.go

build-validate:
	@mkdir -p ./bin
	@go build -o ./bin/validate ./cmd/validate/main.go

build-api-test-service:
	@echo -e "INFO\tCompiling api-test-service..."
	@go build -C services/speakeasy-api-test-service -o ../../bin/api-test-service ./cmd/server/main.go

build-json-schema-%:
	@go run ./cmd/jsonschema/main.go -o ./jsonschemas -l $*

build: build-generate build-validate

wait-services-boot:
	@bash ./scripts/wait-for-services.sh

boot-services: build-api-test-service
	@bash ./scripts/boot-services.sh

test-boot: boot-services wait-services-boot
	@:

stop-services:
	@bash ./scripts/stop-services.sh

restart-services:
	@bash ./scripts/restart-services.sh

changelog:
	@go run cmd/changelog/main.go $(filter-out $@,$(MAKECMDGOALS))

apply-changesets:
	@go run cmd/changeset/apply/main.go

permissions:
	@./scripts/update-permissions.sh

features:
	@go generate ./internal/features/...

tracing:
	docker run --rm --name jaeger \
	-p 16686:16686 \
	-p 4317:4317 \
	-p 4318:4318 \
	-p 5778:5778 \
	-p 9411:9411 \
	jaegertracing/jaeger:2.3.0

%:
ifeq ($(words $(MAKECMDGOALS)),1)
	@echo invalid action: $@
else
	@:
endif

.PHONY: lint
lint:
	./scripts/lint.sh

.PHONY: tidy
tidy:
	@bash ./scripts/tidy-modules.sh
