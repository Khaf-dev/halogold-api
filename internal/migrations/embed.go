package migrations

import _ "embed"

// InitSchema berisi skema awal yang di-embed ke binary, sehingga migrasi
// bisa dijalankan otomatis saat startup tanpa file eksternal.
//
//go:embed 001_init.sql
var InitSchema string
