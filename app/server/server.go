package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Server wraps an http.Server and a logger.
type Server struct {
	httpServer *http.Server
	router     *mux.Router
	addr       string
}

// New creates a configured server listening on the provided address.
func New(addr string) *Server {
	r := mux.NewRouter()

	r.Use(hlog.NewHandler(log.Logger))
	r.Use(hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		hlog.FromRequest(r).Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("handled request")
	}))

	r.Use(hlog.RemoteAddrHandler("ip"))
	r.Use(hlog.UserAgentHandler("user_agent"))
	r.Use(hlog.RefererHandler("referer"))
	r.Use(hlog.RequestIDHandler("request_id", "X-Request-ID"))

	// public routes
	r.HandleFunc("/healthz", healthHandler).Methods("GET")
	r.HandleFunc("/hello", helloHandler).Methods("GET")

	// instrument the router with OpenTelemetry HTTP middleware

	return &Server{
		router: r,
		addr:   addr,
	}
}

func (s *Server) GetRouter() *mux.Router {
	return s.router
}

// Start runs the HTTP server. It returns when the server stops.
func (s *Server) Start() error {
	handler := otelhttp.NewHandler(s.router, "stthmauto-server")

	s.httpServer = &http.Server{
		Addr:         s.addr,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts the server down within the provided context.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World!"))
}
