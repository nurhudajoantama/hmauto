package hmstt

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/nurhudajoantama/stthmauto/app/worker"
	"github.com/rs/zerolog/log"
)

type hmsttWorker struct {
	service *hmsttService
}

func RegisterWorkers(s *worker.Worker, svc *hmsttService) {
	hw := &hmsttWorker{
		service: svc,
	}

	s.Go(func(ctx context.Context) func() error {
		return hw.internetWorker(ctx)
	})
}

func (w *hmsttWorker) internetWorker(ctx context.Context) func() error {
	return func() error {
		ticker := time.NewTicker(INTERVAL_NET_CHECK * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("hmstt internet worker stopped")
				return nil
			case <-ticker.C:
				pingCheckNetOk := PingInternet(INTERNET_CHECK_ADDRESS)
				if !pingCheckNetOk {
					log.Print("modem connection is down, just wait")
					err := w.internetWorkerSwitchModem(ctx)
					if err != nil {
						log.Error().Err(err).Msg("hmstt internet worker switch error")
					}
				}
			}
		}
	}
}

func (w *hmsttWorker) internetWorkerSwitchModem(ctx context.Context) error {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = 30 * time.Second
	exp.MaxInterval = 10 * time.Minute
	exp.MaxElapsedTime = 0
	exp.RandomizationFactor = 0.3
	exp.Multiplier = 3.0

	bo := backoff.WithContext(exp, ctx)

	return backoff.Retry(func() error {

		pingCheckModemOk := PingInternet(INTERNET_MODEM_ADDRESS)
		if !pingCheckModemOk {
			log.Print("modem connection is down")
			return errors.New("modem connection is down, cannot restart modem (will retry)")
		}

		pingCheckNetOk := PingInternet(INTERNET_CHECK_ADDRESS)
		if pingCheckNetOk {
			log.Print("internet connection is down")
			return nil
		}

		log.Print("internet connection is down, restarting modem")

		err := w.service.RestartModem(ctx)
		if err != nil {
			log.Printf("restart modem failed: %v (will retry)", err)
			return err
		}
		log.Print("restart modem success")

		return errors.New("internet still down after modem restart (will retry)")
	}, bo)

}
