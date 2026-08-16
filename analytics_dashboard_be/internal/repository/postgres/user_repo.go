package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"analytics-dashboard-be/internal/domain"
)

// UserRepo is the Postgres/sqlx implementation of domain.UserRepository.
type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) ByUsername(ctx context.Context, username string) (domain.User, error) {
	var u domain.User
	err := r.db.GetContext(ctx, &u,
		`SELECT id, username, password_hash, role, created_at FROM users WHERE username = $1`, username)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, err
}

func (r *UserRepo) Upsert(ctx context.Context, username, passwordHash string, role domain.Role) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (username, password_hash, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (username) DO UPDATE
		SET password_hash = EXCLUDED.password_hash, role = EXCLUDED.role`,
		username, passwordHash, role)
	return err
}

func (r *UserRepo) List(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	// password_hash is intentionally omitted from the projection.
	err := r.db.SelectContext(ctx, &users,
		`SELECT id, username, role, created_at FROM users ORDER BY created_at`)
	return users, err
}
