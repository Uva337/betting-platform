package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository отвечает за работу с таблицей users
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository создает новый экземпляр репозитория
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// GetBalance запрашивает текущий баланс пользователя
func (r *UserRepository) GetBalance(ctx context.Context, userID int) (float64, error) {
	var balance float64

	// Нам не нужна вся строка пользователя, запрашиваем только одну колонку
	query := `SELECT balance FROM users WHERE id = $1`

	err := r.pool.QueryRow(ctx, query, userID).Scan(&balance)
	if err != nil {
		// Печатаем чистую ошибку от базы прямо в терминал сервера!
		log.Printf("DEBUG DB ERROR: %v for userID=%d", err, userID)
		return 0, fmt.Errorf("ошибка получения баланса: %w", err)
	}

	return balance, nil
}
