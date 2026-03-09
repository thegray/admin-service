package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"admin-service/pkg/config"
)

const (
	dbConnectAttempts = 3
	dbConnectDelay    = 5 * time.Second
)

func Connect(ctx context.Context, cfg config.Config) (*gorm.DB, error) {
	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s",
			url.QueryEscape(cfg.DatabaseUser),
			url.QueryEscape(cfg.DatabasePassword),
			cfg.DatabaseHost,
			cfg.DatabasePort,
			url.QueryEscape(cfg.DatabaseName),
			cfg.DatabaseSSLMode,
		)
	}

	var lastErr error
	for attempt := 1; attempt <= dbConnectAttempts; attempt++ {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			sqlDB, err := db.DB()
			if err != nil {
				return nil, fmt.Errorf("accessing underlying sql db: %w", err)
			}

			configureConnPool(sqlDB)
			return db, nil
		}

		lastErr = err
		if attempt < dbConnectAttempts {
			time.Sleep(dbConnectDelay)
		}
	}

	return nil, fmt.Errorf("connecting to postgres after %d attempts: %w", dbConnectAttempts, lastErr)
}

func configureConnPool(sqlDB *sql.DB) {
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
}
