package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
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
	daCtrl := NewDAController(rounds)
	mux.Handle(fmt.Sprintf("GET %s/getRequests/{votingRoundID}", da), auth(daCtrl.getRequests))
	mux.Handle(fmt.Sprintf("GET %s/getAttestations/{votingRoundID}", da), auth(daCtrl.getAttestations))

	ic := &infoController{source: status}
	mux.Handle("GET /info", auth(ic.info))

	srv := &http.Server{
		Handler:           recoveryMiddleware(corsMiddleware(serverConfig.CORSOrigin, serverConfig.APIKeyName, mux)),
		Addr:              serverConfig.Addr,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		ReadTimeout:       15 * time.Second,
		IdleTimeout:       60 * time.Second,
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
		logger.Errorf("Server shutdown failed: %v", err)
	} else {
		logger.Info("Server gracefully stopped")
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		logger.Errorf("failed to marshal JSON response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(data); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func apiKeyMiddleware(keyName string, keys []string, next http.Handler) http.Handler {
	hashedKeys := make([][32]byte, len(keys))
	for i, k := range keys {
		hashedKeys[i] = sha256.Sum256([]byte(k))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyHash := sha256.Sum256([]byte(r.Header.Get(keyName)))
		valid := 0
		for _, hk := range hashedKeys {
			valid |= subtle.ConstantTimeCompare(keyHash[:], hk[:])
		}
		if valid != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware recovers from panics in downstream handlers, logs the panic with a
// stack trace, and returns a generic 500 to the client instead of dropping the connection.
// It re-panics on http.ErrAbortHandler so net/http can handle that intentional abort.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec)
			}
			logger.Errorf("panic serving %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}()
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(origin, keyName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Frame-Options", "DENY")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, "+keyName)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
