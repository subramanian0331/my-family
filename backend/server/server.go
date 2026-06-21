package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/subbu/family_tree/config"
	"github.com/subbu/family_tree/handlers"
	postgresclient "github.com/subbu/family_tree/client/postgres"
)

type Server struct {
	cfg    config.Config
	db     postgresclient.Client
	router *handlers.Router
	http   *http.Server
}

func New(cfg config.Config, db postgresclient.Client, router *handlers.Router) *Server {
	return &Server{
		cfg:    cfg,
		db:     db,
		router: router,
		http: &http.Server{
			Addr:              ":" + cfg.Port,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.http.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.http.Shutdown(shutdownCtx)
		s.db.Close()
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return fmt.Errorf("http server failed: %w", err)
	}
}