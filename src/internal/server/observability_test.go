package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twistingmercury/mnemonic/internal/middleware"
	"github.com/twistingmercury/otelx/v2"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// useTraceExporter installs an in-memory tracer provider as the global for
// the test, the way otelx does in production, and returns the span exporter.
func useTraceExporter(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	old := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(old)
		require.NoError(t, tp.Shutdown(context.Background()))
	})
	return exporter
}

// newTestRouter builds the production router with /health and /version routes.
func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	requestMetrics, err := middleware.NewRequestMetrics(sdkmetric.NewMeterProvider().Meter("test"))
	require.NoError(t, err)
	router := setupRouter(&otelx.Telemetry{Logger: zerolog.Nop()}, requestMetrics)
	for _, path := range []string{"/health", "/version"} {
		router.GET(path, func(c *gin.Context) { c.Status(http.StatusOK) })
	}
	return router
}

// TestTracingSkipsHealth verifies that a normal route records a server span
// and the /health probe does not.
func TestTracingSkipsHealth(t *testing.T) {
	exporter := useTraceExporter(t)
	router := newTestRouter(t)

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health", nil))
	assert.Empty(t, exporter.GetSpans(), "/health must not be traced")

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/version", nil))
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, trace.SpanKindServer, spans[0].SpanKind)
	assert.Contains(t, spans[0].Name, "/version")
}
