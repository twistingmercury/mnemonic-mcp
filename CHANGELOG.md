# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Tracing for the `/mcp` endpoint. Each request gets an otelhttp server span (continuing any incoming W3C trace context) and each tool call a `tools/call <tool>` span beneath it, with child spans for PostgreSQL queries (otelpgx), Neo4j operations and OpenAI embedding calls (otelhttp). Retried embedding calls add an `openai.embed.retry` span event.
- `mnemonic.mcp.tool_calls` (by `tool` and `outcome`: ok, invalid_input, not_found, unavailable) and `mnemonic.mcp.tool_duration`, plus the standard `http.server.*` metrics for `/mcp`.
- PostgreSQL client metrics from otelpgx: `db.client.operation.duration` per operation and `pgxpool.*` pool statistics.
- `mnemonic.db.query_latency` and `mnemonic.db.query_errors` are now recorded for every Neo4j operation, and `mnemonic.patterns.query_latency` and `mnemonic.patterns.returned` for every vector search. Previously these instruments were created but never recorded.
- Tool-call log lines carry `trace_id` and `span_id`.

### Changed

- Updated Go modules and Docker build images to Go 1.27.2.
- Upgraded `golang.org/x/net` to v0.60.0 to clear five govulncheck findings in v0.59.0.
- Custom latency histograms record fractional milliseconds. Previously durations were truncated to whole milliseconds, so sub-millisecond calls recorded 0.

### Removed

- The unused connection pool gauges, replaced by the otelpgx pool statistics.

## [0.3.2] - 2026-09-09

### Changed

- Updated Go modules and Docker build images to Go 1.27.1 and refreshed dependencies.
- Disabled test-file analysis in golangci-lint.

### Fixed

- Moved test-only constructors into `export_test.go` files to avoid unused-function
  lint errors when test analysis is disabled.
- Kept pgx at v5.10.0 for compatibility with pgxmock v4.9.0, restoring unit-test
  compilation during `make build`.

## [0.3.1]

### Changed

- Updated build to use latest go version 1.26.6
- Updated packages as needed
- Updated docker build to use latest golang-tooling image

## [0.3.0]

### Added

- GitHub Actions CI workflow for Mnemonic service (`mnemonic-ci.yaml`)
- GitHub Actions CD workflow for Mnemonic service (`mnemonic-cd.yaml`)
- Docker image artifact passing between CI and CD workflows
- E2E test execution during CI builds
- Cleanup trap in build script for docker compose
- Configuration package implementation (`internal/config`)
- Layered configuration loading (defaults → file → environment variables)
- Comprehensive validation with clear error messages for all configuration sections
- Server integration with configurable timeouts, TLS, and graceful shutdown
- Telemetry package (`internal/telemetry`) with otelx integration for unified OpenTelemetry initialization
- Middleware package (`internal/middleware`) with tracing and request metrics middleware for Gin
- Metrics package (`internal/metrics`) with domain-specific metrics for routing, patterns, and database operations
- Distributed tracing support via otelgin middleware
- Structured logging with trace correlation via otelx
- Request metrics: count, duration histograms, in-flight counters
- Version package (`internal/version`) for build metadata, eliminating upward dependencies
- Agent repository package (`internal/repository`) with PostgreSQL implementation
- Agent schema migrations (002_create_agents.up.sql and 002_create_agents.down.sql)
- Repository error types: ErrAgentExists, ErrAgentNotFound, ErrAgentInUse
- List options for pagination support in repository queries
- Comprehensive unit tests for agent repository with pgxmock
- RegexMatcher implementation with compiled pattern caching and sliding TTL
- Background cleanup goroutine for expired regex patterns
- `Close()` method added to `RuleMatcher` interface for resource cleanup
- `CloseAll()` method added to `MatcherRegistry` for graceful matcher shutdown
- `PatternIDs []uuid.UUID` field on `SimilarityOptions` for filtering similarity search to specific pattern IDs
- Unit and integration tests for PatternIDs filter on `FindSimilar`
- Startup timeout for rule cache (`routing.cache.startup_timeout`) bounds
  initial rule load to prevent indefinite hangs (default 30s)

### Changed

- Build script now uses identical behavior for local and CI execution
- Push logic moved from build script to CD workflow
- Server now initializes telemetry and registers observability middleware
- Configuration validation includes log level validation (fail-fast on invalid level)
- Moved MetricsRegistry from server package to telemetry package for proper dependency direction
- Refactored cmd/version to delegate to internal/version, fixing internal->cmd dependency violation
- `NormalizePrompt` now trims only (no longer lowercases); matchers own case-folding for their specific needs
- Routing engine design doc updated: Pattern Matcher section corrected from in-Go cosine similarity to pgvector-delegated approach via `<=>` operator
- `data-storage.md` updated: `SimilarityOptions` struct includes `PatternIDs` field

### Fixed

- Typo in Mnemonic E2E tests docker-compose.yaml (`menmonic_tests` → `mnemonic_tests`)
- `NormalizePrompt` changed from lowercase+trim to trim-only, fixing case-sensitive regex matching (regex flags now control case folding)
- Unknown regex flags now rejected with clear error message instead of being silently ignored
