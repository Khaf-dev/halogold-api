package service

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"halogold-api/internal/domain"
)

// mockTxRepo adalah implementasi in-memory dari TransactionRepository
// untuk keperluan unit test (tanpa PostgreSQL).
type mockTxRepo struct {
	txs    []domain.Transaction
	nextID int64
}

func newMockTxRepo() *mockTxRepo { return &mockTxRepo{nextID: 1} }

func (m *mockTxRepo) Create(_ context.Context, tx *domain.Transaction) (*domain.Transaction, error) {
	tx.ID = m.nextID
	m.nextID++
	m.txs = append(m.txs, *tx)
	return tx, nil
}

func (m *mockTxRepo) ListByUser(_ context.Context, userID int64, limit int) ([]domain.Transaction, error) {
	var out []domain.Transaction
	for i := len(m.txs) - 1; i >= 0 && len(out) < limit; i-- {
		if m.txs[i].UserID == userID {
			out = append(out, m.txs[i])
		}
	}
	return out, nil
}

func (m *mockTxRepo) GoldBalance(_ context.Context, userID int64) (decimal.Decimal, error) {
	bal := decimal.Zero
	for _, t := range m.txs {
		if t.UserID != userID {
			continue
		}
		if t.Type == domain.TransactionBuy {
			bal = bal.Add(t.Gram)
		} else {
			bal = bal.Sub(t.Gram)
		}
	}
	return bal, nil
}

const testPrice int64 = 1_945_200
const testUser int64 = 1

func newService() *GoldService {
	return NewGoldService(newMockTxRepo(), NewStaticPriceProvider(testPrice))
}

// TestBuy_BRDExampleAmount menguji contoh nominal dari BRD (amount 500000).
//
// CATATAN PENTING: BRD mencantumkan hasil 0.2571 gram, TAPI secara aritmetika
// 500000 / 1945200 = 0.257043..., yang jika dibulatkan 4 desimal KE BAWAH = 0.2570.
// Nilai 0.2571 di BRD adalah pembulatan KE ATAS, yang berarti nasabah menerima
// emas senilai ~Rp500.111 padahal hanya membayar Rp500.000 — kebocoran nilai bagi
// platform. Kami sengaja memilih ROUND DOWN (0.2570) sebagai perilaku yang benar
// secara finansial, dan menandai inkonsistensi contoh di BRD.
func TestBuy_BRDExampleAmount(t *testing.T) {
	svc := newService()
	tx, err := svc.Buy(context.Background(), testUser, 500_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := decimal.RequireFromString("0.2570") // round-down dari 0.257043...
	if !tx.Gram.Equal(want) {
		t.Errorf("gram = %s, want %s", tx.Gram, want)
	}
	if tx.Price != testPrice {
		t.Errorf("price = %d, want %d", tx.Price, testPrice)
	}
}

// TestBuy_RoundsDown memastikan pembulatan selalu ke bawah (tidak pernah
// memberi emas lebih dari seharusnya).
func TestBuy_RoundsDown(t *testing.T) {
	svc := newService()
	// 1.000.000 / 1.945.200 = 0.514085... -> truncate 4 desimal = 0.5140
	tx, err := svc.Buy(context.Background(), testUser, 1_000_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := decimal.RequireFromString("0.5140")
	if !tx.Gram.Equal(want) {
		t.Errorf("gram = %s, want %s", tx.Gram, want)
	}
}

// TestBuy_BelowMinimum menolak nominal di bawah minimum.
func TestBuy_BelowMinimum(t *testing.T) {
	svc := newService()
	_, err := svc.Buy(context.Background(), testUser, 5_000)
	appErr, ok := domain.AsAppError(err)
	if !ok || appErr.Code != domain.CodeInvalidAmount {
		t.Fatalf("expected INVALID_AMOUNT, got %v", err)
	}
}

// TestSell_MatchesBRDExample memverifikasi contoh persis dari BRD:
// gram 1 -> amount 1945200. (Didahului buy agar saldo cukup.)
func TestSell_MatchesBRDExample(t *testing.T) {
	svc := newService()
	// beli dulu 2 gram (butuh nominal ~ 2 * harga) supaya saldo cukup
	if _, err := svc.Buy(context.Background(), testUser, 4_000_000); err != nil {
		t.Fatalf("setup buy failed: %v", err)
	}
	tx, err := svc.Sell(context.Background(), testUser, decimal.RequireFromString("1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Amount != 1_945_200 {
		t.Errorf("amount = %d, want 1945200", tx.Amount)
	}
}

// TestSell_InsufficientBalance menolak jual melebihi saldo.
func TestSell_InsufficientBalance(t *testing.T) {
	svc := newService()
	_, err := svc.Sell(context.Background(), testUser, decimal.RequireFromString("1"))
	appErr, ok := domain.AsAppError(err)
	if !ok || appErr.Code != domain.CodeInsufficientGold {
		t.Fatalf("expected INSUFFICIENT_BALANCE, got %v", err)
	}
}

// TestSell_InvalidGram menolak gram <= 0.
func TestSell_InvalidGram(t *testing.T) {
	svc := newService()
	_, err := svc.Sell(context.Background(), testUser, decimal.Zero)
	appErr, ok := domain.AsAppError(err)
	if !ok || appErr.Code != domain.CodeInvalidGram {
		t.Fatalf("expected INVALID_GRAM, got %v", err)
	}
}
