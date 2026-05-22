// Package event defines the shared real-time event types and Redis envelope format.
// Both the service layer (publisher) and the ws layer (subscriber) depend on this
// package to avoid an import cycle: service → ws → service.
package event

import "encoding/json"

type Type string

const (
	MessageNew     Type = "message.new"
	MessageEdited  Type = "message.edited"
	MessageDeleted Type = "message.deleted"
	TypingStart    Type = "typing.start"
	TypingStop     Type = "typing.stop"
)

const RedisChannel = "richtalk:events"

// Envelope is the Redis Pub/Sub wire format used by all API instances.
// TargetUserIDs lets the hub dispatch only to relevant connected clients.
type Envelope struct {
	Type          Type            `json:"type"`
	Payload       json.RawMessage `json:"payload"`
	TargetUserIDs []string        `json:"target_user_ids"`
}
