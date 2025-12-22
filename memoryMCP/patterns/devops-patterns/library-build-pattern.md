---
entity_name: Library Build Pattern
entity_type: devops-pattern
language: go
domain: devops
description: Containerized build pattern for Go libraries/packages with unit tests, build validation, and E2E tests on host
tags:
  - go
  - library
  - docker
  - containerized-build
  - CI/CD
  - github-actions
---

# Library Build Pattern

## Philosophy

Library builds differ fundamentally from service builds. A library has no deployable artifact - no binary, no container image pushed to a registry. Success means the code compiles and tests pass.

Key principles:

- **No binary output**: Libraries produce no executable - validation is the only goal
- **Success = tests pass + code compiles**: The build proves the library works
- **Containerized for local/CI parity**: Running builds in Docker ensures developers and CI see identical environments
- **E2E tests run on host**: Avoid Docker-in-Docker complexity by running E2E tests outside the container

The containerized build handles unit tests and compilation. E2E tests, which often require Docker Compose or external services, run on the host after the container exits successfully.

## Build Dockerfile

The Dockerfile creates a consistent build environment for unit tests and compilation.

**build/Dockerfile**:

```dockerfile
FROM golang:1.24-alpine

RUN apk add --no-cache git bash

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENTRYPOINT ["/workspace/build/build.sh"]
```

Key design decisions:

- **golang:1.24-alpine**: Minimal image with Go toolchain
- **git and bash**: Required for scripts and module operations
- **Layer caching**: `go.mod` and `go.sum` copied first so dependencies cache separately from source
- **Entrypoint**: Runs the build script directly

## Build Script

The build script orchestrates unit tests, compilation, and optional E2E tests. The `SKIP_E2E` environment variable controls whether E2E tests run.

**build/build.sh**:

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# shellcheck source=../scripts/print.sh
source "${PROJECT_DIR}/scripts/print.sh"

SKIP_E2E="${SKIP_E2E:-false}"

unit_test() {
    go mod tidy
    print::info "running unit tests..."
    if ! go test -v "${PROJECT_DIR}/..."; then
        print::error "unit tests failed"
        return 1
    fi
}

build() {
    print::info "building package..."
    if ! go build ./...; then
        print::error "build failed"
        return 1
    fi
}

e2e_test() {
    print::info "running end-to-end tests..."
    if ! "${PROJECT_DIR}/tests/e2e/test-runner.sh" run; then
        print::error "e2e tests failed"
        return 1
    fi
}

main() {
    unit_test && build || return 1

    if [ "${SKIP_E2E}" = "true" ]; then
        print::info "Skipping E2E tests (SKIP_E2E=true)"
    else
        e2e_test || return 1
    fi

    print::info "build completed successfully"
    return 0
}

main "$@"
```

Key design decisions:

- **SKIP_E2E environment variable**: Defaults to `false`, set to `true` when running in container
- **Modular functions**: `unit_test()`, `build()`, `e2e_test()` for clear separation
- **Fail fast**: `set -e` and explicit return codes ensure failures propagate
- **Shared print utilities**: Consistent logging across all scripts

## Docker Runner Script

This script builds and runs the containerized build. It always skips E2E tests since those run on the host.

**build/build-docker.sh**:

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# shellcheck source=../scripts/print.sh
source "${PROJECT_DIR}/scripts/print.sh"

IMAGE_NAME="heartbeat-build:latest"

print::info "Building Docker image: ${IMAGE_NAME}"
docker build -t "${IMAGE_NAME}" -f "${SCRIPT_DIR}/Dockerfile" "${PROJECT_DIR}"

print::info "Running containerized build (unit tests + build, skipping E2E)..."
docker run --rm -e SKIP_E2E=true "${IMAGE_NAME}"

print::info "Containerized build completed successfully"
```

Key design decisions:

- **SKIP_E2E=true**: Always passed to container - E2E tests run separately on host
- **--rm flag**: Removes container after execution for clean builds
- **Descriptive logging**: Clear indication of what runs in container vs host

## GitHub Actions Workflow

The CI workflow runs the containerized build first, then E2E tests on the host.

**.github/workflows/ci.yaml**:

```yaml
name: CI

on:
  workflow_dispatch:
  push:
    branches:
      - main
  pull_request:
    branches:
      - main

jobs:
  build:
    name: Build and Test
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Run containerized build
        run: ./build/build-docker.sh

      - name: Set up Go for E2E tests
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
          cache: true

      - name: Run E2E tests
        run: ./tests/e2e/test-runner.sh run
```

Key design decisions:

- **Containerized build first**: Unit tests and compilation run in Docker for consistency
- **Go setup for E2E**: Go installed on runner for E2E tests that may need to compile test binaries
- **E2E on host**: Avoids Docker-in-Docker complexity, allows Docker Compose in E2E tests
- **Caching enabled**: `cache: true` speeds up Go dependency downloads

