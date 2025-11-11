package hmstt

import (
	"context"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/rs/zerolog/log"
)

func PingInternet(address string) bool {
	pinger, err := probing.NewPinger(address)
	if err != nil {
		panic(err)
	}
	pinger.Count = 3

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	c := make(chan struct{})

	go func() {
		_ = pinger.Run()
		close(c)
	}()

	select {
	case <-ctx.Done():
	case <-c:
	}

	pinger.Stop()

	cancel()

	stats := pinger.Statistics()

	ok := stats.PacketsRecv > 0
	if !ok {
		log.Warn().Msgf("Internet ping to %s failed", address)
	} else {
		log.Info().Msgf("Internet ping to %s successful", address)
	}

	return ok
}

// func
