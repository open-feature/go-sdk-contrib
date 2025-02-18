ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)
MODULE_TYPE ?= providers
FLAGD_TESTBED = flagd-testbed
FLAGD_SYNC = sync-testbed

workspace-init:
	go work init
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module);)

workspace-update:
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module);)

test:
	go list -f '{{.Dir}}/...' -m | xargs -I{} go test -v {}

e2e-start-helpers:
	docker run --name $(FLAGD_TESTBED) -d -p 8013:8013 ghcr.io/open-feature/flagd-testbed:v0.5.6
	docker run --name $(FLAGD_SYNC) -d -p 9090:9090 ghcr.io/open-feature/sync-testbed:v0.5.6

e2e-remove-helpers:
	docker stop $(FLAGD_TESTBED)
	docker stop $(FLAGD_SYNC)
	docker rm $(FLAGD_TESTBED)
	docker rm $(FLAGD_SYNC)

e2e:
	go clean -testcache && go list -f '{{.Dir}}/...' -m | xargs -I{} go test -tags=e2e {}

lint:
	go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5
	$(foreach module, $(ALL_GO_MOD_DIRS), ${GOPATH}/bin/golangci-lint run $(module)/...;)

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
