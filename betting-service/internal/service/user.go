package service

import (
	"context"

	"github.com/uva337/betting-platform/betting-service/internal/repository"
)

// UserService содержит бизнес-логику для работы с пользователями
type UserService struct {
	repo *repository.UserRepository
}

// NewUserService конструктор сервиса
func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// GetBalance запрашивает баланс пользователя
func (s *UserService) GetBalance(ctx context.Context, userID int) (float64, error) {
	return s.repo.GetBalance(ctx, userID)
}
