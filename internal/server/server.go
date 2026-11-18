package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
	"github.com/NebojsaJovanovic95/VISP/internal/database"
)

// Run starts the VISP Server.
func Run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer db.Close()

	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: newRouter(),
	}

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	fmt.Println("VISP Server listening on :8080")

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server: %w", err)
		}

	case <-ctx.Done():
		fmt.Println("Shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		_ = httpServer.Close()
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	fmt.Println("VISP Server stopped")

	return nil
}

// newRouter creates the VISP HTTP request router.
func newRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)

	return mux
}

// healthHandler handles health check requests.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("OK")); err != nil {
		return
	}
}
