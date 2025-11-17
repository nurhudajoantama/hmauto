package hmalert

import (
	"context"
	"encoding/json"

	"github.com/nurhudajoantama/hmauto/app/worker"
	"github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type HmalertWorker struct {
	Event   *HmalertEvent
	Service *HmalerService
}

func RegisterWorkers(s *worker.Worker, event *HmalertEvent, service *HmalerService) {
	hw := &HmalertWorker{
		Event:   event,
		Service: service,
	}

	s.Go(hw.alertConsumer)
}

func (w *HmalertWorker) alertConsumer(ctx context.Context) error {
	l := zerolog.Ctx(ctx)

	msg, err := w.Event.ConsumeAlerts(ctx)
	if err != nil {
		l.Error().Err(err).Msg("Failed to consume alert messages")
		return err
	}

	go func() {
		for d := range msg {
			l.Info().Msgf("Received an alert message: %s", d.Body)

			err := w.processAlert(d)
			if err != nil {
				l.Error().Err(err).Msg("Failed to process alert message")
				d.Nack(false, false)
				continue
			}

			d.Ack(false)
		}
	}()
	<-ctx.Done()

	l.Info().Msg("Hmalert alert consumer stopped")
	return nil
}

func (w *HmalertWorker) processAlert(d amqp091.Delivery) error {
	// Process the alert message here

	var body alertEvent
	// Unmarshal the message body into the alertEvent struct
	// Handle any errors during unmarshaling
	err := json.Unmarshal(d.Body, &body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal alert message")
		return err
	}

	// Further processing can be done here, such as sending notifications
	err = w.Service.SendDiscordNotification(context.Background(), body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to send Discord message")
		return err
	}

	return nil
}
