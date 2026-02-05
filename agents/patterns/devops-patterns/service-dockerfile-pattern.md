---
entity_name: Service Dockerfile Pattern
entity_type: devops-pattern
language: docker
domain: devops
description: Multi-stage Dockerfile pattern for Go services using Alpine for build and scratch for runtime with security best practices and version embedding
tags:
  - Docker
  - services
  - multi-stage
  - Alpine
  - Go
  - security
---

# Service Dockerfile Pattern

## Philosophy

Use multi-stage Dockerfiles with Alpine for build and scratch for runtime to minimize attack surface and image size. Embed version information via build arguments and ldflags. Follow security best practices.

## Multi-Stage Dockerfile (Alpine → Scratch)

```dockerfile
FROM golang:alpine AS build

ENV CGO_ENABLED=0

WORKDIR /src

ARG BUILD_DATE
ARG BUILD_VER
ARG BUILD_COMMIT
ARG VERSION_PKG="github.com/org/project/internal/version"

COPY . .

RUN apk update && \
    apk --no-cache add git && \
    apk --no-cache add ca-certificates

# Install Swagger for API documentation generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger docs (if REST API)
RUN go get -u ./... && go mod tidy
RUN $(go env GOPATH)/bin/swag init \
    -dir cmd/main,internal/api/handler,internal/types \
    -o internal/swagger

# Build with version information embedded
RUN go build \
    -ldflags "-s -w \
      -X '${VERSION_PKG}.ApiVersion=${BUILD_VER}' \
      -X '${VERSION_PKG}.BuildDate=${BUILD_DATE}' \
      -X '${VERSION_PKG}.GitCommit=${BUILD_COMMIT}'" \
    -o ./bin/servicename \
    ./cmd/main/main.go

#-------------------------------------------------------------
# final stage
#-------------------------------------------------------------
FROM scratch AS final

ARG BUILD_DATE
ARG BUILD_VER
ARG BUILD_COMMIT

# OCI labels for metadata
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.version="${BUILD_VER}" \
      org.opencontainers.image.vendor="Your Organization" \
      org.opencontainers.image.source="repository-url" \
      org.opencontainers.image.description="Service description" \
      com.company.git_commit="${BUILD_COMMIT}"

WORKDIR /app
COPY --from=build /src/bin/ /app/
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Environment variables (set at runtime)
ENV LOG_LEVEL="" \
    ENVIRONMENT="" \
    DATABASE_CONNECTION_STRING=""

EXPOSE 8080

ENTRYPOINT ["/app/servicename"]
```

## Key Practices

- **Use scratch for minimal attack surface**: No shell, no package manager, only your binary
- **Copy CA certificates**: Required for HTTPS calls to external services
- **Embed version info via ldflags**: `-X 'package.Variable=value'` pattern
- **Use OCI labels**: Standard metadata format for container images
- **Multi-stage to keep build tools out**: Only runtime artifacts in final image
- **CGO_ENABLED=0**: Static binaries that don't depend on C libraries
- **Build arguments**: VERSION, COMMIT, BUILD_DATE for traceability

## Version Package Pattern

Create `internal/version/version.go`:

```go
package version

var (
    ApiVersion = "dev"      // Set via ldflags
    GitCommit  = "unknown"  // Set via ldflags
    BuildDate  = "unknown"  // Set via ldflags
)
```

## Build Command

```bash
docker build \
    --build-arg BUILD_VER="v1.0.0" \
    --build-arg BUILD_COMMIT="abc123" \
    --build-arg BUILD_DATE="2024-01-15T10:30:00" \
    --progress=plain \
    -f build/Dockerfile \
    -t service-name:v1.0.0 \
    -t service-name:latest \
    .
```

## Security Considerations

- **Scratch base image**: No unnecessary packages or files
- **Non-root user**: Consider adding USER directive if binary supports it
- **CA certificates**: Copy for HTTPS but nothing else
- **No secrets in image**: Use runtime environment variables
- **Minimal exposed ports**: Only what service actually uses
- **OCI labels**: Full traceability without inspecting filesystem
