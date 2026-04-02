package api

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/danrichardson/pdarr/internal/config"
	"github.com/danrichardson/pdarr/internal/db"
	"github.com/danrichardson/pdarr/internal/queue"
	"github.com/danrichardson/pdarr/internal/scanner"
	"github.com/danrichardson/pdarr/internal/transcoder"
)

//go:embed frontend_dist
var frontendFS embed.FS

// Server is the HTTP API server.
type Server struct {
	cfg        *config.Config
	db         *db.DB
	worker     *queue.Worker
	scanner    *scanner.Scanner
	encoder    *transcoder.Encoder
	hub        *wsHub
	log        *slog.Logger
	httpServer *http.Server
}

// New creates a Server.
func New(
	cfg *config.Config,
	database *db.DB,
	w *queue.Worker,
	s *scanner.Scanner,
	enc *transcoder.Encoder,
	log *slog.Logger,
) *Server {
	srv := &Server{
		cfg:     cfg,
		db:      database,
		worker:  w,
		scanner: s,
		encoder: enc,
		hub:     newWSHub(),
		log:     log,
	}

	// Subscribe worker events to WebSocket hub.
	w.Subscribe(func(e queue.Event) {
		srv.hub.broadcast(e)
	})

	mux := http.NewServeMux()
	srv.registerRoutes(mux)

	srv.httpServer = &http.Server{
		Addr:         cfg.Addr(),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // 0 = no timeout for streaming/WebSocket
		IdleTimeout:  120 * time.Second,
	}
	return srv
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	api := http.NewServeMux()

	// Status
	api.HandleFunc("GET /status", s.handleGetStatus)

	// Jobs
	api.HandleFunc("GET /jobs", s.handleListJobs)
	api.HandleFunc("POST /jobs", s.handleCreateJob)
	api.HandleFunc("GET /jobs/{id}", s.handleGetJob)
	api.HandleFunc("DELETE /jobs/{id}", s.handleCancelJob)
	api.HandleFunc("POST /jobs/{id}/retry", s.handleRetryJob)

	// Directories
	api.HandleFunc("GET /directories", s.handleListDirectories)
	api.HandleFunc("POST /directories", s.handleCreateDirectory)
	api.HandleFunc("GET /directories/{id}", s.handleGetDirectory)
	api.HandleFunc("PUT /directories/{id}", s.handleUpdateDirectory)
	api.HandleFunc("DELETE /directories/{id}", s.handleDeleteDirectory)

	// Scanner
	api.HandleFunc("POST /scan", s.handleTriggerScan)

	// Worker control
	api.HandleFunc("POST /queue/pause", s.handlePauseQueue)
	api.HandleFunc("POST /queue/resume", s.handleResumeQueue)

	// Stats
	api.HandleFunc("GET /stats", s.handleGetStats)

	// WebSocket
	api.HandleFunc("GET /ws", s.handleWebSocket)

	// Auth
	api.HandleFunc("POST /auth/login", s.handleLogin)

	// Wrap all API routes with auth middleware.
	mux.Handle("/api/v1/", http.StripPrefix("/api/v1",
		s.authMiddleware(s.requestLogger(api))))

	// SPA fallback — serve embedded frontend.
	fsys, err := fs.Sub(frontendFS, "frontend_dist")
	if err != nil {
		panic(fmt.Sprintf("embed frontend: %v", err))
	}
	fileServer := http.FileServer(http.FS(fsys))
	mux.Handle("/", spaHandler(fileServer))
}

// ServeHTTP implements http.Handler, used in tests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.httpServer.Handler.ServeHTTP(w, r)
}

// Start begins listening. Returns when ctx is cancelled.
func (s *Server) Start(ctx context.Context) error {
	go s.hub.run(ctx)

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("HTTP server listening", "addr", s.cfg.Addr())
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutCtx)
	case err := <-errCh:
		return err
	}
}

// spaHandler serves the SPA index.html for any path that doesn't match a
// static file — enabling client-side routing.
func spaHandler(fileServer http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileServer.ServeHTTP(w, r)
	})
}
