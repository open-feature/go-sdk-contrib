.PHONY: generate test lint
GOPATH_LOC = ${GOPATH}

OF_SDK_DIR := $(shell go list -f '{{.Dir}}' github.com/open-feature/go-sdk/openfeature)

generate:
	go generate ./...
	go mod download
	mockgen -source=${OF_SDK_DIR}/provider.go -write_source_comment=false -write_command_comment=false -package=mocks -destination=./internal/mocks/provider_mock.go
	mockgen -source=${OF_SDK_DIR}/hooks.go -write_source_comment=false -write_command_comment=false -package=mocks -destination=./internal/mocks/hook_mock.go

lint:
	go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5
	${GOPATH}/bin/golangci-lint run ./...

test:
	go test ./...