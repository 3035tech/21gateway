package database

import (
	"context"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/21mobile/sms-gateway/internal/config"
)

func Open(ctx context.Context, dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{DisableAutomaticPing: true, TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(config.DatabaseMaxOpenConnections)
	pool.SetMaxIdleConns(config.DatabaseMaxIdleConnections)
	pool.SetConnMaxLifetime(config.DatabaseConnectionLifetime)
	pool.SetConnMaxIdleTime(config.DatabaseConnectionIdleTime)
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, err
	}
	return db, nil
}
