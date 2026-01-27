---
entity_name: Service Build Script Pattern
entity_type: devops-pattern
language: bash
domain: devops
description: Shell script pattern for building Go services with version embedding, compilation flags, and artifact generation for CI/CD pipelines
tags:
  - bash
  - build-script
  - Go
  - compilation
  - CI/CD
---

# Service Build Script Pattern

## Philosophy

Create a comprehensive build pipeline script that orchestrates all quality gates: tool installation, code analysis, unit testing, Docker build, E2E testing, and coverage reporting. Use shared utility libraries for consistent logging.

## Complete Build Script

**build/build.sh**:

```bash
#!/usr/bin/env bash

set -e

# Load utility libraries
BUILD_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJ_ROOT="$(cd "${BUILD_DIR}/.." && pwd)"

source "${PROJ_ROOT}/scripts/lib/print.sh"
source "${PROJ_ROOT}/scripts/lib/path.sh"

# Build configuration from environment or git
export BUILD_VERSION="${BUILD_VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo 'v0.0.0')}"
export BUILD_COMMIT="${BUILD_COMMIT:-$(git rev-parse --short HEAD)}"
export BUILD_DATE="${BUILD_DATE:-$(date +%Y-%m-%dT%H:%M:%S)}"
export SERVICE_NAME="${SERVICE_NAME:-service-name}"

# Verify prerequisites
print::info "verifying azure cli authentication"
if ! az account show >/dev/null 2>&1; then
    print::error "not logged into azure cli - run 'az login' first"
    exit 1
fi

install_golang_tools() {
    print::section "Installing Go Tools"

    print::info "installing golangci-lint@v2.4.0"
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.4.0

    print::info "installing govulncheck@v1.1.4"
    go install golang.org/x/vuln/cmd/govulncheck@v1.1.4

    print::info "installing gosec@latest"
    go install github.com/securego/gosec/v2/cmd/gosec@latest

    # Test reporting tools
    print::info "installing go-junit-report@v1.0.0"
    go install github.com/jstemmer/go-junit-report@v1.0.0

    print::info "installing gocov@v1.1.0"
    go install github.com/axw/gocov/gocov@v1.1.0

    print::info "installing gocov-xml@v1.0.0"
    go install github.com/AlekSi/gocov-xml@v1.0.0

    print::success "Go tools installed"
}

analyze_and_unit_test() {
    print::section "Code Analysis and Unit Testing"

    local gobin="$(go env GOPATH)/bin"

    print::info "updating dependencies"
    go get -u ./... && go mod tidy

    print::info "performing linting"
    "${gobin}/golangci-lint" run -c ./.golangci.yaml

    print::info "performing vulnerability checks"
    "${gobin}/govulncheck" -json ./... > govulncheck-report.json

    print::info "performing security scan"
    "${gobin}/gosec" -quiet -exclude-generated -exclude-dir=tests \
        -fmt json -out gosec-report.json ./...

    print::info "performing unit tests"
    go test -v -count=1 -coverprofile=coverage.out \
        -covermode=count "./cmd/..." "./internal/..." 2>&1 | tee test_output.log

    print::success "Code analysis and unit tests completed"
}

build_image() {
    print::section "Building Docker Image"

    local image_tag="${SERVICE_NAME}:${BUILD_VERSION}"
    print::info "building ${image_tag}"

    docker build \
        --build-arg BUILD_VER="${BUILD_VERSION}" \
        --build-arg BUILD_COMMIT="${BUILD_COMMIT}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --progress=plain \
        -f "${BUILD_DIR}/Dockerfile" \
        -t "${SERVICE_NAME}:${BUILD_VERSION}" \
        -t "${SERVICE_NAME}:latest" \
        "${PROJ_ROOT}"

    print::success "Docker image built: ${image_tag}"
}

run_e2e_tests() {
    print::section "Running E2E Tests"

    # Cleanup function for docker compose resources
    cleanup() {
        docker compose -f tests/docker-compose.yaml down --remove-orphans > /dev/null 2>&1 || true
    }
    trap cleanup EXIT

    if ! docker compose -f tests/docker-compose.yaml up \
        --build --exit-code-from app_tests; then
        print::error "end-to-end tests failed"
        return 1
    fi

    print::success "E2E tests passed"
}

create_coverage_reports() {
    print::section "Creating Coverage Reports"

    local gobin="$(go env GOPATH)/bin"

    print::info "converting coverage to xml format"
    "${gobin}/gocov" convert coverage.out | \
        "${gobin}/gocov-xml" > coverage.xml

    print::info "converting test output to junit xml"
    "${gobin}/go-junit-report" < test_output.log > junit.xml

    print::success "Coverage reports created"
}

prep_image_for_upload() {
    print::section "Preparing Image for ACR"

    if [ -z "${CONTAINER_REGISTRY:-}" ]; then
        print::warn "CONTAINER_REGISTRY not set, skipping ACR tagging"
        return 0
    fi

    print::info "tagging for ${CONTAINER_REGISTRY}"
    docker tag "${SERVICE_NAME}:${BUILD_VERSION}" \
        "${CONTAINER_REGISTRY}/${SERVICE_NAME}:${BUILD_VERSION}"
    docker tag "${SERVICE_NAME}:latest" \
        "${CONTAINER_REGISTRY}/${SERVICE_NAME}:latest"

    print::success "Images tagged for ACR"
}

main() {
    print::section "Service Build Pipeline"
    print::info "Service: ${SERVICE_NAME}"
    print::info "Version: ${BUILD_VERSION}"
    print::info "Commit: ${BUILD_COMMIT}"
    print::info "Build Date: ${BUILD_DATE}"

    install_golang_tools
    analyze_and_unit_test
    build_image
    run_e2e_tests
    create_coverage_reports
    prep_image_for_upload

    print::section "Build Complete"
    print::success "All steps completed successfully"
    print::info "Next steps:"
    print::info "  - Push to ACR: docker push ${CONTAINER_REGISTRY}/${SERVICE_NAME}:${BUILD_VERSION}"
    print::info "  - View coverage: open coverage.xml"
    print::info "  - View test results: open junit.xml"
}

main "$@"
```

