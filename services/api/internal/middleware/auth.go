package middleware

import (
	"context"
	"net/http"
	"strings"

	"richtalk/api/internal/httpx"
	"richtalk/api/internal/service"
)

type contextKey string

const userIDKey contextKey = "userID"

func Auth(jwtSvc *service.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Требуется авторизация")
				return
			}

			claims, err := jwtSvc.Validate(strings.TrimPrefix(authHeader, "Bearer "))
			if err != nil {
				httpx.Error(w, http.StatusUnauthorized, "invalid_token", "Недействительный или истёкший токен")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID retrieves the authenticated user's ID from the request context.
func GetUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok && id != ""
}
