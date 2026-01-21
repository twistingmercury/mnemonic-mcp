package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/twistingmercury/mnemonic/internal/handlers/operations"
)

// ListenAndServer starts the server
func ListenAndServe() error {
	router := gin.Default()

	operations.SetupHandlers(router)

	server := CreateHttpServer(router)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to shutdown server: %s", err.Error())
		}
	}()

	log.Println("mnemonic is running...")

	shutdown, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdown.Done()
	fmt.Print("\r") // hide the ugly ^C

	log.Println("mnemonic is shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown gracefully: %w", err)
	}
	return nil
}

// CreateHttpServer creates a new http.Server that uses the gin.Engine for
// its handler.
func CreateHttpServer(r *gin.Engine) *http.Server {
	return &http.Server{
		Addr:           ":8080",
		Handler:        r,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
}
