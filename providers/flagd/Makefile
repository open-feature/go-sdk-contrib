.PHONY: mockgen e2e-test-build e2e-test-run

mockgen:
	mockgen -source=pkg/iservice.go -destination=internal/mock/service_mock.go -package=mock

e2e-test-build:
	cd providers/flagd/e2e && go build -tags=e2e

e2e-test-run:
	cd providers/flagd/e2e && go test -tags=e2e