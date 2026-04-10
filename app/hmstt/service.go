package hmstt

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
)

var ErrStateAlreadyExists = errors.New("STATE ALREADY EXISTS")
var ErrStateNotFound = errors.New("STATE NOT FOUND")
var ErrNothingToUpdate = errors.New("NOTHING TO UPDATE")

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

func (s *HmsttService) GetStatesByKeys(ctx context.Context, tipe string, keys []string) ([]StateEntry, error) {
	l := zerolog.Ctx(ctx)

	if tipe == "" {
		return nil, errors.New("INVALID TYPE")
	}
	if len(keys) == 0 {
		return nil, errors.New("NO KEYS PROVIDED")
	}

	entries, err := s.store.GetAllByType(ctx, tipe)
	if err != nil {
		return nil, errors.New("GET ALL BY TYPE ERROR")
	}

	entryByKey := make(map[string]StateEntry, len(entries))
	for _, entry := range entries {
		entryByKey[entry.K] = entry
	}

	results := make([]StateEntry, 0, len(keys))
	for _, key := range keys {
		entry, ok := entryByKey[key]
		if !ok {
			l.Warn().Str("hmstt_type", tipe).Str("hmstt_key", key).Msg("batch sync key not found")
			continue
		}
		results = append(results, entry)
	}

	return results, nil
}

func (s *HmsttService) CreateState(ctx context.Context, tipe, key, value, description string) error {
	l := zerolog.Ctx(ctx)
	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe).Str("hmstt_key", key).Str("hmstt_value", value).Str("hmstt_description", description)
	})
	l.Info().Msg("Handling CreateState service")

	if !canTypeChangedWithKey(tipe, key, value) {
		l.Error().Msg("CreateState: invalid type or key")
		return errors.New("INVALID TYPE OR KEY")
	}

	if _, err := s.store.GetState(ctx, tipe, key); err == nil {
		return ErrStateAlreadyExists
	}

	if err := s.store.SetState(ctx, tipe, key, value, description); err != nil {
		l.Error().Err(err).Msg("CreateState failed")
		return errors.New("SET STATE ERROR")
	}
	hmsttStateChangesTotal.WithLabelValues(tipe).Inc()

	generatedKey := PREFIX_HMSTT + KEY_DELIMITER + tipe + KEY_DELIMITER + key
	if err := s.event.StateChange(ctx, generatedKey, value); err != nil {
		l.Error().Err(err).Msg("StateChange event failed")
	}

	return nil
}

// SetState updates the value of an existing state entry (creates if not exists).
// If description is nil, the existing description is preserved.
// MQTT event is fired only when the value actually changes.
func (s *HmsttService) SetState(ctx context.Context, tipe, key, value string, description *string) error {
	l := zerolog.Ctx(ctx)
	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe).Str("hmstt_key", key).Str("hmstt_value", value)
	})
	l.Info().Msg("Handling SetState service")

	if !canTypeChangedWithKey(tipe, key, value) {
		l.Error().Err(errors.New("invalid type or key")).Msg("SetState failed")
		return errors.New("INVALID TYPE OR KEY")
	}

	current, err := s.store.GetState(ctx, tipe, key)
	valueChanged := err != nil || current.Value != value

	desc := current.Description
	if description != nil {
		desc = *description
		l.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("hmstt_description", desc)
		})
	}

	if err := s.store.SetState(ctx, tipe, key, value, desc); err != nil {
		l.Error().Err(err).Msg("SetState failed")
		return errors.New("SET STATE ERROR")
	}

	if valueChanged {
		hmsttStateChangesTotal.WithLabelValues(tipe).Inc()
		generatedKey := PREFIX_HMSTT + KEY_DELIMITER + tipe + KEY_DELIMITER + key
		if err := s.event.StateChange(ctx, generatedKey, value); err != nil {
			l.Error().Err(err).Msg("StateChange event failed")
		}
	}

	return nil
}

// PatchState partially updates value and/or description of an existing state entry.
// At least one of value or description must be non-nil.
// MQTT event is fired only when the value actually changes.
func (s *HmsttService) PatchState(ctx context.Context, tipe, key string, value *string, description *string) error {
	l := zerolog.Ctx(ctx)
	l.UpdateContext(func(c zerolog.Context) zerolog.Context {
		return c.Str("hmstt_type", tipe).Str("hmstt_key", key)
	})
	l.Info().Msg("Handling PatchState service")

	if value == nil && description == nil {
		return ErrNothingToUpdate
	}

	current, err := s.store.GetState(ctx, tipe, key)
	if err != nil {
		l.Error().Err(err).Msg("PatchState: state not found")
		return ErrStateNotFound
	}

	newValue := current.Value
	newDesc := current.Description
	valueChanged := false

	if value != nil {
		if !canTypeChangedWithKey(tipe, key, *value) {
			l.Error().Msg("PatchState: invalid type or key")
			return errors.New("INVALID TYPE OR KEY")
		}
		if current.Value != *value {
			valueChanged = true
			newValue = *value
		}
		l.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("hmstt_value", newValue)
		})
	}

	if description != nil {
		newDesc = *description
		l.UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("hmstt_description", newDesc)
		})
	}

	if err := s.store.SetState(ctx, tipe, key, newValue, newDesc); err != nil {
		l.Error().Err(err).Msg("PatchState failed")
		return errors.New("SET STATE ERROR")
	}

	if valueChanged {
		hmsttStateChangesTotal.WithLabelValues(tipe).Inc()
		generatedKey := PREFIX_HMSTT + KEY_DELIMITER + tipe + KEY_DELIMITER + key
		if err := s.event.StateChange(ctx, generatedKey, newValue); err != nil {
			l.Error().Err(err).Msg("StateChange event failed")
		}
	}

	return nil
}
