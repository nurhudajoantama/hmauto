package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nurhudajoantama/stthmauto/app/hmmon"
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

	errgrp.Go(func() error {
		srv.Start(ctx)
		return nil
	})

	// start workers
	if err := errgrp.Wait(); err != nil {
		log.Error().Err(err).Msg("closing application due to error")
	}

	// Close Connection
	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rabbitmq.Close(closeCtx, rabbitMQConn)
	postgres.Close(closeCtx, gormPostgres)
}
