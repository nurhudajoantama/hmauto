package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nurhudajoantama/hmauto/internal/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// ServerConfig holds configuration for the server
type ServerConfig struct {
	APIKeys        map[string]bool
	EnableAuth     bool
	MaxRequestSize int64
	RateLimiter    *middleware.RateLimiter
}

// Server wraps an http.Server and a logger.
type Server struct {
	httpServer *http.Server
	router     *mux.Router
	addr       string
	config     *ServerConfig
}

// New creates a configured server listening on the provided address.
func New(addr string) *Server {
	return NewWithConfig(addr, nil)
}

// NewWithConfig creates a configured server with custom config.
func NewWithConfig(addr string, config *ServerConfig) *Server {
	if config == nil {
		config = &ServerConfig{
			EnableAuth:     false,
			MaxRequestSize: 1024 * 1024, // 1MB default
		}
	}

	r := mux.NewRouter()

	// Security headers middleware (always enabled)
	r.Use(middleware.SecurityHeaders)

	// Request size limiting (always enabled)
	r.Use(middleware.MaxBytesReader(config.MaxRequestSize))

	// Logging middleware
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

	// Rate limiting (if configured)
	if config.RateLimiter != nil {
		r.Use(middleware.RateLimit(config.RateLimiter))
	}

	// public routes
	r.HandleFunc("/healthz", healthHandler).Methods("GET")
	r.HandleFunc("/hello", helloHandler).Methods("GET")

	return &Server{
		router: r,
		addr:   addr,
		config: config,
	}
}

func (s *Server) GetRouter() *mux.Router {
	return s.router
}

// GetConfig returns the server configuration
func (s *Server) GetConfig() *ServerConfig {
	return s.config
}

// ApplyAuthMiddleware applies authentication to a subrouter
func (s *Server) ApplyAuthMiddleware(subrouter *mux.Router) {
	if s.config != nil && s.config.EnableAuth && len(s.config.APIKeys) > 0 {
		subrouter.Use(middleware.APIKeyAuth(s.config.APIKeys))
	}
}

// Start runs the HTTP server. It returns when the server stops.
func (s *Server) Start(ctx context.Context) error {
	handler := otelhttp.NewHandler(s.router, "stthmauto-server")

	s.httpServer = &http.Server{
		Addr:         s.addr,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- s.httpServer.ListenAndServe()
	}()
	log.Info().Msgf("HTTP server started on %s", s.addr)

	select {
	case <-ctx.Done():
		// Shutdown server
	case err := <-srvErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
		return err
	}

	return nil
}

// Shutdown gracefully shuts the server down within the provided context.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info().Msg("Shutting down HTTP server")
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
