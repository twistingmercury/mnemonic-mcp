package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Routing holds instruments for routing-related metrics.
// It tracks routing decisions, rule matches, and cache performance.
type Routing struct {
	routingDecisions metric.Int64Counter
	ruleMatches      metric.Int64Counter
	cacheHits        metric.Int64Counter
	cacheMisses      metric.Int64Counter
}

// NewRouting creates routing metric instruments using the provided meter.
func NewRouting(meter metric.Meter) (*Routing, error) {
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

	return &Routing{
		routingDecisions: routingDecisions,
		ruleMatches:      ruleMatches,
		cacheHits:        cacheHits,
		cacheMisses:      cacheMisses,
	}, nil
}

// RecordRoutingDecision records that a routing decision was made for the specified agent.
// The agentName should be one of the predefined agent types (bounded cardinality).
// Do not use user-provided or dynamic values to avoid metric explosion.
func (m *Routing) RecordRoutingDecision(ctx context.Context, agentName string) {
	m.routingDecisions.Add(ctx, 1, metric.WithAttributes(
		attribute.String("agent", agentName),
	))
}

// RecordRuleMatch records a rule match by type.
// The ruleType should be one of the predefined rule types (bounded cardinality).
// Do not use user-provided or dynamic values to avoid metric explosion.
func (m *Routing) RecordRuleMatch(ctx context.Context, ruleType string) {
	m.ruleMatches.Add(ctx, 1, metric.WithAttributes(
		attribute.String("rule_type", ruleType),
	))
}

// RecordCacheHit records a routing cache hit.
func (m *Routing) RecordCacheHit(ctx context.Context) {
	m.cacheHits.Add(ctx, 1)
}

// RecordCacheMiss records a routing cache miss.
func (m *Routing) RecordCacheMiss(ctx context.Context) {
	m.cacheMisses.Add(ctx, 1)
}
