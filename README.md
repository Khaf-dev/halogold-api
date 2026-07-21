# HaloGold API

REST API sederhana untuk transaksi emas digital (beli/jual), dibangun sesuai
BRD Proyek HaloGold. Fokus utama: **struktur bersih, validasi, error handling,
dan presisi finansial**.

- **Bahasa:** Go (build toolchain 1.25; lihat [Catatan Environment / Troubleshooting](#catatan-environment--troubleshooting))
- **Framework HTTP:** Gin
- **Database:** PostgreSQL (via pgx v5)
- **Presisi angka:** shopspring/decimal (bukan `float64`)

---

## Daftar Isi

1. [Cara Menjalankan](#cara-menjalankan)
2. [Endpoint](#endpoint)
3. [Arsitektur](#arsitektur)
4. [Keputusan Desain Penting](#keputusan-desain-penting)
5. [Testing](#testing)
6. [Catatan Environment / Troubleshooting](#catatan-environment--troubleshooting)
7. [Yang Belum Diimplementasikan](#yang-belum-diimplementasikan)

---

## Cara Menjalankan

### Opsi A — Docker (paling mudah)

```bash
docker compose up --build
```

Perintah ini menjalankan PostgreSQL + API sekaligus. API otomatis menjalankan
migrasi (buat tabel + seed user) saat startup. Setelah jalan, akses `http://localhost:8080`.

> **Port bentrok?** Jika `5432` di host sudah dipakai proses lain, ubah **angka kiri** pada mapping port service `db` di `docker-compose.yml` (mis. `5544:5432`). Angka kanan jangan diubah - koneksi antar-container tetap lewat `db:5432`, jadi aplikasi tidak terpengaruh.

### Opsi B — Lokal (butuh Go > 1.25 + PostgreSQL)

```bash
# 1. Ambil dependency (WAJIB dijalankan pertama kali)
go mod tidy

# 2. Siapkan PostgreSQL lalu set koneksi
cp .env.example .env   # sesuaikan bila perlu, atau export manual:
export DATABASE_URL="postgres://halogold:halogold@localhost:5544/halogold?sslmode=disable"

# 3. Jalankan (migrasi otomatis saat start)
go run ./cmd/api
```

> Migrasi di-embed ke dalam binary dan dijalankan otomatis secara idempotent,
> jadi tidak ada langkah SQL manual.

---

## Endpoint

Base URL: `http://localhost:8080`

| Method | Path            | Deskripsi                 |
| ------ | --------------- | ------------------------- |
| GET    | `/health`       | Health check              |
| GET    | `/price`        | Harga emas saat ini       |
| GET    | `/transactions` | Daftar transaksi          |
| POST   | `/buy`          | Beli emas (rupiah → gram) |
| POST   | `/sell`         | Jual emas (gram → rupiah) |
| GET    | `/balance`      | Saldo emas user   |

### Contoh (PowerShell / Windows)

```bash
# Harga emas
curl localhost:8080/price
# -> {"price":1945200}

# Beli emas Rp500.000
curl -X POST localhost:8080/buy -H 'Content-Type: application/json' -d '{"amount":500000}'
# -> {"gram":0.257,"price":1945200}

# Jual 0.1 gram
curl -X POST localhost:8080/sell -H 'Content-Type: application/json' -d '{"gram":0.1}'
# -> {"amount":194520}

# Daftar transaksi
curl localhost:8080/transactions

# Saldo emas
curl localhost:8080/balance
# -> {"gram":0.157}
```

#### Di PowerShell, gunakan `Invoke-RestMethod` (alias `irm`) dan **sertakan `http://`**:

```powershell
# Harga Emas
irm http://localhost:8080/price

# Beli Emas Rp500.000
irm -Method Post http://localhost:8080/buy -ContentType 'application/json' -Body '{"amount":500000}'

# Jual 0.1 gram
irm -Method Post http://localhost:8080/sell -ContentType 'application/json' -Body '{"gram":0.1}'

# Daftar transaksi
irm http://localhost:8080/transactions

# Saldo emas
irm http://localhost:8080/balance

# Error dan Validasi
try { irm -Method Post http://localhost:8080/sell -ContentType 'application/json' -Body '{"gram":999}' } catch { $_.ErrorDetails.Message }
```

### Format Error

Response sukses mengikuti format persis di BRD (tanpa envelope). Response error
memakai format konsisten:

```json
{
  "error": {
    "code": "INSUFFICIENT_BALANCE",
    "message": "Saldo emas tidak mencukupi. Saldo saat ini 0 gram"
  }
}
```

| Kode                   | HTTP | Kapan                                                    |
| ---------------------- | ---- | -------------------------------------------------------- |
| `VALIDATION_ERROR`     | 400  | Body tidak valid / field wajib kosong                    |
| `INVALID_AMOUNT`       | 400  | Nominal beli < minimum                                   |
| `INVALID_GRAM`         | 400  | Gram jual ≤ 0                                            |
| `INSUFFICIENT_BALANCE` | 422  | Jual melebihi saldo emas                                 |
| `INTERNAL_ERROR`       | 500  | Kesalahan tak terduga (detail tidak dibocorkan ke klien) |

---

## Arsitektur

Pendekatan **Clean Architecture pragmatis** dengan Repository Pattern — cukup berlapis untuk memisahkan tanggung jawab, tanpa over-engineering.

```
cmd/api/main.go            → composition root: wiring, migrasi, graceful shutdown
internal/
  domain/                  → entity + interface (port). TIDAK bergantung ke framework/DB
    transaction.go, user.go, errors.go, repository.go
  service/                 → logic bisnis emas (kalkulasi beli/jual). Bisa di-unit-test murni
    gold_service.go, price_service.go, gold_service_test.go
  repository/              → implementasi PostgreSQL dari interface domain
    postgres.go, transaction_repo.go, user_repo.go
  handler/                 → HTTP: parsing, validasi, mapping error → status
    gold_handler.go, router.go, response.go
  config/                  → loader konfigurasi dari environment
  migrations/              → skema SQL yang di-embed
```

**Aliran dependency:** `handler → service → repository`, dengan interface
didefinisikan di `domain`. Service hanya kenal abstraksi, sehingga:

- Logic bisnis bisa diuji tanpa DB atau HTTP (lihat `gold_service_test.go`).
- Driver DB / sumber harga bisa diganti tanpa menyentuh logic.

---

## Keputusan Desain Penting

### 1. Tidak pakai `float64` untuk uang & emas

Perhitungan finansial dengan `float64` menimbulkan galat pembulatan (mis. `0.1 + 0.2 != 0.3`). Semua gram memakai `decimal.Decimal`, disimpan di DB sebagai
`NUMERIC(20,8)`. Rupiah memakai `int64` (bilangan bulat, tanpa sen).

### 2. Pembulatan gram: ROUND DOWN — dan catatan inkonsistensi BRD

BRD memberi contoh: `amount 500000 → gram 0.2571`. Namun secara aritmetika:

```
500000 / 1945200 = 0.257043...
```

Dibulatkan 4 desimal **ke bawah** = `0.2570`. Nilai `0.2571` di BRD adalah pembulatan **ke atas**, yang berarti nasabah menerima emas senilai ~Rp500.111
padahal hanya membayar Rp500.000 — **kebocoran nilai** bagi platform.

Karena itu API ini sengaja memakai **ROUND DOWN** (favor the platform), yang merupakan praktik standar aplikasi finansial. Aturan ini terpusat di satu tempat
(`GoldService.Buy`) dan mudah diubah bila kebijakan bisnis berbeda. Sama halnya, `sell` membulatkan rupiah ke bawah.

### 3. Menyimpan harga per transaksi

Skema minimum di BRD tidak menyertakan kolom `price`, tapi kami menambahkannya.
Harga emas berubah-ubah; menyimpan harga saat transaksi terjadi, penting untuk **audit trail** dan rekonstruksi riwayat yang akurat.

### 4. User default (login tidak diwajibkan)

Endpoint `/buy` dan `/sell` di BRD tidak menyertakan `user_id`, dan login tidak diwajibkan. Kami seed satu user default (`id=1`) via migrasi dan memakainya sebagai konteks transaksi. Ini keputusan sadar, bukan kelalaian — struktur sudah siap menerima auth (tinggal ambil user dari token, bukan dari config).

### 5. Validasi saldo saat menjual

Meski tidak diminta eksplisit, `sell` memvalidasi bahwa gram yang dijual tidak melebihi saldo emas user (`SUM(buy) - SUM(sell)`), dihitung langsung di DB.

### 6. Keamanan & robustness

- Semua query memakai **parameter binding** (`$1..`) → aman dari SQL injection.
- Error internal tidak dibocorkan ke klien (hanya `INTERNAL_ERROR`).
- **Graceful shutdown** + timeout server.
- Connection pool dengan batas & lifetime yang wajar.

---

## Testing

**Unit test** fokus pada jantung aplikasi — logic kalkulasi emas — dan berjalan tanpa DB (memakai repository in-memory):

```bash
go test ./... -v
```

Cakupan: contoh nominal BRD, pembulatan ke bawah, nominal di bawah minimum, konversi jual, saldo tidak cukup, dan gram invalid.

**Integration test** memverifikasi lapisan database (pgx + service) terhadap PostgreSQL sungguhan. Dipisah dengan build tag agar tidak ikut `go test ./...`:

```bash
# pastikan PostgreSQL aktif (mis. `docker compose up -d db`)
TEST_DATABASE_URL="postgres://halogold:halogold@localhost:5544/halogold?sslmode=disable" \
    go test -tags=integration ./internal/repository/ -v
```

Cakupan: migrasi, insert/select, presisi `NUMERIC(20,8)`, perhitungan saldo, dan penolakan jual melebihi saldo.

---

## Catatan Environment / Troubleshooting

### Versi Go (toolchain 1.25)

`go.mod` menetapkan versi bahasa minimum, namun graf dependensi (khususnya sebuah *test-dependency* transitif) ter-resolve ke versi yang menuntut **toolcahin Go saat `go mod tidy` dijalankan pada Go 1.25 Karena itu:

- **Dockerfile** memakai `golang:1.25-alpine` sebagai builder agar build deterministik. Menjalankan lewat Docker **tidak butuh Go lokal** sama sekali.
- Untuk build/run **tanpa Docker**, gunakan **Go > 1.25**.

Ini pembedaan antara *versi bahasa* (`go` directive) dan *versi toolchain* yang diperlukan dependensi -- Docker sengaja dipakai agar reviewer tidak perlu menyamakan versi Go secara manual.

### Port PostgreSQL bentrok

Jika muncul `Bind for 0.0.0.0:5432 failed: port is already allocated`, ubah **angka kiri** mapping port service `db` di `docker-compose.yml` ke port bebas (mis. `"5544:5432"`), lalu `docker compose up` lagi. Koneksi aplikasi ke DB tidak terpengaruh (antar-container lewat `db:5432`).

### PowerShell (Windows)

- Gunakan `Invoke-RestMethod` (`irm`), bukan `curl`
- Selalu sertakan skema `http://` pada URL, jika tidak muncul `The URI prefix is not recognized`.

## Yang Belum Diimplementasikan

Sesuai BRD, item berikut **tidak wajib**.

- **JWT / Refresh Token / Role**

- **Swagger** — bisa ditambahkan cepat via anotasi `swaggo`; disiapkan sebagai
  langkah lanjutan.

Item yang **sudah** dikerjakan dari daftar opsional: Repository Pattern, Clean Architecture, Docker, dan Unit Test.
