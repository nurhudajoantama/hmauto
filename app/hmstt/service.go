package hmstt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nurhudajoantama/hmauto/app/hmalert"
	"github.com/rs/zerolog"
)

type HmsttService struct {
	store *HmsttStore
	event *HmsttEvent

	hmalertService *hmalert.HmalerService
}

func NewService(hmsttStore *HmsttStore, hmsttEvent *HmsttEvent, hmalertService *hmalert.HmalerService) *HmsttService {
	return &HmsttService{
		store:          hmsttStore,
		event:          hmsttEvent,
		hmalertService: hmalertService,
	}
}

type GetStateResponse struct {
	States string `json:"state"`
}

func (s *HmsttService) GetState(ctx context.Context, tipe, key string) (string, error) {
	l := zerolog.Ctx(ctx)

	generatedKey, ok := generateKey(tipe, key)
	if !ok {
		l.Error().Err(errors.New("INVALID TYPE OR KEY")).Msg("GetState failed")
		return "", errors.New("INVALID TYPE OR KEY")
	}

	result, err := s.store.GetState(ctx, generatedKey)
	if err != nil {
		l.Error().Err(err).Msg("GetState failed")
		return "", errors.New("GET STATE ERROR")
	}

	return result.Value, nil
}

func (s *HmsttService) GetStateDetail(ctx context.Context, tipe, key string) (hmsttState, error) {
	generatedKey, ok := generateKey(tipe, key)
	if !ok {
		return hmsttState{}, errors.New("INVALID TYPE OR KEY")
	}

	result, err := s.store.GetState(ctx, generatedKey)
	if err != nil {
		return hmsttState{}, errors.New("GET STATE ERROR")
	}

	return result, nil
}

func (s *HmsttService) GetAllStates(ctx context.Context) ([]hmsttState, error) {
	results, err := s.store.GetAllStates(ctx)
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

	generatedKey, ok := canTypeChangedWithKey(tipe, key, value)
	if !ok {
		l.Error().Err(errors.New("INVALID TYPE OR KEY")).Msg("SetState failed")
		return errors.New("INVALID TYPE OR KEY")
	}

	tx := s.store.Transaction()

	state, err := s.store.GetState(ctx, generatedKey)
	if err != nil {
		l.Error().Err(err).Msg("GetState before SetState failed")
		return errors.New("GET STATE BEFORE SET ERROR")
	}

	state.Value = value

	err = s.store.SetStateTx(ctx, tx, &state)
	if err != nil {
		l.Error().Err(err).Msg("SetState failed")
		s.store.Rollback(tx)
		return errors.New("SET STATE ERROR")
	}

	err = s.event.StateChange(ctx, generatedKey, value)
	if err != nil {
		l.Error().Err(err).Msg("StateChange failed")
		s.store.Rollback(tx)
		return errors.New("STATE CHANGE ERROR")
	}
	s.store.Commit(tx)

	go s.hmalertService.PublishAlert(ctx, hmalert.PublishAlertBody{
		Tipe:    "Hmstate Change",
		Level:   hmalert.LEVEL_INFO,
		Message: fmt.Sprintf("State %s changed to %s", generatedKey, value),
	})

	return nil
}

func (s *HmsttService) RestartSwitchByKey(ctx context.Context, key string) (err error) {
	l := zerolog.Ctx(ctx)
	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_key", key)
	})
	l.Info().Msg("Handling RestartSwitchByKey service")

	err = s.SetState(ctx, PREFIX_SWITCH, key, STATE_OFF)
	if err != nil {
		l.Error().Err(err).Msg("SetState to OFF failed in RestartSwitchByKey")
		return
	}

	time.Sleep(500 * time.Millisecond)
	err = s.SetState(ctx, PREFIX_SWITCH, key, STATE_ON)
	if err != nil {
		l.Error().Err(err).Msg("SetState to ON failed in RestartSwitchByKey")
		return
	}
	return
}
