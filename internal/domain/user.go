package domain

import "time"

// User merepresentasikan pemilik akun emas.
// Sesuai BRD skema minimum: id, nama, email.
type User struct {
	ID        int64     `json:"id"`
	Nama      string    `json:"nama"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
