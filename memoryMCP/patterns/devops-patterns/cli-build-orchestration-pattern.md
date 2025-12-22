---
entity_name: CLI Build Orchestration Pattern
entity_type: devops-pattern
language: shell
domain: devops
description: Build script pattern that orchestrates Docker multi-stage builds, running E2E tests before exporting binaries
tags:
  - Docker
  - CLI
  - E2E
  - build-script
  - CI
  - multi-stage
---

# CLI Build Orchestration Pattern

## Philosophy

The build script orchestrates Docker build targets in sequence: first run E2E tests inside Docker using the linux binary, then export all platform binaries only if tests pass. This ensures broken code never gets exported.

## The Pattern

1. Run `docker build --target e2e_tests` to build and test
2. If tests pass (exit code 0), run `docker build --target export` to export binaries
3. If tests fail, stop immediately - don't export broken binaries

## Dockerfile Structure

Multi-stage Dockerfile with three key stages:

- `builder` stage: builds all platform binaries (darwin, linux, windows)
- `e2e_tests` stage: copies linux binary, runs E2E tests with `CLI_BINARY_PATH` env var
- `export` stage: scratch image that copies binaries from builder

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Install git for version info
RUN apk add --no-cache git

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build arguments for version embedding
ARG VERSION
ARG COMMIT
ARG BUILD_DATE
ENV VERSION_PATH="github.com/org/project/cmd/version"

# Create output directory structure
RUN mkdir -p /out/amd64/linux /out/amd64/darwin /out/amd64/windows \
             /out/arm64/linux /out/arm64/darwin

# Build for all platforms
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X ${VERSION_PATH}.Version=${VERSION} \
              -X ${VERSION_PATH}.Commit=${COMMIT} \
              -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}" \
    -o /out/amd64/linux/cli-tool ./cmd/main

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags "-X ${VERSION_PATH}.Version=${VERSION} \
              -X ${VERSION_PATH}.Commit=${COMMIT} \
              -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}" \
    -o /out/arm64/linux/cli-tool ./cmd/main

RUN CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -ldflags "-X ${VERSION_PATH}.Version=${VERSION} \
              -X ${VERSION_PATH}.Commit=${COMMIT} \
              -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}" \
    -o /out/amd64/darwin/cli-tool ./cmd/main

RUN CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
    -ldflags "-X ${VERSION_PATH}.Version=${VERSION} \
              -X ${VERSION_PATH}.Commit=${COMMIT} \
              -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}" \
    -o /out/arm64/darwin/cli-tool ./cmd/main

RUN CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -ldflags "-X ${VERSION_PATH}.Version=${VERSION} \
              -X ${VERSION_PATH}.Commit=${COMMIT} \
              -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}" \
    -o /out/amd64/windows/cli-tool.exe ./cmd/main

FROM golang:1.25-alpine AS e2e_tests
WORKDIR /e2e

# Copy E2E test module files and download dependencies
COPY tests/e2e/go.mod tests/e2e/go.sum ./
RUN go mod download

