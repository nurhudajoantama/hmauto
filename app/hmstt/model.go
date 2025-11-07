package hmstt

import "time"

type hmsttState struct {
	Key       string    `gorm:"primaryKey"`
	Type      string    `gorm:"default:''"`
	K         string    `gorm:"default:''"`
	Title     string    `gorm:"default:''"`
	Value     string    `gorm:"default:''"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (hmsttState) TableName() string {
	return "hmstt_states"
}
