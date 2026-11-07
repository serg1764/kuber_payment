CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL,
    provider VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    amount NUMERIC(12, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO payments (order_id, provider, status, amount, currency)
VALUES
    (1001, 'paypal', 'created', 120.50, 'EUR'),
    (1002, 'stripe', 'paid', 89.99, 'EUR'),
    (1003, 'klarna', 'failed', 250.00, 'EUR'),
    (1004, 'payu', 'paid', 310.20, 'EUR');
