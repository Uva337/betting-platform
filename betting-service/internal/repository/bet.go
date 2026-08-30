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

// CreateBet сохраняет ставку и списывает деньги в рамках одной транзакции
func (r *BetRepository) CreateBet(ctx context.Context, bet *models.Bet) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка старта транзакции: %w", err)
	}
	defer tx.Rollback(ctx)

	updateQuery := `
		UPDATE users 
		SET balance = balance - $1 
		WHERE id = $2 AND balance >= $1
	`
	commandTag, err := tx.Exec(ctx, updateQuery, bet.Amount, bet.UserID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении баланса: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("недостаточно средств на балансе или пользователь не найден")
	}

	// ДОБАВЛЕНО: поле prediction (хардкод 'Team A' для новых ставок)
	insertQuery := `
		INSERT INTO bets (user_id, match_id, amount, odds, prediction, status, created_at)
		VALUES ($1, $2, $3, $4, 'Team A', $5, CURRENT_TIMESTAMP)
		RETURNING id, created_at
	`
	err = tx.QueryRow(ctx, insertQuery,
		bet.UserID,
		bet.MatchID,
		bet.Amount,
		bet.Odds,
		bet.Status,
	).Scan(&bet.ID, &bet.CreatedAt)

	if err != nil {
		return fmt.Errorf("ошибка при сохранении ставки: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// GetBetByID ищет ставку в базе по её ID
func (r *BetRepository) GetBetByID(ctx context.Context, id int) (*models.Bet, error) {
	// ДОБАВЛЕНО: колонка prediction
	query := `
		SELECT id, user_id, match_id, amount, odds, prediction, status, created_at
		FROM bets
		WHERE id = $1
	`

	var bet models.Bet

	// ДОБАВЛЕНО: &bet.Prediction
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&bet.ID,
		&bet.UserID,
		&bet.MatchID,
		&bet.Amount,
		&bet.Odds,
		&bet.Prediction,
		&bet.Status,
		&bet.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("ставка не найдена: %w", err)
	}

	return &bet, nil
}

// GetBetsByUserID получает историю ставок
func (r *BetRepository) GetBetsByUserID(ctx context.Context, userID int) ([]models.Bet, error) {
	// ДОБАВЛЕНО: колонка prediction
	query := `
        SELECT id, user_id, match_id, amount, odds, prediction, status, created_at
        FROM bets
        WHERE user_id = $1
        ORDER BY created_at DESC
    `
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения истории ставок: %w", err)
	}
	defer rows.Close()

	var bets []models.Bet
	for rows.Next() {
		var bet models.Bet
		// ДОБАВЛЕНО: &bet.Prediction
		if err := rows.Scan(&bet.ID, &bet.UserID, &bet.MatchID, &bet.Amount, &bet.Odds, &bet.Prediction, &bet.Status, &bet.CreatedAt); err != nil {
			return nil, fmt.Errorf("ошибка сканирования ставки: %w", err)
		}
		bets = append(bets, bet)
	}
	return bets, nil
}

// SettleBets обрабатывает ставки после завершения матча внутри транзакции.
func (r *BetRepository) SettleBets(ctx context.Context, matchID int, matchWinner string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка старта транзакции расчета ставок: %w", err)
	}
	defer tx.Rollback(ctx)

	querySelect := `
		SELECT id, user_id, amount, odds, status, prediction
		FROM bets
		WHERE match_id = $1 AND status = 'PENDING'
		FOR UPDATE
	`
	rows, err := tx.Query(ctx, querySelect, matchID)
	if err != nil {
		return fmt.Errorf("ошибка получения pending ставок: %w", err)
	}

	var betsToProcess []models.Bet
	for rows.Next() {
		var b models.Bet
		if err := rows.Scan(&b.ID, &b.UserID, &b.Amount, &b.Odds, &b.Status, &b.Prediction); err != nil {
			rows.Close()
			return fmt.Errorf("ошибка сканирования ставки при расчете: %w", err)
		}
		betsToProcess = append(betsToProcess, b)
	}
	rows.Close()

	if len(betsToProcess) == 0 {
		return tx.Commit(ctx)
	}

	for _, bet := range betsToProcess {
		isWinner := bet.Prediction == matchWinner

		if isWinner {
			newStatus := "WON"
			winnings := bet.Amount * bet.Odds

			_, err = tx.Exec(ctx, `UPDATE bets SET status = $1 WHERE id = $2`, newStatus, bet.ID)
			if err != nil {
				return fmt.Errorf("ошибка обновления статуса выигравшей ставки %d: %w", bet.ID, err)
			}

			_, err = tx.Exec(ctx, `UPDATE users SET balance = balance + $1 WHERE id = $2`, winnings, bet.UserID)
			if err != nil {
				return fmt.Errorf("ошибка зачисления выигрыша юзеру %d: %w", bet.UserID, err)
			}
		} else {
			newStatus := "LOST"
			_, err = tx.Exec(ctx, `UPDATE bets SET status = $1 WHERE id = $2`, newStatus, bet.ID)
			if err != nil {
				return fmt.Errorf("ошибка обновления статуса проигравшей ставки %d: %w", bet.ID, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита транзакции расчета: %w", err)
	}

	return nil
}
