# Observability Implementation Design

[Back to Architecture Overview](../../architecture/00-overview.md) | [Back to Project README](../../../README.md)

**Note:** This document reflects the actual Phase 3 implementation. See [Design Change Log](design-changelog.md) for details on how implementation differs from original design.

## Table of Contents

- [Overview](#overview)
- [otelx Package Integration](#otelx-package-integration)
- [Initialization and Configuration](#initialization-and-configuration)
- [Distributed Tracing Implementation](#distributed-tracing-implementation)
- [Metrics Implementation](#metrics-implementation)
- [Structured Logging Implementation](#structured-logging-implementation)
- [Handler Instrumentation Patterns](#handler-instrumentation-patterns)
- [Database Instrumentation](#database-instrumentation)
- [Gaps and Additional Implementation](#gaps-and-additional-implementation)
- [Testing Strategy](#testing-strategy)
- [Implementation Checklist](#implementation-checklist)

## Overview

> **Architecture Reference:** [Observability Architecture](../../architecture/07-observability-architecture.md) | [Requirements - Quality Attributes](../../architecture/01-requirements.md#quality-attributes)

This document provides the detailed Go implementation design for Phase 1 (MVP) observability in Mnemonic, as defined in the [Observability Architecture](../../architecture/07-observability-architecture.md).

**Phase 1 Scope:**

- OpenTelemetry SDK integration via `otelx`
- Structured logging with trace correlation
- Metrics emission (counters, histograms, gauges)
- Distributed tracing with span creation
- OTLP export configuration

**Primary Package:** `github.com/twistingmercury/otelx`

The otelx package provides unified OpenTelemetry initialization with:

- Zerolog-based structured logging with automatic trace correlation
- Prometheus metrics exporter
- OTLP gRPC trace exporter
- Gin middleware for HTTP request instrumentation

### Current Mnemonic Architecture

```text
src/mnemonic/
├── cmd/
│   ├── main/main.go           # Application entrypoint
│   └── version/version.go     # Version information
└── internal/
    ├── handlers/
    │   ├── agents/            # Agent CRUD endpoints
    │   ├── operations/        # Health and version endpoints
    │   ├── patterns/          # Pattern CRUD endpoints
    │   └── routes/
    │       ├── routes.go      # Routing endpoint
    │       └── rules/         # Routing rules CRUD
    └── server/server.go       # HTTP server setup
```

**Key Integration Points:**

1. `cmd/main/main.go` - Telemetry initialization and shutdown
2. `internal/server/server.go` - Middleware registration
3. All handler packages - Span creation and logging

## otelx Package Integration

> **Architecture Reference:** [Observability Architecture - Observability Stack](../../architecture/07-observability-architecture.md#observability-stack)

### Package Capabilities

The `otelx` package (v1.0.0) provides:

| Capability                 | otelx Support | Notes                              |
| -------------------------- | ------------- | ---------------------------------- |
| Unified initialization     | Yes           | Single `Initialize()` call         |
| Zerolog logging            | Yes           | With automatic trace correlation   |
| Prometheus metrics         | Yes           | Exposes `/metrics` endpoint        |
| OTLP tracing               | Yes           | gRPC exporter to collector         |
| Gin logging middleware     | Yes           | `middleware/gin.LoggingMiddleware` |
| Gin tracing middleware     | No            | Requires additional implementation |
| Request metrics middleware | No            | Requires additional implementation |
| Database instrumentation   | No            | Requires additional implementation |

### Required Dependencies

Add to `go.mod`:

```go
require (
    github.com/twistingmercury/otelx v1.0.0
    go.opentelemetry.io/otel v1.35.0
    go.opentelemetry.io/otel/metric v1.35.0
    go.opentelemetry.io/otel/trace v1.35.0
    go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.60.0
)
```

The `otelgin` package provides the tracing middleware that otelx does not include.

## Initialization and Configuration

> **Architecture Reference:** [Observability Architecture - Implementation Phases](../../architecture/07-observability-architecture.md#implementation-phases) | [Deployment Architecture - Operational Considerations](../../architecture/05-deployment-architecture.md#operational-considerations)
>
> **Note:** This configuration aligns with the established patterns in [Configuration Design](configuration.md). Environment variables use the `MNEMONIC_OBSERVABILITY_*` prefix for observability settings.

### Configuration Package

Observability configuration is integrated into the main `MnemonicConfig` structure following the Phase 2 unified configuration pattern. The observability settings are nested under the `Observability` section.

**Actual Implementation:**

Configuration structure from `internal/config/config.go`:

```go
type MnemonicConfig struct {
    Server        ServerConfig
    Logging       LoggingConfig
    Observability ObservabilityConfig
}

type ObservabilityConfig struct {
    Metrics MetricsConfig
    Tracing TracingConfig
}

type MetricsConfig struct {
    Enabled bool
    Port    int
    Path    string
}

type TracingConfig struct {
    Enabled      bool
    Endpoint     string
    OTLPInsecure bool
    SampleRate   float64
}

type LoggingConfig struct {
    Level string // Parsed to zerolog.Level in telemetry package
}
```

The configuration is loaded via the Phase 2 `config.Load()` function which handles environment variables and defaults.

### Telemetry Initialization

**Actual Implementation** from `internal/telemetry/telemetry.go`:

```go
package telemetry

import (
    "context"
    "fmt"

    "github.com/rs/zerolog"
    "github.com/twistingmercury/mnemonic/cmd/version"
    "github.com/twistingmercury/mnemonic/internal/config"
    "github.com/twistingmercury/otelx"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/trace"
)

// Telemetry wraps the otelx.Telemetry with application-specific helpers.
type Telemetry struct {
    otel   *otelx.Telemetry
    logger zerolog.Logger
}

// Initialize creates and configures the telemetry system using otelx.
func Initialize(ctx context.Context, cfg *config.MnemonicConfig) (*Telemetry, error) {
    opts, err := buildOptions(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to build telemetry options: %w", err)
    }

    tel, err := otelx.Initialize(ctx, opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize telemetry: %w", err)
    }

    return &Telemetry{
        otel:   tel,
        logger: tel.Logger,
    }, nil
}

func buildOptions(cfg *config.MnemonicConfig) ([]otelx.Option, error) {
    logLevel, err := parseLogLevel(cfg.Logging.Level)
    if err != nil {
        return nil, err
    }

    opts := []otelx.Option{
        otelx.WithService(
            "mnemonic",
            version.Version(),
            getEnvironment(cfg),
        ),
        otelx.WithLogLevel(logLevel),
    }

    // Metrics configuration - otelx uses opt-in pattern
    if cfg.Observability.Metrics.Enabled {
        opts = append(opts, otelx.WithMetrics(cfg.Observability.Metrics.Port))
        opts = append(opts, otelx.WithMetricsPath(cfg.Observability.Metrics.Path))
    }

    // Tracing configuration - otelx uses opt-in pattern
    if cfg.Observability.Tracing.Enabled {
        opts = append(opts, otelx.WithTracing())
        opts = append(opts, otelx.WithTraceSampleRate(cfg.Observability.Tracing.SampleRate))
        if cfg.Observability.Tracing.Endpoint != "" {
            opts = append(opts, otelx.WithOTLPEndpoint(cfg.Observability.Tracing.Endpoint))
        }
        if cfg.Observability.Tracing.OTLPInsecure {
            opts = append(opts, otelx.WithOTLPInsecure())
        }
    }

    return opts, nil
}

// Shutdown gracefully shuts down telemetry, flushing pending data.
func (t *Telemetry) Shutdown(ctx context.Context) error {
    return t.otel.Shutdown(ctx)
}

// Logger returns the zerolog logger with trace correlation support.
func (t *Telemetry) Logger() zerolog.Logger {
    return t.logger
}

// Tracer returns an OpenTelemetry tracer for creating spans.
func (t *Telemetry) Tracer(name string) trace.Tracer {
    if t.otel.TracerProvider != nil {
        return t.otel.TracerProvider.Tracer(name)
    }
    return otel.Tracer(name)
}

// Meter returns an OpenTelemetry meter for creating metrics.
func (t *Telemetry) Meter(name string) metric.Meter {
    if t.otel.MeterProvider != nil {
        return t.otel.MeterProvider.Meter(name)
    }
    return otel.Meter(name)
}

// TracerProvider returns the underlying trace provider.
func (t *Telemetry) TracerProvider() trace.TracerProvider {
    if t.otel.TracerProvider != nil {
        return t.otel.TracerProvider
    }
    return otel.GetTracerProvider()
}

// MeterProvider returns the underlying meter provider.
func (t *Telemetry) MeterProvider() metric.MeterProvider {
    if t.otel.MeterProvider != nil {
        return t.otel.MeterProvider
    }
    return otel.GetMeterProvider()
}

// Otelx returns the underlying otelx.Telemetry instance.
func (t *Telemetry) Otelx() *otelx.Telemetry {
    return t.otel
}
```

### Server Lifecycle and Telemetry

**Actual Implementation:**

In Phase 3, telemetry initialization is owned by the `server` package, not `main.go`. This follows the Phase 2 pattern where the server package owns configuration loading and lifecycle management.

From `internal/server/server.go`:

```go
// ListenAndServeWithConfig starts the server using the provided configuration.
func ListenAndServeWithConfig(cfg *config.MnemonicConfig) error {
    shutdown, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    // Initialize telemetry
    tel, err := telemetry.Initialize(shutdown, cfg)
    if err != nil {
        return fmt.Errorf("failed to initialize telemetry: %w", err)
    }
    defer func() {
        if shutdownErr := tel.Shutdown(context.Background()); shutdownErr != nil {
            logger := tel.Logger()
            logger.Error().Err(shutdownErr).Msg("telemetry shutdown error")
        }
    }()

    logger := tel.Logger()
    logger.Info().
        Str("host", cfg.Server.Host).
        Int("port", cfg.Server.Port).
        Bool("metrics_enabled", cfg.Observability.Metrics.Enabled).
        Bool("tracing_enabled", cfg.Observability.Tracing.Enabled).
        Msg("mnemonic starting")

    // Create request metrics middleware
    requestMetrics, err := middleware.NewRequestMetrics(tel.Meter("mnemonic/http"))
    if err != nil {
        return fmt.Errorf("failed to create request metrics: %w", err)
    }

    router := setupRouter(tel, requestMetrics)
    operations.SetupHandlers(router)

    // Server startup and graceful shutdown...
}
```

The `main.go` remains simple and delegates to the server package.

## Distributed Tracing Implementation

> **Architecture Reference:** [Observability Architecture - Distributed Tracing (Jaeger)](../../architecture/07-observability-architecture.md#distributed-tracing-jaeger)

### Trace Structure

Based on the architecture document, traces should capture:

```text
POST /v1/api/route (45ms)
├── Validate Request (2ms)
├── Apply Routing Rules (8ms)
├── Fetch Patterns (30ms)
│   ├── Postgres Query (10ms)
│   ├── PGVector Search (12ms)
│   └── Neo4j Query (8ms)
└── Build Response (5ms)
```

### Tracing Middleware

**Actual Implementation** from `internal/middleware/tracing.go`:

```go
package middleware

import (
    "net/http"
    "slices"

    "github.com/gin-gonic/gin"
    "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// skipPaths defines paths to exclude from tracing to reduce noise.
var defaultSkipPaths = []string{
    "/health",
    "/ops/health",
    "/metrics",
}

// TracingMiddleware returns Gin middleware that creates spans for HTTP requests.
// It uses W3C Trace Context for trace propagation and automatically instruments
// incoming requests with OpenTelemetry spans.
func TracingMiddleware(serviceName string) gin.HandlerFunc {
    return otelgin.Middleware(serviceName,
        otelgin.WithFilter(func(req *http.Request) bool {
            // Skip tracing for health checks and metrics to reduce noise
            return !slices.Contains(defaultSkipPaths, req.URL.Path)
        }),
    )
}

// TracingMiddlewareWithSkipPaths returns Gin middleware with custom skip paths.
// This allows callers to specify which paths should not be traced.
func TracingMiddlewareWithSkipPaths(serviceName string, skipPaths []string) gin.HandlerFunc {
    return otelgin.Middleware(serviceName,
        otelgin.WithFilter(func(req *http.Request) bool {
            return !slices.Contains(skipPaths, req.URL.Path)
        }),
    )
}
```

The implementation provides two functions for flexibility:

- `TracingMiddleware()` with default skip paths
- `TracingMiddlewareWithSkipPaths()` for custom skip path configuration

### Creating Child Spans in Handlers

Handlers create child spans for logical operations:

```go
package routes

import (
    "context"

    "github.com/gin-gonic/gin"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("mnemonic/handlers/routes")

func RoutePrompt(c *gin.Context) {
    ctx := c.Request.Context()

    // Validate request
    ctx, validateSpan := tracer.Start(ctx, "validate_request")
    req, err := validateRouteRequest(ctx, c)
    if err != nil {
        validateSpan.RecordError(err)
        validateSpan.SetStatus(codes.Error, err.Error())
        validateSpan.End()
        // Return error response...
        return
    }
    validateSpan.SetAttributes(
        attribute.String("prompt.preview", truncate(req.Prompt, 100)),
    )
    validateSpan.End()

    // Apply routing rules
    ctx, routeSpan := tracer.Start(ctx, "apply_routing_rules")
    agent, rule, err := applyRoutingRules(ctx, req)
    if err != nil {
        routeSpan.RecordError(err)
        routeSpan.SetStatus(codes.Error, err.Error())
        routeSpan.End()
        // Return error response...
        return
    }
    routeSpan.SetAttributes(
        attribute.String("routing.agent", agent.Name),
        attribute.String("routing.rule_type", rule.Type),
        attribute.Int("routing.rule_priority", rule.Priority),
    )
    routeSpan.End()

    // Fetch patterns
    ctx, patternSpan := tracer.Start(ctx, "fetch_patterns")
    patterns, err := fetchPatterns(ctx, agent, req)
    patternSpan.SetAttributes(
        attribute.Int("patterns.count", len(patterns)),
    )
    if err != nil {
        patternSpan.RecordError(err)
        patternSpan.SetStatus(codes.Error, err.Error())
    }
    patternSpan.End()

    // Build and return response...
}
```

### Span Naming Conventions

| Operation          | Span Name               | Attributes                           |
| ------------------ | ----------------------- | ------------------------------------ |
| HTTP request       | `HTTP {METHOD} {route}` | Auto by otelgin                      |
| Request validation | `validate_request`      | `prompt.preview`                     |
| Routing rules      | `apply_routing_rules`   | `routing.agent`, `routing.rule_type` |
| Pattern fetch      | `fetch_patterns`        | `patterns.count`                     |
| Postgres query     | `postgres.query`        | `db.statement`, `db.operation`       |
| PGVector search    | `pgvector.search`       | `db.statement`, `vector.dimensions`  |
| Neo4j query        | `neo4j.query`           | `db.statement`, `db.operation`       |

## Metrics Implementation

> **Architecture Reference:** [Observability Architecture - Metrics (Prometheus)](../../architecture/07-observability-architecture.md#metrics-prometheus)

### Metrics Categories

Based on the architecture document, implement these metric categories:

1. **Request metrics** - HTTP request counts, durations, in-flight
2. **Routing metrics** - Routing decisions, pattern matches, cache stats
3. **Pattern metrics** - Query latency, patterns returned
4. **Database metrics** - Connection pools, query latency, errors

### Request Metrics Middleware

**Actual Implementation** from `internal/middleware/metrics.go`:

```go
package middleware

import (
    "strconv"
    "time"

    "github.com/gin-gonic/gin"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

// RequestMetrics holds the instruments for HTTP request metrics.
type RequestMetrics struct {
    requestCount    metric.Int64Counter
    requestDuration metric.Float64Histogram
    requestInFlight metric.Int64UpDownCounter
}

// NewRequestMetrics creates request metric instruments.
func NewRequestMetrics(meter metric.Meter) (*RequestMetrics, error) {
    requestCount, err := meter.Int64Counter(
        "mnemonic.http.request.count",
        metric.WithDescription("Total number of HTTP requests"),
        metric.WithUnit("{request}"),
    )
    if err != nil {
        return nil, err
    }

    requestDuration, err := meter.Float64Histogram(
        "mnemonic.http.request.duration",
        metric.WithDescription("HTTP request duration in milliseconds"),
        metric.WithUnit("ms"),
        metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000),
    )
    if err != nil {
        return nil, err
    }

    requestInFlight, err := meter.Int64UpDownCounter(
        "mnemonic.http.request.in_flight",
        metric.WithDescription("Number of HTTP requests currently in flight"),
        metric.WithUnit("{request}"),
    )
    if err != nil {
        return nil, err
    }

    return &RequestMetrics{
        requestCount:    requestCount,
        requestDuration: requestDuration,
        requestInFlight: requestInFlight,
    }, nil
}

// Middleware returns Gin middleware that records request metrics.
func (m *RequestMetrics) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        // Track in-flight requests
        m.requestInFlight.Add(c.Request.Context(), 1)
        defer m.requestInFlight.Add(c.Request.Context(), -1)

        // Process request
        c.Next()

        // Record metrics
        duration := float64(time.Since(start).Milliseconds())
        attrs := []attribute.KeyValue{
            attribute.String("http.method", c.Request.Method),
            attribute.String("http.route", c.FullPath()),
            attribute.String("http.status_code", strconv.Itoa(c.Writer.Status())),
        }

        m.requestCount.Add(c.Request.Context(), 1, metric.WithAttributes(attrs...))
        m.requestDuration.Record(c.Request.Context(), duration, metric.WithAttributes(attrs...))
    }
}
```

### Routing Metrics

Create `internal/metrics/routing.go`:

```go
package metrics

import (
    "context"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

// Routing holds instruments for routing-related metrics.
type Routing struct {
    routingDecisions metric.Int64Counter
    ruleMatches      metric.Int64Counter
    cacheHits        metric.Int64Counter
    cacheMisses      metric.Int64Counter
}

// NewRouting creates routing metric instruments.
func NewRouting(meter metric.Meter) (*Routing, error) {
    routingDecisions, err := meter.Int64Counter(
        "mnemonic.routing.decisions",
        metric.WithDescription("Number of routing decisions made"),
        metric.WithUnit("{decision}"),
    )
    if err != nil {
        return nil, err
    }

    ruleMatches, err := meter.Int64Counter(
        "mnemonic.routing.rule_matches",
        metric.WithDescription("Number of rule matches by type"),
        metric.WithUnit("{match}"),
    )
    if err != nil {
        return nil, err
    }

    cacheHits, err := meter.Int64Counter(
        "mnemonic.routing.cache_hits",
        metric.WithDescription("Number of routing cache hits"),
        metric.WithUnit("{hit}"),
    )
    if err != nil {
        return nil, err
    }

    cacheMisses, err := meter.Int64Counter(
        "mnemonic.routing.cache_misses",
        metric.WithDescription("Number of routing cache misses"),
        metric.WithUnit("{miss}"),
    )
    if err != nil {
        return nil, err
    }

    return &Routing{
        routingDecisions: routingDecisions,
        ruleMatches:      ruleMatches,
        cacheHits:        cacheHits,
        cacheMisses:      cacheMisses,
    }, nil
}

// RecordRoutingDecision records a routing decision was made.
func (m *Routing) RecordRoutingDecision(ctx context.Context, agentName string) {
    m.routingDecisions.Add(ctx, 1, metric.WithAttributes(
        attribute.String("agent", agentName),
    ))
}

// RecordRuleMatch records a rule match by type.
func (m *Routing) RecordRuleMatch(ctx context.Context, ruleType string) {
    m.ruleMatches.Add(ctx, 1, metric.WithAttributes(
        attribute.String("rule_type", ruleType),
    ))
}

// RecordCacheHit records a cache hit.
func (m *Routing) RecordCacheHit(ctx context.Context) {
    m.cacheHits.Add(ctx, 1)
}

// RecordCacheMiss records a cache miss.
func (m *Routing) RecordCacheMiss(ctx context.Context) {
    m.cacheMisses.Add(ctx, 1)
}
```

### Pattern Metrics

Create `internal/metrics/patterns.go`:

```go
package metrics

import (
    "context"
    "time"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

// Pattern holds instruments for pattern retrieval metrics.
type Pattern struct {
    queryLatency     metric.Float64Histogram
    patternsReturned metric.Int64Histogram
}

// NewPattern creates pattern metric instruments.
func NewPattern(meter metric.Meter) (*Pattern, error) {
    queryLatency, err := meter.Float64Histogram(
        "mnemonic.patterns.query_latency",
        metric.WithDescription("Pattern query latency in milliseconds"),
        metric.WithUnit("ms"),
        metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500),
    )
    if err != nil {
        return nil, err
    }

    patternsReturned, err := meter.Int64Histogram(
        "mnemonic.patterns.returned",
        metric.WithDescription("Number of patterns returned per query"),
        metric.WithUnit("{pattern}"),
        metric.WithExplicitBucketBoundaries(0, 1, 5, 10, 25, 50, 100),
    )
    if err != nil {
        return nil, err
    }

    return &Pattern{
        queryLatency:     queryLatency,
        patternsReturned: patternsReturned,
    }, nil
}

// RecordQuery records a pattern query with its latency and result count.
func (m *Pattern) RecordQuery(ctx context.Context, database string, duration time.Duration, count int) {
    attrs := metric.WithAttributes(attribute.String("database", database))
    m.queryLatency.Record(ctx, float64(duration.Milliseconds()), attrs)
    m.patternsReturned.Record(ctx, int64(count), attrs)
}
```

### Database Metrics

Create `internal/metrics/database.go`:

```go
package metrics

import (
    "context"
    "time"

    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/metric"
)

// Database holds instruments for database-related metrics.
type Database struct {
    connectionPoolSize   metric.Int64Gauge
    connectionPoolInUse  metric.Int64Gauge
    queryLatency         metric.Float64Histogram
    queryErrors          metric.Int64Counter
}

// NewDatabase creates database metric instruments.
func NewDatabase(meter metric.Meter) (*Database, error) {
    connectionPoolSize, err := meter.Int64Gauge(
        "mnemonic.db.connection_pool.size",
        metric.WithDescription("Total size of the database connection pool"),
        metric.WithUnit("{connection}"),
    )
    if err != nil {
        return nil, err
    }

    connectionPoolInUse, err := meter.Int64Gauge(
        "mnemonic.db.connection_pool.in_use",
        metric.WithDescription("Number of connections currently in use"),
        metric.WithUnit("{connection}"),
    )
    if err != nil {
        return nil, err
    }

    queryLatency, err := meter.Float64Histogram(
        "mnemonic.db.query_latency",
        metric.WithDescription("Database query latency in milliseconds"),
        metric.WithUnit("ms"),
        metric.WithExplicitBucketBoundaries(1, 5, 10, 25, 50, 100, 250, 500, 1000),
    )
    if err != nil {
        return nil, err
    }

    queryErrors, err := meter.Int64Counter(
        "mnemonic.db.query_errors",
        metric.WithDescription("Number of database query errors"),
        metric.WithUnit("{error}"),
    )
    if err != nil {
        return nil, err
    }

    return &Database{
        connectionPoolSize:  connectionPoolSize,
        connectionPoolInUse: connectionPoolInUse,
        queryLatency:        queryLatency,
        queryErrors:         queryErrors,
    }, nil
}

// RecordPoolStats records connection pool statistics.
func (m *Database) RecordPoolStats(ctx context.Context, database string, size, inUse int64) {
    attrs := metric.WithAttributes(attribute.String("database", database))
    m.connectionPoolSize.Record(ctx, size, attrs)
    m.connectionPoolInUse.Record(ctx, inUse, attrs)
}

// RecordQuery records a database query with latency.
func (m *Database) RecordQuery(ctx context.Context, database, operation string, duration time.Duration) {
    m.queryLatency.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(
        attribute.String("database", database),
        attribute.String("operation", operation),
    ))
}

// RecordError records a database error.
func (m *Database) RecordError(ctx context.Context, database, operation string) {
    m.queryErrors.Add(ctx, 1, metric.WithAttributes(
        attribute.String("database", database),
        attribute.String("operation", operation),
    ))
}
```

### Metrics Registry

Create `internal/metrics/registry.go` to centralize metric initialization:

```go
package metrics

import (
    "go.opentelemetry.io/otel/metric"
)

// Registry holds all metric instruments for the application.
type Registry struct {
    Routing  *Routing
    Patterns *Pattern
    Database *Database
}

// NewRegistry creates all metric instruments.
func NewRegistry(meter metric.Meter) (*Registry, error) {
    routing, err := NewRouting(meter)
    if err != nil {
        return nil, err
    }

    patterns, err := NewPattern(meter)
    if err != nil {
        return nil, err
    }

    database, err := NewDatabase(meter)
    if err != nil {
        return nil, err
    }

    return &Registry{
        Routing:  routing,
        Patterns: patterns,
        Database: database,
    }, nil
}
```

## Structured Logging Implementation

> **Architecture Reference:** [Observability Architecture - Logs (Loki)](../../architecture/07-observability-architecture.md#logs-loki)

### Using otelx Gin Middleware

**Actual Implementation:**

otelx provides `middleware/gin.LoggingMiddleware` for request logging with trace correlation. From `internal/server/server.go`:

```go
package server

import (
    "github.com/gin-gonic/gin"
    "github.com/twistingmercury/mnemonic/internal/middleware"
    "github.com/twistingmercury/mnemonic/internal/telemetry"
    otelxgin "github.com/twistingmercury/otelx/middleware/gin"
)

// setupRouter creates and configures the Gin router with middleware.
func setupRouter(tel *telemetry.Telemetry, requestMetrics *middleware.RequestMetrics) *gin.Engine {
    // Use gin.New() instead of gin.Default() to avoid duplicate logging
    router := gin.New()

    // Recovery middleware (keep this)
    router.Use(gin.Recovery())

    // Paths to skip for tracing and metrics
    skipPaths := []string{"/health", "/ops/health", "/metrics"}

    // Tracing middleware using otelgin
    router.Use(middleware.TracingMiddlewareWithSkipPaths("mnemonic", skipPaths))

    // otelx logging middleware with trace correlation
    router.Use(otelxgin.LoggingMiddleware(tel.Otelx(),
        otelxgin.WithSkipPaths("/health", "/ops/health", "/metrics"),
        otelxgin.WithRequestHeaders("X-Request-ID", "X-Correlation-ID"),
    ))

    // Request metrics middleware
    router.Use(requestMetrics.MiddlewareWithSkipPaths(skipPaths))

    return router
}
```

**Key Differences:**

- Import alias `otelxgin` distinguishes from contrib `otelgin`
- `WithSkipPaths()` uses variadic parameters, not slices
- Additional skip paths for `/ops/health` and `/metrics`
- `WithRequestHeaders()` uses variadic parameters
- Middleware registration includes tracing and metrics

### Handler Logging

Use the logger from context in handlers:

```go
package agents

import (
    "github.com/gin-gonic/gin"
    otelgin "github.com/twistingmercury/otelx/middleware/gin"
    "github.com/twistingmercury/mnemonic/internal/telemetry"
)

func ListAgents(c *gin.Context, tel *telemetry.Telemetry) {
    logger := otelgin.Logger(c, tel.Telemetry)

    logger.Info().Msg("listing agents")

    // Business logic...

    logger.Debug().
        Int("count", len(agents)).
        Msg("agents retrieved")
}
```

### Log Entry Structure

All log entries automatically include (via otelx):

```json
{
  "level": "info",
  "time": "2024-01-21T10:30:00Z",
  "service": "mnemonic",
  "trace_id": "abc123def456...",
  "span_id": "789xyz...",
  "message": "request completed",
  "http.method": "POST",
  "http.path": "/v1/api/route",
  "http.status_code": 200,
  "latency_ms": 45
}
```

### Log Levels by Event Type

| Event Type              | Level | Example                     |
| ----------------------- | ----- | --------------------------- |
| Request received        | Debug | Start of request processing |
| Request completed (2xx) | Info  | Successful response         |
| Request completed (4xx) | Warn  | Client error                |
| Request completed (5xx) | Error | Server error                |
| Routing decision        | Info  | Agent selected              |
| Pattern query           | Debug | Database query executed     |
| Configuration loaded    | Info  | Startup configuration       |
| Service lifecycle       | Info  | Start/stop events           |
| Validation failure      | Warn  | Invalid input               |
| Database error          | Error | Connection/query failure    |

## Handler Instrumentation Patterns

> **Architecture Reference:** [System Architecture - Mnemonic](../../architecture/03-system-architecture.md#mnemonic) | [Observability Architecture - Key Takeaways](../../architecture/07-observability-architecture.md#key-takeaways)

### Handler Dependencies (Future Phase)

**Status:** Not implemented in Phase 3.

This section is deferred to Phase 1D per the implementation checklist. Handler instrumentation will be implemented when handlers require metrics and tracing.

**Planned Implementation:**

Create `internal/handlers/deps.go`:

```go
package handlers

import (
    "github.com/twistingmercury/mnemonic/internal/metrics"
    "github.com/twistingmercury/mnemonic/internal/telemetry"
    "go.opentelemetry.io/otel/trace"
)

// Dependencies holds shared dependencies for all handlers.
type Dependencies struct {
    Tel     *telemetry.Telemetry
    Metrics *metrics.Registry
    Tracer  trace.Tracer
}

// NewDependencies creates handler dependencies.
func NewDependencies(tel *telemetry.Telemetry, metrics *metrics.Registry) *Dependencies {
    return &Dependencies{
        Tel:     tel,
        Metrics: metrics,
        Tracer:  tel.TracerProvider.Tracer("mnemonic/handlers"),
    }
}
```

### Instrumented Handler Pattern

Example of a fully instrumented handler:

```go
package routes

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    otelgin "github.com/twistingmercury/otelx/middleware/gin"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"

    "github.com/twistingmercury/mnemonic/internal/handlers"
)

// RoutePrompt handles POST /v1/api/route with full observability.
func RoutePrompt(deps *handlers.Dependencies) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := c.Request.Context()
        logger := otelgin.Logger(c, deps.Tel.Telemetry)

        // 1. Validate request with span
        ctx, validateSpan := deps.Tracer.Start(ctx, "validate_request")
        req, err := validateRequest(c)
        if err != nil {
            validateSpan.RecordError(err)
            validateSpan.SetStatus(codes.Error, "validation failed")
            validateSpan.End()

            logger.Warn().Err(err).Msg("request validation failed")
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        validateSpan.End()

        // 2. Apply routing rules with span and metrics
        ctx, routeSpan := deps.Tracer.Start(ctx, "apply_routing_rules")
        start := time.Now()

        agent, rule, cached, err := applyRoutingRules(ctx, req)
        if err != nil {
            routeSpan.RecordError(err)
            routeSpan.SetStatus(codes.Error, "routing failed")
            routeSpan.End()

            logger.Error().Err(err).Msg("routing rules failed")
            c.JSON(http.StatusInternalServerError, gin.H{"error": "routing failed"})
            return
        }

        routeSpan.SetAttributes(
            attribute.String("routing.agent", agent.Name),
            attribute.String("routing.rule_type", rule.Type),
            attribute.Bool("routing.cached", cached),
        )
        routeSpan.End()

        // Record routing metrics
        deps.Metrics.Routing.RecordRoutingDecision(ctx, agent.Name)
        deps.Metrics.Routing.RecordRuleMatch(ctx, rule.Type)
        if cached {
            deps.Metrics.Routing.RecordCacheHit(ctx)
        } else {
            deps.Metrics.Routing.RecordCacheMiss(ctx)
        }

        logger.Info().
            Str("agent", agent.Name).
            Str("rule_type", rule.Type).
            Dur("duration", time.Since(start)).
            Msg("routing decision made")

        // 3. Fetch patterns with span and metrics
        ctx, patternSpan := deps.Tracer.Start(ctx, "fetch_patterns")
        patternStart := time.Now()

        patterns, err := fetchPatterns(ctx, deps, agent, req)
        patternDuration := time.Since(patternStart)

        patternSpan.SetAttributes(attribute.Int("patterns.count", len(patterns)))
        if err != nil {
            patternSpan.RecordError(err)
            patternSpan.SetStatus(codes.Error, "pattern fetch failed")
        }
        patternSpan.End()

        deps.Metrics.Patterns.RecordQuery(ctx, "combined", patternDuration, len(patterns))

        // 4. Build and return response
        response := buildResponse(agent, patterns)

        logger.Debug().
            Int("pattern_count", len(patterns)).
            Msg("response built")

        c.JSON(http.StatusOK, response)
    }
}
```

### Handler Registration Update

Update handler registration to use dependencies:

```go
package routes

import (
    "github.com/gin-gonic/gin"
    "github.com/twistingmercury/mnemonic/internal/handlers"
)

// SetupHandlers registers route handlers with dependencies.
func SetupHandlers(r *gin.Engine, deps *handlers.Dependencies) {
    r.POST("/v1/api/route", RoutePrompt(deps))
}
```

## Database Instrumentation

> **Architecture Reference:** [System Architecture - Mnemonic](../../architecture/03-system-architecture.md#mnemonic) | [Observability Architecture - Metrics (Prometheus)](../../architecture/07-observability-architecture.md#metrics-prometheus)

### Postgres/PGVector Instrumentation (Future Phase)

**Status:** Not implemented in Phase 3.

This section is deferred to Phase 1E per the implementation checklist. Database instrumentation will be implemented with the repository layer.

**Planned Implementation:**

Create `internal/repository/postgres/instrumented.go`:

```go
package postgres

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgconn"
    "github.com/jackc/pgx/v5/pgxpool"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"

    "github.com/twistingmercury/mnemonic/internal/metrics"
)

var tracer = otel.Tracer("mnemonic/repository/postgres")

// InstrumentedPool wraps pgxpool.Pool with observability.
type InstrumentedPool struct {
    pool    *pgxpool.Pool
    metrics *metrics.Database
}

// NewInstrumentedPool creates an instrumented database pool.
func NewInstrumentedPool(pool *pgxpool.Pool, metrics *metrics.Database) *InstrumentedPool {
    return &InstrumentedPool{
        pool:    pool,
        metrics: metrics,
    }
}

// Query executes a query with tracing and metrics.
func (p *InstrumentedPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
    ctx, span := tracer.Start(ctx, "postgres.query",
        trace.WithAttributes(
            attribute.String("db.system", "postgresql"),
            attribute.String("db.statement", sql),
        ),
    )
    defer span.End()

    start := time.Now()
    rows, err := p.pool.Query(ctx, sql, args...)
    duration := time.Since(start)

    p.metrics.RecordQuery(ctx, "postgres", "query", duration)

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        p.metrics.RecordError(ctx, "postgres", "query")
        return nil, err
    }

    return rows, nil
}

// QueryRow executes a query that returns a single row with tracing.
func (p *InstrumentedPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
    ctx, span := tracer.Start(ctx, "postgres.query_row",
        trace.WithAttributes(
            attribute.String("db.system", "postgresql"),
            attribute.String("db.statement", sql),
        ),
    )

    start := time.Now()
    row := p.pool.QueryRow(ctx, sql, args...)
    duration := time.Since(start)

    p.metrics.RecordQuery(ctx, "postgres", "query_row", duration)
    span.End()

    return row
}

// Exec executes a command with tracing and metrics.
func (p *InstrumentedPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
    ctx, span := tracer.Start(ctx, "postgres.exec",
        trace.WithAttributes(
            attribute.String("db.system", "postgresql"),
            attribute.String("db.statement", sql),
        ),
    )
    defer span.End()

    start := time.Now()
    tag, err := p.pool.Exec(ctx, sql, args...)
    duration := time.Since(start)

    p.metrics.RecordQuery(ctx, "postgres", "exec", duration)

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        p.metrics.RecordError(ctx, "postgres", "exec")
        return tag, err
    }

    return tag, nil
}

// RecordPoolStats records connection pool statistics.
func (p *InstrumentedPool) RecordPoolStats(ctx context.Context) {
    stats := p.pool.Stat()
    p.metrics.RecordPoolStats(ctx, "postgres",
        int64(stats.MaxConns()),
        int64(stats.AcquiredConns()),
    )
}
```

### Neo4j Instrumentation (Future Phase)

**Status:** Not implemented in Phase 3.

This section is deferred to Phase 1E per the implementation checklist. Database instrumentation will be implemented with the repository layer.

**Planned Implementation:**

Create `internal/repository/neo4j/instrumented.go`:

```go
package neo4j

import (
    "context"
    "time"

    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"

    "github.com/twistingmercury/mnemonic/internal/metrics"
)

var tracer = otel.Tracer("mnemonic/repository/neo4j")

// InstrumentedSession wraps neo4j.SessionWithContext with observability.
type InstrumentedSession struct {
    session neo4j.SessionWithContext
    metrics *metrics.Database
}

// NewInstrumentedSession creates an instrumented Neo4j session.
func NewInstrumentedSession(session neo4j.SessionWithContext, metrics *metrics.Database) *InstrumentedSession {
    return &InstrumentedSession{
        session: session,
        metrics: metrics,
    }
}

// Run executes a Cypher query with tracing and metrics.
func (s *InstrumentedSession) Run(ctx context.Context, cypher string, params map[string]any) (neo4j.ResultWithContext, error) {
    ctx, span := tracer.Start(ctx, "neo4j.query",
        trace.WithAttributes(
            attribute.String("db.system", "neo4j"),
            attribute.String("db.statement", cypher),
        ),
    )
    defer span.End()

    start := time.Now()
    result, err := s.session.Run(ctx, cypher, params)
    duration := time.Since(start)

    s.metrics.RecordQuery(ctx, "neo4j", "query", duration)

    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        s.metrics.RecordError(ctx, "neo4j", "query")
        return nil, err
    }

    return result, nil
}

// Close closes the session.
func (s *InstrumentedSession) Close(ctx context.Context) error {
    return s.session.Close(ctx)
}
```

### Connection Pool Monitoring

Create a background goroutine to periodically record pool stats:

```go
package telemetry

import (
    "context"
    "time"
)

// StartPoolStatsRecorder starts a background routine to record pool stats.
func StartPoolStatsRecorder(ctx context.Context, pool *postgres.InstrumentedPool, interval time.Duration) {
    go func() {
        ticker := time.NewTicker(interval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                pool.RecordPoolStats(ctx)
            }
        }
    }()
}
```

## Gaps and Additional Implementation

### What otelx Provides

| Capability                   | Status   | Notes                      |
| ---------------------------- | -------- | -------------------------- |
| Unified initialization       | Provided | Single `Initialize()` call |
| Structured logging (zerolog) | Provided | With trace correlation     |
| Prometheus metrics endpoint  | Provided | Configurable port/path     |
| OTLP trace export            | Provided | gRPC to collector          |
| Gin logging middleware       | Provided | Request completion logging |
| Gin correlation middleware   | Provided | Logger in context          |

### What Requires Additional Implementation

| Capability                   | Status | Implementation Needed          |
| ---------------------------- | ------ | ------------------------------ |
| Gin tracing middleware       | Gap    | Use `otelgin` from contrib     |
| Request metrics middleware   | Gap    | Custom `middleware/metrics.go` |
| Application-specific metrics | Gap    | Custom `metrics/*.go`          |
| Database instrumentation     | Gap    | Custom repository wrappers     |
| Connection pool monitoring   | Gap    | Background stat recorder       |
| Custom sampling logic        | Gap    | Configure via otelx options    |

### Recommended Package Structure

```text
src/mnemonic/internal/
├── config/
│   └── config.go              # Configuration including observability
├── telemetry/
│   └── telemetry.go           # otelx initialization wrapper
├── middleware/
│   ├── tracing.go             # otelgin tracing middleware
│   └── metrics.go             # Request metrics middleware
├── metrics/
│   ├── registry.go            # Centralized metric registry
│   ├── routing.go             # Routing-specific metrics
│   ├── patterns.go            # Pattern-specific metrics
│   └── database.go            # Database-specific metrics
├── handlers/
│   ├── deps.go                # Handler dependencies
│   └── ...                    # Existing handlers (updated)
└── repository/
    ├── postgres/
    │   └── instrumented.go    # Instrumented Postgres pool
    └── neo4j/
        └── instrumented.go    # Instrumented Neo4j session
```

## Testing Strategy

> **Architecture Reference:** [Requirements - Success Criteria](../../architecture/01-requirements.md#success-criteria)

### Unit Testing Observability

Test metric recording without external dependencies:

```go
package metrics_test

import (
    "context"
    "testing"

    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/metric/metricdata"

    mmetrics "github.com/twistingmercury/mnemonic/internal/metrics"
)

func TestRouting(t *testing.T) {
    // Create a test meter provider with in-memory reader
    reader := metric.NewManualReader()
    provider := metric.NewMeterProvider(metric.WithReader(reader))
    meter := provider.Meter("test")

    // Create metrics
    rm, err := mmetrics.NewRouting(meter)
    if err != nil {
        t.Fatalf("failed to create routing metrics: %v", err)
    }

    // Record some metrics
    ctx := context.Background()
    rm.RecordRoutingDecision(ctx, "go-engineer")
    rm.RecordRuleMatch(ctx, "keyword")
    rm.RecordCacheHit(ctx)

    // Collect and verify
    var data metricdata.ResourceMetrics
    if err := reader.Collect(ctx, &data); err != nil {
        t.Fatalf("failed to collect metrics: %v", err)
    }

    // Assert expected metrics exist with correct values
    // ...
}
```

### Integration Testing

Test middleware integration with Gin:

```go
package middleware_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/twistingmercury/mnemonic/internal/middleware"
    "go.opentelemetry.io/otel/trace"
)

func TestTracingMiddleware(t *testing.T) {
    gin.SetMode(gin.TestMode)

    router := gin.New()
    router.Use(middleware.TracingMiddleware("test-service"))
    router.GET("/test", func(c *gin.Context) {
        // Verify span is in context
        span := trace.SpanFromContext(c.Request.Context())
        if !span.SpanContext().IsValid() {
            t.Error("expected valid span in context")
        }
        c.Status(http.StatusOK)
    })

    req := httptest.NewRequest(http.MethodGet, "/test", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("expected status 200, got %d", w.Code)
    }
}
```

### E2E Observability Verification

Verify telemetry emission in E2E tests:

```go
func TestObservabilityEmission(t *testing.T) {
    // Start test server with telemetry
    // Make requests
    // Verify:
    // 1. Prometheus metrics endpoint returns expected metrics
    // 2. Logs contain trace IDs
    // 3. (With collector) Traces are exported
}
```

## Implementation Checklist

### Phase 1A: Foundation

- [ ] Create `internal/config/config.go` with observability configuration
- [ ] Create `internal/telemetry/telemetry.go` wrapping otelx
- [ ] Update `cmd/main/main.go` to initialize telemetry
- [ ] Add required dependencies to `go.mod`

### Phase 1B: Middleware

- [ ] Create `internal/middleware/tracing.go` using otelgin
- [ ] Create `internal/middleware/metrics.go` for request metrics
- [ ] Update `internal/server/server.go` to register middleware
- [ ] Configure otelx logging middleware with skip paths

### Phase 1C: Application Metrics

- [ ] Create `internal/metrics/registry.go`
- [ ] Create `internal/metrics/routing.go`
- [ ] Create `internal/metrics/patterns.go`
- [ ] Create `internal/metrics/database.go`

### Phase 1D: Handler Instrumentation

- [ ] Create `internal/handlers/deps.go` for dependency injection
- [ ] Update handler signatures to accept dependencies
- [ ] Add span creation in handlers for logical operations
- [ ] Add metric recording calls in handlers
- [ ] Use otelgin.Logger for trace-correlated logging

### Phase 1E: Database Instrumentation

- [ ] Create `internal/repository/postgres/instrumented.go`
- [ ] Create `internal/repository/neo4j/instrumented.go`
- [ ] Implement connection pool stat recording
- [ ] Wrap all database calls with instrumentation

### Phase 1F: Testing and Verification

- [ ] Unit tests for metric instruments
- [ ] Integration tests for middleware
- [ ] Verify log output contains trace IDs
- [ ] Verify Prometheus metrics endpoint works
- [ ] Document local development setup (disable/stdout exporters)

---

Copyright (c) 2025 Jeremy K. Johnson. All rights reserved.
