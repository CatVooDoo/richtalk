package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"richtalk/api/internal/config"
	"richtalk/api/internal/db"
	"richtalk/api/internal/handler"
	"richtalk/api/internal/repository"
	"richtalk/api/internal/service"
	"richtalk/api/internal/ws"
)

type App struct {
	server *http.Server
	pgPool *pgxpool.Pool
	redis  *redis.Client
	hub    *ws.Hub
	log    *slog.Logger
}

func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	ctx := context.Background()

	log.Info("running migrations", "path", cfg.MigrationsPath)
	if err := db.RunMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}
	log.Info("migrations complete")

	pgPool, err := db.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	log.Info("connected to postgres")

	rdb, err := db.NewRedisClient(ctx, cfg.RedisAddr, cfg.RedisPassword)
	if err != nil {
		pgPool.Close()
		return nil, fmt.Errorf("redis: %w", err)
	}
	log.Info("connected to redis")

	userRepo := repository.NewUserRepo(pgPool)
	refreshRepo := repository.NewRefreshTokenRepo(pgPool)
	chatRepo := repository.NewChatRepo(pgPool)
	messageRepo := repository.NewMessageRepo(pgPool)

	jwtSvc := service.NewJWTService(cfg.JWTSecret, cfg.AccessTokenTTL)
	authSvc := service.NewAuthService(userRepo, refreshRepo, chatRepo, jwtSvc, cfg.RefreshTokenTTL, log)
	userSvc := service.NewUserService(userRepo, log)
	chatSvc := service.NewChatService(chatRepo, log)
	messageSvc := service.NewMessageService(messageRepo, chatRepo, rdb, log)

	hub := ws.NewHub(rdb, chatRepo, log)

	router := handler.NewRouter(handler.Deps{
		AuthSvc: authSvc,
		UserSvc: userSvc,
		ChatSvc: chatSvc,
		MsgSvc:  messageSvc,
		JWTSvc:  jwtSvc,
		Hub:     hub,
		PGPool:  pgPool,
		Redis:   rdb,
		Log:     log,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{
		server: srv,
		pgPool: pgPool,
		redis:  rdb,
		hub:    hub,
		log:    log,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go a.hub.Run(ctx)

	go func() {
		a.log.Info("http server started", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server error: %w", err)
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a.log.Info("shutting down http server")
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.log.Error("http server shutdown error", "error", err)
	}

	a.log.Info("closing postgres pool")
	a.pgPool.Close()

	a.log.Info("closing redis client")
	if err := a.redis.Close(); err != nil {
		a.log.Error("redis close error", "error", err)
	}

	a.log.Info("shutdown complete")
	return nil
}
