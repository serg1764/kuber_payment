package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrPaymentNotFound = errors.New("payment not found")

type Payment struct {
	ID        int64   `json:"id"`
	OrderID   int64   `json:"order_id"`
	Provider  string  `json:"provider"`
	Status    string  `json:"status"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	CreatedAt string  `json:"created_at"`
}

type CreatePaymentRequest struct {
	OrderID  int64   `json:"order_id"`
	Provider string  `json:"provider"`
	Status   string  `json:"status"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type PaymentRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

func (r *PaymentRepository) GetPayments(ctx context.Context) ([]Payment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, order_id, provider, status, amount::float8, currency, created_at::text
		FROM payments
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	payments := make([]Payment, 0)
	for rows.Next() {
		var p Payment
		if err := rows.Scan(&p.ID, &p.OrderID, &p.Provider, &p.Status, &p.Amount, &p.Currency, &p.CreatedAt); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return payments, nil
}

func (r *PaymentRepository) GetPaymentByID(ctx context.Context, id int64) (Payment, error) {
	var p Payment
	err := r.pool.QueryRow(ctx, `
		SELECT id, order_id, provider, status, amount::float8, currency, created_at::text
		FROM payments
		WHERE id = $1
	`, id).Scan(&p.ID, &p.OrderID, &p.Provider, &p.Status, &p.Amount, &p.Currency, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Payment{}, ErrPaymentNotFound
		}
		return Payment{}, err
	}

	return p, nil
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, req CreatePaymentRequest) (Payment, error) {
	var p Payment
	err := r.pool.QueryRow(ctx, `
		INSERT INTO payments (order_id, provider, status, amount, currency)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, order_id, provider, status, amount::float8, currency, created_at::text
	`, req.OrderID, req.Provider, req.Status, req.Amount, req.Currency).Scan(
		&p.ID,
		&p.OrderID,
		&p.Provider,
		&p.Status,
		&p.Amount,
		&p.Currency,
		&p.CreatedAt,
	)
	if err != nil {
		return Payment{}, err
	}

	return p, nil
}
