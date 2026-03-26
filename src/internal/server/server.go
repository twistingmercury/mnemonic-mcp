package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/twistingmercury/mnemonic/internal/config"
	"github.com/twistingmercury/mnemonic/internal/database"
	"github.com/twistingmercury/mnemonic/internal/handlers/operations"
	"github.com/twistingmercury/mnemonic/internal/health"
	"github.com/twistingmercury/mnemonic/internal/mcpserver"
	"github.com/twistingmercury/mnemonic/internal/middleware"
	agentrepo "github.com/twistingmercury/mnemonic/internal/repository/agent"
	chunkrepo "github.com/twistingmercury/mnemonic/internal/repository/chunk"
	graphrepo "github.com/twistingmercury/mnemonic/internal/repository/graph"
	patternrepo "github.com/twistingmercury/mnemonic/internal/repository/pattern"
	openaisvc "github.com/twistingmercury/mnemonic/internal/service/openai"
	patternsvc "github.com/twistingmercury/mnemonic/internal/service/pattern"
	searchsvc "github.com/twistingmercury/mnemonic/internal/service/search"
	"github.com/twistingmercury/mnemonic/internal/telemetry"
	otelxgin "github.com/twistingmercury/otelx/middleware/gin"
)

// ListenAndServe starts the mnemonic server. It initializes telemetry,
// establishes database connections, wires dependencies for MCP search tools,
// and runs the operations HTTP server plus MCP server. It blocks until a
// shutdown signal is received or a component returns a fatal error.
func ListenAndServe(cfg *config.MnemonicConfig) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize telemetry.
	tel, err := telemetry.Initialize(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	defer func() {
		logger := tel.Logger()
		if shutdownErr := tel.Shutdown(context.Background()); shutdownErr != nil {
			logger.Error().Err(shutdownErr).Msg("telemetry shutdown error")
		}
	}()

	logger := tel.Logger()
	logger.Info().
		Str("host", cfg.Server.Host).
		Int("admin_port", cfg.Server.Port).
		Int("mcp_port", cfg.MCP.Port).
		Bool("metrics_enabled", cfg.Observability.Metrics.Enabled).
		Bool("tracing_enabled", cfg.Observability.Tracing.Enabled).
		Msg("mnemonic starting")

	// Establish database connections.
	pgPool, neo4jDriver, err := openDatabases(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer closeDatabases(pgPool, neo4jDriver, logger)

	// Initialize health checks with real database connections.
	if err := health.Initialize(health.Dependencies{
		PGPool:      pgPool,
		Neo4jDriver: neo4jDriver,
	}); err != nil {
		return fmt.Errorf("failed to initialize health checks: %w", err)
	}

	// Wire MCP dependencies.
	toolDeps, err := wireDependencies(pgPool, neo4jDriver, cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to wire dependencies: %w", err)
	}

	// Build operations HTTP server.
	requestMetrics, err := middleware.NewRequestMetrics(tel.Meter("mnemonic/http"))
	if err != nil {
		return fmt.Errorf("failed to create request metrics: %w", err)
	}
	router := setupRouter(tel, requestMetrics)
	operations.SetupHandlers(router, health.Descriptors())
	adminServer := CreateHTTPServer(router, cfg)

	// Build the MCP HTTP server.
	mcpSrv := mcpserver.NewMCPServer(toolDeps, logger, cfg.MCP)
	mcpHandler := mcpserver.NewMCPHTTPHandler(mcpSrv)
	mcpHTTPServer := mcpserver.NewMCPHTTPServer(cfg.MCP, cfg.Server.Host, mcpHandler)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return runHTTPServer(gCtx, adminServer, cfg, logger, "operations")
	})
	g.Go(func() error {
		return runMCPServer(gCtx, mcpHTTPServer, logger)
	})

	if err := g.Wait(); err != nil {
		return err
	}

	logger.Info().Msg("mnemonic shutdown complete")
	return nil
}

// openDatabases creates the Postgres pool and Neo4j driver, logging safe
// connection details. Returns both connections or an error if either fails.
func openDatabases(ctx context.Context, cfg *config.MnemonicConfig, logger zerolog.Logger) (*pgxpool.Pool, neo4j.DriverWithContext, error) {
	logger.Info().
		Str("dsn", cfg.Database.Postgres.SafeDSN()).
		Msg("connecting to PostgreSQL")

	pgPool, err := database.NewPostgresPool(ctx, cfg.Database.Postgres)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	logger.Info().Msg("PostgreSQL connected")

	logger.Info().
		Str("uri", cfg.Database.Neo4j.SafeURI()).
		Str("database", cfg.Database.Neo4j.Database).
		Msg("connecting to Neo4j")

	neo4jDriver, err := database.NewNeo4jDriver(ctx, cfg.Database.Neo4j)
	if err != nil {
		pgPool.Close()
		return nil, nil, fmt.Errorf("failed to connect to Neo4j: %w", err)
	}
	logger.Info().Msg("Neo4j connected")

	return pgPool, neo4jDriver, nil
}

