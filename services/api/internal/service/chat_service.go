package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"richtalk/api/internal/model"
	"richtalk/api/internal/repository"
)

type ChatService struct {
	chats *repository.ChatRepo
	log   *slog.Logger
}

func NewChatService(chats *repository.ChatRepo, log *slog.Logger) *ChatService {
	return &ChatService{chats: chats, log: log}
}

func (s *ChatService) GetOrCreateDirect(ctx context.Context, requesterID, targetUserID uuid.UUID) (*model.ChatWithLastMessage, error) {
	chat, err := s.chats.CreateDirect(ctx, requesterID, targetUserID)
	if err != nil {
		return nil, err
	}
	// Fetch full details (other_user, last message) now that the chat exists.
	return s.chats.GetByID(ctx, chat.ID, requesterID)
}

func (s *ChatService) ListChats(ctx context.Context, userID uuid.UUID) ([]model.ChatWithLastMessage, error) {
	chats, err := s.chats.ListByUserID(ctx, userID)
	if err != nil {
		s.log.Error("list chats", "user_id", userID, "error", err)
		return nil, err
	}
	return chats, nil
}

func (s *ChatService) GetChat(ctx context.Context, chatID, requesterID uuid.UUID) (*model.ChatWithLastMessage, error) {
	chat, err := s.chats.GetByID(ctx, chatID, requesterID)
	if err != nil {
		s.log.Error("get chat", "chat_id", chatID, "requester_id", requesterID, "error", err)
		return nil, err
	}
	return chat, nil
}

func (s *ChatService) GetNotesChat(ctx context.Context, userID uuid.UUID) (*model.ChatWithLastMessage, error) {
	chat, err := s.chats.GetNotesChat(ctx, userID)
	if err != nil {
		s.log.Error("get notes chat", "user_id", userID, "error", err)
		return nil, err
	}
	return chat, nil
}
