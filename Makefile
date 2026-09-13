ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
MODULE_TYPE ?= providers
FLAGD_TESTBED = flagd-testbed
FLAGD_SYNC = sync-testbed
GOLANGCI_LINT_VERSION := v2.13.2
GOBIN := $(or $(shell go env GOBIN),$(shell go env GOPATH | cut -d: -f1)/bin)

# The OpenFeature Provider Conformance Suite is selected by test name, and `e2e`
# and `tck` below are the two halves of that one filter: `e2e` skips every test
# whose name matches, `tck` runs those and nothing else. Every conformance suite
# in this repository must therefore be named so that this pattern selects it;
# each adoption has a test that fails if one is not.
#
# Both targets still compile every module under -tags=e2e, which is the reason
# the split is a test-name filter and not a second build tag: a tag would take
# the adoption out of the build, and CI would stop typechecking it against
# tools/tck. See tools/tck/README.md, "What a default build runs, and what it
# does not".
TCK_FILTER := Conformance
TCK_TIMEOUT ?= 20m

workspace-init:
	go work init
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module) &&) true

workspace-update:
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module) &&) true

test:
	go list -f '{{.Dir}}/...' -m | xargs -I{} go test -v {}

# call with TESTCONTAINERS_RYUK_DISABLED="true" to avoid problems with podman on Macs
e2e:
	go clean -testcache && go list -f '{{.Dir}}/...' -m | xargs -I{} go test -timeout=3m -tags=e2e -skip '$(TCK_FILTER)' {}

# The OpenFeature Provider Conformance Suite, and nothing else. Needs Docker and
# takes minutes, so it is not part of `e2e` and is not run on every pull request;
# a maintainer runs it deliberately. It is a step of its own so that a red result
# says conformance failed rather than that a test failed -- the two mean
# different things, because a conformance suite fails a scenario by design
# wherever a known deviation is declared.
tck:
	go list -f '{{.Dir}}/...' -m | xargs -I{} go test -count=1 -timeout=$(TCK_TIMEOUT) -tags=e2e -run '$(TCK_FILTER)' {}

lint:
	go install -v github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(foreach module, $(ALL_GO_MOD_DIRS), ${GOBIN}/golangci-lint run $(module)/...;)

vulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	$(foreach module, $(ALL_GO_MOD_DIRS), cd $(module) && ${GOBIN}/govulncheck ./... && cd - > /dev/null;)

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
