package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/73ai/openbotkit/store"
)

type Server struct {
	cfg   Config
	store *Store
}

func NewServer(cfg Config, st *Store) *Server {
	return &Server{cfg: cfg, store: st}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	s.routes(mux)

	handler := s.cors(limitBody(mux))

	srv := &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(_ net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("registry server listening", "addr", s.cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		slog.Info("shutting down registry server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", s.handleHealth)

	mux.HandleFunc("GET /api/usecases", s.handleBrowse)
	mux.HandleFunc("GET /api/usecases/{id}", s.handleGetUseCase)

	mux.HandleFunc("GET /api/auth/google", s.handleAuthGoogle)
	mux.HandleFunc("GET /api/auth/google/callback", s.handleAuthGoogleCallback)
	mux.HandleFunc("GET /api/auth/me", s.handleAuthMe)
	mux.HandleFunc("POST /api/auth/logout", s.handleAuthLogout)

	if s.cfg.DemoLogin {
		mux.HandleFunc("GET /api/auth/demo", s.handleAuthDemo)
	}

	auth := s.authRequired
	mux.Handle("POST /api/usecases", auth(http.HandlerFunc(s.handleCreateUseCase)))
	mux.Handle("PUT /api/usecases/{id}", auth(http.HandlerFunc(s.handleUpdateUseCase)))
	mux.Handle("DELETE /api/usecases/{id}", auth(http.HandlerFunc(s.handleDeleteUseCase)))
	mux.Handle("POST /api/usecases/{id}/fork", auth(http.HandlerFunc(s.handleForkUseCase)))
	mux.Handle("GET /api/dashboard", auth(http.HandlerFunc(s.handleDashboard)))
	mux.Handle("PUT /api/users/me", auth(http.HandlerFunc(s.handleUpdateUser)))
}

func openDB(cfg Config) (*store.DB, error) {
	return store.Open(store.SQLiteConfig(cfg.DBPath))
}
