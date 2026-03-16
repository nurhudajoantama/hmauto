package hmapikey

import (
	"context"

	"github.com/nurhudajoantama/hmauto/internal/apikey"
)

type Service struct {
	store apikey.Store
}

func NewService(store apikey.Store) *Service {
	return &Service{store: store}
}

func (s *Service) ListKeys(ctx context.Context) ([]apikey.KeyMetadata, error) {
	return s.store.ListKeys(ctx)
}

func (s *Service) CreateKey(ctx context.Context, label string) (string, error) {
	return s.store.CreateKey(ctx, label)
}

func (s *Service) RevokeKey(ctx context.Context, key string) error {
	return s.store.RevokeKey(ctx, key)
}
