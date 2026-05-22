package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"richtalk/api/internal/model"
)

type MessageRepo struct {
	db *pgxpool.Pool
}

func NewMessageRepo(db *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{db: db}
}

// Create inserts a message and returns it with author info in one CTE query.
func (r *MessageRepo) Create(ctx context.Context, chatID, senderID uuid.UUID, content string) (*model.Message, error) {
	const q = `
		WITH ins AS (
			INSERT INTO messages (chat_id, author_id, content)
			VALUES ($1, $2, $3)
			RETURNING id, chat_id, author_id, content, created_at, updated_at, deleted_at
		)
		SELECT i.id, i.chat_id, i.content, i.created_at, i.updated_at, i.deleted_at,
		       u.id AS author_id, u.username
		FROM ins i
		JOIN users u ON u.id = i.author_id`

	return r.scanMessage(ctx, r.db.QueryRow(ctx, q, chatID.String(), senderID.String(), content))
}

// ListByChatID returns messages in a chat older than `before`, newest first.
// If before is nil, starts from now.
func (r *MessageRepo) ListByChatID(ctx context.Context, chatID uuid.UUID, before *time.Time, limit int) ([]model.Message, error) {
	cursor := time.Now().Add(time.Second) // just after now
	if before != nil {
		cursor = *before
	}

	const q = `
		SELECT m.id, m.chat_id, m.content, m.created_at, m.updated_at, m.deleted_at,
		       u.id AS author_id, u.username
		FROM messages m
		JOIN users u ON u.id = m.author_id
		WHERE m.chat_id = $1 AND m.created_at < $2
		ORDER BY m.created_at DESC
		LIMIT $3`

	rows, err := r.db.Query(ctx, q, chatID.String(), cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var msgs []model.Message
	for rows.Next() {
		msg, err := r.scanMessageRow(rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

// Update edits message content. Returns ErrForbidden if senderID is not the author,
// ErrMessageNotFound if the message does not exist or is already deleted.
func (r *MessageRepo) Update(ctx context.Context, messageID, senderID uuid.UUID, content string) (*model.Message, error) {
	const q = `
		WITH upd AS (
			UPDATE messages
			SET content = $1, updated_at = now()
			WHERE id = $2 AND author_id = $3 AND deleted_at IS NULL
			RETURNING id, chat_id, author_id, content, created_at, updated_at, deleted_at
		)
		SELECT u.id, u.chat_id, u.content, u.created_at, u.updated_at, u.deleted_at,
		       usr.id AS author_id, usr.username
		FROM upd u
		JOIN users usr ON usr.id = u.author_id`

	msg, err := r.scanMessage(ctx, r.db.QueryRow(ctx, q, content, messageID.String(), senderID.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, r.checkOwnership(ctx, messageID, senderID)
	}
	return msg, err
}

// SoftDelete marks a message as deleted. Only the author may delete.
func (r *MessageRepo) SoftDelete(ctx context.Context, messageID, senderID uuid.UUID) error {
	const q = `
		UPDATE messages SET deleted_at = now()
		WHERE id = $1 AND author_id = $2 AND deleted_at IS NULL`

	tag, err := r.db.Exec(ctx, q, messageID.String(), senderID.String())
	if err != nil {
		return fmt.Errorf("soft delete message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return r.checkOwnership(ctx, messageID, senderID)
	}
	return nil
}

// GetByID returns a message with its author info.
func (r *MessageRepo) GetByID(ctx context.Context, messageID uuid.UUID) (*model.Message, error) {
	const q = `
		SELECT m.id, m.chat_id, m.content, m.created_at, m.updated_at, m.deleted_at,
		       u.id AS author_id, u.username
		FROM messages m
		JOIN users u ON u.id = m.author_id
		WHERE m.id = $1`

	msg, err := r.scanMessage(ctx, r.db.QueryRow(ctx, q, messageID.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrMessageNotFound
	}
	return msg, err
}

// checkOwnership distinguishes "not found" from "wrong author" after a failed update/delete.
func (r *MessageRepo) checkOwnership(ctx context.Context, messageID, senderID uuid.UUID) error {
	const q = `SELECT author_id FROM messages WHERE id = $1 AND deleted_at IS NULL`
	var authorIDStr string
	err := r.db.QueryRow(ctx, q, messageID.String()).Scan(&authorIDStr)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrMessageNotFound
	}
	if err != nil {
		return fmt.Errorf("check ownership: %w", err)
	}
	if authorIDStr != senderID.String() {
		return model.ErrForbidden
	}
	return nil
}

// scanMessage scans a single message row from QueryRow.
func (r *MessageRepo) scanMessage(_ context.Context, row pgx.Row) (*model.Message, error) {
	var (
		idStr, chatIDStr        string
		content                 string
		createdAt, updatedAt    time.Time
		deletedAt               *time.Time
		authorIDStr, authorName string
	)
	if err := row.Scan(&idStr, &chatIDStr, &content, &createdAt, &updatedAt, &deletedAt, &authorIDStr, &authorName); err != nil {
		return nil, err
	}
	return &model.Message{
		ID:     uuid.MustParse(idStr),
		ChatID: uuid.MustParse(chatIDStr),
		Author: model.Author{
			ID:       uuid.MustParse(authorIDStr),
			Username: authorName,
		},
		Content:   content,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
	}, nil
}

// scanMessageRow scans from an open Rows cursor.
func (r *MessageRepo) scanMessageRow(rows pgx.Rows) (model.Message, error) {
	var (
		idStr, chatIDStr        string
		content                 string
		createdAt, updatedAt    time.Time
		deletedAt               *time.Time
		authorIDStr, authorName string
	)
	if err := rows.Scan(&idStr, &chatIDStr, &content, &createdAt, &updatedAt, &deletedAt, &authorIDStr, &authorName); err != nil {
		return model.Message{}, fmt.Errorf("scan message: %w", err)
	}
	return model.Message{
		ID:     uuid.MustParse(idStr),
		ChatID: uuid.MustParse(chatIDStr),
		Author: model.Author{
			ID:       uuid.MustParse(authorIDStr),
			Username: authorName,
		},
		Content:   content,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
	}, nil
}
