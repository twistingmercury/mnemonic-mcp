---
entity_name: CLI Dockerfile Pattern
entity_type: devops-pattern
language: docker
domain: devops
description: Multi-stage Dockerfile pattern for CLI tools using Alpine and scratch base images with minimal attack surface and embedded version information
tags:
  - Docker
  - CLI
  - multi-stage
  - Alpine
  - security
---

# CLI Dockerfile Pattern

## Philosophy

Build CLI tools for multiple platforms (Linux, macOS, Windows) using Docker cross-compilation. Generate static binaries with CGO_ENABLED=0. Embed version information for traceability.

## Cross-Compilation Dockerfile

```dockerfile
# Multi-stage Dockerfile for building CLI tools
# Builds for all platforms inside Docker for portability

FROM golang:1.25-alpine AS base
RUN apk add --no-cache git ca-certificates
WORKDIR /app
COPY . .

FROM base AS builder-all
ARG VERSION
ARG COMMIT
ARG BUILD_DATE
ENV VERSION_PATH="github.com/org/project/cmd/version"

# Validate required build arguments
RUN if [ -z "$VERSION" ]; then \
        echo "ERROR: VERSION build argument is required"; \
        exit 1; \
    fi

RUN if [ -z "$COMMIT" ]; then \
        echo "ERROR: COMMIT build argument is required"; \
        exit 1; \
    fi

RUN if [ -z "$BUILD_DATE" ]; then \
        echo "ERROR: BUILD_DATE build argument is required"; \
        exit 1; \
    fi

# Create build directory
RUN mkdir -p build

RUN go get -u ./... && go mod tidy

# Build for all platforms
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags "-X ${VERSION_PATH}.ApiVersion=${VERSION} \
              -X ${VERSION_PATH}.GitCommit=${COMMIT} \
              -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}" \
    -o build/app-linux-amd64 ./cmd/main

RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags "-X ${VERSION_PATH}.ApiVersion=${VERSION} \
              -X ${VERSION_PATH}.GitCommit=${COMMIT} \
              -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}" \
    -o build/app-linux-arm64 ./cmd/main

RUN CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
    -ldflags "-X ${VERSION_PATH}.ApiVersion=${VERSION} \
              -X ${VERSION_PATH}.GitCommit=${COMMIT} \
              -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}" \
    -o build/app-darwin-amd64 ./cmd/main

RUN CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
    -ldflags "-X ${VERSION_PATH}.ApiVersion=${VERSION} \
              -X ${VERSION_PATH}.GitCommit=${COMMIT} \
              -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}" \
    -o build/app-darwin-arm64 ./cmd/main

RUN CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
    -ldflags "-X ${VERSION_PATH}.ApiVersion=${VERSION} \
              -X ${VERSION_PATH}.GitCommit=${COMMIT} \
              -X ${VERSION_PATH}.BuildDate=${BUILD_DATE}" \
    -o build/app-windows-amd64.exe ./cmd/main
```

## Supported Platforms

- **Linux AMD64**: Standard x86_64 Linux servers and desktops
- **Linux ARM64**: ARM-based Linux (Raspberry Pi, ARM servers)
- **macOS AMD64**: Intel-based Macs
- **macOS ARM64**: Apple Silicon Macs (M1, M2, M3)
- **Windows AMD64**: Standard x86_64 Windows

## Build Script Pattern

```bash
#!/usr/bin/env bash

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BINARY_NAME="appname"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
BIN_DIR="${PROJECT_ROOT}/.bin"
DOCKERFILE_PATH="$SCRIPT_DIR/Dockerfile"

# Require environment variables
if [ -z "$VERSION" ] || [ -z "$COMMIT" ] || [ -z "$BUILD_DATE" ]; then
    echo "ERROR: VERSION, COMMIT, and BUILD_DATE env vars must be set"
    exit 1
fi

echo "Building ALL Platforms"
echo "Version: $VERSION"

# Clean previous builds
rm -rf "${BIN_DIR}"
mkdir -p "${BIN_DIR}"

# Build using Docker
IMAGE_TAG="${BINARY_NAME}-builder-all:$VERSION"

docker build \
    --target builder-all \
    --build-arg VERSION="$VERSION" \
    --build-arg COMMIT="$COMMIT" \
    --build-arg BUILD_DATE="$BUILD_DATE" \
    --progress=plain \
    -f "$DOCKERFILE_PATH" \
    -t "${IMAGE_TAG}" \
    "$PROJECT_ROOT"

# Extract all binaries from container
docker run --rm \
    -v "$BIN_DIR:/output" "${IMAGE_TAG}" \
    sh -c "cp /app/build/* /output/"

# Organize into platform directories
mkdir -p "$BIN_DIR/linux/amd64"
mkdir -p "$BIN_DIR/linux/arm64"
mkdir -p "$BIN_DIR/darwin/amd64"
mkdir -p "$BIN_DIR/darwin/arm64"
mkdir -p "$BIN_DIR/windows/amd64"

cp "$BIN_DIR/$BINARY_NAME-linux-amd64" "$BIN_DIR/linux/amd64/$BINARY_NAME"
cp "$BIN_DIR/$BINARY_NAME-linux-arm64" "$BIN_DIR/linux/arm64/$BINARY_NAME"
cp "$BIN_DIR/$BINARY_NAME-darwin-amd64" "$BIN_DIR/darwin/amd64/$BINARY_NAME"
cp "$BIN_DIR/$BINARY_NAME-darwin-arm64" "$BIN_DIR/darwin/arm64/$BINARY_NAME"
cp "$BIN_DIR/$BINARY_NAME-windows-amd64.exe" "$BIN_DIR/windows/amd64/$BINARY_NAME.exe"

echo "Build completed for ALL platforms!"
```

## Output Structure

```text
.bin/
├── linux/amd64/appname
├── linux/arm64/appname
├── darwin/amd64/appname
├── darwin/arm64/appname
└── windows/amd64/appname.exe
```

## Key Practices

- **CGO_ENABLED=0**: Static binaries, no C dependencies
- **All platforms built in single Docker stage**: Consistency across builds
- **Version injection via ldflags**: Traceability in deployed binaries
- **Argument validation at build time**: Fail fast if required args missing
- **Organized output structure**: Easy to package for distribution
- **Docker-based builds**: Reproducible across developer machines

## Distribution

After building, binaries can be:

- Uploaded to GitHub Releases
- Packaged in platform-specific installers
- Distributed via package managers (Homebrew, Chocolatey, apt)
- Uploaded to Azure Blob Storage or S3
