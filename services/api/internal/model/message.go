package model

import (
	"time"

	"github.com/google/uuid"
)

type Author struct {
	ID       uuid.UUID
	Username string
}

type Message struct {
	ID             uuid.UUID
	ChatID         uuid.UUID
	Author         Author
	Content        string
	AttachmentType *string
	AttachmentURL  *string
	AttachmentName *string
	AttachmentSize *int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

func (m *Message) IsDeleted() bool {
	return m.DeletedAt != nil
}

// MessageEventPayload is the JSON shape sent over WebSocket — matches the REST response format.
type MessageEventPayload struct {
	ID             string              `json:"id"`
	ChatID         string              `json:"chat_id"`
	Author         AuthorEventPayload  `json:"author"`
	Content        string              `json:"content"`
	Deleted        bool                `json:"deleted"`
	AttachmentType *string             `json:"attachment_type,omitempty"`
	AttachmentURL  *string             `json:"attachment_url,omitempty"`
	AttachmentName *string             `json:"attachment_name,omitempty"`
	AttachmentSize *int64              `json:"attachment_size,omitempty"`
	CreatedAt      string              `json:"created_at"`
	UpdatedAt      string              `json:"updated_at"`
}

type AuthorEventPayload struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (m *Message) ToEventPayload() MessageEventPayload {
	content := m.Content
	deleted := m.IsDeleted()
	if deleted {
		content = ""
	}
	return MessageEventPayload{
		ID:     m.ID.String(),
		ChatID: m.ChatID.String(),
		Author: AuthorEventPayload{
			ID:       m.Author.ID.String(),
			Username: m.Author.Username,
		},
		Content:        content,
		Deleted:        deleted,
		AttachmentType: m.AttachmentType,
		AttachmentURL:  m.AttachmentURL,
		AttachmentName: m.AttachmentName,
		AttachmentSize: m.AttachmentSize,
		CreatedAt:      m.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      m.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