// closeDatabases closes both database connections, logging any errors.
func closeDatabases(pgPool *pgxpool.Pool, neo4jDriver neo4j.DriverWithContext, logger zerolog.Logger) {
	pgPool.Close()
	logger.Debug().Msg("PostgreSQL pool closed")

	if err := neo4jDriver.Close(context.Background()); err != nil {
		logger.Error().Err(err).Msg("neo4j driver close error")
	} else {
		logger.Debug().Msg("Neo4j driver closed")
	}
}

// wireDependencies creates the repositories and read-side services required by
// the MCP search tools and returns MCP ToolDependencies.
func wireDependencies(
	pgPool *pgxpool.Pool,
	neo4jDriver neo4j.DriverWithContext,
	cfg *config.MnemonicConfig,
	logger zerolog.Logger,
) (mcpserver.ToolDependencies, error) {
	// Repositories.
	agentRepo := agentrepo.NewRepository(pgPool)
	patternRepo := patternrepo.NewRepository(pgPool)
	graphRepo := graphrepo.NewRepository(neo4jDriver, cfg.Database.Neo4j.Database)
	chunkRepo := chunkrepo.NewRepository(pgPool)

	// External services.
	embeddingSvc := openaisvc.NewEmbeddingService(cfg.OpenAI)

	// Domain services used by MCP tools.
	searchSvc := searchsvc.New(embeddingSvc, patternRepo, agentRepo, chunkRepo, graphRepo, logger)
	patternSvc := patternsvc.New(patternRepo, nil, graphRepo, agentRepo, pgPool, chunkRepo, logger)

	// MCP facade.
	toolDeps := mcpserver.NewToolDependencies(searchSvc, patternSvc)

	return toolDeps, nil
}

// runHTTPServer starts the operations HTTP server and gracefully shuts it down
// when the context is cancelled.
func runHTTPServer(ctx context.Context, srv *http.Server, cfg *config.MnemonicConfig, logger zerolog.Logger, name string) error {
	errCh := make(chan error, 1)

	go func() {
		logger.Info().
			Str("addr", srv.Addr).
			Str("component", name).
			Msg("HTTP server listening")

		var err error
		if cfg.Server.TLS.Enabled {
			err = srv.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("%s server error: %w", name, err)
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	logger.Info().Str("component", name).Msg("shutting down HTTP server")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("%s shutdown error: %w", name, err)
	}
	return nil
}

// runMCPServer starts the MCP HTTP server and gracefully shuts it down when
// the context is cancelled.
func runMCPServer(ctx context.Context, srv *http.Server, logger zerolog.Logger) error {
	errCh := make(chan error, 1)

	go func() {
		logger.Info().
			Str("addr", srv.Addr).
			Str("component", "mcp").
			Msg("MCP server listening")

		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("mcp server error: %w", err)
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	// Use a fixed 5s timeout for MCP shutdown; it has no long-running requests.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), mcpShutdownTimeout)
	defer cancel()

	logger.Info().Str("component", "mcp").Msg("shutting down MCP server")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("mcp shutdown error: %w", err)
	}
	return nil
}

// mcpShutdownTimeout is the grace period for MCP server shutdown.
const mcpShutdownTimeout = 5 * time.Second

// setupRouter creates and configures the Gin router with middleware.
func setupRouter(tel *telemetry.Telemetry, requestMetrics *middleware.RequestMetrics) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// Correlation ID middleware: echo X-Request-ID from request to response.
	router.Use(func(c *gin.Context) {
		if rid := c.GetHeader("X-Request-ID"); rid != "" {
			c.Header("X-Request-ID", rid)
		}
		c.Next()
	})

	skipPaths := middleware.DefaultSkipPaths
	router.Use(middleware.TracingMiddlewareWithSkipPaths("mnemonic", skipPaths))
	router.Use(otelxgin.LoggingMiddleware(tel.Otelx(),
		otelxgin.WithSkipPaths("/health", "/metrics"),
		otelxgin.WithRequestHeaders("X-Request-ID", "X-Correlation-ID"),
	))
	router.Use(requestMetrics.MiddlewareWithSkipPaths(skipPaths))

	return router
}

// CreateHTTPServer creates an operations http.Server from config.
func CreateHTTPServer(r *gin.Engine, cfg *config.MnemonicConfig) *http.Server {
	return &http.Server{
		Addr:           cfg.Server.Address(),
		Handler:        r,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: 1 << 20,
	}
}
