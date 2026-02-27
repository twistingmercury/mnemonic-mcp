.PHONY: local build tests-unit tests-bench analyze help tests-db tests-db-agent tests-db-pattern tests-db-graph tests-e2e

GIT_COMMIT := $(shell git rev-parse --short=8 HEAD 2>/dev/null || echo "unknown")
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +%Y-%m-%d)
LD_FLAGS="-s -w \
	-X 'github.com/twistingmercury/mnemonic/cmd/version.version=${GIT_TAG}' \
	-X 'github.com/twistingmercury/mnemonic/cmd/version.buildDate=${BUILD_DATE}' \
	-X 'github.com/twistingmercury/mnemonic/cmd/version.commit=${GIT_COMMIT}'"

default: help

local:
	go build --ldflags ${LD_FLAGS} -o .bin/mnemonic cmd/main/main.go

build: ## Build Docker image locally using the full CI build script
	./build/build.sh 
	docker system prune -f

analyze: ## Run linters, formatters, security scanners, etc
	goimports -w .
	golangci-lint run
	govulncheck ./cmd/... ./internal/...
	gosec -quiet -exclude-dir=tests ./...

tests-db: tests-db-agent tests-db-pattern tests-db-graph ## Run all database integration tests

tests-db-agent: ## Run agent repository integration tests
	./internal/repository/tests/run-agent-integration-tests.sh

tests-db-pattern: ## Run pattern repository integration tests
	./internal/repository/tests/run-pattern-integration-tests.sh

tests-db-graph: ## Run graph repository integration tests
	./internal/repository/tests/run-graph-integration-tests.sh
	
tests-e2e: ## Runs the end-to-end tests
	docker compose -f "./tests/docker-compose-dev.yaml" up \
        --build \
        --abort-on-container-exit \
        --exit-code-from mnemonic_tests \
        mnemonic_api mnemonic_tests
		docker system prune -f

tests-unit: ## Run the unit tests
	go test ./internal/... -coverprofile=coverage.out
	go tool cover -html=coverage.out

tests-bench: ## Run benchmark tests
	go test ./internal/... -bench=. -benchmem -run=^$

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nAvailable targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
