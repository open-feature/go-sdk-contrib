ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)

workspace-init:
	go work init
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module);)

workspace-update:
	$(foreach module, $(ALL_GO_MOD_DIRS), go work use $(module);)

test:
	$(foreach module, $(ALL_GO_MOD_DIRS), go test $(module)/...;)

lint:
	go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(foreach module, $(ALL_GO_MOD_DIRS), ${GOPATH}/bin/golangci-lint run --deadline=3m --timeout=3m $(module)/...;)

