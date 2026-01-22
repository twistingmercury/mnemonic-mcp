# Observability Implementation Design

[Back to Architecture Overview](../../architecture/00-overview.md) | [Back to Project README](../../../README.md)

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

Create `internal/config/config.go` to manage observability configuration:

```go
package config

import (
    "os"
    "strconv"

    "github.com/rs/zerolog"
)

// ObservabilityConfig holds all observability-related configuration.
type ObservabilityConfig struct {
    // Service identity
    ServiceName    string
    ServiceVersion string
    Environment    string

    // Logging
    LogLevel zerolog.Level

    // Metrics
    MetricsEnabled bool
    MetricsPort    int
    MetricsPath    string

    // Health
    HealthEnabled bool
    HealthPath    string

    // Tracing
    TracingEnabled  bool
    OTLPEndpoint    string
    OTLPInsecure    bool
    TraceSampleRate float64
}

// DefaultObservabilityConfig returns configuration with sensible defaults.
func DefaultObservabilityConfig() ObservabilityConfig {
    return ObservabilityConfig{
        ServiceName:     "mnemonic",
        ServiceVersion:  version.Version(),
        Environment:     getEnvOrDefault("MNEMONIC_ENV", "development"),
        LogLevel:        parseLogLevel(getEnvOrDefault("MNEMONIC_LOGGING_LEVEL", "info")),
        MetricsEnabled:  getEnvBool("MNEMONIC_OBSERVABILITY_METRICS_ENABLED", true),
        MetricsPort:     getEnvInt("MNEMONIC_OBSERVABILITY_METRICS_PORT", 9090),
        MetricsPath:     getEnvOrDefault("MNEMONIC_OBSERVABILITY_METRICS_PATH", "/metrics"),
        HealthEnabled:   getEnvBool("MNEMONIC_OBSERVABILITY_HEALTH_ENABLED", true),
        HealthPath:      getEnvOrDefault("MNEMONIC_OBSERVABILITY_HEALTH_PATH", "/health"),
        TracingEnabled:  getEnvBool("MNEMONIC_OBSERVABILITY_TRACING_ENABLED", false),
        OTLPEndpoint:    getEnvOrDefault("MNEMONIC_OBSERVABILITY_TRACING_ENDPOINT", ""),
        OTLPInsecure:    getEnvBool("MNEMONIC_OBSERVABILITY_TRACING_OTLP_INSECURE", true),
        TraceSampleRate: getEnvFloat("MNEMONIC_OBSERVABILITY_TRACING_SAMPLE_RATE", 0.1),
    }
}

func getEnvOrDefault(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
    if val := os.Getenv(key); val != "" {
        b, err := strconv.ParseBool(val)
        if err == nil {
            return b
        }
    }
    return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
    if val := os.Getenv(key); val != "" {
        i, err := strconv.Atoi(val)
        if err == nil {
            return i
        }
    }
    return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
    if val := os.Getenv(key); val != "" {
        f, err := strconv.ParseFloat(val, 64)
        if err == nil {
            return f
        }
    }
    return defaultVal
}

func parseLogLevel(level string) zerolog.Level {
    l, err := zerolog.ParseLevel(level)
    if err != nil {
        return zerolog.InfoLevel
    }
    return l
}
```

### Telemetry Initialization

Create `internal/telemetry/telemetry.go`:

```go
package telemetry

import (
    "context"
    "fmt"

    "github.com/twistingmercury/otelx"
    "github.com/twistingmercury/mnemonic/internal/config"
)

// Telemetry wraps the otelx.Telemetry with application-specific helpers.
type Telemetry struct {
    *otelx.Telemetry
    cfg config.ObservabilityConfig
}

// Initialize creates and configures the telemetry system using otelx.
func Initialize(ctx context.Context, cfg config.ObservabilityConfig) (*Telemetry, error) {
    opts := buildOptions(cfg)

    tel, err := otelx.Initialize(ctx, opts...)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize telemetry: %w", err)
    }

    return &Telemetry{
        Telemetry: tel,
        cfg:       cfg,
    }, nil
}

func buildOptions(cfg config.ObservabilityConfig) []otelx.Option {
    opts := []otelx.Option{
        otelx.WithService(cfg.ServiceName, cfg.ServiceVersion, cfg.Environment),
        otelx.WithLogLevel(cfg.LogLevel),
    }

    // Metrics configuration
    if cfg.MetricsEnabled {
        opts = append(opts, otelx.WithMetrics(cfg.MetricsPort))
        if cfg.MetricsPath != "/metrics" {
            opts = append(opts, otelx.WithMetricsPath(cfg.MetricsPath))
        }
    } else {
        opts = append(opts, otelx.WithoutMetrics())
    }

    // Tracing configuration
    if cfg.TracingEnabled {
        opts = append(opts, otelx.WithTracing())
        opts = append(opts, otelx.WithTraceSampleRate(cfg.TraceSampleRate))
        opts = append(opts, otelx.WithOTLPEndpoint(cfg.OTLPEndpoint))
        if cfg.OTLPInsecure {
            opts = append(opts, otelx.WithOTLPInsecure())
        }
    } else {
        opts = append(opts, otelx.WithoutTracing())
    }

    return opts
}

// Shutdown gracefully shuts down telemetry, flushing pending data.
func (t *Telemetry) Shutdown(ctx context.Context) error {
    return t.Telemetry.Shutdown(ctx)
}
```

