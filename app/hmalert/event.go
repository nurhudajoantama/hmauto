package hmalert

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nurhudajoantama/hmauto/internal/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type HmalertEvent struct {
	ch *amqp.Channel
	q  amqp.Queue
}

func NewEvent(conn *amqp.Connection) *HmalertEvent {
	ch := rabbitmq.NewRabbitMQChannel(conn)

	q, err := ch.QueueDeclare(
		MQ_CHANNEL_HMALERT, // name
		false,              // durable (queue survives broker restart)
		false,              // delete when unused
		false,              // exclusive (used by only one connection)
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to declare a queue")
	}

	return &HmalertEvent{
		ch: ch,
		q:  q,
	}
}

func (e *HmalertEvent) PublishAlert(ctx context.Context, body alertEvent) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	l := zerolog.Ctx(ctx)

	b, err := json.Marshal(body)
	if err != nil {
		l.Error().Err(err).Msg("Failed to marshal alert event")
		return err
	}

	err = e.ch.PublishWithContext(
		ctx,
		"",       // exchange
		e.q.Name, // routing key
		false,    // mandatory
		false,    // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(b),
		},
	)
	if err != nil {
		l.Error().Err(err).Msg("Failed to publish a message")
	}

	l.Info().Msgf("Published alert event: %s", body.Type)

	return err
}

func (e *HmalertEvent) ConsumeAlerts(ctx context.Context) (<-chan amqp.Delivery, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	l := zerolog.Ctx(ctx)

	msgs, err := e.ch.Consume(
		e.q.Name, // queue
		"",       // consumer
		false,    // auto-ack
		false,    // exclusive
		false,    // no-local
		false,    // no-wait
		nil,      // args
	)
	if err != nil {
		l.Error().Err(err).Msg("Failed to register a consumer")
		return nil, err
	}

	l.Info().Msg("Consumer registered for alert events")

	return msgs, nil
}

func (e *HmalertEvent) Close() error {
	return e.ch.Close()
}
