// Package health provides HTTP health check endpoints.
// Implements Kubernetes-style liveness and readiness probes.
package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"sync/atomic"
	"time"
)

// ReadinessChecker is implemented by components that can report readiness.
type ReadinessChecker interface {
	Ready() bool
}

// Server provides health check HTTP endpoints.
type Server struct {
	addr    string
	checker ReadinessChecker
	logger  *slog.Logger
	server  *http.Server
	ready   atomic.Bool
}

// NewServer creates a new health server.
func NewServer(addr string, checker ReadinessChecker, logger *slog.Logger) *Server {
	s := &Server{
		addr:    addr,
		checker: checker,
		logger:  logger.With("component", "health"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleLiveness)
	mux.HandleFunc("/readyz", s.handleReadiness)
	mux.HandleFunc("/", s.handleRoot)

	// Register pprof handlers for profiling
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Run starts the health server. Blocks until context is canceled.
func (s *Server) Run(ctx context.Context) error {
	s.ready.Store(true)

	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("health server starting", "addr", s.addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.ready.Store(false)
	s.logger.Info("health server shutting down")
	return s.server.Shutdown(ctx)
}

// handleLiveness responds to liveness probes.
// Returns 200 if the process is alive (always, unless crashed).
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "alive",
	})
}

// handleReadiness responds to readiness probes.
// Returns 200 if the service is ready to accept traffic.
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ready := s.ready.Load() && s.checker.Ready()

	if ready {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ready",
		})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not_ready",
		})
	}
}

// handleRoot provides a simple index page.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"service": "gas-estimator",
		"endpoints": map[string]string{
			"/healthz": "Liveness probe",
			"/readyz":  "Readiness probe",
		},
	})
}
