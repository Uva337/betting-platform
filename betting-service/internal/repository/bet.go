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
	// 1. Начинаем транзакцию
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ошибка старта транзакции: %w", err)
	}

	// defer tx.Rollback() - это страховка. Если мы выйдем из функции с ошибкой (до Commit),
	// Rollback автоматически откатит все изменения в базе.
	// Если же транзакция завершится успешно (Commit), Rollback простSо ничего не сделает.
	defer tx.Rollback(ctx)

	// 2. Списываем деньги с баланса (ТОЛЬКО если их хватает)
	// Условие "AND balance >= $1" защищает нас от ухода баланса в минус
	updateQuery := `
		UPDATE users 
		SET balance = balance - $1 
		WHERE id = $2 AND balance >= $1
	`
	// tx.Exec выполняет запрос, не ожидая возврата строк данных
	commandTag, err := tx.Exec(ctx, updateQuery, bet.Amount, bet.UserID)
	if err != nil {
		return fmt.Errorf("ошибка при обновлении баланса: %w", err)
	}

	// Проверяем, обновилась ли хоть одна строка.
	// Если 0, значит либо юзера нет, либо у него недостаточно денег!
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("недостаточно средств на балансе или пользователь не найден")
	}

	// 3. Сохраняем саму ставку
	insertQuery := `
		INSERT INTO bets (user_id, match_id, amount, odds, status, created_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		RETURNING id, created_at
	`
	// Обрати внимание: теперь мы используем tx.QueryRow, а не r.pool.QueryRow
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

	// 4. Фиксируем транзакцию (окончательно сохраняем всё в БД)
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return nil
}

// GetBetByID ищет ставку в базе по её ID
func (r *BetRepository) GetBetByID(ctx context.Context, id int) (*models.Bet, error) {
	// Запрос на поиск одной строки (SELECT)
	query := `
		SELECT id, user_id, match_id, amount, odds, status, created_at
		FROM bets
		WHERE id = $1
	`

	var bet models.Bet

	// Выполняем запрос и считываем колонки в структуру.
	// Порядок переменных в Scan() должен строго совпадать с порядком колонок в SELECT!
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&bet.ID,
		&bet.UserID,
		&bet.MatchID,
		&bet.Amount,
		&bet.Odds,
		&bet.Status,
		&bet.CreatedAt,
	)

	if err != nil {
		// Если база ответит "нет таких строк" (pgx.ErrNoRows), вернется эта ошибка
		return nil, fmt.Errorf("ставка не найдена: %w", err)
	}

	return &bet, nil
}
