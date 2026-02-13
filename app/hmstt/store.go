package hmstt

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type HmsttStore struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *HmsttStore {
	if err := db.AutoMigrate(&hmsttState{}); err != nil {
		panic(fmt.Errorf("failed to migrate hmstt state: %w", err))
	}

	return &HmsttStore{db: db}
}

func (s *HmsttStore) SetStateTx(ctx context.Context, tx *gorm.DB, state *hmsttState) error {
	return tx.WithContext(ctx).Save(state).Error
}

func (s *HmsttStore) GetState(ctx context.Context, key string) (hmsttState, error) {
	var state hmsttState
	err := s.db.WithContext(ctx).First(&state, "key = ?", key).Error
	if err != nil {
		return hmsttState{}, err
	}
	return state, nil
}

func (s *HmsttStore) GetAllStates(ctx context.Context) ([]hmsttState, error) {
	var states []hmsttState
	err := s.db.WithContext(ctx).Order("key asc").Find(&states).Error
	if err != nil {
		return nil, err
	}
	return states, nil
}

func (s *HmsttStore) Transaction() (tx *gorm.DB) {
	return s.db.Begin()
}

func (s *HmsttStore) Commit(tx *gorm.DB) error {
	return tx.Commit().Error
}

func (s *HmsttStore) Rollback(tx *gorm.DB) error {
	return tx.Rollback().Error
}
