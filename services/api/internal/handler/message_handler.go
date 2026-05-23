package handler

import (
	"errors"
	"net/http"
	"time"

	"richtalk/api/internal/httpx"
	"richtalk/api/internal/model"
	"richtalk/api/internal/service"
)

type MessageHandler struct {
	messages *service.MessageService
}

func NewMessageHandler(messages *service.MessageService) *MessageHandler {
	return &MessageHandler{messages: messages}
}

type sendMessageRequest struct {
	Content        string  `json:"content"         validate:"max=4096"`
	AttachmentURL  *string `json:"attachment_url"`
	AttachmentType *string `json:"attachment_type" validate:"omitempty,oneof=image audio"`
	AttachmentName *string `json:"attachment_name"`
	AttachmentSize *int64  `json:"attachment_size"`
}

type editMessageRequest struct {
	Content string `json:"content" validate:"required,min=1,max=4096"`
}

func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	chatID, ok := pathUUID(r, "chatID")
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "Некорректный ID чата")
		return
	}

	var req sendMessageRequest
	if validErrs, err := httpx.DecodeValidate(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "Некорректный JSON")
		return
	} else if validErrs != nil {
		httpx.ValidationErrors(w, validErrs)
		return
	}

	if req.Content == "" && req.AttachmentURL == nil {
		httpx.Error(w, http.StatusBadRequest, "empty_message", "Сообщение должно содержать текст или вложение")
		return
	}

	var att *service.MessageAttachment
	if req.AttachmentURL != nil && req.AttachmentType != nil {
		name := ""
		if req.AttachmentName != nil {
			name = *req.AttachmentName
		}
		var size int64
		if req.AttachmentSize != nil {
			size = *req.AttachmentSize
		}
		att = &service.MessageAttachment{
			Type: *req.AttachmentType,
			URL:  *req.AttachmentURL,
			Name: name,
			Size: size,
		}
	}

	msg, err := h.messages.SendMessage(r.Context(), chatID, userID, req.Content, att)
	if err != nil {
		mapMessageError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, mapMessage(msg))
}

func (h *MessageHandler) Edit(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	msgID, ok := pathUUID(r, "messageID")
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "Некорректный ID сообщения")
		return
	}

	var req editMessageRequest
	if validErrs, err := httpx.DecodeValidate(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "Некорректный JSON")
		return
	} else if validErrs != nil {
		httpx.ValidationErrors(w, validErrs)
		return
	}

	msg, err := h.messages.EditMessage(r.Context(), msgID, userID, req.Content)
	if err != nil {
		mapMessageError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, mapMessage(msg))
}

func (h *MessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := mustUserID(r)
	msgID, ok := pathUUID(r, "messageID")
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "invalid_id", "Некорректный ID сообщения")
		return
	}

	if err := h.messages.DeleteMessage(r.Context(), msgID, userID); err != nil {
		mapMessageError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- mapping helpers ---

type messageResponse struct {
	ID             string         `json:"id"`
	ChatID         string         `json:"chat_id"`
	Author         authorResponse `json:"author"`
	Content        string         `json:"content"`
	Deleted        bool           `json:"deleted"`
	AttachmentType *string        `json:"attachment_type,omitempty"`
	AttachmentURL  *string        `json:"attachment_url,omitempty"`
	AttachmentName *string        `json:"attachment_name,omitempty"`
	AttachmentSize *int64         `json:"attachment_size,omitempty"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

type authorResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func mapMessage(m *model.Message) messageResponse {
	content := m.Content
	if m.IsDeleted() {
		content = ""
	}
	return messageResponse{
		ID:     m.ID.String(),
		ChatID: m.ChatID.String(),
		Author: authorResponse{
			ID:       m.Author.ID.String(),
			Username: m.Author.Username,
		},
		Content:        content,
		Deleted:        m.IsDeleted(),
		AttachmentType: m.AttachmentType,
		AttachmentURL:  m.AttachmentURL,
		AttachmentName: m.AttachmentName,
		AttachmentSize: m.AttachmentSize,
		CreatedAt:      m.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      m.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func mapMessages(msgs []model.Message) []messageResponse {
	out := make([]messageResponse, len(msgs))
	for i := range msgs {
		out[i] = mapMessage(&msgs[i])
	}
	return out
}

func mapMessageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrChatNotFound):
		httpx.Error(w, http.StatusNotFound, "chat_not_found", "Чат не найден")
	case errors.Is(err, model.ErrMessageNotFound):
		httpx.Error(w, http.StatusNotFound, "message_not_found", "Сообщение не найдено")
	case errors.Is(err, model.ErrForbidden):
		httpx.Error(w, http.StatusForbidden, "forbidden", "Нет доступа")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
	}
}
