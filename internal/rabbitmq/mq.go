package rabbitmq

import (
	"context"

	log "github.com/rs/zerolog/log"

	"github.com/nurhudajoantama/hmauto/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

func NewRabbitMQConn(c config.MQTT) *amqp.Connection {
	conn, err := amqp.Dial(c.BrokerURL())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to RabbitMQ")
	}
	log.Info().Msg("Connected to RabbitMQ")

	return conn
}

func NewRabbitMQChannel(conn *amqp.Connection) *amqp.Channel {
	if conn == nil {
		log.Fatal().Msg("rabbitmq connection is nil")
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open rabbitmq channel")
	}
	log.Info().Msg("Opened a channel to RabbitMQ")

	return ch
}

func Close(ctx context.Context, conn *amqp.Connection) {
	if conn == nil {
		log.Warn().Msg("rabbitmq connection is nil, skipping close")
		return
	}

	c := make(chan struct{})
	go func() {
		defer close(c)
		if err := conn.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close RabbitMQ connection")
		}
	}()

	select {
	case <-ctx.Done():
		log.Warn().Msg("timeout while closing RabbitMQ connection")
	case <-c:
		log.Info().Msg("RabbitMQ connection closed")
	}
}
