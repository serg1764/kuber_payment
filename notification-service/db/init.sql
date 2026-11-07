CREATE TABLE payment_status_notifications (
    id SERIAL PRIMARY KEY,
    payment_id BIGINT NOT NULL,
    order_id BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    event_time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
