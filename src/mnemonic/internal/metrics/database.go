package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// DatabaseMetrics holds instruments for database-related metrics.
// It tracks connection pool statistics, query latency, and errors.
type DatabaseMetrics struct {
	connectionPoolSize  metric.Int64Gauge
	connectionPoolInUse metric.Int64Gauge
	queryLatency        metric.Float64Histogram
	queryErrors         metric.Int64Counter
}

// NewDatabaseMetrics creates database metric instruments using the provided meter.
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

// RecordPoolStats records connection pool statistics for the specified database.
// Call this periodically (e.g., every 30 seconds) to track pool health.
func (m *DatabaseMetrics) RecordPoolStats(ctx context.Context, database string, size, inUse int64) {
	attrs := metric.WithAttributes(attribute.String("database", database))
	m.connectionPoolSize.Record(ctx, size, attrs)
	m.connectionPoolInUse.Record(ctx, inUse, attrs)
}

// RecordQuery records a database query with its latency and operation type.
// The operation parameter identifies the type of query (e.g., "select", "insert", "update").
func (m *DatabaseMetrics) RecordQuery(ctx context.Context, database, operation string, duration time.Duration) {
	m.queryLatency.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(
		attribute.String("database", database),
		attribute.String("operation", operation),
	))
}

// RecordError records a database query error.
func (m *DatabaseMetrics) RecordError(ctx context.Context, database, operation string) {
	m.queryErrors.Add(ctx, 1, metric.WithAttributes(
		attribute.String("database", database),
		attribute.String("operation", operation),
	))
}

// RecordQueryWithError records a database query, including an error if one occurred.
// This is a convenience method for recording both latency and potential errors.
func (m *DatabaseMetrics) RecordQueryWithError(ctx context.Context, database, operation string, duration time.Duration, err error) {
	m.RecordQuery(ctx, database, operation, duration)
	if err != nil {
		m.RecordError(ctx, database, operation)
	}
}
