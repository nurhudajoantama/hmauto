package hmstt

import (
	"context"

	"gorm.io/gorm"
)

type hmsttStore struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *hmsttStore {
	// Auto migrate the hmsttState model
	db.AutoMigrate(&hmsttState{})

	return &hmsttStore{db: db}
}

func (s *hmsttStore) SetStateTx(ctx context.Context, tx *gorm.DB, state *hmsttState) error {
	return tx.WithContext(ctx).Save(state).Error
}

func (s *hmsttStore) GetState(ctx context.Context, key string) (hmsttState, error) {
	var state hmsttState
	err := s.db.WithContext(ctx).First(&state, "key = ?", key).Error
	if err != nil {
		return hmsttState{}, err
	}
	return state, nil
}

func (s *hmsttStore) GetAllStates(ctx context.Context) ([]hmsttState, error) {
	var states []hmsttState
	err := s.db.WithContext(ctx).Order("key asc").Find(&states).Error
	if err != nil {
		return nil, err
	}
	return states, nil
}

func (s *hmsttStore) Transaction() (tx *gorm.DB) {
	return s.db.Begin()
}

func (s *hmsttStore) Commit(tx *gorm.DB) error {
	return tx.Commit().Error
}

func (s *hmsttStore) Rollback(tx *gorm.DB) error {
	return tx.Rollback().Error
}
