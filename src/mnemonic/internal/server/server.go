package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/mnemonic/internal/config"
	"github.com/twistingmercury/mnemonic/internal/handlers/operations"
	"github.com/twistingmercury/mnemonic/internal/middleware"
	"github.com/twistingmercury/mnemonic/internal/telemetry"
	otelxgin "github.com/twistingmercury/otelx/middleware/gin"
)

// // ListenAndServe starts the server using configuration loaded from config sources.
// func ListenAndServe() error {
// 	cfg, err := config.Load()
// 	if err != nil {
// 		return fmt.Errorf("failed to load configuration: %w", err)
// 	}

// 	return ListenAndServeWithConfig(cfg)
// }

// ListenAndServeWithConfig starts the server using the provided configuration.
func ListenAndServe(cfg *config.MnemonicConfig) error {
	shutdown, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Initialize telemetry
	tel, err := telemetry.Initialize(shutdown, cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}
	defer func() {
		logger := tel.Logger() // Capture before shutdown to avoid nil pointer if shutdown fails
		if shutdownErr := tel.Shutdown(context.Background()); shutdownErr != nil {
			logger.Error().Err(shutdownErr).Msg("telemetry shutdown error")
		}
	}()

	logger := tel.Logger()
	logger.Info().
		Str("host", cfg.Server.Host).
		Int("port", cfg.Server.Port).
		Bool("metrics_enabled", cfg.Observability.Metrics.Enabled).
		Bool("tracing_enabled", cfg.Observability.Tracing.Enabled).
		Msg("mnemonic starting")

	// Create request metrics middleware
	requestMetrics, err := middleware.NewRequestMetrics(tel.Meter("mnemonic/http"))
	if err != nil {
		return fmt.Errorf("failed to create request metrics: %w", err)
	}

	logger.Debug().Msg("metrics registry initialized")

	router := setupRouter(tel, requestMetrics)

	operations.SetupHandlers(router)

	server := CreateHTTPServer(router, cfg)

	errChan := make(chan error, 1)

	go func(ch chan error) {
		var err error
		if cfg.Server.TLS.Enabled {
			err = server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("net/http server error: %w", err)
		}
	}(errChan)

	select {
	case err := <-errChan:
		return err
	case <-shutdown.Done():
		fmt.Print("\r") // hide that ugly ^C
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	logger.Info().Msg("mnemonic shutting down")

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown gracefully: %w", err)
	}

	logger.Info().Msg("mnemonic shutdown complete")
	return nil
}

// setupRouter creates and configures the Gin router with middleware.
func setupRouter(tel *telemetry.Telemetry, requestMetrics *middleware.RequestMetrics) *gin.Engine {
	// Use gin.New() instead of gin.Default() to avoid duplicate logging
	router := gin.New()

	// Recovery middleware (keep this)
	router.Use(gin.Recovery())

	// Use exported DefaultSkipPaths from middleware package
	skipPaths := middleware.DefaultSkipPaths

	// Tracing middleware using otelgin
	router.Use(middleware.TracingMiddlewareWithSkipPaths("mnemonic", skipPaths))

	// otelx logging middleware with trace correlation
	router.Use(otelxgin.LoggingMiddleware(tel.Otelx(),
		otelxgin.WithSkipPaths("/health", "/ops/health", "/metrics"),
		otelxgin.WithRequestHeaders("X-Request-ID", "X-Correlation-ID"),
	))

	// Request metrics middleware
	router.Use(requestMetrics.MiddlewareWithSkipPaths(skipPaths))

	return router
}

// CreateHTTPServer creates a new http.Server configured with settings from
// the provided configuration.
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
