// Package main is the entry point for the hmauto home automation backend.
//
//	@title			hmauto API
//	@version		1.0
//	@description	Home automation backend API for IoT state management.
//
//	@host		localhost:8080
//	@BasePath	/v1
//	@schemes	http https
//
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
//	@description				Format: Bearer {api-key}
//
//	@securityDefinitions.apikey	AdminKeyAuth
//	@in							header
//	@name						Authorization
//	@description				Format: Bearer {admin-key}
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/nurhudajoantama/hmauto/app/hmapikey"
	"github.com/nurhudajoantama/hmauto/app/hmstt"
	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/nurhudajoantama/hmauto/internal/apikey"
	"github.com/nurhudajoantama/hmauto/internal/config"
	_ "github.com/nurhudajoantama/hmauto/docs"
	"github.com/nurhudajoantama/hmauto/internal/health"
	"github.com/nurhudajoantama/hmauto/internal/instrumentation"
	"github.com/nurhudajoantama/hmauto/internal/middleware"
	internalredis "github.com/nurhudajoantama/hmauto/internal/redis"
	"github.com/nurhudajoantama/hmauto/internal/rabbitmq"
	"golang.org/x/sync/errgroup"

	log "github.com/rs/zerolog/log"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "conf/conf.yaml"
	}
	cfg, err := config.InitializeConfig(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Initialize Sentry (before everything else)
	if cfg.Sentry.DSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.Sentry.DSN,
			Environment:      cfg.Sentry.Environment,
			TracesSampleRate: cfg.Sentry.SampleRate,
		}); err != nil {
			log.Fatal().Err(err).Msg("failed to initialize Sentry")
		}
	}

	// Initialize logger
	cleanupLog := instrumentation.InitializeLogger(cfg.Log)

	// Initialize OTEL
	otelShutdown, err := instrumentation.SetupOTelSDK(ctx, cfg.OTel)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize OpenTelemetry")
	}

	// Initialize Redis
	rdb := internalredis.NewClient(cfg.Redis)

	// Initialize RabbitMQ
	rabbitMQConn := rabbitmq.NewRabbitMQConn(cfg.MQTT)

	rateLimiter := middleware.NewRateLimiter(cfg.Security.GetRateLimitPerMin(), time.Minute, cfg.Security.GetRateLimitBurst())

	// API key store (Redis-backed)
	keyStore := apikey.NewRedisStore(rdb, cfg.GetRedisKeyPrefix())

	// Initialize server
	serverConfig := &server.ServerConfig{
		KeyStore:       keyStore,
		AdminKey:       cfg.Security.AdminKey,
		EnableAuth:     cfg.Security.EnableAuth,
		MaxRequestSize: cfg.Security.GetMaxRequestSize(),
		RateLimiter:    rateLimiter,
	}
	srv := server.NewWithConfig(cfg.HTTP.Addr(), serverConfig)

	// Health check endpoints (unversioned — used by K8s probes)
	healthChecker := health.NewHealthChecker(rdb, rabbitMQConn)
	r := srv.GetRouter()
	r.HandleFunc("/healthz", health.LivenessHandler()).Methods("GET")
	r.HandleFunc("/health", healthChecker.Handler()).Methods("GET")
	r.HandleFunc("/ready", healthChecker.ReadinessHandler()).Methods("GET")
	r.HandleFunc("/live", health.LivenessHandler()).Methods("GET")

	// HMSTT
	hmsttStore := hmstt.NewStore(rdb, cfg.GetRedisKeyPrefix())
	hmsttEvent := hmstt.NewEvent(rabbitMQConn)
	hmsttService := hmstt.NewService(hmsttStore, hmsttEvent)
	hmstt.RegisterHandlers(srv, hmsttService)

	// Admin API key management
	apikeyService := hmapikey.NewService(keyStore)
	hmapikey.RegisterHandlers(srv, apikeyService)

	errgrp, ctx := errgroup.WithContext(ctx)
	errgrp.Go(func() error {
		return srv.Start(ctx)
	})

	if err := errgrp.Wait(); err != nil {
		log.Error().Err(err).Msg("closing application due to error")
	}

	// Graceful shutdown
	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(closeCtx); err != nil {
		log.Error().Err(err).Msg("failed to shutdown http server")
	}
	rabbitmq.Close(closeCtx, rabbitMQConn)
	internalredis.Close(closeCtx, rdb)

	if err := otelShutdown(closeCtx); err != nil {
		log.Error().Err(err).Msg("failed to shutdown OpenTelemetry")
	}
	cleanupLog(closeCtx)
}
