package api

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/alexandrmotologa/agentloop/internal/engine"
	"github.com/alexandrmotologa/agentloop/internal/store"
)

// Server encapsulates the HTTP server lifecycle.
type Server struct {
	httpServer *http.Server
}

// StartServer initializes and begins listening on the configured address.
func StartServer(addr string, db *store.DB, executor *engine.Executor, webFS fs.FS) (*Server, error) {
	router := NewRouter(db, executor, webFS)

	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	server := &Server{
		httpServer: httpSrv,
	}

	go func() {
		log.Printf("[SERVER] agentloop cockpit listening at http://%s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[SERVER] listen error: %v", err)
		}
	}()

	return server, nil
}

// Shutdown gracefully terminates the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
