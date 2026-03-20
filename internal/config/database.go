package config

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// NewDatabase membuat koneksi database PostgreSQL dengan connection pooling
func NewDatabase(cfg *DBConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi database: %w", err)
	}

	// Connection pooling
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Cek koneksi
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("gagal ping database: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"host":          cfg.Host,
		"database":      cfg.Name,
		"max_open_conn": cfg.MaxOpenConns,
	}).Info("Database terhubung")

	return db, nil
}
