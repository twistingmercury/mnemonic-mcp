package mcpserver

import (
	"context"
	"errors"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/twistingmercury/mnemonic/internal/metrics"
)

var tracer = otel.Tracer("mnemonic/mcp")

// Tool call outcomes recorded on spans and metrics.
const (
	outcomeOK          = "ok"
	outcomeInvalid     = "invalid_input"
	outcomeNotFound    = "not_found"
	outcomeUnavailable = "unavailable"
)

// toolTelemetry returns middleware that gives every tools/call request a span
// (a child of the /mcp HTTP span) and, when toolMetrics is non-nil, a count
// and latency record by tool and outcome. Only unavailable outcomes mark the
// span as an error; invalid input and unknown patterns are caller mistakes.
func toolTelemetry(toolMetrics *metrics.Tools) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			call, ok := req.(*mcp.CallToolRequest)
			if !ok {
				return next(ctx, method, req)
			}

			tool := call.Params.Name
			start := time.Now()
			ctx, span := tracer.Start(ctx, "tools/call "+tool,
				trace.WithAttributes(attribute.String("mcp.tool.name", tool)),
			)
			defer span.End()

			res, err := next(ctx, method, req)

			toolErr := err
			if r, ok := res.(*mcp.CallToolResult); ok && r != nil && r.IsError && toolErr == nil {
				toolErr = r.GetError()
				if toolErr == nil {
					toolErr = errors.New("tool returned an error result")
				}
			}

			outcome := toolOutcome(toolErr)
			span.SetAttributes(attribute.String("mcp.tool.outcome", outcome))
			if toolErr != nil {
				span.RecordError(toolErr)
				if outcome == outcomeUnavailable {
					span.SetStatus(codes.Error, toolErr.Error())
				}
			}
			if toolMetrics != nil {
				toolMetrics.Record(ctx, tool, outcome, time.Since(start))
			}
			return res, err
		}
	}
}

// toolOutcome maps a tool error to a fixed outcome value.
func toolOutcome(err error) string {
	switch {
	case err == nil:
		return outcomeOK
	case errors.Is(err, ErrInvalidInput):
		return outcomeInvalid
	case errors.Is(err, ErrPatternNotFound):
		return outcomeNotFound
	default:
		return outcomeUnavailable
	}
}

// withTrace adds the active trace and span IDs to logger, so tool logs can be
// joined to their trace.
func withTrace(ctx context.Context, logger zerolog.Logger) *zerolog.Logger {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		logger = logger.With().Str("trace_id", sc.TraceID().String()).Str("span_id", sc.SpanID().String()).Logger()
	}
	return &logger
}
