package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"richtalk/api/internal/model"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, username, passwordHash string) (*model.User, error) {
	const q = `
		INSERT INTO users (username, password_hash)
		VALUES ($1, $2)
		RETURNING id, username, password_hash, created_at, updated_at`

	var u model.User
	err := r.db.QueryRow(ctx, q, username, passwordHash).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, model.ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	const q = `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users WHERE username = $1`

	var u model.User
	err := r.db.QueryRow(ctx, q, username).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	const q = `
		SELECT id, username, password_hash, created_at, updated_at
		FROM users WHERE id = $1`

	var u model.User
	err := r.db.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) Search(ctx context.Context, query string, limit int) ([]*model.User, error) {
	const q = `
		SELECT id, username, created_at, updated_at
		FROM users
		WHERE username LIKE $1
		ORDER BY username
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
