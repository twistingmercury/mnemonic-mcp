# E2E Test Runner Dockerfile
# Builds and runs end-to-end tests against the Mnemonic API

FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy test source files
COPY *.go ./

# Build test binary
RUN CGO_ENABLED=0 go test -c -o /e2e-tests .

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Copy test binary
COPY --from=builder /e2e-tests /app/e2e-tests

# Set environment variables
ENV API_URL=http://api:8080

# Run tests
ENTRYPOINT ["/app/e2e-tests", "-test.v"]
