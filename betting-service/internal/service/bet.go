package service

import (
	"context"

	"github.com/uva337/betting-platform/betting-service/internal/models"
	"github.com/uva337/betting-platform/betting-service/internal/repository"
)

// BetService содержит бизнес-логику для ставок
type BetService struct {
	repo *repository.BetRepository
}

// NewBetService конструктор сервиса
func NewBetService(repo *repository.BetRepository) *BetService {
	return &BetService{repo: repo}
}

// ProcessBet обрабатывает бизнес-правила и передает данные на сохранение
func (s *BetService) ProcessBet(ctx context.Context, userID, matchID int, amount float64) (*models.Bet, error) {

	// 1. Бизнес-логика: получаем текущий коэффициент матча
	// Пока используем заглушку, в реальности здесь был бы поход в другой сервис или таблицу БД
	currentOdds := 1.95

	// 2. Формируем модель ставки
	bet := &models.Bet{
		UserID:  userID,
		MatchID: matchID,
		Amount:  amount,
		Odds:    currentOdds,
		Status:  "PENDING", // Начальный статус для любой новой ставки
	}

	// 3. Передаем готовую модель в слой Repository для сохранения в БД
	err := s.repo.CreateBet(ctx, bet)
	if err != nil {
		return nil, err
	}

	return bet, nil
}
