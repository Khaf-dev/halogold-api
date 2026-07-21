-- Skema awal HaloGold API (idempotent — aman dijalankan berulang).

CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL PRIMARY KEY,
    nama       VARCHAR(150) NOT NULL,
    email      VARCHAR(150) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS transactions (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id),
    type       VARCHAR(10)  NOT NULL CHECK (type IN ('buy', 'sell')),
    amount     BIGINT       NOT NULL CHECK (amount >= 0),   -- rupiah
    gram       NUMERIC(20,8) NOT NULL CHECK (gram >= 0),    -- presisi tinggi, hindari float
    price      BIGINT       NOT NULL CHECK (price > 0),     -- harga per gram saat transaksi
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Index untuk mempercepat query daftar transaksi & saldo per user.
CREATE INDEX IF NOT EXISTS idx_transactions_user_created
    ON transactions (user_id, created_at DESC);

-- Seed user default (login tidak diwajibkan di BRD).
INSERT INTO users (id, nama, email)
VALUES (1, 'Budi Investor', 'budi@halogold.id')
ON CONFLICT (id) DO NOTHING;

-- Sinkronkan sequence agar insert user berikutnya tidak bentrok dengan id seed.
SELECT setval(pg_get_serial_sequence('users', 'id'), GREATEST((SELECT MAX(id) FROM users), 1));
