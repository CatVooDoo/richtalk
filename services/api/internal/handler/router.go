package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	mw "richtalk/api/internal/middleware"
	"richtalk/api/internal/service"
)

type Deps struct {
	AuthSvc  *service.AuthService
	UserSvc  *service.UserService
	JWTSvc   *service.JWTService
	PGPool   *pgxpool.Pool
	Redis    *redis.Client
	Log      *slog.Logger
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(mw.Logger(d.Log))
	r.Use(mw.Recover(d.Log))

	authH := NewAuthHandler(d.AuthSvc)
	userH := NewUserHandler(d.UserSvc)
	healthH := NewHealthHandler(d.PGPool, d.Redis)

	r.Get("/api/health", healthH.Check)

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authH.Register)
		r.Post("/login", authH.Login)
		r.Post("/refresh", authH.Refresh)
		// Logout requires a valid JWT to prevent anonymous token invalidation
		r.With(mw.Auth(d.JWTSvc)).Post("/logout", authH.Logout)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(mw.Auth(d.JWTSvc))

		r.Get("/users/me", userH.Me)
		r.Get("/users/search", userH.Search)
	})

	return r
}
