package hmmon

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/nurhudajoantama/hmauto/app/hmalert"
	"github.com/nurhudajoantama/hmauto/app/hmstt"
	"github.com/nurhudajoantama/hmauto/app/worker"
	"github.com/nurhudajoantama/hmauto/internal/config"
	"github.com/rs/zerolog/log"
)

const (
	DISCORD_TIPE = "Cek Internet"
)

type HmmonWorker struct {
	hmsttService   *hmstt.HmsttService
	hmalertService *hmalert.HmalerService
	intercheckCfg  config.InternetCheck
}

func RegisterWorkers(s *worker.Worker, hmsttService *hmstt.HmsttService, hmalertService *hmalert.HmalerService, intercheckCfg config.InternetCheck) {
	hw := &HmmonWorker{
		hmsttService:   hmsttService,
		hmalertService: hmalertService,
		intercheckCfg:  intercheckCfg,
	}

	s.Go(hw.internetWorker)
}

func (w *HmmonWorker) internetWorker(ctx context.Context) error {
	interval, err := time.ParseDuration(w.intercheckCfg.Interval)
	if err != nil {
		log.Error().Err(err).Msg("invalid internet check interval duration, using default 1 minute")
		interval = 2 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("hmstt internet worker stopped")
			return nil
		case <-ticker.C:
			pingCheckNetOk := pingInternet(w.intercheckCfg.CheckAddress)
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

func (w *HmmonWorker) internetWorkerSwitchModem(ctx context.Context) error {
	exp := backoff.NewExponentialBackOff()
	exp.InitialInterval = 30 * time.Second
	exp.MaxInterval = 10 * time.Minute
	exp.MaxElapsedTime = 0
	exp.RandomizationFactor = 0.3
	exp.Multiplier = 3.0

	bo := backoff.WithContext(exp, ctx)

	return backoff.Retry(func() error {
		pingCheckModemOk := pingInternet(w.intercheckCfg.ModemAddress)
		if !pingCheckModemOk {
			log.Print("modem connection is down")
			w.hmalertService.PublishAlert(context.Background(), DISCORD_TIPE, hmalert.LEVEL_WARNING, fmt.Sprintf("Fail Ping %s, Modem connection is down ❌, cannot restart modem 🔄", w.intercheckCfg.ModemAddress))
			return errors.New("modem connection is down, cannot restart modem (will retry)")
		}

		pingCheckNetOk := pingInternet(w.intercheckCfg.CheckAddress)
		if pingCheckNetOk {
			w.hmalertService.PublishAlert(context.Background(), DISCORD_TIPE, hmalert.LEVEL_INFO, fmt.Sprintf("Ping %s success, Internet connection is up ✅", w.intercheckCfg.CheckAddress))
			log.Print("internet connection is up")
			return nil
		}
		w.hmalertService.PublishAlert(context.Background(), DISCORD_TIPE, hmalert.LEVEL_INFO, fmt.Sprintf("Ping %s failed, Internet connection is down ❌, restarting modem 🔄", w.intercheckCfg.CheckAddress))

		log.Print("internet connection is down, restarting modem")

		err := w.hmsttService.RestartSwitchByKey(ctx, w.intercheckCfg.SwitchKey)
		if err != nil {
			log.Printf("restart modem failed: %v (will retry)", err)
			return err
		}
		log.Print("restart modem success")

		return errors.New("internet still down after modem restart (will retry)")
	}, bo)

}