# Copy E2E test source files
COPY tests/e2e/*.go ./

# Copy the linux binary from builder
COPY --from=builder /out/amd64/linux/cli-tool /usr/local/bin/cli-tool

# Run E2E tests with binary path environment variable
RUN CLI_BINARY_PATH=/usr/local/bin/cli-tool go test -v ./...

FROM scratch AS export
COPY --from=builder /out/ .
```

## Build Script

The orchestration script extracts version metadata, runs tests first, and only exports binaries if tests pass.

**build/build.sh**:

```bash
#!/usr/bin/env bash

set -e

# Script location and project paths
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BIN_DIR="${PROJECT_ROOT}/.bin"
DOCKERFILE_PATH="${SCRIPT_DIR}/Dockerfile"

# Logging functions
log_info() {
    printf "INFO: %s\n" "$*"
}

log_success() {
    printf "SUCCESS: %s\n" "$*"
}

log_error() {
    printf "ERROR: %s\n" "$*" >&2
}

log_section() {
    printf "\n"
    printf "==========================================\n"
    printf "%s\n" "$*"
    printf "==========================================\n"
}

# Extract version metadata from git
VERSION="${VERSION:-$(git describe --tags --abbrev=0 2>/dev/null || echo 'v0.0.0')}"
COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

log_section "CLI Build Pipeline"
log_info "Version: ${VERSION}"
log_info "Commit: ${COMMIT}"
log_info "Build Date: ${BUILD_DATE}"

# Clean previous builds
rm -rf "${BIN_DIR}"
mkdir -p "${BIN_DIR}"

# Phase 1: Run E2E tests
log_section "Running E2E Tests"
log_info "Building and testing with Docker..."

if ! docker build \
    --target e2e_tests \
    --build-arg VERSION="${VERSION}" \
    --build-arg COMMIT="${COMMIT}" \
    --build-arg BUILD_DATE="${BUILD_DATE}" \
    --progress=plain \
    -f "${DOCKERFILE_PATH}" \
    "${PROJECT_ROOT}"; then
    log_error "E2E tests failed - aborting build"
    exit 1
fi

log_success "E2E tests passed"

# Phase 2: Export binaries (only reached if tests pass)
log_section "Exporting Binaries"
log_info "Building and exporting all platform binaries..."

docker build \
    --target export \
    --build-arg VERSION="${VERSION}" \
    --build-arg COMMIT="${COMMIT}" \
    --build-arg BUILD_DATE="${BUILD_DATE}" \
    --progress=plain \
    --output "${BIN_DIR}" \
    -f "${DOCKERFILE_PATH}" \
    "${PROJECT_ROOT}"

log_success "Binaries exported to ${BIN_DIR}"

# Summary
log_section "Build Complete"
log_info "Output directory: ${BIN_DIR}"
log_info "Platforms built:"
log_info "  - linux/amd64"
log_info "  - linux/arm64"
log_info "  - darwin/amd64"
log_info "  - darwin/arm64"
log_info "  - windows/amd64"
```

## Why This Works

- **E2E tests run in Docker against actual linux binary**: Tests validate the real compiled artifact, not just the source code
- **Tests use same environment as CI (containerized)**: No discrepancies between local and CI test runs
- **Build fails fast if tests fail - no broken exports**: The `--target export` phase is never reached if tests fail
- **CI config becomes trivial**: Just run the build script

## Integration with CI

CI platforms (GitHub Actions, GitLab CI, Azure Pipelines) simply call the build script. All complexity lives in the script and Dockerfile.

### GitHub Actions Example

```yaml
name: Build

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for git describe

      - name: Build and Test
        run: ./build/build.sh

      - name: Upload Artifacts
        uses: actions/upload-artifact@v4
        with:
          name: binaries
          path: .bin/
```

### GitLab CI Example

```yaml
build:
  stage: build
  image: docker:24
  services:
    - docker:24-dind
  script:
    - ./build/build.sh
  artifacts:
    paths:
      - .bin/
```

### Azure Pipelines Example

```yaml
trigger:
  - main

pool:
  vmImage: ubuntu-latest

steps:
  - checkout: self
    fetchDepth: 0

  - script: ./build/build.sh
    displayName: Build and Test

  - publish: .bin/
    artifact: binaries
```

## Output Structure

```text
.bin/
├── amd64/
│   ├── darwin/
│   │   └── cli-tool
│   ├── linux/
│   │   └── cli-tool
│   └── windows/
│       └── cli-tool.exe
└── arm64/
    ├── darwin/
    │   └── cli-tool
    └── linux/
        └── cli-tool
```

## Key Practices

- **Test before export**: E2E tests gate the binary export
- **Version from git**: Use `git describe --tags` for semantic versioning
- **Single Dockerfile**: All stages in one file for clarity
- **Scratch export stage**: Minimal layer, just the binaries
- **Progress output**: `--progress=plain` for readable CI logs
- **Fail fast**: `set -e` exits on first error
