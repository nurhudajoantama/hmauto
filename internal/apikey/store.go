package apikey

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrKeyNotFound is returned when an API key does not exist.
var ErrKeyNotFound = errors.New("api key not found")

type KeyMetadata struct {
	KeyHint   string    `json:"key_hint"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used,omitempty"`
}

type keyRecord struct {
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used,omitempty"`
}

// Store defines the API key management interface.
type Store interface {
	CreateKey(ctx context.Context, label string) (key string, err error)
	ValidateKey(ctx context.Context, key string) (bool, error)
	RevokeKey(ctx context.Context, key string) error
	ListKeys(ctx context.Context) ([]KeyMetadata, error)
}

// RedisStore implements Store using Redis.
type RedisStore struct {
	rdb    *redis.Client
	prefix string
}

func NewRedisStore(rdb *redis.Client, prefix string) *RedisStore {
	return &RedisStore{rdb: rdb, prefix: prefix}
}

func (s *RedisStore) redisKey(key string) string {
	return s.prefix + ":apikey:" + key
}

func (s *RedisStore) indexKey() string {
	return s.prefix + ":apikeys:index"
}

func keyHint(key string) string {
	if len(key) <= 8 {
		return key
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func (s *RedisStore) CreateKey(ctx context.Context, label string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	key := hex.EncodeToString(raw)

	record := keyRecord{
		Label:     label,
		CreatedAt: time.Now().UTC(),
	}
	data, err := json.Marshal(record)
	if err != nil {
		return "", fmt.Errorf("marshal key record: %w", err)
	}

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, s.redisKey(key), data, 0)
	pipe.SAdd(ctx, s.indexKey(), key)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("redis pipeline: %w", err)
	}

	return key, nil
}

func (s *RedisStore) ValidateKey(ctx context.Context, key string) (bool, error) {
	data, err := s.rdb.Get(ctx, s.redisKey(key)).Bytes()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis GET: %w", err)
	}

	// Update last_used asynchronously — non-blocking.
	go func() {
		var record keyRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return
		}
		record.LastUsed = time.Now().UTC()
		updated, err := json.Marshal(record)
		if err != nil {
			return
		}
		// Use a background context since the request context may be cancelled.
		bgCtx := context.Background()
		s.rdb.Set(bgCtx, s.redisKey(key), updated, 0) //nolint:errcheck
	}()

	return true, nil
}

func (s *RedisStore) RevokeKey(ctx context.Context, key string) error {
	n, err := s.rdb.Del(ctx, s.redisKey(key)).Result()
	if err != nil {
		return fmt.Errorf("redis DEL: %w", err)
	}
	if n == 0 {
		return ErrKeyNotFound
	}
	if err := s.rdb.SRem(ctx, s.indexKey(), key).Err(); err != nil {
		return fmt.Errorf("redis SREM: %w", err)
	}
	return nil
}

func (s *RedisStore) ListKeys(ctx context.Context) ([]KeyMetadata, error) {
	members, err := s.rdb.SMembers(ctx, s.indexKey()).Result()
	if err != nil {
		return nil, fmt.Errorf("redis SMEMBERS: %w", err)
	}

	result := make([]KeyMetadata, 0, len(members))
	for _, member := range members {
		data, err := s.rdb.Get(ctx, s.redisKey(member)).Bytes()
		if err == redis.Nil {
			// Key was revoked but index not cleaned up — skip.
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("redis GET for key %s: %w", member, err)
		}
		var record keyRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("unmarshal key record: %w", err)
		}
		result = append(result, KeyMetadata{
			KeyHint:   keyHint(member),
			Label:     record.Label,
			CreatedAt: record.CreatedAt,
			LastUsed:  record.LastUsed,
		})
	}
	return result, nil
}
