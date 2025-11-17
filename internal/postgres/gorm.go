package postgres

import (
	"context"
	"time"

	log "github.com/rs/zerolog/log"

	"github.com/nurhudajoantama/hmauto/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewGorm(c config.SQL) *gorm.DB {
	db, err := gorm.Open(postgres.Open(c.DataSourceName()), &gorm.Config{})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get database instance")
	}

	sqlDB.SetMaxIdleConns(c.MaxIdleConn)
	sqlDB.SetMaxOpenConns(c.MaxOpenConn)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db
}

func Close(ctx context.Context, db *gorm.DB) {
	c := make(chan struct{})
	go func() {
		sqlDB, err := db.DB()
		if err != nil {
			log.Error().Err(err).Msg("failed to get database instance for closing")
			return
		}
		if err := sqlDB.Close(); err != nil {
			log.Error().Err(err).Msg("failed to close database connection")
		}
		close(c)
	}()

	select {
	case <-ctx.Done():
		log.Warn().Msg("timeout while closing database connection")
	case <-c:
		log.Info().Msg("database connection closed")
	}
}
