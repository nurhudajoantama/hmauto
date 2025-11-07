package hmstt

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
)

type hmsttService struct {
	store *hmsttStore
	event *hmsttEvent
}

func NewService(hmsttStore *hmsttStore, hmsttEvent *hmsttEvent) *hmsttService {
	return &hmsttService{
		store: hmsttStore,
		event: hmsttEvent,
	}
}

type GetStateResponse struct {
	States string `json:"state"`
}

func (s *hmsttService) GetState(ctx context.Context, tipe, key string) (string, error) {
	generatedKey, ok := generateKey(tipe, key)
	if !ok {
		return "", errors.New("INVALID TYPE OR KEY")
	}

	result, err := s.store.GetState(ctx, generatedKey)
	if err != nil {
		return "", errors.New("GET STATE ERROR")
	}

	return result.Value, nil
}

func (s *hmsttService) GetStateDetail(ctx context.Context, tipe, key string) (hmsttState, error) {
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

func (s *hmsttService) GetAllStates(ctx context.Context) ([]hmsttState, error) {
	results, err := s.store.GetAllStates(ctx)
	if err != nil {
		return nil, errors.New("GET ALL STATES ERROR")
	}

	return results, nil
}

func (s *hmsttService) SetState(ctx context.Context, tipe, key, value string) error {
	generatedKey, ok := canTypeChangedWithKey(tipe, key, value)
	if !ok {
		log.Error().Err(errors.New("INVALID TYPE OR KEY")).Msg("SetState failed")
		return errors.New("INVALID TYPE OR KEY")
	}

	tx := s.store.Transaction()

	state, err := s.store.GetState(ctx, generatedKey)
	if err != nil {
		return errors.New("GET STATE BEFORE SET ERROR")
	}

	state.Value = value

	err = s.store.SetStateTx(ctx, tx, &state)
	if err != nil {
		log.Error().Err(err).Msg("SetState failed")
		s.store.Rollback(tx)
		return errors.New("SET STATE ERROR")
	}

	err = s.event.StateChange(ctx, generatedKey, value)
	if err != nil {
		log.Error().Err(err).Msg("StateChange failed")
		s.store.Rollback(tx)
		return errors.New("STATE CHANGE ERROR")
	}
	s.store.Commit(tx)

	return nil
}
