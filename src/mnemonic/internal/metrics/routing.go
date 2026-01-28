package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RoutingMetrics holds instruments for routing-related metrics.
// It tracks routing decisions, pattern matches, and cache performance.
type RoutingMetrics struct {
	routingDecisions metric.Int64Counter
	patternMatches   metric.Int64Counter
	cacheHits        metric.Int64Counter
	cacheMisses      metric.Int64Counter
}

// NewRoutingMetrics creates routing metric instruments using the provided meter.
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

// RecordRoutingDecision records that a routing decision was made for the specified agent.
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

// RecordCacheHit records a routing cache hit.
func (m *RoutingMetrics) RecordCacheHit(ctx context.Context) {
	m.cacheHits.Add(ctx, 1)
}

// RecordCacheMiss records a routing cache miss.
func (m *RoutingMetrics) RecordCacheMiss(ctx context.Context) {
	m.cacheMisses.Add(ctx, 1)
}
