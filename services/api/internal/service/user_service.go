package service

import (
	"context"
	"log/slog"

	"richtalk/api/internal/model"
	"richtalk/api/internal/repository"
)

type UserService struct {
	users *repository.UserRepo
	log   *slog.Logger
}

func NewUserService(users *repository.UserRepo, log *slog.Logger) *UserService {
	return &UserService{users: users, log: log}
}

func (s *UserService) GetByID(ctx context.Context, id string) (*model.User, error) {
	return s.users.GetByID(ctx, id)
}

func (s *UserService) Search(ctx context.Context, query string) ([]*model.User, error) {
	if len(query) < 2 {
		return nil, nil
	}
	return s.users.Search(ctx, query, 20)
}
