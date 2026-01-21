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
	fmt.Print("\r") // hide that ugly ^C

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
