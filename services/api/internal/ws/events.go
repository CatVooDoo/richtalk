package ws

import (
	"richtalk/api/internal/event"
)

// Re-export event types so callers only need to import ws.
type EventType = event.Type

const (
	EventMessageNew     = event.MessageNew
	EventMessageEdited  = event.MessageEdited
	EventMessageDeleted = event.MessageDeleted
	EventTypingStart    = event.TypingStart
	EventTypingStop     = event.TypingStop
)

// ClientEvent is a message received FROM a connected WebSocket client.
type ClientEvent struct {
	Type    event.Type `json:"type"`
	Payload struct {
		ChatID string `json:"chat_id"`
	} `json:"payload"`
}

// TypingPayload is broadcast to other chat members when someone is typing.
type TypingPayload struct {
	ChatID string `json:"chat_id"`
	UserID string `json:"user_id"`
}
