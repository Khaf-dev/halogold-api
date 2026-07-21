package service

import (
	"context"

	"github.com/shopspring/decimal"

	"halogold-api/internal/domain"
)

// GramScale = jumlah desimal gram yang dipakai untuk penyimpanan & tampilan.
// Contoh di BRD: amount 500000 -> gram 0.2571 (4 desimal).
const GramScale int32 = 4

// MinBuyAmount = nominal pembelian minimum (rupiah). Contoh validasi bisnis.
const MinBuyAmount int64 = 10_000

// GoldService berisi seluruh aturan bisnis emas.
// Ia hanya bergantung pada abstraksi (repository & price provider),
// sehingga bisa di-unit-test tanpa DB atau HTTP.
type GoldService struct {
	txRepo domain.TransactionRepository
	price  PriceProvider
}

func NewGoldService(txRepo domain.TransactionRepository, price PriceProvider) *GoldService {
	return &GoldService{txRepo: txRepo, price: price}
}

// CurrentPrice mengembalikan harga emas terkini (rupiah/gram).
func (s *GoldService) CurrentPrice() int64 {
	return s.price.CurrentPrice()
}

// Buy mengonversi nominal rupiah menjadi gram emas lalu mencatat transaksi.
//
// Aturan presisi (PENTING untuk aplikasi finansial):
//   - Perhitungan memakai decimal, bukan float64, agar tidak ada galat pembulatan.
//   - gram = amount / price, dibulatkan KE BAWAH (ROUND_DOWN) ke GramScale desimal.
//     Membulatkan ke bawah memastikan sistem tidak pernah "memberi" emas lebih
//     banyak dari yang seharusnya (favor the house / mencegah kebocoran nilai).
func (s *GoldService) Buy(ctx context.Context, userID, amount int64) (*domain.Transaction, error) {
	if amount < MinBuyAmount {
		return nil, domain.NewAppError(
			domain.CodeInvalidAmount,
			"Nominal pembelian minimal Rp"+decimal.NewFromInt(MinBuyAmount).String(),
			nil,
		)
	}

	price := s.price.CurrentPrice()
	if price <= 0 {
		return nil, domain.NewAppError(domain.CodeInternal, "Harga emas tidak tersedia", nil)
	}

	gram := decimal.NewFromInt(amount).
		DivRound(decimal.NewFromInt(price), GramScale+2). // buffer presisi
		Truncate(GramScale)                               // round down ke 4 desimal

	if gram.IsZero() {
		return nil, domain.NewAppError(
			domain.CodeInvalidAmount,
			"Nominal terlalu kecil untuk mendapatkan emas pada harga saat ini",
			nil,
		)
	}

	tx := &domain.Transaction{
		UserID: userID,
		Type:   domain.TransactionBuy,
		Amount: amount,
		Gram:   gram,
		Price:  price,
	}
	return s.txRepo.Create(ctx, tx)
}

// Sell mengonversi gram emas menjadi rupiah lalu mencatat transaksi.
//
// Aturan:
//   - Validasi gram > 0.
//   - Validasi saldo: gram yang dijual tidak boleh melebihi saldo emas user.
//   - amount = gram * price, dibulatkan KE BAWAH ke rupiah bulat (int64).
func (s *GoldService) Sell(ctx context.Context, userID int64, gram decimal.Decimal) (*domain.Transaction, error) {
	if gram.LessThanOrEqual(decimal.Zero) {
		return nil, domain.NewAppError(domain.CodeInvalidGram, "Jumlah gram harus lebih dari 0", nil)
	}

	// Normalisasi gram ke skala yang sama dengan penyimpanan.
	gram = gram.Truncate(GramScale)

	balance, err := s.txRepo.GoldBalance(ctx, userID)
	if err != nil {
		return nil, err
	}
	if gram.GreaterThan(balance) {
		return nil, domain.NewAppError(
			domain.CodeInsufficientGold,
			"Saldo emas tidak mencukupi. Saldo saat ini "+balance.String()+" gram",
			nil,
		)
	}

	price := s.price.CurrentPrice()
	amount := gram.Mul(decimal.NewFromInt(price)).
		Truncate(0). // round down ke rupiah bulat
		IntPart()

	tx := &domain.Transaction{
		UserID: userID,
		Type:   domain.TransactionSell,
		Amount: amount,
		Gram:   gram,
		Price:  price,
	}
	return s.txRepo.Create(ctx, tx)
}

// ListTransactions mengembalikan daftar transaksi user (terbaru dulu).
func (s *GoldService) ListTransactions(ctx context.Context, userID int64, limit int) ([]domain.Transaction, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.txRepo.ListByUser(ctx, userID, limit)
}

// Balance mengembalikan saldo emas user dalam gram.
func (s *GoldService) Balance(ctx context.Context, userID int64) (decimal.Decimal, error) {
	return s.txRepo.GoldBalance(ctx, userID)
}
