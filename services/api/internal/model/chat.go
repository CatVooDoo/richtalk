package model

import (
	"time"

	"github.com/google/uuid"
)

type ChatType string

const (
	ChatTypeDirect  ChatType = "direct"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
	ChatTypeNotes   ChatType = "notes"
)

type Chat struct {
	ID        uuid.UUID
	Type      ChatType
	Name      *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OtherUser struct {
	ID       uuid.UUID
	Username string
}

type LastMessage struct {
	ID        uuid.UUID
	Content   string // empty string when Deleted == true
	AuthorID  uuid.UUID
	CreatedAt time.Time
	Deleted   bool
}

type ChatWithLastMessage struct {
	Chat
	OtherUser   *OtherUser   // nil for group chats
	LastMessage *LastMessage // nil when chat has no messages
}
