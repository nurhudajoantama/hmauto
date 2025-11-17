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
		log.Error().Err(err).Msg("Failed to connect to RabbitMQ")
	}
	log.Info().Msg("Connected to RabbitMQ")

	return conn
}

func NewRabbitMQChannel(conn *amqp.Connection) *amqp.Channel {
	ch, err := conn.Channel()
	if err != nil {
		log.Error().Err(err).Msg("Failed to open a channel")
	}
	log.Info().Msg("Opened a channel to RabbitMQ")

	return ch
}

func Close(ctx context.Context, conn *amqp.Connection) {
	c := make(chan struct{})
	go func() {
		if err := conn.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close RabbitMQ connection")
		}
		close(c)
	}()

	select {
	case <-ctx.Done():
		log.Warn().Msg("timeout while closing RabbitMQ connection")
	case <-c:
		log.Info().Msg("RabbitMQ connection closed")
	}
}
