.PHONY: generate test
GOPATH_LOC = ${GOPATH}

generate:
	go generate ./...
	mockgen -source=${GOPATH}/github.com/open-feature/go-sdk@v1.14.1/openfeature/provider.go -package=mocks -destination=./internal/mocks/provider_mock.go

test:
	go test ./...