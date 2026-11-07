package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"payment-service/internal/config"
)

func NewPostgresPoolWithRetry(ctx context.Context, cfg config.Config, maxAttempts int, delay time.Duration) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			lastErr = err
			log.Printf("db connection attempt %d/%d failed: %v", attempt, maxAttempts, err)
			time.Sleep(delay)
			continue
		}

		if err := pool.Ping(ctx); err != nil {
			lastErr = err
			pool.Close()
			log.Printf("db ping attempt %d/%d failed: %v", attempt, maxAttempts, err)
			time.Sleep(delay)
			continue
		}

		log.Printf("db connection established on attempt %d", attempt)
		return pool, nil
	}

	return nil, fmt.Errorf("database is unavailable after %d attempts: %w", maxAttempts, lastErr)
}
