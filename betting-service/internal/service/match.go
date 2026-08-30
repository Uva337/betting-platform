package service

import (
	"context"
	"fmt"

	"github.com/uva337/betting-platform/betting-service/internal/repository"
)

type MatchService struct {
	matchRepo *repository.MatchRepository
	betRepo   *repository.BetRepository
}

// NewMatchService принимает оба репозитория
func NewMatchService(matchRepo *repository.MatchRepository, betRepo *repository.BetRepository) *MatchService {
	return &MatchService{
		matchRepo: matchRepo,
		betRepo:   betRepo,
	}
}

// FinishMatch завершает матч и сразу запускает перерасчет всех ставок
func (s *MatchService) FinishMatch(ctx context.Context, matchID int, winner string) error {
	// 1. Сначала переводим матч в статус FINISHED в базе
	err := s.matchRepo.FinishMatch(ctx, matchID, winner)
	if err != nil {
		return fmt.Errorf("ошибка завершения матча: %w", err)
	}

	// 2. Затем вызываем расчет ставок в BetRepository
	err = s.betRepo.SettleBets(ctx, matchID, winner)
	if err != nil {
		return fmt.Errorf("ошибка расчета ставок: %w", err)
	}

	return nil
}
