#------------------------------------------------------------
# Stage 1: Run linters, formatters, security scanners, 
# unit tests, then build binary
#------------------------------------------------------------
FROM ghcr.io/twistingmercury/golang-tooling:v1.0-go1.25-alpine AS build

ENV CGO_ENABLED=0 \
    GOARCH=amd64 \
    GOOS=linux

ARG BUILD_VER=dev
ARG BUILD_DATE=unknown
ARG BUILD_COMMIT=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Run linters, formatters, security scanners, etc
RUN goimports -w .
RUN golangci-lint run
RUN govulncheck ./cmd/... ./internal/...
RUN gosec -quiet -exclude-dir=tests ./...
RUN go test -v ./... -coverprofile=coverage.out

ARG LD_FLAGS="-s -w \
    -X 'github.com/twistingmercury/mnemonic/cmd/version.version=${BUILD_VER}' \
    -X 'github.com/twistingmercury/mnemonic/cmd/version.buildDate=${BUILD_DATE}' \
    -X 'github.com/twistingmercury/mnemonic/cmd/version.commit=${BUILD_COMMIT}'"

RUN go build --ldflags "${LD_FLAGS}" -o /out/mnemonic cmd/main/main.go


#------------------------------------------------------------
# Stage 2: Runtime - minimal production image
#------------------------------------------------------------
FROM scratch AS final

LABEL org.opencontainers.image.title="mnemonic" \
    org.opencontainers.image.description="Backend server providing routing and pattern retrieval via REST API for ACE CLI" \
    org.opencontainers.image.source="https://github.com/twistingmercury/ace/tree/develop/src/mnemonic" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.authors="Jeremy K. Johnson" \
    org.opencontainers.image.created="${BUILD_DATE}" \
    org.opencontainers.image.revision="${VCS_REF}" \
    org.opencontainers.image.version="${VERSION}"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /out/mnemonic /mnemonic

EXPOSE 8080

ENTRYPOINT ["/mnemonic"]