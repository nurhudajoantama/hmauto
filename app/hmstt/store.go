package hmstt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
)

type StateEntry struct {
	Type      string
	K         string
	Value     string
	UpdatedAt time.Time
}

type StateStore interface {
	GetState(ctx context.Context, tipe, k string) (StateEntry, error)
	SetState(ctx context.Context, tipe, k, value string) error
	GetAllByType(ctx context.Context, tipe string) ([]StateEntry, error)
	GetAll(ctx context.Context) ([]StateEntry, error)
}

type stateEntryJSON struct {
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HmsttStore struct {
	rdb *redis.Client
}

func NewStore(rdb *redis.Client) *HmsttStore {
	return &HmsttStore{rdb: rdb}
}

func redisKey(tipe string) string {
	return "hmstt:" + tipe
}

func (s *HmsttStore) SetState(ctx context.Context, tipe, k, value string) error {
	ctx, span := otel.Tracer("hmstt").Start(ctx, "store.SetState")
	defer span.End()

	entry := stateEntryJSON{
		Value:     value,
		UpdatedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal state entry: %w", err)
	}
	if err := s.rdb.HSet(ctx, redisKey(tipe), k, data).Err(); err != nil {
		return fmt.Errorf("redis HSET: %w", err)
	}
	return nil
}

func (s *HmsttStore) GetState(ctx context.Context, tipe, k string) (StateEntry, error) {
	ctx, span := otel.Tracer("hmstt").Start(ctx, "store.GetState")
	defer span.End()

	data, err := s.rdb.HGet(ctx, redisKey(tipe), k).Bytes()
	if err != nil {
		return StateEntry{}, fmt.Errorf("redis HGET: %w", err)
	}
	var entry stateEntryJSON
	if err := json.Unmarshal(data, &entry); err != nil {
		return StateEntry{}, fmt.Errorf("unmarshal state entry: %w", err)
	}
	return StateEntry{Type: tipe, K: k, Value: entry.Value, UpdatedAt: entry.UpdatedAt}, nil
}

func (s *HmsttStore) GetAllByType(ctx context.Context, tipe string) ([]StateEntry, error) {
	ctx, span := otel.Tracer("hmstt").Start(ctx, "store.GetAllByType")
	defer span.End()

	result, err := s.rdb.HGetAll(ctx, redisKey(tipe)).Result()
	if err != nil {
		return nil, fmt.Errorf("redis HGETALL: %w", err)
	}
	entries := make([]StateEntry, 0, len(result))
	for k, v := range result {
		var entry stateEntryJSON
		if err := json.Unmarshal([]byte(v), &entry); err != nil {
			return nil, fmt.Errorf("unmarshal state entry for key %s: %w", k, err)
		}
		entries = append(entries, StateEntry{Type: tipe, K: k, Value: entry.Value, UpdatedAt: entry.UpdatedAt})
	}
	return entries, nil
}

func (s *HmsttStore) GetAll(ctx context.Context) ([]StateEntry, error) {
	ctx, span := otel.Tracer("hmstt").Start(ctx, "store.GetAll")
	defer span.End()

	keys, err := s.rdb.Keys(ctx, "hmstt:*").Result()
	if err != nil {
		return nil, fmt.Errorf("redis KEYS: %w", err)
	}
	var all []StateEntry
	for _, key := range keys {
		tipe := strings.TrimPrefix(key, "hmstt:")
		entries, err := s.GetAllByType(ctx, tipe)
		if err != nil {
			return nil, err
		}
		all = append(all, entries...)
	}
	return all, nil
}
