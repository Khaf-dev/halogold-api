//go:build integration

// Integration test lapisan database: mengeksekusi alur beli/jual/saldo terhadap
// PostgreSQL SUNGGUHAN (bukan mock). Dipisah dengan build tag `integration`
// supaya tidak ikut jalan pada `go test ./...` biasa (yang tidak butuh DB).
//
// Cara menjalankan (butuh PostgreSQL aktif):
//
//	# via docker compose (jalankan `docker compose up -d db` dulu), atau DB lokal
//	TEST_DATABASE_URL="postgres://halogold:halogold@localhost:5432/halogold?sslmode=disable" \
//	    go test -tags=integration ./internal/repository/ -v
package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/shopspring/decimal"

	"halogold-api/internal/domain"
	"halogold-api/internal/migrations"
	"halogold-api/internal/repository"
	"halogold-api/internal/service"
)

const defaultDSN = "postgres://halogold:halogold@localhost:5432/halogold?sslmode=disable"

func dsn() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return defaultDSN
}

const (
	testUserID = 1
	testPrice  = int64(1_945_200)
)

// TestEndToEnd_RealPostgres memverifikasi bahwa repository (pgx) + service bekerja
// benar terhadap PostgreSQL nyata: migrasi, insert, select, precision NUMERIC,
// perhitungan saldo, dan validasi saldo kurang.
func TestEndToEnd_RealPostgres(t *testing.T) {
	ctx := context.Background()

	pool, err := repository.NewPool(ctx, dsn())
	if err != nil {
		t.Fatalf("connect DB (set TEST_DATABASE_URL?): %v", err)
	}
	defer pool.Close()

	if err := repository.Migrate(ctx, pool, migrations.InitSchema); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM transactions"); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	txRepo := repository.NewTransactionRepo(pool)
	svc := service.NewGoldService(txRepo, service.NewStaticPriceProvider(testPrice))

	// BUY 500.000 -> 0.2570 (round down)
	buy, err := svc.Buy(ctx, testUserID, 500_000)
	if err != nil {
		t.Fatalf("buy: %v", err)
	}
	if !buy.Gram.Equal(decimal.RequireFromString("0.2570")) {
		t.Errorf("buy gram = %s, want 0.2570", buy.Gram)
	}

	// BUY lagi agar saldo cukup
	if _, err := svc.Buy(ctx, testUserID, 4_000_000); err != nil {
		t.Fatalf("buy2: %v", err)
	}

	// SELL 1 gram -> 1.945.200
	sell, err := svc.Sell(ctx, testUserID, decimal.RequireFromString("1"))
	if err != nil {
		t.Fatalf("sell: %v", err)
	}
	if sell.Amount != 1_945_200 {
		t.Errorf("sell amount = %d, want 1945200", sell.Amount)
	}

	// SELL melebihi saldo -> ditolak
	if _, err := svc.Sell(ctx, testUserID, decimal.RequireFromString("999")); err != nil {
		if appErr, ok := domain.AsAppError(err); !ok || appErr.Code != domain.CodeInsufficientGold {
			t.Errorf("expected INSUFFICIENT_BALANCE, got %v", err)
		}
	} else {
		t.Errorf("expected error selling beyond balance, got nil")
	}

	// LIST -> 3 transaksi
	list, err := svc.ListTransactions(ctx, testUserID, 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("list len = %d, want 3", len(list))
	}
}
