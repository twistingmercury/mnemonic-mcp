package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/heartbeat"
	"github.com/twistingmercury/mnemonic/internal/config"
	"github.com/twistingmercury/mnemonic/internal/handlers/operations"
)

// ListenAndServe starts the server using configuration loaded from config sources.
func ListenAndServe() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	return ListenAndServeWithConfig(cfg)
}

// ListenAndServeWithConfig starts the server using the provided configuration.
func ListenAndServeWithConfig(cfg *config.MnemonicConfig) error {
	router := gin.Default()

	operations.SetupHandlers(router)

	server := CreateHTTPServer(router, cfg)

	go func() {
		var err error
		if cfg.Server.TLS.Enabled {
			err = server.ListenAndServeTLS(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start server: %s", err.Error())
		}
	}()

	log.Printf("mnemonic is running on %s...", cfg.Server.Address())

	shutdown, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdown.Done()
	fmt.Print("\r") // hide that ugly ^C

	log.Println("mnemonic is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown gracefully: %w", err)
	}
	return nil
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

// CreateHttpServer creates a new http.Server using default configuration.
// Deprecated: Use CreateHTTPServer with explicit configuration instead.
func CreateHttpServer(r *gin.Engine) *http.Server {
	return &http.Server{
		Addr:           ":8080",
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}

func CheckHealth() error {
	//--> THIS IS A HACK UNTIL I CAN UPDATE THE HEARTBEAT PACKAGE!!!
	mode := gin.Mode()
	defer gin.SetMode(mode)
	gin.SetMode(gin.TestMode)

	statCheck := heartbeat.Handler("mnemonic", operations.DefineDependencies()...)

	writer := httptest.NewRecorder()
	hackContext, _ := gin.CreateTestContext(writer)
	hackContext.Request = httptest.NewRequest("GET", "/ops/health", nil)

	statCheck(hackContext)

	if writer.Code != http.StatusOK {
		return fmt.Errorf("unhealthy: %s", writer.Body.String())
	}
	log.Println("healthy")
	return nil
	//<-- END HACK
}
