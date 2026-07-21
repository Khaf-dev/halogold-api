package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"halogold-api/internal/domain"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// Pastikan UserRepo memenuhi kontrak domain.UserRepository (compile-time check).
var _ domain.UserRepository = (*UserRepo)(nil)

func (r *UserRepo) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	const q = `SELECT id, nama, email, created_at FROM users WHERE id = $1`

	var u domain.User
	err := r.pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.Nama, &u.Email, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewAppError(domain.CodeUserNotFound, "User tidak ditemukan", nil)
		}
		return nil, fmt.Errorf("find user: %w", err)
	}
	return &u, nil
}
