package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/flare-foundation/go-flare-common/pkg/logger"
	"github.com/flare-foundation/go-flare-common/pkg/storage"

	"github.com/flare-foundation/fdc-client/client/config"
	"github.com/flare-foundation/fdc-client/client/round"
)

const shutdownTimeout = 5 * time.Second

// Server wraps an HTTP server.
type Server struct {
	srv *http.Server
}

// New creates a new Server with routes, API key auth, and CORS.
func New(
	rounds *storage.Cyclic[uint32, *round.Round],
	protocolID uint8,
	serverConfig config.RestServer,
) Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	keySet := make(map[string]bool, len(serverConfig.APIKeys))
	for _, k := range serverConfig.APIKeys {
		keySet[k] = true
	}
	auth := func(h http.HandlerFunc) http.Handler {
		return apiKeyMiddleware(serverConfig.APIKeyName, keySet, h)
	}

	fsp := serverConfig.FSPSubpath
	controller := newFDCProtocolProviderController(rounds, protocolID)
	mux.Handle(fmt.Sprintf("GET %s/submit1/{votingRoundID}/{submitAddress}", fsp), auth(controller.submit1))
	mux.Handle(fmt.Sprintf("GET %s/submit2/{votingRoundID}/{submitAddress}", fsp), auth(controller.submit2))
	mux.Handle(fmt.Sprintf("GET %s/submitSignatures/{votingRoundID}/{submitAddress}", fsp), auth(controller.submitSignatures))

	da := serverConfig.DAPSubpath
	daCtrl := DAController{Rounds: rounds}
	mux.Handle(fmt.Sprintf("GET %s/getRequests/{votingRoundID}", da), auth(daCtrl.getRequests))
	mux.Handle(fmt.Sprintf("GET %s/getAttestations/{votingRoundID}", da), auth(daCtrl.getAttestations))

	srv := &http.Server{
		Handler:           corsMiddleware(mux),
		Addr:              serverConfig.Addr,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
	}

	return Server{srv: srv}
}

// Run starts the HTTP server.
func (s *Server) Run(_ context.Context) {
	logger.Infof("Starting server on %s", s.srv.Addr)

	err := s.srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Panicf("server: %v", err)
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		logger.Errorf("server shutdown failed: %v", err)
	} else {
		logger.Info("Server gracefully stopped")
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Errorf("failed to write JSON response: %v", err)
	}
}

func apiKeyMiddleware(keyName string, keys map[string]bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get(keyName)
		if !keys[key] {
			http.Error(w, fmt.Sprintf("Unauthorized, provide valid %s api key", keyName), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-KEY")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
