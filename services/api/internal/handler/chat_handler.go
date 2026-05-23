package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"richtalk/api/internal/httpx"
	"richtalk/api/internal/middleware"
	"richtalk/api/internal/model"
	"richtalk/api/internal/service"
)

type ChatHandler struct {
	chats    *service.ChatService
	messages *service.MessageService
}

func NewChatHandler(chats *service.ChatService, messages *service.MessageService) *ChatHandler {
	return &ChatHandler{chats: chats, messages: messages}
}

// --- request / response types ---

type createDirectRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

// --- handlers ---

func (h *ChatHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	chats, err := h.chats.ListChats(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Ошибка получения чатов")
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"chats": mapChats(chats)})
}

func (h *ChatHandler) CreateDirect(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)

	var req createDirectRequest
	if validErrs, err := httpx.DecodeValidate(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "Некорректный JSON")
		return
	} else if validErrs != nil {
		httpx.ValidationErrors(w, validErrs)
		return
	}

	targetID, err := uuid.Parse(req.UserID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_user_id", "Некорректный UUID пользователя")
		return
	}
	if targetID == userID {
		httpx.Error(w, http.StatusBadRequest, "self_chat", "Нельзя создать чат с самим собой")
		return
	}

	chat, err := h.chats.GetOrCreateDirect(r.Context(), userID, targetID)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			httpx.Error(w, http.StatusNotFound, "user_not_found", "Пользователь не найден")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Ошибка создания чата")
		return
	}

	httpx.JSON(w, http.StatusOK, mapChat(chat))
}

func (h *ChatHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	chatID, ok := pathUUID(r, "chatID")
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "Некорректный ID чата")
		return
	}

	chat, err := h.chats.GetChat(r.Context(), chatID, userID)
	if err != nil {
		mapChatError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, mapChat(chat))
}

func (h *ChatHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	chatID, ok := pathUUID(r, "chatID")
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "Некорректный ID чата")
		return
	}

	var before *time.Time
	if raw := r.URL.Query().Get("before"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid_before", "Параметр before должен быть в формате RFC3339")
			return
		}
		before = &t
	}

	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			httpx.Error(w, http.StatusBadRequest, "invalid_limit", "limit должен быть от 1 до 100")
			return
		}
		limit = n
	}

	msgs, err := h.messages.ListMessages(r.Context(), chatID, userID, before, limit)
	if err != nil {
		mapChatError(w, err)
		return
	}

	var hasMore bool
	var nextCursor *string
	if len(msgs) == limit {
		hasMore = true
		cursor := msgs[len(msgs)-1].CreatedAt.UTC().Format(time.RFC3339Nano)
		nextCursor = &cursor
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"messages":    mapMessages(msgs),
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	})
}

// --- mapping helpers ---

type chatResponse struct {
	ID        string             `json:"id"`
	Type      string             `json:"type"`
	Name      *string            `json:"name,omitempty"`
	OtherUser *otherUserResponse `json:"other_user,omitempty"`
	LastMsg   *lastMsgResponse   `json:"last_message,omitempty"`
	CreatedAt string             `json:"created_at"`
}

type otherUserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type lastMsgResponse struct {
	ID        string `json:"id"`
	Content   string `json:"content"` // empty when deleted
	AuthorID  string `json:"author_id"`
	CreatedAt string `json:"created_at"`
	Deleted   bool   `json:"deleted"`
}

func mapChat(c *model.ChatWithLastMessage) chatResponse {
	resp := chatResponse{
		ID:        c.ID.String(),
		Type:      string(c.Type),
		Name:      c.Name,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339),
	}
	if c.OtherUser != nil {
		resp.OtherUser = &otherUserResponse{
			ID:       c.OtherUser.ID.String(),
			Username: c.OtherUser.Username,
		}
	}
	if c.LastMessage != nil {
		content := c.LastMessage.Content
		if c.LastMessage.Deleted {
			content = ""
		}
		resp.LastMsg = &lastMsgResponse{
			ID:        c.LastMessage.ID.String(),
			Content:   content,
			AuthorID:  c.LastMessage.AuthorID.String(),
			CreatedAt: c.LastMessage.CreatedAt.UTC().Format(time.RFC3339),
			Deleted:   c.LastMessage.Deleted,
		}
	}
	return resp
}

func mapChats(chats []model.ChatWithLastMessage) []chatResponse {
	out := make([]chatResponse, len(chats))
	for i, c := range chats {
		c := c
		out[i] = mapChat(&c)
	}
	return out
}

func mapChatError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrChatNotFound):
		httpx.Error(w, http.StatusNotFound, "chat_not_found", "Чат не найден")
	case errors.Is(err, model.ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "Нет доступа к чату")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
	}
}

// pathUUID parses a chi URL param as UUID.
func pathUUID(r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	return id, err == nil
}

// mustUserID extracts the authenticated user ID from context (set by Auth middleware).
func mustUserID(r *http.Request) uuid.UUID {
	raw, _ := middleware.GetUserID(r.Context())
	id, _ := uuid.Parse(raw)
	return id
}
