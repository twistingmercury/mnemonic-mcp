.PHONY: local build unit-test analyze help

GIT_COMMIT := $(shell git rev-parse --short=8 HEAD 2>/dev/null || echo "unknown")
GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +%Y-%m-%d)
LD_FLAGS="-s -w \
	-X 'github.com/twistingmercury/mnemonic/cmd/version.version=${GIT_TAG}' \
	-X 'github.com/twistingmercury/mnemonic/cmd/version.buildDate=${BUILD_DATE}' \
	-X 'github.com/twistingmercury/mnemonic/cmd/version.commit=${GIT_COMMIT}'"

default: help

local:
	LOCAL_BUILD=true go build --ldflags ${LD_FLAGS} -o .bin/mnemonic cmd/main/main.go

build: ## Build Docker image locally using the full CI build script
	./build/build.sh 
	docker system prune -f

analyze: ## Run linters, formatters, security scanners, etc
	goimports -w .
	golangci-lint run
	govulncheck ./cmd/... ./internal/...
	gosec -quiet -exclude-dir=tests ./...

unit-test: ## Run the unit tests
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

image: ## Build the image locally
	docker build --rm \
        --file "build/Dockerfile" \
        --build-arg BUILD_VER="${GIT_COMMIT}-local" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg BUILD_COMMIT="${GIT_COMMIT}" \
        --target final \
        --tag "mnemonic:${GIT_COMMIT}-local" .

	docker run --rm mnemonic:${GIT_COMMIT}-local --version

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nAvailable targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
