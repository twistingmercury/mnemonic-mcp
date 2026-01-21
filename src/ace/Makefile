.PHONY: build unit-test analyze help

default: help

build: ## Build the ace binary
	go mod tidy
	go build -o ./bin/ace cmd/ace/main.go

analyze: ## Run linters, formatters, security scanners, etc
	goimports -w .
	golangci-lint run
	govulncheck ./cmd/... ./internal/...
	gosec -quiet -exclude-dir=tests ./...

unit-test: ## Run the unit tests
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nAvailable targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
