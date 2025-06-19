ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
MODULE_TYPE ?= providers
FLAGD_TESTBED = flagd-testbed
FLAGD_SYNC = sync-testbed
GOLANGCI_LINT_VERSION := v1.64.8

workspace-init:
	go work init
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module);)

workspace-update:
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module);)

# call with TESTCONTAINERS_RYUK_DISABLED="true" to avoid problems with podman on Macs
e2e:
	go clean -testcache && go list -f '{{.Dir}}/...' -m | xargs -I{} go test -tags=e2e {}

new-provider:
	mkdir ./providers/$(MODULE_NAME)
	cd ./providers/$(MODULE_NAME) && go mod init github.com/open-feature/go-sdk-contrib/providers/$(MODULE_NAME) && touch README.md
	$(MAKE) append-to-release-please MODULE_TYPE=providers MODULE_NAME=$(MODULE_NAME)

new-hook:
	mkdir ./hooks/$(MODULE_NAME)
	cd ./hooks/$(MODULE_NAME) && go mod init github.com/open-feature/go-sdk-contrib/hooks/$(MODULE_NAME) && touch README.md
	$(MAKE) append-to-release-please MODULE_TYPE=hooks MODULE_NAME=$(MODULE_NAME)

append-to-release-please:
	jq '.packages += {"${MODULE_TYPE}/${MODULE_NAME}": {"release-type":"go","package-name":"${MODULE_TYPE}/${MODULE_NAME}","bump-minor-pre-major":true,"bump-patch-for-minor-pre-major":true,"versioning":"default","extra-files": []}}' release-please-config.json > tmp.json
	mv tmp.json release-please-config.json

.PHONY: go-mod-tidy
go-mod-tidy: $(ALL_GO_MOD_DIRS:%=go-mod-tidy/%)
go-mod-tidy/%: DIR=$*
go-mod-tidy/%:
	@echo "go mod tidy in $(DIR)" \
		&& cd $(DIR) \
		&& go mod tidy

.PHONY: lint
lint: golangci-lint

.PHONY: golangci-lint golangci-lint-fix
golangci-lint-fix: ARGS=--fix
golangci-lint-fix: golangci-lint
golangci-lint: $(ALL_GO_MOD_DIRS:%=golangci-lint/%)
golangci-lint/%: DIR=$*
golangci-lint/%:
	@echo 'golangci-lint $(if $(ARGS),$(ARGS) ,)$(DIR)' \
		&& cd $(DIR) \
		&& go run github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION} run --allow-serial-runners $(ARGS)

.PHONY: test
test-verbose: ARGS=-v
test: $(ALL_GO_MOD_DIRS:%=test/%)
test/%: DIR=$*
test/%:
	@echo "go test $(ARGS) $(DIR)/..." \
		&& cd $(DIR) \
		&& go test $(ARGS) ./...
