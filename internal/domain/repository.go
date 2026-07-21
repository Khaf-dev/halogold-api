package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

// UserRepository adalah "port" untuk akses data user.
// Interface didefinisikan di sisi domain (yang memakainya), implementasi
// konkret (PostgreSQL) berada di package repository. Ini menerapkan
// Dependency Inversion: service bergantung pada abstraksi, bukan pada driver DB.
type UserRepository interface {
	FindByID(ctx context.Context, id int64) (*User, error)
}

// TransactionRepository adalah port untuk akses data transaksi.
type TransactionRepository interface {
	Create(ctx context.Context, tx *Transaction) (*Transaction, error)
	ListByUser(ctx context.Context, userID int64, limit int) ([]Transaction, error)
	// GoldBalance mengembalikan saldo emas (gram) user = SUM(buy) - SUM(sell).
	GoldBalance(ctx context.Context, userID int64) (decimal.Decimal, error)
}
