package server

import (
	"context"
	"crypto/subtle"
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
	status InfoSource,
) Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	auth := func(h http.HandlerFunc) http.Handler {
		return apiKeyMiddleware(serverConfig.APIKeyName, serverConfig.APIKeys, h)
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

	ic := &infoController{source: status}
	mux.Handle("GET /info", auth(ic.info))

	srv := &http.Server{
		Handler:           corsMiddleware(serverConfig.CORSOrigin, mux),
		Addr:              serverConfig.Addr,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
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

func apiKeyMiddleware(keyName string, keys []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := []byte(r.Header.Get(keyName))
		valid := false
		for _, k := range keys {
			if subtle.ConstantTimeCompare(key, []byte(k)) == 1 {
				valid = true
				break
			}
		}
		if !valid {
			http.Error(w, fmt.Sprintf("Unauthorized, provide valid %s api key", keyName), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(origin string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-KEY")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
