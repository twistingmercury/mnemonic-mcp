package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RoutingMetrics holds instruments for routing-related metrics.
// It tracks routing decisions, rule matches, and cache performance.
type RoutingMetrics struct {
	routingDecisions metric.Int64Counter
	ruleMatches      metric.Int64Counter
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
		return nil, fmt.Errorf("routing decisions counter: %w", err)
	}

	ruleMatches, err := meter.Int64Counter(
		"mnemonic.routing.rule_matches",
		metric.WithDescription("Number of rule matches by type"),
		metric.WithUnit("{match}"),
	)
	if err != nil {
		return nil, fmt.Errorf("rule matches counter: %w", err)
	}

	cacheHits, err := meter.Int64Counter(
		"mnemonic.routing.cache_hits",
		metric.WithDescription("Number of routing cache hits"),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		return nil, fmt.Errorf("cache hits counter: %w", err)
	}

	cacheMisses, err := meter.Int64Counter(
		"mnemonic.routing.cache_misses",
		metric.WithDescription("Number of routing cache misses"),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		return nil, fmt.Errorf("cache misses counter: %w", err)
	}

	return &RoutingMetrics{
		routingDecisions: routingDecisions,
		ruleMatches:      ruleMatches,
		cacheHits:        cacheHits,
		cacheMisses:      cacheMisses,
	}, nil
}

// RecordRoutingDecision records that a routing decision was made for the specified agent.
// The agentName should be one of the predefined agent types (bounded cardinality).
// Do not use user-provided or dynamic values to avoid metric explosion.
func (m *RoutingMetrics) RecordRoutingDecision(ctx context.Context, agentName string) {
	m.routingDecisions.Add(ctx, 1, metric.WithAttributes(
		attribute.String("agent", agentName),
	))
}

// RecordRuleMatch records a rule match by type.
// The ruleType should be one of the predefined rule types (bounded cardinality).
// Do not use user-provided or dynamic values to avoid metric explosion.
func (m *RoutingMetrics) RecordRuleMatch(ctx context.Context, ruleType string) {
	m.ruleMatches.Add(ctx, 1, metric.WithAttributes(
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
