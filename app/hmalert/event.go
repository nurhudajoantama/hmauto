package hmalert

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nurhudajoantama/hmauto/internal/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
)

type HmalertEvent struct {
	ch *amqp.Channel
	q  amqp.Queue
}

func NewEvent(conn *amqp.Connection) *HmalertEvent {
	ch := rabbitmq.NewRabbitMQChannel(conn)

	q, err := ch.QueueDeclare(
		MQ_CHANNEL_HMALERT,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to declare hmalert queue")
	}

	return &HmalertEvent{
		ch: ch,
		q:  q,
	}
}

func (e *HmalertEvent) PublishAlert(ctx context.Context, body alertEvent) error {
	ctx, span := otel.Tracer("hmalert").Start(ctx, "event.PublishAlert")
	defer span.End()

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
		"",
		e.q.Name,
		false,
		false,
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
		e.q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
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
