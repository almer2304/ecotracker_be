package config

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// NewRedis membuat Redis client. Mengembalikan nil jika Redis tidak dikonfigurasi.
func NewRedis(cfg *RedisConfig) (*redis.Client, error) {
	if cfg.Host == "" {
		logrus.Warn("Redis tidak dikonfigurasi, rate limiting & caching akan dinonaktifkan")
		return nil, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Cek koneksi
	if err := client.Ping(context.Background()).Err(); err != nil {
		logrus.WithError(err).Warn("Redis tidak dapat terhubung, melanjutkan tanpa Redis")
		return nil, nil
	}

	logrus.WithFields(logrus.Fields{
		"host": cfg.Host,
		"db":   cfg.DB,
	}).Info("Redis terhubung")

	return client, nil
}
