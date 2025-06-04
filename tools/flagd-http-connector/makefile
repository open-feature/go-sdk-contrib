test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -tags=integration ./pkg/... -timeout 5m
test: ## Run unit tests
	@echo "Running unit tests..."
	@go test ./pkg/... -timeout 1m -race