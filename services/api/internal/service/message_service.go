package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"richtalk/api/internal/event"
	"richtalk/api/internal/model"
	"richtalk/api/internal/repository"
)

const maxMessageContent = 4096

type MessageService struct {
	messages *repository.MessageRepo
	chats    *repository.ChatRepo
	rdb      *redis.Client
	log      *slog.Logger
}

func NewMessageService(
	messages *repository.MessageRepo,
	chats *repository.ChatRepo,
	rdb *redis.Client,
	log *slog.Logger,
) *MessageService {
	return &MessageService{
		messages: messages,
		chats:    chats,
		rdb:      rdb,
		log:      log,
	}
}

func (s *MessageService) SendMessage(ctx context.Context, chatID, senderID uuid.UUID, content string) (*model.Message, error) {
	if len([]rune(content)) > maxMessageContent {
		return nil, fmt.Errorf("content exceeds %d characters", maxMessageContent)
	}

	// Verify sender is a member by fetching chat (returns ErrChatNotFound if not member)
	if _, err := s.chats.GetByID(ctx, chatID, senderID); err != nil {
		return nil, err
	}

	msg, err := s.messages.Create(ctx, chatID, senderID, content)
	if err != nil {
		s.log.Error("create message", "chat_id", chatID, "error", err)
		return nil, err
	}

	s.fanout(ctx, event.MessageNew, msg, chatID)
	return msg, nil
}

func (s *MessageService) ListMessages(ctx context.Context, chatID, requesterID uuid.UUID, before *time.Time, limit int) ([]model.Message, error) {
	if _, err := s.chats.GetByID(ctx, chatID, requesterID); err != nil {
		return nil, err
	}

	msgs, err := s.messages.ListByChatID(ctx, chatID, before, limit)
	if err != nil {
		s.log.Error("list messages", "chat_id", chatID, "error", err)
		return nil, err
	}
	return msgs, nil
}

func (s *MessageService) EditMessage(ctx context.Context, messageID, requesterID uuid.UUID, content string) (*model.Message, error) {
	if len([]rune(content)) > maxMessageContent {
		return nil, fmt.Errorf("content exceeds %d characters", maxMessageContent)
	}

	msg, err := s.messages.Update(ctx, messageID, requesterID, content)
	if err != nil {
		s.log.Error("edit message", "message_id", messageID, "error", err)
		return nil, err
	}

	s.fanout(ctx, event.MessageEdited, msg, msg.ChatID)
	return msg, nil
}

func (s *MessageService) DeleteMessage(ctx context.Context, messageID, requesterID uuid.UUID) error {
	// Fetch first to get chatID for fan-out
	msg, err := s.messages.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	if err := s.messages.SoftDelete(ctx, messageID, requesterID); err != nil {
		s.log.Error("delete message", "message_id", messageID, "error", err)
		return err
	}

	type deletedPayload struct {
		ID        string `json:"id"`
		ChatID    string `json:"chat_id"`
		DeletedAt string `json:"deleted_at"`
	}
	payload := deletedPayload{
		ID:        messageID.String(),
		ChatID:    msg.ChatID.String(),
		DeletedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.fanoutPayload(ctx, event.MessageDeleted, payload, msg.ChatID)
	return nil
}

// fanout publishes an event with the message serialised in REST-compatible format.
func (s *MessageService) fanout(ctx context.Context, evType event.Type, msg *model.Message, chatID uuid.UUID) {
	payloadJSON, err := json.Marshal(msg.ToEventPayload())
	if err != nil {
		s.log.Error("marshal message payload", "error", err)
		return
	}
	s.publishEnvelope(ctx, evType, json.RawMessage(payloadJSON), chatID)
}

// fanoutPayload publishes an event with an arbitrary payload struct.
func (s *MessageService) fanoutPayload(ctx context.Context, evType event.Type, payload any, chatID uuid.UUID) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		s.log.Error("marshal event payload", "error", err)
		return
	}
	s.publishEnvelope(ctx, evType, json.RawMessage(payloadJSON), chatID)
}

func (s *MessageService) publishEnvelope(ctx context.Context, evType event.Type, payload json.RawMessage, chatID uuid.UUID) {
	memberIDs, err := s.chats.GetMemberIDs(ctx, chatID)
	if err != nil {
		s.log.Error("get member ids for fanout", "chat_id", chatID, "error", err)
		return
	}

	targetIDs := make([]string, len(memberIDs))
	for i, id := range memberIDs {
		targetIDs[i] = id.String()
	}

	env := event.Envelope{
		Type:          evType,
		Payload:       payload,
		TargetUserIDs: targetIDs,
	}
	data, err := json.Marshal(env)
	if err != nil {
		s.log.Error("marshal event envelope", "error", err)
		return
	}

	if err := s.rdb.Publish(ctx, event.RedisChannel, data).Err(); err != nil {
		s.log.Error("publish to redis", "channel", event.RedisChannel, "error", err)
	}
}
