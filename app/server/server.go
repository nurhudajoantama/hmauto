package server

import (
	"context"
	"net/http"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gorilla/mux"
	"github.com/nurhudajoantama/hmauto/internal/apikey"
	"github.com/nurhudajoantama/hmauto/internal/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// ServerConfig holds configuration for the server.
type ServerConfig struct {
	KeyStore       apikey.Store
	AdminKey       string
	EnableAuth     bool
	MaxRequestSize int64
	RateLimiter    *middleware.RateLimiter
}

// Server wraps an http.Server and a mux.Router.
type Server struct {
	httpServer *http.Server
	router     *mux.Router
	addr       string
	config     *ServerConfig
}

// NewWithConfig creates a configured server.
func NewWithConfig(addr string, config *ServerConfig) *Server {
	if config == nil {
		config = &ServerConfig{
			EnableAuth:     false,
			MaxRequestSize: 1024 * 1024,
		}
	}

	r := mux.NewRouter()

	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.MaxBytesReader(config.MaxRequestSize))
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

	if config.RateLimiter != nil {
		r.Use(middleware.RateLimit(config.RateLimiter))
	}

	r.Use(middleware.PrometheusMiddleware)
	r.Use(middleware.TraceIDMiddleware)

	// Public routes
	r.HandleFunc("/healthz", healthHandler).Methods("GET")
	r.HandleFunc("/hello", helloHandler).Methods("GET")
	r.Handle("/metrics", promhttp.Handler()).Methods("GET")

	return &Server{
		router: r,
		addr:   addr,
		config: config,
	}
}

func (s *Server) GetRouter() *mux.Router {
	return s.router
}

func (s *Server) GetConfig() *ServerConfig {
	return s.config
}

// ApplyAuthMiddleware attaches Redis-backed API key auth to a subrouter.
func (s *Server) ApplyAuthMiddleware(subrouter *mux.Router) {
	if s.config != nil && s.config.EnableAuth && s.config.KeyStore != nil {
		subrouter.Use(middleware.APIKeyAuth(s.config.KeyStore))
	}
}

// ApplyAdminMiddleware attaches config-only admin key auth to a subrouter.
func (s *Server) ApplyAdminMiddleware(subrouter *mux.Router) {
	if s.config != nil && s.config.AdminKey != "" {
		subrouter.Use(middleware.AdminKeyAuth(s.config.AdminKey))
	}
}

// Start runs the HTTP server.
func (s *Server) Start(ctx context.Context) error {
	sentryHandler := sentryhttp.New(sentryhttp.Options{Repanic: true})
	handler := sentryHandler.Handle(otelhttp.NewHandler(s.router, "hmauto-server"))

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
	case err := <-srvErr:
		if err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
		return err
	}

	return nil
}

// Shutdown gracefully shuts the server down.
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
