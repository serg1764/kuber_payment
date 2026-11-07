package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"notification-service/internal/config"
)

// NewPostgresPoolWithRetry создает пул подключений к Postgres с повторными попытками.
func NewPostgresPoolWithRetry(ctx context.Context, cfg config.Config, maxAttempts int, delay time.Duration) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	var lastErr error
	// Повторяем подключение к БД заданное количество раз, чтобы пережить медленный старт контейнера.
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err := pgxpool.New(ctx, dsn)
		// Если не удалось создать пул, запоминаем ошибку и пробуем еще раз.
		if err != nil {
			lastErr = err
			log.Printf("db connection attempt %d/%d failed: %v", attempt, maxAttempts, err)
			time.Sleep(delay)
			continue
		}

		// Если ping не прошел, закрываем текущий пул и пробуем подключиться заново.
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

	// Если все попытки исчерпаны, возвращаем итоговую ошибку с контекстом.
	return nil, fmt.Errorf("database is unavailable after %d attempts: %w", maxAttempts, lastErr)
}
