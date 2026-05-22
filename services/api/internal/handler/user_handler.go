package handler

import (
	"errors"
	"net/http"

	"richtalk/api/internal/httpx"
	"richtalk/api/internal/middleware"
	"richtalk/api/internal/model"
	"richtalk/api/internal/service"
)

type UserHandler struct {
	users *service.UserService
}

func NewUserHandler(users *service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Требуется авторизация")
		return
	}

	user, err := h.users.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			httpx.Error(w, http.StatusNotFound, "user_not_found", "Пользователь не найден")
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	httpx.JSON(w, http.StatusOK, toUserResponse(user))
}

func (h *UserHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		httpx.Error(w, http.StatusBadRequest, "query_too_short", "Запрос должен содержать минимум 2 символа")
		return
	}

	users, err := h.users.Search(r.Context(), q)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
		return
	}

	type searchResult struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}

	results := make([]searchResult, 0, len(users))
	for _, u := range users {
		results = append(results, searchResult{ID: u.ID, Username: u.Username})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"users": results})
}
