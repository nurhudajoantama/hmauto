package hmstt

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
)

var hmsttStateChangesTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "hmstt_state_changes_total",
		Help: "Total number of state changes.",
	},
	[]string{"type"},
)

type HmsttService struct {
	store StateStore
	event *HmsttEvent
}

func NewService(hmsttStore StateStore, hmsttEvent *HmsttEvent) *HmsttService {
	return &HmsttService{
		store: hmsttStore,
		event: hmsttEvent,
	}
}

func (s *HmsttService) GetState(ctx context.Context, tipe, key string) (StateEntry, error) {
	l := zerolog.Ctx(ctx)

	if tipe == "" || key == "" {
		l.Error().Msg("GetState: empty type or key")
		return StateEntry{}, errors.New("INVALID TYPE OR KEY")
	}

	result, err := s.store.GetState(ctx, tipe, key)
	if err != nil {
		l.Error().Err(err).Msg("GetState failed")
		return StateEntry{}, errors.New("GET STATE ERROR")
	}

	return result, nil
}

func (s *HmsttService) GetAllByType(ctx context.Context, tipe string) ([]StateEntry, error) {
	results, err := s.store.GetAllByType(ctx, tipe)
	if err != nil {
		return nil, errors.New("GET ALL BY TYPE ERROR")
	}
	return results, nil
}

func (s *HmsttService) GetAllStates(ctx context.Context) ([]StateEntry, error) {
	results, err := s.store.GetAll(ctx)
	if err != nil {
		return nil, errors.New("GET ALL STATES ERROR")
	}
	return results, nil
}

func (s *HmsttService) SetState(ctx context.Context, tipe, key, value string) error {
	l := zerolog.Ctx(ctx)
	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe).Str("hmstt_key", key).Str("hmstt_value", value)
	})
	l.Info().Msg("Handling SetState service")

	if _, ok := canTypeChangedWithKey(tipe, key, value); !ok {
		l.Error().Err(errors.New("invalid type or key")).Msg("SetState failed")
		return errors.New("INVALID TYPE OR KEY")
	}

	if err := s.store.SetState(ctx, tipe, key, value); err != nil {
		l.Error().Err(err).Msg("SetState failed")
		return errors.New("SET STATE ERROR")
	}
	hmsttStateChangesTotal.WithLabelValues(tipe).Inc()

	generatedKey := PREFIX_HMSTT + KEY_DELIMITER + tipe + KEY_DELIMITER + key
	if err := s.event.StateChange(ctx, generatedKey, value); err != nil {
		l.Error().Err(err).Msg("StateChange event failed")
	}

	return nil
}
