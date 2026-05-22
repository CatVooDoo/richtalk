package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"richtalk/api/internal/httpx"
)

func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						"panic", rec,
						"path", r.URL.Path,
						"stack", string(debug.Stack()),
					)
					httpx.Error(w, http.StatusInternalServerError, "internal_error", "Внутренняя ошибка сервера")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
