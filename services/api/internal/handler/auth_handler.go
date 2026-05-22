package handler

import (
	"errors"
	"net/http"
	"time"

	"richtalk/api/internal/httpx"
	"richtalk/api/internal/model"
	"richtalk/api/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Username string `json:"username" validate:"required,min=3,max=32,username"`
	Password string `json:"password" validate:"required,min=8,max=128"`
}

type loginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type userResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

type authResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         userResponse `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if validErrs, err := httpx.DecodeValidate(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "Некорректный JSON")
		return
	} else if validErrs != nil {
		httpx.ValidationErrors(w, validErrs)
		return
	}

	result, err := h.auth.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		mapAuthError(w, err)
		return
	}

	httpx.JSON(w, http.StatusCreated, toAuthResponse(result))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if validErrs, err := httpx.DecodeValidate(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "Некорректный JSON")
		return
	} else if validErrs != nil {
		httpx.ValidationErrors(w, validErrs)
		return
	}

	result, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		mapAuthError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, toAuthResponse(result))
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if validErrs, err := httpx.DecodeValidate(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "Некорректный JSON")
		return
	} else if validErrs != nil {
		httpx.ValidationErrors(w, validErrs)
		return
	}

	result, err := h.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		mapAuthError(w, err)
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]string{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if validErrs, err := httpx.DecodeValidate(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "Некорректный JSON")
		return
	} else if validErrs != nil {
		httpx.ValidationErrors(w, validErrs)
		return
	}

	if err := h.auth.Logout(r.Context(), req.RefreshToken); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Ошибка при выходе")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func mapAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrUserAlreadyExists):
		httpx.Error(w, http.StatusConflict, "username_taken", "Это имя пользователя уже занято")
	case errors.Is(err, model.ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "Неверный логин или пароль")
	case errors.Is(err, model.ErrTokenExpired):
		httpx.Error(w, http.StatusUnauthorized, "token_expired", "Токен истёк, войдите снова")
	case errors.Is(err, model.ErrTokenNotFound):
		httpx.Error(w, http.StatusUnauthorized, "invalid_token", "Недействительный токен")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
	}
}

func toAuthResponse(result *service.AuthResult) authResponse {
	return authResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         toUserResponse(result.User),
	}
}

func toUserResponse(u *model.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Username:  u.Username,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}
