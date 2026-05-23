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
	"richtalk/api/internal/ws"
)

type Deps struct {
	AuthSvc  *service.AuthService
	UserSvc  *service.UserService
	ChatSvc  *service.ChatService
	MsgSvc   *service.MessageService
	JWTSvc   *service.JWTService
	Hub      *ws.Hub
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
	chatH := NewChatHandler(d.ChatSvc, d.MsgSvc)
	msgH := NewMessageHandler(d.MsgSvc)
	healthH := NewHealthHandler(d.PGPool, d.Redis)

	r.Get("/api/health", healthH.Check)

	// WebSocket — JWT validated inside the handler, not via chi middleware
	r.Get("/ws", ws.ServeWS(d.Hub, d.JWTSvc, d.Log))

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authH.Register)
		r.Post("/login", authH.Login)
		r.Post("/refresh", authH.Refresh)
		r.With(mw.Auth(d.JWTSvc)).Post("/logout", authH.Logout)
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(mw.Auth(d.JWTSvc))

		r.Get("/users/me", userH.Me)
		r.Get("/users/search", userH.Search)

		// Chats
		r.Get("/chats", chatH.List)
		r.Post("/chats/direct", chatH.CreateDirect)
		r.Get("/chats/{chatID}", chatH.Get)
		r.Get("/chats/{chatID}/messages", chatH.ListMessages)

		// Messages (edit / delete by ID)
		r.Post("/chats/{chatID}/messages", msgH.Send)
		r.Patch("/messages/{messageID}", msgH.Edit)
		r.Delete("/messages/{messageID}", msgH.Delete)
	})

	return r
}
