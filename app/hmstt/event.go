package hmstt

import (
	"context"
	"time"

	"github.com/nurhudajoantama/stthmauto/internal/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

type HmsttEvent struct {
	ch *amqp.Channel
}

func NewEvent(conn *amqp.Connection) *HmsttEvent {
	ch := rabbitmq.NewRabbitMQChannel(conn)
	return &HmsttEvent{
		ch: ch,
	}
}

func (e *HmsttEvent) StateChange(ctx context.Context, key string, value string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	l := zerolog.Ctx(ctx)

	routing := MQ_CHANNEL_HMSTT + KEY_DELIMITER + key

	err := e.ch.PublishWithContext(
		ctx,
		"amq.topic", // exchange
		routing,     // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(value),
		},
	)
	if err != nil {
		l.Error().Err(err).Msg("Failed to publish a message")
	}
	l.Info().Msgf("Published state change event %s:%s", key, value)

	return err
}
