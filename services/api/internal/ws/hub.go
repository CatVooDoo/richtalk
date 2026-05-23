package ws

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"richtalk/api/internal/event"
)

// MemberFetcher returns the member IDs of a chat. Implemented by repository.ChatRepo.
type MemberFetcher interface {
	GetMemberIDs(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error)
}

// incomingMessage is a raw message received from a connected client.
type incomingMessage struct {
	client *Client
	data   []byte
}

// Hub manages all active WebSocket connections and fans out Redis events.
//
// Design: a single goroutine owns the clients map — no mutex needed.
// All mutations go through the register / unregister / incoming channels.
type Hub struct {
	// userID → slice of connections (one user can have multiple tabs)
	clients    map[string][]*Client
	register   chan *Client
	unregister chan *Client
	incoming   chan incomingMessage

	rdb     *redis.Client
	members MemberFetcher
	log     *slog.Logger
}

func NewHub(rdb *redis.Client, members MemberFetcher, log *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[string][]*Client),
		register:   make(chan *Client, 16),
		unregister: make(chan *Client, 16),
		incoming:   make(chan incomingMessage, 256),
		rdb:        rdb,
		members:    members,
		log:        log,
	}
}

// Run starts the hub event loop. Blocks until ctx is cancelled.
func (h *Hub) Run(ctx context.Context) {
	pubsub := h.rdb.Subscribe(ctx, event.RedisChannel)
	defer pubsub.Close()

	redisCh := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			// Close all client send channels so writePumps exit cleanly.
			for _, clients := range h.clients {
				for _, c := range clients {
					close(c.send)
				}
			}
			return

		case c := <-h.register:
			h.clients[c.userID] = append(h.clients[c.userID], c)
			h.log.Debug("ws client registered", "user_id", c.userID, "total", len(h.clients[c.userID]))

		case c := <-h.unregister:
			h.removeClient(c)

		case msg := <-redisCh:
			h.dispatch([]byte(msg.Payload))

		case im := <-h.incoming:
			// Handle in a goroutine to avoid blocking the hub on DB calls.
			go h.handleClientEvent(ctx, im.client, im.data)
		}
	}
}

// dispatch delivers a Redis envelope to the target clients.
func (h *Hub) dispatch(raw []byte) {
	var env event.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		h.log.Error("ws dispatch: unmarshal redis envelope", "error", err)
		return
	}

	// Re-serialize as the client wire format: {type, payload}
	wireMsg, err := json.Marshal(map[string]json.RawMessage{
		"type":    json.RawMessage(`"` + string(env.Type) + `"`),
		"payload": env.Payload,
	})
	if err != nil {
		h.log.Error("ws dispatch: marshal wire message", "error", err)
		return
	}

	for _, userID := range env.TargetUserIDs {
		for _, c := range h.clients[userID] {
			select {
			case c.send <- wireMsg:
			default:
				// Slow client: drop the message and close.
				h.log.Warn("ws dispatch: slow client, dropping", "user_id", userID)
				close(c.send)
				h.removeClient(c)
			}
		}
	}
}

// handleClientEvent processes a typing event sent by a connected client.
func (h *Hub) handleClientEvent(ctx context.Context, c *Client, data []byte) {
	var ev ClientEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		h.log.Debug("ws: bad client message", "user_id", c.userID, "error", err)
		return
	}

	switch ev.Type {
	case event.TypingStart, event.TypingStop:
		h.broadcastTyping(ctx, c, ev)
	default:
		h.log.Debug("ws: unknown client event type", "type", ev.Type)
	}
}

// broadcastTyping publishes a typing event to Redis so all instances can fan it out.
func (h *Hub) broadcastTyping(ctx context.Context, sender *Client, ev ClientEvent) {
	chatID, err := uuid.Parse(ev.Payload.ChatID)
	if err != nil {
		return
	}

	memberIDs, err := h.members.GetMemberIDs(ctx, chatID)
	if err != nil {
		h.log.Error("ws typing: get member ids", "chat_id", chatID, "error", err)
		return
	}

	// Exclude the sender from target_user_ids — they don't need their own typing event.
	targets := make([]string, 0, len(memberIDs)-1)
	for _, id := range memberIDs {
		if id.String() != sender.userID {
			targets = append(targets, id.String())
		}
	}
	if len(targets) == 0 {
		return
	}

	payload := TypingPayload{ChatID: ev.Payload.ChatID, UserID: sender.userID}
	payloadJSON, _ := json.Marshal(payload)

	env := event.Envelope{
		Type:          ev.Type,
		Payload:       json.RawMessage(payloadJSON),
		TargetUserIDs: targets,
	}
	envJSON, _ := json.Marshal(env)

	if err := h.rdb.Publish(ctx, event.RedisChannel, envJSON).Err(); err != nil {
		h.log.Error("ws typing: redis publish", "error", err)
	}
}

func (h *Hub) removeClient(c *Client) {
	list := h.clients[c.userID]
	for i, existing := range list {
		if existing == c {
			h.clients[c.userID] = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(h.clients[c.userID]) == 0 {
		delete(h.clients, c.userID)
	}
	h.log.Debug("ws client unregistered", "user_id", c.userID)
}
