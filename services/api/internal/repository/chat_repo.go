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

type ChatRepo struct {
	db *pgxpool.Pool
}

func NewChatRepo(db *pgxpool.Pool) *ChatRepo {
	return &ChatRepo{db: db}
}

// canonicalPair returns (smaller, larger) UUID so direct_chat_lookup
// always stores the same ordering regardless of argument order.
func canonicalPair(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}

// CreateDirect is idempotent: creates a new direct chat or returns the existing one.
// Atomicity is guaranteed by the UNIQUE constraint on direct_chat_lookup(user1_id, user2_id):
// ON CONFLICT DO NOTHING returns no rows → we rollback the new chat and fetch the existing one.
func (r *ChatRepo) CreateDirect(ctx context.Context, userID1, userID2 uuid.UUID) (*model.Chat, error) {
	u1, u2 := canonicalPair(userID1, userID2)

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var chat model.Chat
	var chatIDStr, chatTypeStr string
	err = tx.QueryRow(ctx,
		`INSERT INTO chats (type) VALUES ('direct') RETURNING id, type, created_at, updated_at`,
	).Scan(&chatIDStr, &chatTypeStr, &chat.CreatedAt, &chat.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert chat: %w", err)
	}
	chat.ID = uuid.MustParse(chatIDStr)
	chat.Type = model.ChatType(chatTypeStr)

	if _, err = tx.Exec(ctx,
		`INSERT INTO chat_members (chat_id, user_id) VALUES ($1, $2), ($1, $3)`,
		chatIDStr, u1.String(), u2.String(),
	); err != nil {
		return nil, fmt.Errorf("insert members: %w", err)
	}

	var claimedIDStr string
	err = tx.QueryRow(ctx,
		`INSERT INTO direct_chat_lookup (chat_id, user1_id, user2_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user1_id, user2_id) DO NOTHING
		 RETURNING chat_id`,
		chatIDStr, u1.String(), u2.String(),
	).Scan(&claimedIDStr)

	if errors.Is(err, pgx.ErrNoRows) {
		// Another chat already exists for this pair — discard new and return existing.
		_ = tx.Rollback(ctx)
		return r.FindDirectChat(ctx, userID1, userID2)
	}
	if err != nil {
		return nil, fmt.Errorf("insert direct_chat_lookup: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &chat, nil
}

// FindDirectChat looks up an existing direct chat between two users.
func (r *ChatRepo) FindDirectChat(ctx context.Context, userID1, userID2 uuid.UUID) (*model.Chat, error) {
	u1, u2 := canonicalPair(userID1, userID2)
	const q = `
		SELECT c.id, c.type, c.created_at, c.updated_at
		FROM chats c
		JOIN direct_chat_lookup d ON d.chat_id = c.id
		WHERE d.user1_id = $1 AND d.user2_id = $2`

	var chat model.Chat
	var idStr, typeStr string
	err := r.db.QueryRow(ctx, q, u1.String(), u2.String()).
		Scan(&idStr, &typeStr, &chat.CreatedAt, &chat.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrChatNotFound
		}
		return nil, fmt.Errorf("find direct chat: %w", err)
	}
	chat.ID = uuid.MustParse(idStr)
	chat.Type = model.ChatType(typeStr)
	return &chat, nil
}

// GetByID returns the chat if requesterID is a member, otherwise ErrChatNotFound.
func (r *ChatRepo) GetByID(ctx context.Context, chatID, requesterID uuid.UUID) (*model.ChatWithLastMessage, error) {
	const q = `
		SELECT c.id, c.type, c.name, c.created_at, c.updated_at,
		       other_u.id, other_u.username
		FROM chats c
		JOIN chat_members me ON me.chat_id = c.id AND me.user_id = $2
		LEFT JOIN chat_members other_cm ON other_cm.chat_id = c.id AND other_cm.user_id != $2
		LEFT JOIN users other_u ON other_u.id = other_cm.user_id
		WHERE c.id = $1`

	var (
		idStr, typeStr string
		name           *string
		createdAt      time.Time
		updatedAt      time.Time
		otherUserID    *string
		otherUsername  *string
	)
	err := r.db.QueryRow(ctx, q, chatID.String(), requesterID.String()).
		Scan(&idStr, &typeStr, &name, &createdAt, &updatedAt, &otherUserID, &otherUsername)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrChatNotFound
		}
		return nil, fmt.Errorf("get chat by id: %w", err)
	}

	c := &model.ChatWithLastMessage{
		Chat: model.Chat{
			ID:        uuid.MustParse(idStr),
			Type:      model.ChatType(typeStr),
			Name:      name,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
	}
	if otherUserID != nil && otherUsername != nil {
		c.OtherUser = &model.OtherUser{
			ID:       uuid.MustParse(*otherUserID),
			Username: *otherUsername,
		}
	}
	return c, nil
}

