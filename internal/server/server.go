package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
	"github.com/NebojsaJovanovic95/VISP/internal/database"
	"github.com/NebojsaJovanovic95/VISP/internal/storage"
)

// Server holds the dependencies required by the VISP HTTP API.
type Server struct {
	db       *database.Database
	storage  *storage.Storage
	frameDir string
}

// New creates a Server with its required dependencies.
func New(db *database.Database, objectStorage *storage.Storage) *Server {
	return &Server{
		db:       db,
		storage:  objectStorage,
		frameDir: "data/frames",
	}
}

// Run starts the VISP Server.
func Run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	cfg, err := config.LoadServer()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer db.Close()

	storage, err := storage.New(ctx, cfg.ObjectStorage)
	if err != nil {
		return fmt.Errorf("initialize storage: %w", err)
	}

	server := New(db, storage)

	httpServer := &http.Server{
		Addr:    cfg.Address,
		Handler: loggingMiddleware(server.Router()),
	}

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- httpServer.ListenAndServe()
	}()

	fmt.Printf("VISP Server listening on %s\n", cfg.Address)

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

// Router creates the VISP HTTP request router.
func (s *Server) Router() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/cameras", s.listCamerasHandler)
	mux.HandleFunc("POST /api/frames", s.receiveFrameHandler)
	mux.HandleFunc("POST /api/heartbeat", s.heartbeatHandler)
	mux.HandleFunc("POST /api/cameras/{camera_id}/status", s.cameraStatusHandler)
	return mux
}

// healthHandler handles health check requests.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("OK")); err != nil {
		return
	}
}

// heartbeatHandler handles component heartbeat requests.
func (s *Server) heartbeatHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Component heartbeat received at %s\n", time.Now().Format(time.RFC3339))

	w.WriteHeader(http.StatusOK)
}

type cameraStatusRequest struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// cameraStatusHandler handles camera status reports.
func (s *Server) cameraStatusHandler(w http.ResponseWriter, r *http.Request) {
	cameraID := r.PathValue("camera_id")

	var request cameraStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if request.Status != "connected" && request.Status != "disconnected" {
		http.Error(w, "invalid camera status", http.StatusBadRequest)
		return
	}

	fmt.Printf(
		"Camera status received: camera_id=%s status=%s error=%s\n",
		cameraID,
		request.Status,
		request.Error,
	)

	w.WriteHeader(http.StatusOK)
}
