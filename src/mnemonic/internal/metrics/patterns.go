package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// PatternMetrics holds instruments for pattern retrieval metrics.
// It tracks query latency and the number of patterns returned per query.
type PatternMetrics struct {
	queryLatency     metric.Float64Histogram
	patternsReturned metric.Int64Histogram
}

// NewPatternMetrics creates pattern metric instruments using the provided meter.
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
// The database parameter identifies the data source (e.g., "postgres", "pgvector", "neo4j").
func (m *PatternMetrics) RecordQuery(ctx context.Context, database string, duration time.Duration, count int) {
	attrs := metric.WithAttributes(attribute.String("database", database))
	m.queryLatency.Record(ctx, float64(duration.Milliseconds()), attrs)
	m.patternsReturned.Record(ctx, int64(count), attrs)
}

// RecordQueryLatency records only the query latency without pattern count.
// Use this when the pattern count is not yet known.
func (m *PatternMetrics) RecordQueryLatency(ctx context.Context, database string, duration time.Duration) {
	m.queryLatency.Record(ctx, float64(duration.Milliseconds()),
		metric.WithAttributes(attribute.String("database", database)))
}

// RecordPatternsReturned records only the number of patterns returned.
// Use this when recording pattern count separately from latency.
func (m *PatternMetrics) RecordPatternsReturned(ctx context.Context, database string, count int) {
	m.patternsReturned.Record(ctx, int64(count),
		metric.WithAttributes(attribute.String("database", database)))
}
