package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"halogold-api/internal/domain"
)

type TransactionRepo struct {
	pool *pgxpool.Pool
}

func NewTransactionRepo(pool *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{pool: pool}
}

var _ domain.TransactionRepository = (*TransactionRepo)(nil)

// Create menyimpan transaksi baru.
//
// Catatan robustness: gram dikirim sebagai string (via decimal.String()) dan tipe
// sebagai string biasa, sehingga tidak bergantung pada auto-mapping tipe pgx yang
// bisa berbeda antar versi. Semua nilai memakai parameter binding ($1..) untuk
// mencegah SQL injection.
func (r *TransactionRepo) Create(ctx context.Context, tx *domain.Transaction) (*domain.Transaction, error) {
	const q = `
		INSERT INTO transactions (user_id, type, amount, gram, price)
		VALUES ($1, $2, $3, $4::numeric, $5)
		RETURNING id, created_at`

	err := r.pool.QueryRow(ctx, q,
		tx.UserID,
		string(tx.Type),
		tx.Amount,
		tx.Gram.String(),
		tx.Price,
	).Scan(&tx.ID, &tx.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert transaction: %w", err)
	}
	return tx, nil
}

// ListByUser mengembalikan transaksi user, terbaru dulu.
// gram di-cast ke text lalu di-parse ke decimal secara eksplisit.
func (r *TransactionRepo) ListByUser(ctx context.Context, userID int64, limit int) ([]domain.Transaction, error) {
	const q = `
		SELECT id, user_id, type, amount, gram::text, price, created_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, q, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	txs := make([]domain.Transaction, 0, limit)
	for rows.Next() {
		var (
			t        domain.Transaction
			typeStr  string
			gramText string
		)
		if err := rows.Scan(&t.ID, &t.UserID, &typeStr, &t.Amount, &gramText, &t.Price, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}

		gram, err := decimal.NewFromString(gramText)
		if err != nil {
			return nil, fmt.Errorf("parse gram %q: %w", gramText, err)
		}
		t.Type = domain.TransactionType(typeStr)
		t.Gram = gram
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}
	return txs, nil
}

// GoldBalance menghitung saldo emas user langsung di DB:
// SUM(gram beli) - SUM(gram jual). COALESCE memastikan hasil 0 bila belum ada
// transaksi. Hasil di-cast ke text agar parsing decimal-nya deterministik.
func (r *TransactionRepo) GoldBalance(ctx context.Context, userID int64) (decimal.Decimal, error) {
	const q = `
		SELECT COALESCE(SUM(
			CASE WHEN type = 'buy' THEN gram ELSE -gram END
		), 0)::text
		FROM transactions
		WHERE user_id = $1`

	var balanceText string
	if err := r.pool.QueryRow(ctx, q, userID).Scan(&balanceText); err != nil {
		return decimal.Zero, fmt.Errorf("gold balance: %w", err)
	}

	balance, err := decimal.NewFromString(balanceText)
	if err != nil {
		return decimal.Zero, fmt.Errorf("parse balance %q: %w", balanceText, err)
	}
	return balance, nil
}
