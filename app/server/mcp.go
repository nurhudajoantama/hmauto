package server

import (
	"context"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nurhudajoantama/hmauto/internal/apikey"
	"github.com/nurhudajoantama/hmauto/internal/middleware"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"
)

// MCPServer wraps the MCP server with auth and HTTP lifecycle.
type MCPServer struct {
	server     *mcp.Server
	httpServer *http.Server
	addr       string
	keyStore   apikey.Store
	enableAuth bool
}

// MCPServerConfig holds configuration for the MCP server.
type MCPServerConfig struct {
	KeyStore   apikey.Store
	EnableAuth bool
}

// NewMCPServer creates a new MCP server.
func NewMCPServer(addr string, cfg *MCPServerConfig) *MCPServer {
	if cfg == nil {
		cfg = &MCPServerConfig{}
	}
	s := mcp.NewServer(&mcp.Implementation{Name: "hmauto", Version: "1.0.0"}, nil)
	return &MCPServer{
		server:     s,
		addr:       addr,
		keyStore:   cfg.KeyStore,
		enableAuth: cfg.EnableAuth,
	}
}

// GetServer returns the underlying MCP server for tool registration.
func (m *MCPServer) GetServer() *mcp.Server {
	return m.server
}

// Start runs the MCP HTTP server and blocks until ctx is cancelled.
func (m *MCPServer) Start(ctx context.Context) error {
	mcpHandler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return m.server
	}, nil)

	var h http.Handler = mcpHandler
	h = hlog.NewHandler(log.Logger)(h)
	if m.enableAuth && m.keyStore != nil {
		h = middleware.APIKeyAuth(m.keyStore)(h)
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", h)

	m.httpServer = &http.Server{
		Addr:         m.addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // no write timeout — SSE/streaming connections are long-lived
		IdleTimeout:  120 * time.Second,
	}

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- m.httpServer.ListenAndServe()
	}()
	log.Info().Msgf("MCP server started on %s/mcp", m.addr)

	select {
	case <-ctx.Done():
	case err := <-srvErr:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the MCP HTTP server.
func (m *MCPServer) Shutdown(ctx context.Context) error {
	log.Info().Msg("Shutting down MCP server")
	return m.httpServer.Shutdown(ctx)
}