### Main Function Updates

Update `cmd/main/main.go`:

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/spf13/pflag"
    "github.com/twistingmercury/mnemonic/cmd/version"
    "github.com/twistingmercury/mnemonic/internal/config"
    "github.com/twistingmercury/mnemonic/internal/server"
    "github.com/twistingmercury/mnemonic/internal/telemetry"
)

var verFlag = pflag.Bool("version", false, "Displays current version information for mnemonic")
var healthFlag = pflag.Bool("health", false, "Get the current health of the service")

func main() {
    pflag.Parse()

    if *verFlag {
        println(version.Print())
        os.Exit(0)
    }

    if *healthFlag {
        err := server.CheckHealth()
        if err != nil {
            log.Fatal(err)
        }
        os.Exit(0)
    }

    // Initialize telemetry
    ctx := context.Background()
    cfg := config.DefaultObservabilityConfig()

    tel, err := telemetry.Initialize(ctx, cfg)
    if err != nil {
        log.Fatalf("failed to initialize telemetry: %v", err)
    }
    defer func() {
        if err := tel.Shutdown(ctx); err != nil {
            log.Printf("telemetry shutdown error: %v", err)
        }
    }()

    tel.Logger.Info().
        Str("version", cfg.ServiceVersion).
        Str("environment", cfg.Environment).
        Msg("mnemonic starting")

    if err := server.ListenAndServe(tel); err != nil {
        tel.Logger.Error().Err(err).Msg("server exited with error")
    }

    tel.Logger.Info().Msg("mnemonic shutdown complete")
}
```

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

Since otelx does not provide HTTP tracing middleware, use `otelgin` from the OpenTelemetry contrib packages:

Create `internal/middleware/tracing.go`:

```go
package middleware

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// TracingMiddleware returns Gin middleware that creates spans for HTTP requests.
// It uses W3C Trace Context for trace propagation.
func TracingMiddleware(serviceName string) gin.HandlerFunc {
    return otelgin.Middleware(serviceName,
        otelgin.WithFilter(func(req *http.Request) bool {
            // Skip tracing for health checks to reduce noise
            return req.URL.Path != "/health"
        }),
    )
}
```

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

Since otelx does not provide request metrics middleware, create custom middleware:

Create `internal/middleware/metrics.go`:

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

// RoutingMetrics holds instruments for routing-related metrics.
type RoutingMetrics struct {
    routingDecisions metric.Int64Counter
    patternMatches   metric.Int64Counter
    cacheHits        metric.Int64Counter
    cacheMisses      metric.Int64Counter
}

// NewRoutingMetrics creates routing metric instruments.
func NewRoutingMetrics(meter metric.Meter) (*RoutingMetrics, error) {
    routingDecisions, err := meter.Int64Counter(
        "mnemonic.routing.decisions",
        metric.WithDescription("Number of routing decisions made"),
        metric.WithUnit("{decision}"),
    )
    if err != nil {
        return nil, err
    }

    patternMatches, err := meter.Int64Counter(
        "mnemonic.routing.pattern_matches",
        metric.WithDescription("Number of pattern matches by rule type"),
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

    return &RoutingMetrics{
        routingDecisions: routingDecisions,
        patternMatches:   patternMatches,
        cacheHits:        cacheHits,
        cacheMisses:      cacheMisses,
    }, nil
}

// RecordRoutingDecision records a routing decision was made.
func (m *RoutingMetrics) RecordRoutingDecision(ctx context.Context, agentName string) {
    m.routingDecisions.Add(ctx, 1, metric.WithAttributes(
        attribute.String("agent", agentName),
    ))
}

// RecordPatternMatch records a pattern match by rule type.
func (m *RoutingMetrics) RecordPatternMatch(ctx context.Context, ruleType string) {
    m.patternMatches.Add(ctx, 1, metric.WithAttributes(
        attribute.String("rule_type", ruleType),
    ))
}

// RecordCacheHit records a cache hit.
func (m *RoutingMetrics) RecordCacheHit(ctx context.Context) {
    m.cacheHits.Add(ctx, 1)
}

// RecordCacheMiss records a cache miss.
func (m *RoutingMetrics) RecordCacheMiss(ctx context.Context) {
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

// PatternMetrics holds instruments for pattern retrieval metrics.
type PatternMetrics struct {
    queryLatency     metric.Float64Histogram
    patternsReturned metric.Int64Histogram
}

// NewPatternMetrics creates pattern metric instruments.
func NewPatternMetrics(meter metric.Meter) (*PatternMetrics, error) {
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

    return &PatternMetrics{
        queryLatency:     queryLatency,
        patternsReturned: patternsReturned,
    }, nil
}

// RecordQuery records a pattern query with its latency and result count.
func (m *PatternMetrics) RecordQuery(ctx context.Context, database string, duration time.Duration, count int) {
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

// DatabaseMetrics holds instruments for database-related metrics.
type DatabaseMetrics struct {
    connectionPoolSize   metric.Int64Gauge
    connectionPoolInUse  metric.Int64Gauge
    queryLatency         metric.Float64Histogram
    queryErrors          metric.Int64Counter
}

// NewDatabaseMetrics creates database metric instruments.
func NewDatabaseMetrics(meter metric.Meter) (*DatabaseMetrics, error) {
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

    return &DatabaseMetrics{
        connectionPoolSize:  connectionPoolSize,
        connectionPoolInUse: connectionPoolInUse,
        queryLatency:        queryLatency,
        queryErrors:         queryErrors,
    }, nil
}

// RecordPoolStats records connection pool statistics.
func (m *DatabaseMetrics) RecordPoolStats(ctx context.Context, database string, size, inUse int64) {
    attrs := metric.WithAttributes(attribute.String("database", database))
    m.connectionPoolSize.Record(ctx, size, attrs)
    m.connectionPoolInUse.Record(ctx, inUse, attrs)
}

// RecordQuery records a database query with latency.
func (m *DatabaseMetrics) RecordQuery(ctx context.Context, database, operation string, duration time.Duration) {
    m.queryLatency.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(
        attribute.String("database", database),
        attribute.String("operation", operation),
    ))
}

// RecordError records a database error.
func (m *DatabaseMetrics) RecordError(ctx context.Context, database, operation string) {
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
    Routing  *RoutingMetrics
    Patterns *PatternMetrics
    Database *DatabaseMetrics
}

// NewRegistry creates all metric instruments.
func NewRegistry(meter metric.Meter) (*Registry, error) {
    routing, err := NewRoutingMetrics(meter)
    if err != nil {
        return nil, err
    }

    patterns, err := NewPatternMetrics(meter)
    if err != nil {
        return nil, err
    }

    database, err := NewDatabaseMetrics(meter)
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

otelx provides `middleware/gin.LoggingMiddleware` for request logging with trace correlation:

Update `internal/server/server.go`:

```go
package server

