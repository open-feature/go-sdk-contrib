ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
MODULE_TYPE ?= providers
FLAGD_TESTBED = flagd-testbed
FLAGD_SYNC = sync-testbed
GOLANGCI_LINT_VERSION := v2.13.2
GOBIN := $(or $(shell go env GOBIN),$(shell go env GOPATH | cut -d: -f1)/bin)

# A provider's adoption of the OpenFeature Provider Conformance Suite is a
# module of its own, a sibling of the provider's e2e suite rather than a package
# inside it: providers/<name>/tck. `e2e` and `tck` below are the two halves of
# one partition of the module list, and the directory is the whole of the
# selector -- a suite runs in the conformance step because of where its files
# live, not because of what its tests are called.
#
# tools/tck is the harness, not an adoption. Its own tests are unit tests, they
# need no Docker, and `make test` and `make e2e` run them; it stays on the e2e
# side of the partition.
#
# The adoptions also carry `//go:build tck`, which is a different job from the
# one above and not a second way of doing it. The module path decides which of
# the two targets runs a suite; the tag keeps the suite out of every invocation
# that asks for no tags at all -- `make test`, and a bare `go test ./...` typed
# in the module. Neither substitutes for the other.
TCK_GO_MOD_DIRS := $(filter-out ./tools/%,$(filter %/tck,$(ALL_GO_MOD_DIRS)))
E2E_GO_MOD_DIRS := $(filter-out $(TCK_GO_MOD_DIRS),$(ALL_GO_MOD_DIRS))
TCK_TIMEOUT ?= 20m

workspace-init:
	go work init
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module) &&) true

workspace-update:
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module) &&) true

test:
	go list -f '{{.Dir}}/...' -m | xargs -I{} go test -v {}

# call with TESTCONTAINERS_RYUK_DISABLED="true" to avoid problems with podman on Macs
#
# The second command is the conformance modules, compiled and not run, and it is
# what makes the `tck` build tag safe. Appendix F asks for a suite that keeps
# compiling against tools/tck even when it does not execute, so that a signature
# change in the harness cannot rot an adoption unnoticed -- and a build tag only
# breaks that if nothing in the pipeline builds with the tag. This does: -tags=tck
# builds the test binary, and the empty -run pattern matches no test in it.
e2e:
	go clean -testcache
	status=0; for dir in $(E2E_GO_MOD_DIRS); do go test -timeout=3m -tags=e2e $$dir/... || status=1; done; exit $$status
	status=0; for dir in $(TCK_GO_MOD_DIRS); do go test -timeout=3m -tags=tck -run '^$$' $$dir/... || status=1; done; exit $$status

# The OpenFeature Provider Conformance Suite, and nothing else. Needs Docker and
# takes minutes, so it is not part of `e2e` and is not run on every pull request;
# a maintainer runs it deliberately. It is a step of its own so that a red result
# says conformance failed rather than that a test failed -- the two mean
# different things, because a conformance suite fails a scenario by design
# wherever a known deviation is declared.
tck:
	status=0; for dir in $(TCK_GO_MOD_DIRS); do go test -count=1 -timeout=$(TCK_TIMEOUT) -tags=tck $$dir/... || status=1; done; exit $$status

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
