// Package repository handles data persistence.
//
// Refresh token storage: PostgreSQL over Redis.
//
// Refresh tokens are used infrequently (once per access-token expiry, ~every 15 min),
// so the latency advantage of Redis is negligible. Using Postgres means:
//   - No additional TTL management — expires_at column + periodic cleanup
//   - Atomicity: token rotation and issuance are in the same DB that holds users
//   - No Redis data structure to reason about for auth state
//   - One less moving part in the auth critical path
//
// We store SHA-256(token) rather than bcrypt because:
//   - The raw token is 32 cryptographically random bytes — brute-force is infeasible
//     without key-stretching; SHA-256 is sufficient for a random-value lookup key
//   - bcrypt would add ~100 ms per refresh request with no security benefit
//   - SHA-256 is deterministic and fast for exact-match DB lookups

package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"richtalk/api/internal/model"
)

type RefreshTokenRepo struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepo(db *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

func (r *RefreshTokenRepo) Store(ctx context.Context, userID, rawToken string, expiresAt time.Time) error {
	const q = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	if _, err := r.db.Exec(ctx, q, userID, hashToken(rawToken), expiresAt); err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepo) GetByRawToken(ctx context.Context, rawToken string) (*model.RefreshToken, error) {
	const q = `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	var t model.RefreshToken
	err := r.db.QueryRow(ctx, q, hashToken(rawToken)).Scan(
		&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTokenNotFound
		}
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	return &t, nil
}

func (r *RefreshTokenRepo) Delete(ctx context.Context, rawToken string) error {
	const q = `DELETE FROM refresh_tokens WHERE token_hash = $1`
	if _, err := r.db.Exec(ctx, q, hashToken(rawToken)); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}

// DeleteExpired removes all expired tokens. Call periodically or at startup.
func (r *RefreshTokenRepo) DeleteExpired(ctx context.Context) error {
	const q = `DELETE FROM refresh_tokens WHERE expires_at < now()`
	_, err := r.db.Exec(ctx, q)
	return err
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
