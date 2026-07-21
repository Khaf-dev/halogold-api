package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shopspring/decimal"

	"halogold-api/internal/config"
	"halogold-api/internal/handler"
	"halogold-api/internal/migrations"
	"halogold-api/internal/repository"
	"halogold-api/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	// Secara default shopspring/decimal marshal JSON sebagai string berkutip.
	// BRD mengharapkan gram berupa ANGKA (mis. {"gram":0.257}), jadi kita ubah
	// agar decimal diserialisasi tanpa kutip.
	decimal.MarshalJSONWithoutQuotes = true

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// --- Infrastruktur: DB ---
	pool, err := repository.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := repository.Migrate(ctx, pool, migrations.InitSchema); err != nil {
		return err
	}
	log.Println("migration selesai")

	// --- Dependency wiring (composition root) ---
	txRepo := repository.NewTransactionRepo(pool)
	priceProvider := service.NewStaticPriceProvider(cfg.GoldPrice)
	goldSvc := service.NewGoldService(txRepo, priceProvider)
	goldHandler := handler.NewGoldHandler(goldSvc, cfg.DefaultUserID)

	router := handler.NewRouter(goldHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// --- Jalankan server di goroutine terpisah ---
	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	// --- Graceful shutdown: tunggu sinyal, beri waktu request selesai ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	log.Println("server stopped")
	return nil
}
