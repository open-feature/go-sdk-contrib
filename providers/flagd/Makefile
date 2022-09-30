.PHONY: mockgen
mockgen:
	mockgen -source=pkg/service/iservice.go -destination=pkg/service_mock_test.go -package=flagd_test
	mockgen -source=pkg/service/http/client.go -destination=pkg/service/http/client_mock_test.go -package=http_service_test
