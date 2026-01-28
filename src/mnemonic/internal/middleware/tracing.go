// Package middleware provides HTTP middleware for the Mnemonic server.
// It includes tracing and metrics middleware for observability.
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
