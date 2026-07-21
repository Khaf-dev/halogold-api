package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config menampung seluruh konfigurasi aplikasi yang dibaca dari environment.
// Prinsip 12-factor: konfigurasi via env, bukan hardcode.
type Config struct {
	Port          string
	DatabaseURL   string
	GoldPrice     int64 // harga emas dummy (rupiah/gram)
	DefaultUserID int64 // user default karena login tidak diwajibkan di BRD
}

// Load membaca konfigurasi dari environment dengan nilai default yang aman.
func Load() (*Config, error) {
	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://halogold:halogold@localhost:5432/halogold?sslmode=disable"),
		GoldPrice:     getEnvInt64("GOLD_PRICE", 1_945_200),
		DefaultUserID: getEnvInt64("DEFAULT_USER_ID", 1),
	}

	if cfg.GoldPrice <= 0 {
		return nil, fmt.Errorf("GOLD_PRICE harus > 0")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}
