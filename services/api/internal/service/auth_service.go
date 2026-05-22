package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"richtalk/api/internal/model"
	"richtalk/api/internal/repository"
)

type AuthService struct {
	users         *repository.UserRepo
	refreshTokens *repository.RefreshTokenRepo
	jwt           *JWTService
	refreshTTL    time.Duration
	log           *slog.Logger
}

func NewAuthService(
	users *repository.UserRepo,
	refreshTokens *repository.RefreshTokenRepo,
	jwt *JWTService,
	refreshTTL time.Duration,
	log *slog.Logger,
) *AuthService {
	return &AuthService{
		users:         users,
		refreshTokens: refreshTokens,
		jwt:           jwt,
		refreshTTL:    refreshTTL,
		log:           log,
	}
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	User         *model.User
}

func (s *AuthService) Register(ctx context.Context, username, password string) (*AuthResult, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.users.Create(ctx, username, string(hash))
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*AuthResult, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			// Don't distinguish "not found" from "wrong password" — timing-safe
			return nil, model.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, model.ErrInvalidCredentials
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*AuthResult, error) {
	record, err := s.refreshTokens.GetByRawToken(ctx, rawToken)
	if err != nil {
		if errors.Is(err, model.ErrTokenNotFound) {
			return nil, model.ErrInvalidCredentials
		}
		return nil, err
	}

	if time.Now().After(record.ExpiresAt) {
		_ = s.refreshTokens.Delete(ctx, rawToken)
		return nil, model.ErrTokenExpired
	}

	user, err := s.users.GetByID(ctx, record.UserID)
	if err != nil {
		return nil, err
	}

	// Rotate: invalidate old token before issuing new one
	if err := s.refreshTokens.Delete(ctx, rawToken); err != nil {
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	return s.issueTokens(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	// Ignore ErrTokenNotFound — idempotent logout is fine
	err := s.refreshTokens.Delete(ctx, rawToken)
	if errors.Is(err, model.ErrTokenNotFound) {
		return nil
	}
	return err
}

func (s *AuthService) issueTokens(ctx context.Context, user *model.User) (*AuthResult, error) {
	accessToken, err := s.jwt.Issue(user.ID)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	rawRefresh, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	if err := s.refreshTokens.Store(ctx, user.ID, rawRefresh, time.Now().Add(s.refreshTTL)); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		User:         user,
	}, nil
}

func generateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
