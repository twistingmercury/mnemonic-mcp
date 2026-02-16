# Code Review: Phase 3 - Observability

**Date**: 2026-01-28
**Reviewer**: Code Review Agent
**Status**: COMPLETE

## Files Reviewed

- `internal/telemetry/telemetry.go`
- `internal/middleware/tracing.go`
- `internal/middleware/metrics.go`
- `internal/metrics/registry.go`
- `internal/metrics/routing.go`
- `internal/metrics/patterns.go`
- `internal/metrics/database.go`
- `internal/server/server.go`

## Findings

### CRITICAL Priority

| Finding                                                                    | Resolution                                                 |
| -------------------------------------------------------------------------- | ---------------------------------------------------------- |
| Context may be cancelled before in-flight counter decrements in metrics.go | FIXED: Use context.Background() for decrement              |
| Nil pointer risk if tel.Logger() called after failed shutdown              | FIXED: Capture logger before shutdown attempt              |
| requestCount/requestDuration use potentially-cancelled context             | FIXED: Use context.Background() for post-request recording |

### HIGH Priority

| Finding                                                                   | Resolution                                                                |
| ------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| High-cardinality metric risk using raw URL path as fallback in metrics.go | FIXED: Changed to use constant "unknown"                                  |
| Metrics Registry not instantiated in server.go                            | FIXED: Created registry and wired into server                             |
| Status code cardinality with unknown routes                               | ACCEPTED: Status codes are bounded, route fallback already uses "unknown" |
| Upward dependency: telemetry imports cmd/version                          | FIXED: Created internal/version package                                   |
| Upward dependency: operations imports cmd/version                         | FIXED: Updated to use internal/version                                    |
| Missing error wrapping in routing.go metric creation                      | FIXED: Added fmt.Errorf wrapping                                          |
| Missing error wrapping in patterns.go metric creation                     | FIXED: Added fmt.Errorf wrapping                                          |
| Missing error wrapping in database.go metric creation                     | FIXED: Added fmt.Errorf wrapping                                          |
| Hardcoded "development" environment in getEnvironment()                   | FIXED: Read from MNEMONIC_ENV with fallback                               |
| Skip paths defined in two places (tracing.go and server.go)               | FIXED: Exported DefaultSkipPaths, server.go uses it                       |

### MEDIUM Priority

| Finding                                                       | Resolution                                                    |
| ------------------------------------------------------------- | ------------------------------------------------------------- |
| Errors not wrapped with context in registry.go                | FIXED: Added error wrapping                                   |
| Errors not wrapped in NewRequestMetrics                       | FIXED: Added error wrapping                                   |
| Dead code: metricsRegistry created but discarded              | FIXED: Removed unused code                                    |
| log.Println used instead of structured logging in CheckHealth | FIXED: Removed log.Println                                    |
| parseLogLevel should fail-fast on invalid level               | FIXED: Returns error, propagated to Initialize                |
| Unnecessary conditional for metrics path                      | FIXED: Removed conditional                                    |
| getServiceVersion wrapper unnecessary                         | FIXED: Removed, calls version.Version() directly              |
| telemetry.Config() redundant                                  | FIXED: Removed method                                         |
| Nested loops for skip path checks in tracing.go               | FIXED: Replaced with slices.Contains                          |
| Nil pointer risk in Telemetry.Shutdown()                      | DEFERRED: Low risk, otelx handles internally                  |
| Test resource leak with OTLP exporter                         | DEFERRED: Test isolation acceptable                           |
| Skip path matching too strict (won't match /health/)          | DEFERRED: Current paths are exact matches by design           |
| Tests don't verify tracing is actually skipped                | DEFERRED: Functional testing sufficient for MVP               |
| No nil check on metric recording methods                      | DEFERRED: Callers ensure non-nil                              |
| Error channel size of 1 in server.go                          | ACCEPTED: Single goroutine, single error expected             |
| Unused cfg parameter in getEnvironment                        | FIXED: Removed parameter                                      |
| No cardinality documentation on metric attributes             | FIXED: Added documentation comments                           |
| Test doesn't verify metrics can be recorded                   | FIXED: Added smoke test                                       |
| Duplicate code in Middleware() and MiddlewareWithSkipPaths()  | FIXED: Middleware() delegates to MiddlewareWithSkipPaths(nil) |

### LOW Priority

| Finding                                          | Resolution                                              |
| ------------------------------------------------ | ------------------------------------------------------- |
| Hardcoded environment "development"              | DEFERRED: Track for future config enhancement           |
| Linear search in skip paths                      | ACCEPTED: 3-element list, negligible performance impact |
| Duration precision loss (sub-ms as 0)            | DEFERRED: Millisecond precision acceptable for MVP      |
| Telemetry shutdown uses context.Background()     | DEFERRED: Consider timeout in production hardening      |
| Code duplication in metrics middleware           | DEFERRED: Refactor when adding more middleware          |
| No interface for Registry                        | DEFERRED: Add when mocking needed                       |
| Gauge semantics for connection pool              | DEFERRED: Document calling frequency                    |
| No error type classification in database metrics | DEFERRED: Enhance when database layer implemented       |
