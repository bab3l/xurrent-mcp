.PHONY: all test test-integration test-unit lint build docker clean run run-http

GO ?= go
GOLANGCI_LINT ?= golangci-lint

all: lint test build

# Unit tests (no API calls)
test-unit:
	$(GO) test ./internal/... -count=1 -short

# MCP protocol harness tests (no API calls)
test-harness:
	$(GO) test -run TestHarness -count=1 -v ./cmd/xurrent-mcp/

# Live API tests (needs XURRENT_TOKEN, XURRENT_ACCOUNT)
test-live:
	$(GO) test -tags=integration -run TestLive -count=1 -v -timeout 5m ./cmd/xurrent-mcp/

# Fetch verification tests
test-fetch:
	$(GO) test -tags=integration -run TestFetch -count=1 -v -timeout 3m ./cmd/xurrent-mcp/

# Endpoint coverage tests
test-endpoint:
	$(GO) test -tags=integration -run TestEndpoint -count=1 -v -timeout 8m ./cmd/xurrent-mcp/

# All tests (unit + harness)
test: test-unit test-harness

# Full integration suite (needs credentials)
test-integration: test-unit test-harness test-fetch test-endpoint

# Lint
lint:
	$(GOLANGCI_LINT) run ./...

# Build binary
build:
	$(GO) build -trimpath -ldflags="-s -w" -o xurrent-mcp ./cmd/xurrent-mcp

# Docker build — run from parent directory containing both go-xurrent and xurrent-mcp
docker:
	docker build -t xurrent-mcp:latest -f xurrent-mcp/Dockerfile .

# Clean
clean:
	rm -f xurrent-mcp
	$(GO) clean -cache -testcache

# Run locally (stdio mode)
run:
	$(GO) run ./cmd/xurrent-mcp

# Run with HTTP transport
run-http:
	MCP_TRANSPORT=http PORT=8080 $(GO) run ./cmd/xurrent-mcp

# Install pre-commit hooks
hooks:
	git config core.hooksPath .githooks
