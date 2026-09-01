package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MatchRepository отвечает за работу с таблицей матчей
type MatchRepository struct {
	pool *pgxpool.Pool
}

// NewMatchRepository создает новый экземпляр репозитория
func NewMatchRepository(pool *pgxpool.Pool) *MatchRepository {
	return &MatchRepository{pool: pool}
}

// FinishMatch обновляет статус матча и записывает победителя
func (r *MatchRepository) FinishMatch(ctx context.Context, matchID int, winner string) error {
	query := `
		UPDATE matches
		SET status = 'FINISHED', winner = $1
		WHERE id = $2 AND status != 'FINISHED'
	`

	// Exec используется, когда нам не нужно возвращать строки (в отличие от QueryRow)
	commandTag, err := r.pool.Exec(ctx, query, winner, matchID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении матча: %w", err)
	}

	// Защита от повторного завершения или несуществующего матча
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("матч не найден или уже был завершен")
	}

	return nil
}

func (r *MatchRepository) CreateMatch(ctx context.Context, matchID int, title string, oddsA, oddsB float64) error {
	query := `
		INSERT INTO matches (id, title, team_a_odds, team_b_odds, status) 
		VALUES ($1, $2, $3, $4, 'PENDING')
	`
	_, err := r.pool.Exec(ctx, query, matchID, title, oddsA, oddsB)
	return err
}
