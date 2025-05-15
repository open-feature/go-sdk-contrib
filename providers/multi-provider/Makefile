.PHONY: generate test lint
GOPATH_LOC = ${GOPATH}

OF_SDK_DIR := $(shell go list -f '{{.Dir}}' github.com/open-feature/go-sdk/openfeature)

generate:
	go generate ./...

lint:
	go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5
	${GOPATH}/bin/golangci-lint run ./...

test:
	go test ./...