import (
    "github.com/gin-gonic/gin"
    otelgin "github.com/twistingmercury/otelx/middleware/gin"
    "github.com/twistingmercury/mnemonic/internal/telemetry"
)

func setupRouter(tel *telemetry.Telemetry) *gin.Engine {
    router := gin.New() // Use gin.New() instead of gin.Default() to avoid duplicate logging

    // Recovery middleware (keep this)
    router.Use(gin.Recovery())

    // otelx logging middleware with trace correlation
    router.Use(otelgin.LoggingMiddleware(tel.Telemetry,
        otelgin.WithSkipPaths([]string{"/health"}),
        otelgin.WithRequestHeaders([]string{"X-Request-ID", "X-Correlation-ID"}),
    ))

    return router
}
```

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

### Handler Dependencies

Create a dependencies struct to inject telemetry into handlers:

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
        deps.Metrics.Routing.RecordPatternMatch(ctx, rule.Type)
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

### Postgres/PGVector Instrumentation

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
    metrics *metrics.DatabaseMetrics
}

// NewInstrumentedPool creates an instrumented database pool.
func NewInstrumentedPool(pool *pgxpool.Pool, metrics *metrics.DatabaseMetrics) *InstrumentedPool {
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

### Neo4j Instrumentation

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
    metrics *metrics.DatabaseMetrics
}

// NewInstrumentedSession creates an instrumented Neo4j session.
func NewInstrumentedSession(session neo4j.SessionWithContext, metrics *metrics.DatabaseMetrics) *InstrumentedSession {
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

func TestRoutingMetrics(t *testing.T) {
    // Create a test meter provider with in-memory reader
    reader := metric.NewManualReader()
    provider := metric.NewMeterProvider(metric.WithReader(reader))
    meter := provider.Meter("test")

    // Create metrics
    rm, err := mmetrics.NewRoutingMetrics(meter)
    if err != nil {
        t.Fatalf("failed to create routing metrics: %v", err)
    }

    // Record some metrics
    ctx := context.Background()
    rm.RecordRoutingDecision(ctx, "go-engineer")
    rm.RecordPatternMatch(ctx, "keyword")
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
