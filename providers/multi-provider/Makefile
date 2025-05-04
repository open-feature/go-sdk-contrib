.PHONY: generate test lint
GOPATH_LOC = ${GOPATH}

generate:
	go generate ./...
	go mod download
	mockgen -source=${GOPATH}/pkg/mod/github.com/open-feature/go-sdk@v1.13.1/openfeature/provider.go -package=mocks -destination=./internal/mocks/provider_mock.go
	mockgen -source=${GOPATH}/pkg/mod/github.com/open-feature/go-sdk@v1.13.1/openfeature/hooks.go -package=mocks -destination=./internal/mocks/hook_mock.go

lint:
	go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.5
	${GOPATH}/bin/golangci-lint run ./...

test:
	go test ./...