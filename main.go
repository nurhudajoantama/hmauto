package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nurhudajoantama/hmauto/app/hmalert"
	"github.com/nurhudajoantama/hmauto/app/hmmon"
	"github.com/nurhudajoantama/hmauto/app/hmstt"
	"github.com/nurhudajoantama/hmauto/app/server"
	"github.com/nurhudajoantama/hmauto/app/worker"
	"github.com/nurhudajoantama/hmauto/internal/config"
	"github.com/nurhudajoantama/hmauto/internal/instrumentation"
	"github.com/nurhudajoantama/hmauto/internal/postgres"
	"github.com/nurhudajoantama/hmauto/internal/rabbitmq"
	"golang.org/x/sync/errgroup"

	log "github.com/rs/zerolog/log"
)

func main() {
	// initialize config

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "conf/conf.yaml"
	}
	config, err := config.InitializeConfig(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// initialize logger
	cleanupLog := instrumentation.InitializeLogger(config.Log)

	// initialize otel
	otelShutdown, err := instrumentation.SetupOTelSDK(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize OpenTelemetry")
	}

	// initialize bbolt
	gormPostgres := postgres.NewGorm(config.DB)

	// initialize rabbitmq
	rabbitMQConn := rabbitmq.NewRabbitMQConn(config.MQTT)

	// initialize server
	srv := server.New(config.HTTP.Addr())

	// initialize worker
	errgrp, ctx := errgroup.WithContext(ctx)
	w := worker.New(errgrp, ctx)

	// HTSTT
	hmsttStore := hmstt.NewStore(gormPostgres)
	hmsttEvent := hmstt.NewEvent(rabbitMQConn)
	hmsttService := hmstt.NewService(hmsttStore, hmsttEvent)
	hmstt.RegisterHandlers(srv, hmsttService)

	// HMMON
	hmmon.RegisterWorkers(w, hmsttService, config.InternetCheck)

	// MLALERT
	hmalertEvent := hmalert.NewEvent(rabbitMQConn)
	hmalertDiscord := hmalert.NewDiscord(config.DiscordWebhookInfo, config.DiscordWebhookWarning, config.DiscordWebhookError)
	hmalertService := hmalert.NewService(hmalertDiscord, hmalertEvent)
	hmalert.RegisterHandler(srv, hmalertService)
	hmalert.RegisterWorkers(w, hmalertEvent, hmalertService)

	errgrp.Go(func() error {
		return srv.Start(ctx)
	})

	// start workers
	if err := errgrp.Wait(); err != nil {
		log.Error().Err(err).Msg("closing application due to error")
	}

	// Close Connection
	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = srv.Shutdown(closeCtx)
	rabbitmq.Close(closeCtx, rabbitMQConn)
	postgres.Close(closeCtx, gormPostgres)

	otelShutdown(closeCtx)
	cleanupLog(closeCtx)
}
