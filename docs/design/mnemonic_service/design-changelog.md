# Mnemonic Design Change Log

This document tracks changes to design documents when implementation differs from original design. Each entry explains what changed and why.

## Format

Each entry includes:

- **Date**: When the change was made
- **Design Document**: Which document was updated
- **Section**: Which section changed
- **Original Design**: What the design originally specified
- **Implementation**: What was actually built
- **Justification**: Why the implementation differs

## Change Log

### 2026-01-28: Phase 3 Observability Implementation

**Design Document**: `observability-implementation.md`

#### 1. Configuration Structure

- **Original**: Standalone `ObservabilityConfig` struct with helper functions
- **Implementation**: Integrated into `MnemonicConfig` with nested `Observability` section
- **Justification**: Phase 2 established unified configuration pattern. Observability config follows same pattern for consistency.

#### 2. Telemetry Struct Fields

- **Original**: `Telemetry` wraps `*otelx.Telemetry` with `cfg config.ObservabilityConfig`
- **Implementation**: `Telemetry` has `otel *otelx.Telemetry` and `logger zerolog.Logger`, no cfg field
- **Justification**: Logger cached for direct access. Config not stored since it's only needed during initialization.

#### 3. Telemetry Methods

- **Original**: Only `Shutdown()` method documented
- **Implementation**: Added `Logger()`, `Tracer()`, `Meter()`, `TracerProvider()`, `MeterProvider()`, `Otelx()`
- **Justification**: Practical usage requires accessing these components. Methods provide clean encapsulation.

#### 4. Server Lifecycle Ownership

- **Original**: `main.go` initializes telemetry, passes to `server.ListenAndServe(tel)`
- **Implementation**: Server package owns telemetry initialization internally
- **Justification**: Phase 2 established server owning config loading. Telemetry follows same ownership pattern.

#### 5. Middleware Import Aliases

- **Original**: `otelgin` for otelx middleware
- **Implementation**: `otelxgin` for otelx middleware (distinguishes from contrib otelgin)
- **Justification**: Clarity - two different packages with similar names need distinct aliases.

#### 6. Middleware Option Signatures

- **Original**: `otelgin.WithSkipPaths([]string{"/health"})`
- **Implementation**: `otelxgin.WithSkipPaths("/health", "/ops/health", "/metrics")`
- **Justification**: otelx library uses variadic parameters, not slices. Additional paths for ops endpoints.

#### 7. Tracing Middleware Functions

- **Original**: Single `TracingMiddleware()` function
- **Implementation**: Two functions: `TracingMiddleware()` and `TracingMiddlewareWithSkipPaths()`
- **Justification**: Flexibility - default skip paths for common case, configurable for special cases.

#### 8. Skip Path Implementation

- **Original**: Loop-based path checking
- **Implementation**: Uses `slices.Contains()` for cleaner code
- **Justification**: Standard library function is more idiomatic and readable.

#### 9. Handler Dependencies (Deferred)

- **Original**: `internal/handlers/deps.go` with Dependencies struct
- **Implementation**: Not implemented in Phase 3
- **Justification**: Handler instrumentation is Phase 1D per design checklist. Will be implemented when handlers need metrics/tracing.

#### 10. Database Instrumentation (Deferred)

- **Original**: `internal/repository/postgres/instrumented.go` and `neo4j/instrumented.go`
- **Implementation**: Not implemented in Phase 3
- **Justification**: Database instrumentation is Phase 1E per design checklist. Will be implemented with repository layer.

---

Copyright (c) 2026 Jeremy K. Johnson. All rights reserved.
