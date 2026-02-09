package metrics_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/metrics"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestNewRouting(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := provider.Meter("test")

	rm, err := metrics.NewRouting(meter)
	require.NoError(t, err)
	assert.NotNil(t, rm)
}

func TestRoutingRecordRoutingDecision(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := provider.Meter("test")

	rm, err := metrics.NewRouting(meter)
	require.NoError(t, err)

	ctx := context.Background()
	rm.RecordRoutingDecision(ctx, "go-engineer")
	rm.RecordRoutingDecision(ctx, "go-engineer")
	rm.RecordRoutingDecision(ctx, "python-engineer")

	var data metricdata.ResourceMetrics
	err = reader.Collect(ctx, &data)
	require.NoError(t, err)

	// Verify metrics were recorded
	assert.NotEmpty(t, data.ScopeMetrics)
	foundRoutingDecisions := false
	for _, sm := range data.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "mnemonic.routing.decisions" {
				foundRoutingDecisions = true
			}
		}
	}
	assert.True(t, foundRoutingDecisions, "routing decisions metric should be recorded")
}

func TestRoutingRecordRuleMatch(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := provider.Meter("test")

	rm, err := metrics.NewRouting(meter)
	require.NoError(t, err)

	ctx := context.Background()
	rm.RecordRuleMatch(ctx, "keyword")
	rm.RecordRuleMatch(ctx, "semantic")
	rm.RecordRuleMatch(ctx, "keyword")

	var data metricdata.ResourceMetrics
	err = reader.Collect(ctx, &data)
	require.NoError(t, err)

	assert.NotEmpty(t, data.ScopeMetrics)
	foundRuleMatches := false
	for _, sm := range data.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "mnemonic.routing.rule_matches" {
				foundRuleMatches = true
			}
		}
	}
	assert.True(t, foundRuleMatches, "rule matches metric should be recorded")
}

func TestRoutingRecordCacheHitAndMiss(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := provider.Meter("test")

	rm, err := metrics.NewRouting(meter)
	require.NoError(t, err)

	ctx := context.Background()
	rm.RecordCacheHit(ctx)
	rm.RecordCacheHit(ctx)
	rm.RecordCacheMiss(ctx)

	var data metricdata.ResourceMetrics
	err = reader.Collect(ctx, &data)
	require.NoError(t, err)

	assert.NotEmpty(t, data.ScopeMetrics)
	foundCacheHits := false
	foundCacheMisses := false
	for _, sm := range data.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "mnemonic.routing.cache_hits" {
				foundCacheHits = true
			}
			if m.Name == "mnemonic.routing.cache_misses" {
				foundCacheMisses = true
			}
		}
	}
	assert.True(t, foundCacheHits, "cache hits metric should be recorded")
	assert.True(t, foundCacheMisses, "cache misses metric should be recorded")
}
