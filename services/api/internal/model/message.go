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
	ID        uuid.UUID
	ChatID    uuid.UUID
	Author    Author
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (m *Message) IsDeleted() bool {
	return m.DeletedAt != nil
}