// ListByUserID returns all chats for a user with last message, newest first.
func (r *ChatRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.ChatWithLastMessage, error) {
	const q = `
		SELECT
			c.id, c.type, c.name, c.created_at, c.updated_at,
			other_u.id        AS other_user_id,
			other_u.username  AS other_user_name,
			m.id              AS last_msg_id,
			m.content         AS last_msg_content,
			m.author_id       AS last_msg_author_id,
			m.created_at      AS last_msg_created_at,
			m.deleted_at      AS last_msg_deleted_at
		FROM chats c
		JOIN chat_members me ON me.chat_id = c.id AND me.user_id = $1
		LEFT JOIN chat_members other_cm ON other_cm.chat_id = c.id AND other_cm.user_id != $1
		LEFT JOIN users other_u ON other_u.id = other_cm.user_id
		LEFT JOIN LATERAL (
			SELECT id, content, author_id, created_at, deleted_at
			FROM messages
			WHERE chat_id = c.id
			ORDER BY created_at DESC
			LIMIT 1
		) m ON TRUE
		ORDER BY COALESCE(m.created_at, c.created_at) DESC`

	rows, err := r.db.Query(ctx, q, userID.String())
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer rows.Close()

	var chats []model.ChatWithLastMessage
	for rows.Next() {
		var (
			idStr, typeStr    string
			name              *string
			createdAt         time.Time
			updatedAt         time.Time
			otherUserID       *string
			otherUsername     *string
			lastMsgID         *string
			lastMsgContent    *string
			lastMsgAuthorID   *string
			lastMsgCreatedAt  *time.Time
			lastMsgDeletedAt  *time.Time
		)
		if err := rows.Scan(
			&idStr, &typeStr, &name, &createdAt, &updatedAt,
			&otherUserID, &otherUsername,
			&lastMsgID, &lastMsgContent, &lastMsgAuthorID, &lastMsgCreatedAt, &lastMsgDeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan chat row: %w", err)
		}

		c := model.ChatWithLastMessage{
			Chat: model.Chat{
				ID:        uuid.MustParse(idStr),
				Type:      model.ChatType(typeStr),
				Name:      name,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
		}
		if otherUserID != nil && otherUsername != nil {
			c.OtherUser = &model.OtherUser{
				ID:       uuid.MustParse(*otherUserID),
				Username: *otherUsername,
			}
		}
		if lastMsgID != nil && lastMsgContent != nil && lastMsgAuthorID != nil && lastMsgCreatedAt != nil {
			c.LastMessage = &model.LastMessage{
				ID:        uuid.MustParse(*lastMsgID),
				Content:   *lastMsgContent,
				AuthorID:  uuid.MustParse(*lastMsgAuthorID),
				CreatedAt: *lastMsgCreatedAt,
				Deleted:   lastMsgDeletedAt != nil,
			}
		}
		chats = append(chats, c)
	}
	return chats, rows.Err()
}

// GetMemberIDs returns all member UUIDs for a chat. Used by the WS hub for typing fan-out.
func (r *ChatRepo) GetMemberIDs(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error) {
	const q = `SELECT user_id FROM chat_members WHERE chat_id = $1`
	rows, err := r.db.Query(ctx, q, chatID.String())
	if err != nil {
		return nil, fmt.Errorf("get member ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var idStr string
		if err := rows.Scan(&idStr); err != nil {
			return nil, fmt.Errorf("scan member id: %w", err)
		}
		ids = append(ids, uuid.MustParse(idStr))
	}
	return ids, rows.Err()
}
