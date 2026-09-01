package service

import (
	"context"
	"fmt"
	"time"

	"github.com/uva337/betting-platform/betting-service/internal/models"
	"github.com/uva337/betting-platform/betting-service/internal/repository"
)

type MatchService struct {
	matchRepo *repository.MatchRepository
	betRepo   *repository.BetRepository
	redisRepo *repository.RedisRepository
}

// NewMatchService принимает оба репозитория
func NewMatchService(matchRepo *repository.MatchRepository, betRepo *repository.BetRepository, redisRepo *repository.RedisRepository) *MatchService {
	return &MatchService{
		matchRepo: matchRepo,
		betRepo:   betRepo,
		redisRepo: redisRepo,
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

func (s *MatchService) GetLiveOdds(ctx context.Context, matchID int) (*models.MatchOdds, error) {
	odds, err := s.redisRepo.GetMatchOdds(ctx, matchID)
	if err != nil {
		// В реальной системе тут мог бы быть фоллбэк на БД,
		// но для Live-коэффициентов Redis - единственный источник правды
		return nil, fmt.Errorf("коэффициенты не найдены: %w", err)
	}
	return odds, nil
}

func (s *MatchService) CreateMatch(ctx context.Context, title string, oddsA, oddsB float64) (int, error) {
	// Генерируем ID
	matchID := int(time.Now().Unix() % 10000)

	// Просим репозиторий сохранить данные
	err := s.matchRepo.CreateMatch(ctx, matchID, title, oddsA, oddsB)
	if err != nil {
		return 0, err
	}

	return matchID, nil
}
