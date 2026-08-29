package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/uva337/betting-platform/betting-service/internal/models"
)

// BetRepository отвечает за работу с таблицей bets
type BetRepository struct {
	pool *pgxpool.Pool
}

// NewBetRepository создает новый экземпляр репозитория
func NewBetRepository(pool *pgxpool.Pool) *BetRepository {
	return &BetRepository{pool: pool}
}

// CreateBet сохраняет новую ставку в базу данных
func (r *BetRepository) CreateBet(ctx context.Context, bet *models.Bet) error {
	// SQL-запрос на вставку.
	// Использование $1, $2 защищает нас от SQL-инъекций (база сама безопасно подставит параметры)
	query := `
		INSERT INTO bets (user_id, match_id, amount, odds, status, created_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		RETURNING id, created_at
	`

	// Выполняем запрос и сразу считываем сгенерированные базой ID и время создания обратно в структуру
	err := r.pool.QueryRow(ctx, query,
		bet.UserID,
		bet.MatchID,
		bet.Amount,
		bet.Odds,
		bet.Status,
	).Scan(&bet.ID, &bet.CreatedAt)

	if err != nil {
		return fmt.Errorf("ошибка при сохранении ставки в БД: %w", err)
	}

	return nil
}
