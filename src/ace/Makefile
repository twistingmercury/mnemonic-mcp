.PHONY: build unit-test analyze help

default: help

build: ## Build the project
	go build ./...

analyze: ## Run linters, formatters, security scanners, etc
	goimports -w .
	golangci-lint run
	#govulncheck ./...
	gosec ./...

unit-test: ## Run the unit tests
	go test -v ./...

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nAvailable targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
