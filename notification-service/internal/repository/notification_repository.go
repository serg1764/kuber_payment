package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PaymentStatusRecord хранит одну запись статуса платежа из БД notification-service.
type PaymentStatusRecord struct {
	ID        int64
	PaymentID int64
	OrderID   int64
	Status    string
	Provider  string
	EventTime time.Time
	CreatedAt time.Time
}

// NotificationRepository инкапсулирует SQL-операции для таблицы статусов.
type NotificationRepository struct {
	pool *pgxpool.Pool
}

// NewNotificationRepository создает репозиторий с готовым пулом подключений к Postgres.
func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

// CreatePaymentStatus записывает новый статус платежа и возвращает id созданной строки.
func (r *NotificationRepository) CreatePaymentStatus(ctx context.Context, paymentID int64, orderID int64, status string, provider string, eventTime time.Time) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO payment_status_notifications (payment_id, order_id, status, provider, event_time)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, paymentID, orderID, status, provider, eventTime).Scan(&id)
	// Если INSERT завершился ошибкой, возвращаем ее вызывающему слою.
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetPaymentStatusesByPaymentID возвращает историю статусов по payment_id в порядке создания.
func (r *NotificationRepository) GetPaymentStatusesByPaymentID(ctx context.Context, paymentID int64) ([]PaymentStatusRecord, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, payment_id, order_id, status, provider, event_time, created_at
		FROM payment_status_notifications
		WHERE payment_id = $1
		ORDER BY id
	`, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PaymentStatusRecord, 0)
	// Проходим по всем найденным строкам и превращаем их в структуры доменной модели.
	for rows.Next() {
		var item PaymentStatusRecord
		err := rows.Scan(
			&item.ID,
			&item.PaymentID,
			&item.OrderID,
			&item.Status,
			&item.Provider,
			&item.EventTime,
			&item.CreatedAt,
		)
		// Если не удалось прочитать текущую строку, останавливаем обработку и возвращаем ошибку.
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	// Проверяем накопленную ошибку итератора после завершения цикла rows.Next().
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
