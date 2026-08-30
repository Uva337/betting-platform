package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/uva337/betting-platform/betting-service/internal/models"
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

// SetMatchOdds сохраняет коэффициенты матча в Redis
func (r *RedisRepository) SetMatchOdds(ctx context.Context, odds models.MatchOdds) error {
	key := fmt.Sprintf("match:%d:odds", odds.MatchID)

	// Упаковываем структуру в JSON
	data, err := json.Marshal(odds)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга odds: %w", err)
	}

	// Кладем в Redis (без TTL, так как матч может идти долго, удалим при завершении)
	err = r.client.Set(ctx, key, data, 0).Err()
	if err != nil {
		return fmt.Errorf("ошибка записи в redis: %w", err)
	}

	return nil
}

// GetMatchOdds получает актуальные коэффициенты из Redis
func (r *RedisRepository) GetMatchOdds(ctx context.Context, matchID int) (*models.MatchOdds, error) {
	key := fmt.Sprintf("match:%d:odds", matchID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("коэффициенты для матча %d не найдены в кэше", matchID)
		}
		return nil, fmt.Errorf("ошибка чтения из redis: %w", err)
	}

	var odds models.MatchOdds
	if err := json.Unmarshal(data, &odds); err != nil {
		return nil, fmt.Errorf("ошибка анмаршалинга odds: %w", err)
	}

	return &odds, nil
}
