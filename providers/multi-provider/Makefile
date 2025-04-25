.PHONY: generate test
GOPATH_LOC = ${GOPATH}

generate:
	go generate ./...
	go mod download
	mockgen -source=${GOPATH}/pkg/mod/github.com/open-feature/go-sdk@v1.13.1/openfeature/provider.go -package=mocks -destination=./internal/mocks/provider_mock.go

test:
	go test ./...