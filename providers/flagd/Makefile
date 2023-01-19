.PHONY: mockgen
mockgen:
	mockgen -source=pkg/service/iservice.go -destination=internal/mock/service_mock_test.go -package=mock
	mockgen -source=pkg/service/client.go -destination=pkg/service/client_mock_test.go -package=service_test