## Utility Library: print.sh

**scripts/lib/print.sh**:

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

print::section() {
    printf "\n"
    printf "==========================================\n"
    printf "%s\n" "$*"
    printf "==========================================\n"
}
```

## Build Workflow Stages

### 1. Tool Installation

- golangci-lint: Comprehensive linting
- govulncheck: Vulnerability scanning
- gosec: Security scanning
- go-junit-report: JUnit XML generation
- gocov/gocov-xml: Cobertura coverage XML

### 2. Code Analysis

- Dependency updates: `go get -u ./... && go mod tidy`
- Linting: `golangci-lint run -c ./.golangci.yaml`
- Vulnerability scan: `govulncheck -json ./...`
- Security scan: `gosec -quiet -exclude-generated -fmt json ./...`

### 3. Unit Testing

- Run with coverage: `go test -coverprofile=coverage.out`
- Test all packages: `./cmd/...` and `./internal/...`
- Capture output: `tee test_output.log`
- Fail on first error: `set -e`

### 4. Docker Build

- Pass build arguments: VERSION, COMMIT, BUILD_DATE
- Tag with version and latest
- Progress output: `--progress=plain`

### 5. E2E Testing

- Run via Docker Compose: `docker-compose up --build`
- Propagate test failures: `--exit-code-from app_tests`
- Clean up after: `docker-compose down --remove-orphans`

### 6. Coverage Reporting

- Convert to Cobertura XML for Azure DevOps
- Convert test output to JUnit XML
- Both formats for CI/CD integration

### 7. ACR Preparation

- Tag images with full registry path
- Only if CONTAINER_REGISTRY is set
- Ready for `docker push`

## Environment Variables

```bash
# Required
BUILD_VERSION    # Version tag (from git or env)
BUILD_COMMIT     # Git commit hash
BUILD_DATE       # ISO 8601 timestamp
SERVICE_NAME     # Name of service

# Optional
CONTAINER_REGISTRY  # ACR URL (e.g., registry.azurecr.io)
```

## Quality Gates

The script fails fast (`set -e`) if any stage fails:

1. Linting errors → Build fails
2. Vulnerabilities found → Build fails
3. Security issues → Build fails
4. Unit tests fail → Build fails
5. Docker build fails → Build fails
6. E2E tests fail → Build fails

## Cleanup Trap Pattern

When using docker compose for E2E tests, always use a cleanup trap to ensure resources are removed even if tests fail or the script is interrupted:

```bash
run_e2e_tests() {
    # Define cleanup function
    cleanup() {
        docker compose -f tests/docker-compose.yaml down --remove-orphans > /dev/null 2>&1 || true
    }
    # Register cleanup to run on EXIT (covers success, failure, and interrupts)
    trap cleanup EXIT

    # Run tests - cleanup happens automatically after this
    if ! docker compose -f tests/docker-compose.yaml up \
        --build --exit-code-from app_tests; then
        print::error "end-to-end tests failed"
        return 1  # Cleanup still runs due to trap
    fi

    print::success "E2E tests passed"
    # Cleanup runs automatically on function exit
}
```

**Why This Pattern**:

- **Reliability**: Cleanup runs regardless of how the function exits
- **CI-friendly**: Prevents orphaned containers in CI environments
- **Interrupt handling**: Ctrl+C triggers EXIT, so cleanup still runs
- **Silent failures**: `|| true` prevents cleanup errors from masking test results
- **Quiet output**: `> /dev/null 2>&1` keeps logs focused on test output

**Common Trap Signals**:

| Signal | When | Use Case |
|--------|------|----------|
| `EXIT` | Function/script exit | General cleanup (most common) |
| `INT` | Ctrl+C | Interactive interrupt handling |
| `TERM` | kill command | Graceful termination |
| `ERR` | Command failure (with `set -e`) | Error-specific cleanup |

**Best Practice**: Use `trap cleanup EXIT` for docker compose cleanup. It covers all exit scenarios including success, failure, and signals.

## Key Patterns

- **Fail fast**: `set -e` exits on first error
- **Utility libraries**: Centralized logging functions
- **Consistent logging**: `print::*` functions for all output
- **Version from source**: Git tags, not manual versioning
- **Quality gates**: Lint → Security → Tests → Build
- **Coverage reporting**: JUnit and Cobertura for CI integration
- **Conditional ACR tagging**: Only if registry configured
- **Cleanup traps**: Use `trap cleanup EXIT` for docker compose teardown

## Usage

From project root:

```bash
# Use defaults from git
./build/build.sh

# Override version
BUILD_VERSION=v1.2.3 ./build/build.sh

# With ACR
CONTAINER_REGISTRY=myregistry.azurecr.io ./build/build.sh

# All custom
BUILD_VERSION=v1.2.3 \
BUILD_COMMIT=abc123 \
BUILD_DATE=2024-01-15T10:30:00 \
SERVICE_NAME=my-service \
CONTAINER_REGISTRY=myregistry.azurecr.io \
./build/build.sh
```

## Output Artifacts

- `govulncheck-report.json` - Vulnerability scan results
- `gosec-report.json` - Security scan results
- `coverage.out` - Go coverage profile
- `coverage.xml` - Cobertura XML for CI/CD
- `test_output.log` - Test execution log
- `junit.xml` - JUnit XML for CI/CD
- Docker images: `service-name:version` and `service-name:latest`
