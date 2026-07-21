package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransactionType membedakan transaksi beli vs jual emas.
type TransactionType string

const (
	TransactionBuy  TransactionType = "buy"
	TransactionSell TransactionType = "sell"
)

// Valid memastikan tipe transaksi hanya salah satu dari nilai yang diizinkan.
func (t TransactionType) Valid() bool {
	return t == TransactionBuy || t == TransactionSell
}

// Transaction merepresentasikan satu baris transaksi emas.
//
// Catatan desain:
//   - Amount & Price disimpan sebagai int64 (rupiah, tanpa pecahan sen) karena
//     harga emas per gram di sistem ini bilangan bulat.
//   - Gram memakai decimal.Decimal (bukan float64) untuk menghindari galat
//     floating-point pada perhitungan finansial. Di DB dipetakan ke NUMERIC(20,8).
//   - Price disimpan per-transaksi (harga saat transaksi terjadi) demi audit trail
//     yang akurat — ini penambahan dari skema minimum di BRD, dan disengaja.
type Transaction struct {
	ID        int64           `json:"id"`
	UserID    int64           `json:"user_id"`
	Type      TransactionType `json:"type"`
	Amount    int64           `json:"amount"` // rupiah
	Gram      decimal.Decimal `json:"gram"`
	Price     int64           `json:"price"` // rupiah per gram saat transaksi
	CreatedAt time.Time       `json:"created_at"`
}
