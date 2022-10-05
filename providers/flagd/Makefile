.PHONY: mockgen
mockgen:
	mockgen -source=pkg/service/iservice.go -destination=pkg/service_mock_test.go -package=flagd_test
	mockgen -source=pkg/service/client.go -destination=pkg/service/client_mock_test.go -package=service_test
