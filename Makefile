ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
MODULE_TYPE ?= providers
FLAGD_TESTBED = flagd-testbed
FLAGD_SYNC = sync-testbed
GOLANGCI_LINT_VERSION := v2.4.0
GOBIN := $(shell go env GOBIN)

workspace-init:
	go work init
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module);)

workspace-update:
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module);)

test:
	go list -f '{{.Dir}}/...' -m | xargs -I{} go test -v {}

# call with TESTCONTAINERS_RYUK_DISABLED="true" to avoid problems with podman on Macs
e2e:
	go clean -testcache && go list -f '{{.Dir}}/...' -m | xargs -I{} go test -timeout=3m -tags=e2e {}

lint:
	go install -v github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(foreach module, $(ALL_GO_MOD_DIRS), ${GOBIN}/golangci-lint run $(module)/...;)

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
