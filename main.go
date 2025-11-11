package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nurhudajoantama/stthmauto/app/hmstt"
	"github.com/nurhudajoantama/stthmauto/app/server"
	"github.com/nurhudajoantama/stthmauto/app/worker"
	"github.com/nurhudajoantama/stthmauto/internal/config"
	"github.com/nurhudajoantama/stthmauto/internal/instrumentation"
	"github.com/nurhudajoantama/stthmauto/internal/postgres"
	"github.com/nurhudajoantama/stthmauto/internal/rabbitmq"
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
	defer cleanupLog()

	// initialize bbolt
	gormPostgres := postgres.NewGorm(config.DB)

	// initialize rabbitmq
	rabbitMQConn := rabbitmq.NewRabbitMQConn(config.MQTT)
	// defer rabbitMQConn.Close()

	// initialize server
	srv := server.New(config.HTTP.Addr())

	// initialize worker
	errgrp, ctx := errgroup.WithContext(ctx)
	w := worker.New(errgrp, ctx)

	// HTSTT
	{
		hmsttStore := hmstt.NewStore(gormPostgres)
		hmsttEvent := hmstt.NewEvent(rabbitMQConn)
		hmsttService := hmstt.NewService(hmsttStore, hmsttEvent)
		hmstt.RegisterHandlers(srv, hmsttService)

		hmstt.RegisterWorkers(w, hmsttService)
	}

	// start server implemented graceful shutdown
	// go func() {
	// 	log.Info().Msgf("starting server on %s", config.HTTP.Addr())
	// 	if err := srv.Start(); err != nil {
	// 		log.Error().Err(err).Msg("server stopped with error")
	// 	}
	// }()
	errgrp.Go(func() error {
		srv.Start(ctx)
		return nil
	})

	// start workers

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	<-quit

	log.Info().Msg("shutting down server...")
	{
		gracefulPeriod := 10 * time.Second
		shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulPeriod)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("failed to gracefully shutdown server")
		}
		log.Info().Msg("server stopped")
	}
}
