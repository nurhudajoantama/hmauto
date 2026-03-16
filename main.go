// Package main is the entry point for the hmauto home automation backend.
//
//	@title			hmauto API
//	@version		1.0
//	@description	Home automation backend API for IoT state management, alerting, and internet monitoring.
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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/nurhudajoantama/hmauto/app/hmalert"
	"github.com/nurhudajoantama/hmauto/app/hmapikey"
	"github.com/nurhudajoantama/hmauto/app/hmmon"
	"github.com/nurhudajoantama/hmauto/app/hmstt"
	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/nurhudajoantama/hmauto/app/worker"
	"github.com/nurhudajoantama/hmauto/internal/apikey"
	"github.com/nurhudajoantama/hmauto/internal/config"
	"github.com/nurhudajoantama/hmauto/internal/discord"
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

	httpClient := http.DefaultClient

	// Initialize Discord webhooks
	discordWebhookInfo := discord.NewDiscordWebhook(httpClient, cfg.DiscordWebhookInfo)
	discordWebhookWarning := discord.NewDiscordWebhook(httpClient, cfg.DiscordWebhookWarning)
	discordWebhookError := discord.NewDiscordWebhook(httpClient, cfg.DiscordWebhookError)

	// Set default values
	maxRequestSize := cfg.Security.MaxRequestSize
	if maxRequestSize == 0 {
		maxRequestSize = 1024 * 1024
	}
	rateLimitPerMin := cfg.Security.RateLimitPerMin
	if rateLimitPerMin == 0 {
		rateLimitPerMin = 60
	}
	rateLimitBurst := cfg.Security.RateLimitBurst
	if rateLimitBurst == 0 {
		rateLimitBurst = 10
	}

	rateLimiter := middleware.NewRateLimiter(rateLimitPerMin, time.Minute, rateLimitBurst)

	// API key store (Redis-backed)
	keyStore := apikey.NewRedisStore(rdb)

	// Initialize server
	serverConfig := &server.ServerConfig{
		KeyStore:       keyStore,
		AdminKey:       cfg.Security.AdminKey,
		EnableAuth:     cfg.Security.EnableAuth,
		MaxRequestSize: maxRequestSize,
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

	// Initialize worker
	errgrp, ctx := errgroup.WithContext(ctx)
	w := worker.New(errgrp, ctx)

	// HMALERT
	hmalertProducerEvent := hmalert.NewEvent(rabbitMQConn)
	hmalertService := hmalert.NewService(discordWebhookInfo, discordWebhookWarning, discordWebhookError, hmalertProducerEvent)
	hmalert.RegisterHandler(srv, hmalertService)

	hmalertConsumerEvent := hmalert.NewEvent(rabbitMQConn)
	hmalert.RegisterWorkers(w, hmalertConsumerEvent, hmalertService)

	// HMSTT
	hmsttStore := hmstt.NewStore(rdb)
	hmsttEvent := hmstt.NewEvent(rabbitMQConn)
	hmsttService := hmstt.NewService(hmsttStore, hmsttEvent, hmalertService)
	hmstt.RegisterHandlers(srv, hmsttService)

	// HMMON
	hmmon.RegisterWorkers(w, hmsttService, hmalertService, cfg.InternetCheck)

	// Admin API key management
	apikeyService := hmapikey.NewService(keyStore)
	hmapikey.RegisterHandlers(srv, apikeyService)

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
