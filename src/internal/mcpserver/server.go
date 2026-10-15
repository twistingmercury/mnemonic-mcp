package mcpserver

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/twistingmercury/mnemonic/internal/config"
	"github.com/twistingmercury/mnemonic/internal/metrics"
	"github.com/twistingmercury/mnemonic/internal/version"
)

// NewMCPServer creates a configured MCP server with all 3 pattern search
// tools registered. The returned server is ready to be wrapped in an HTTP
// handler via NewMCPHTTPHandler. Every tool call is traced and, when
// toolMetrics is non-nil, counted and timed.
func NewMCPServer(deps ToolDependencies, logger zerolog.Logger, mcpCfg config.MCPConfig, toolMetrics *metrics.Tools) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "mnemonic",
		Version: version.Version(),
	}, &mcp.ServerOptions{
		SchemaCache: mcp.NewSchemaCache(),
	})

	server.AddReceivingMiddleware(toolTelemetry(toolMetrics))
	mcp.AddTool(server, searchPatternsTool, handleSearchPatterns(deps, logger, mcpCfg.DefaultSearchThreshold))
	mcp.AddTool(server, findRelatedPatternsTool, handleFindRelatedPatterns(deps, logger))
	mcp.AddTool(server, getPatternTool, handleGetPattern(deps, logger))

	return server
}

// NewMCPHTTPHandler creates a StreamableHTTPHandler wrapping the MCP server.
// The handler implements http.Handler and serves JSON-RPC 2.0 over HTTP POST.
// Stateless mode is enabled because all tools are stateless.
func NewMCPHTTPHandler(server *mcp.Server) *mcp.StreamableHTTPHandler {
	return mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless: true,
	})
}

// NewMCPHTTPServer creates the net/http.Server for the MCP endpoint. Requests
// get an otelhttp server span (continuing any incoming W3C trace context) and
// the standard http.server.* metrics.
// The caller is responsible for starting and stopping the server
// (typically via errgroup).
func NewMCPHTTPServer(mcpCfg config.MCPConfig, host string, mcpHandler http.Handler) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/mcp", otelhttp.NewHandler(mcpHandler, "/mcp"))

	return &http.Server{
		Addr:         mcpCfg.Address(host),
		Handler:      mux,
		ReadTimeout:  mcpCfg.ReadTimeout,
		WriteTimeout: mcpCfg.WriteTimeout,
		IdleTimeout:  mcpCfg.IdleTimeout,
	}
}
