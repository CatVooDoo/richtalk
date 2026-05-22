package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"richtalk/api/internal/httpx"
)

type HealthHandler struct {
	pg    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(pg *pgxpool.Pool, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{pg: pg, redis: rdb}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	pgOK := h.pg.Ping(ctx) == nil
	redisOK := h.redis.Ping(ctx).Err() == nil

	status := http.StatusOK
	if !pgOK || !redisOK {
		status = http.StatusServiceUnavailable
	}

	httpx.JSON(w, status, map[string]any{
		"status":   statusStr(pgOK && redisOK),
		"postgres": statusStr(pgOK),
		"redis":    statusStr(redisOK),
	})
}

func statusStr(ok bool) string {
	if ok {
		return "ok"
	}
	return "error"
}
