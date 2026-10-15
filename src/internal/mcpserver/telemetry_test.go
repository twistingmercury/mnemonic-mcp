package mcpserver

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/twistingmercury/mnemonic/internal/config"
	"github.com/twistingmercury/mnemonic/internal/metrics"
	"github.com/twistingmercury/mnemonic/internal/service"
	searchsvc "github.com/twistingmercury/mnemonic/internal/service/search"
)

// TestToolCallTelemetry drives real tool calls over Streamable HTTP and checks
// the HTTP span, the tool span beneath it, and the tool metrics. It swaps the
// global tracer provider, so it must not run in parallel.
func TestToolCallTelemetry(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prevTP, prevProp := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTP)
		otel.SetTextMapPropagator(prevProp)
	})

	reader := sdkmetric.NewManualReader()
	toolMetrics, err := metrics.NewTools(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test"))
	require.NoError(t, err)

	missing := uuid.New()
	deps := new(mockToolDeps)
	deps.On("SearchPatterns", mock.Anything, mock.Anything).Return(&searchsvc.SearchResult{}, nil)
	deps.On("GetPatternWithGraph", mock.Anything, missing).Return(nil, nil, service.ErrNotFound)
	deps.On("FindRelatedPatterns", mock.Anything, mock.Anything, mock.Anything).
		Return("", nil, errors.Join(service.ErrServiceUnavailable, errors.New("neo4j down")))

	cfg := config.MCPConfig{DefaultSearchThreshold: 0.5}
	srv := NewMCPHTTPServer(cfg, "localhost", NewMCPHTTPHandler(NewMCPServer(deps, noopLogger(), cfg, toolMetrics)))
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)

	ctx := context.Background()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL + "/mcp", MaxRetries: -1}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	call := func(name string, args map[string]any) {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		require.NoError(t, err)
	}
	call("search_patterns", map[string]any{"query": "retry with backoff"})
	call("get_pattern", map[string]any{"id": missing.String()})
	call("get_pattern", map[string]any{"id": "not-a-uuid"})
	call("find_related_patterns", map[string]any{"pattern_id": uuid.NewString()})

	// Metrics: one record per call, by tool and outcome.
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))
	got := map[string]int64{}
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "mnemonic.mcp.tool_calls" {
				continue
			}
			for _, dp := range m.Data.(metricdata.Sum[int64]).DataPoints {
				tool, _ := dp.Attributes.Value("tool")
				outcome, _ := dp.Attributes.Value("outcome")
				got[tool.AsString()+"/"+outcome.AsString()] = dp.Value
			}
		}
	}
	assert.Equal(t, map[string]int64{
		"search_patterns/ok":                1,
		"get_pattern/not_found":             1,
		"get_pattern/invalid_input":         1,
		"find_related_patterns/unavailable": 1,
	}, got)

	// Traces: each tool span is a child of an /mcp HTTP server span. otelhttp
	// ends that span after the handler returns, which can be after the client
	// has read the reply, so wait until every tool span's parent has ended.
	var spans []sdktrace.ReadOnlySpan
	byID := map[string]sdktrace.ReadOnlySpan{}
	require.Eventually(t, func() bool {
		spans = recorder.Ended()
		clear(byID)
		for _, s := range spans {
			byID[s.SpanContext().SpanID().String()] = s
		}
		for _, s := range spans {
			if _, ok := byID[s.Parent().SpanID().String()]; s.Parent().IsValid() && !ok {
				return false
			}
		}
		return true
	}, 2*time.Second, 5*time.Millisecond)
	status := map[string]codes.Code{}
	for _, s := range spans {
		if s.Name() != "tools/call search_patterns" && s.Name() != "tools/call find_related_patterns" {
			continue
		}
		parent, ok := byID[s.Parent().SpanID().String()]
		require.True(t, ok, "%s has a recorded parent", s.Name())
		assert.Equal(t, "POST /mcp", parent.Name())
		assert.Equal(t, s.SpanContext().TraceID(), parent.SpanContext().TraceID())
		status[s.Name()] = s.Status().Code
	}
	assert.Equal(t, map[string]codes.Code{
		"tools/call search_patterns":       codes.Unset,
		"tools/call find_related_patterns": codes.Error,
	}, status)
}

func TestToolOutcome(t *testing.T) {
	t.Parallel()
	assert.Equal(t, outcomeOK, toolOutcome(nil))
	assert.Equal(t, outcomeInvalid, toolOutcome(ErrInvalidInput))
	assert.Equal(t, outcomeNotFound, toolOutcome(mapServiceError(service.ErrNotFound)))
	assert.Equal(t, outcomeUnavailable, toolOutcome(mapServiceError(errors.New("boom"))))
}