## Makefile Targets

The Makefile provides convenient entry points for local development.

**Makefile**:

```makefile
.PHONY: build build-docker test e2e-up e2e-down e2e-test e2e-run e2e-logs e2e-clean help

default: help

help: ## Show this help
    @awk 'BEGIN {FS = ":.*##"; printf "\n\033[1mAvailable targets:\033[0m\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
    @echo ""

build: ## Run the full build process; unit tests, build, e2e tests
    ./build/build.sh

build-docker: ## Run unit tests and build inside Docker container
    ./build/build-docker.sh

test: ## Run unit tests with coverage report
    go clean -testcache
    go test . -v -coverprofile=coverage.out
    go tool cover -html=coverage.out

e2e-up: ## Start E2E test infrastructure
    @./tests/e2e/test-runner.sh up

e2e-down: ## Stop E2E test infrastructure
    @./tests/e2e/test-runner.sh down

e2e-test: ## Run E2E tests (requires e2e-up first)
    @./tests/e2e/test-runner.sh test

e2e-run: ## Full E2E cycle: start, test, cleanup
    @./tests/e2e/test-runner.sh run

e2e-logs: ## Show E2E service logs
    @./tests/e2e/test-runner.sh logs

e2e-clean: ## Force cleanup E2E Docker resources
    cd tests/e2e && docker compose down -v --remove-orphans 2>/dev/null || true
```

Note: Makefiles require tabs for recipe indentation. The spaces shown above are for documentation display only. Use tabs when copying to actual Makefile.

Key targets:

- **build**: Full local build with E2E tests
- **build-docker**: Containerized build (unit tests + compile only)
- **test**: Quick unit tests with coverage
- **e2e-***: Granular E2E test control

## Key Differences from Service Pattern

| Aspect | Library Pattern | Service Pattern |
|--------|-----------------|-----------------|
| **Output** | None (validation only) | Binary + Docker image |
| **Registry push** | No | Yes (to ACR/registry) |
| **Focus** | Tests pass, code compiles | Deployable artifact |
| **Docker usage** | Build environment only | Build + runtime image |
| **E2E location** | Host (avoids DinD) | Often in Docker Compose |
| **Version embedding** | Not applicable | ldflags with version/commit |
| **Artifacts** | None | Images, binaries, coverage reports |

## The SKIP_E2E Pattern

The `SKIP_E2E` pattern solves a common problem: E2E tests often need Docker Compose or external services that are difficult to run inside a container.

**How it works**:

1. Build script checks `SKIP_E2E` environment variable (defaults to `false`)
2. Docker runner always sets `SKIP_E2E=true` when running container
3. CI runs containerized build first, then E2E tests on host
4. Local `make build` runs everything including E2E

**Benefits**:

- Avoids Docker-in-Docker complexity
- E2E tests can use Docker Compose normally
- Containerized build remains fast and isolated
- Same scripts work locally and in CI

## Usage

### Local Development

```bash
# Full build with E2E tests
make build

# Containerized build only (unit tests + compile)
make build-docker

# Quick unit tests with coverage
make test

# E2E tests separately
make e2e-run
```

### CI/CD

The GitHub Actions workflow handles everything automatically:

1. Checkout code
2. Run containerized build (unit tests + compile)
3. Install Go on runner
4. Run E2E tests on host

### Direct Script Execution

```bash
# Full build (default, includes E2E)
./build/build.sh

# Skip E2E tests
SKIP_E2E=true ./build/build.sh

# Containerized build
./build/build-docker.sh
```

## Directory Structure

```text
project/
  build/
    Dockerfile          # Build environment
    build.sh            # Main build script
    build-docker.sh     # Docker runner
  scripts/
    print.sh            # Logging utilities
  tests/
    e2e/
      test-runner.sh    # E2E test orchestration
  .github/
    workflows/
      ci.yaml           # GitHub Actions workflow
  Makefile              # Developer entry points
```

## Quality Gates

The build fails fast if any step fails:

1. **go mod tidy** - Dependencies must be clean
2. **Unit tests** - All tests must pass
3. **Compilation** - Code must compile (`go build ./...`)
4. **E2E tests** - Integration tests must pass (when not skipped)

## Print Utilities

The build scripts use shared print utilities for consistent logging.

**scripts/print.sh**:

```bash
#!/usr/bin/env bash

print::info() {
    printf "INFO: %s\n" "$*"
}

print::success() {
    printf "SUCCESS: %s\n" "$*"
}

print::error() {
    printf "ERROR: %s\n" "$*" >&2
}

print::warn() {
    printf "WARN: %s\n" "$*"
}
```
