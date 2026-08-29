package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresDB создает пул соединений с базой данных
func NewPostgresDB(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	// Парсим строку подключения
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("не удалось распарсить конфиг БД: %w", err)
	}

	// Настройки для Highload (Пул соединений)
	// Держим максимум 20 открытых соединений
	config.MaxConns = 20
	// Держим минимум 5 соединений всегда открытыми (чтобы не тратить время на хэндшейк при всплеске нагрузки)
	config.MinConns = 5
	// Если соединение простаивает 5 минут - закрываем его
	config.MaxConnIdleTime = time.Minute * 5

	// Создаем пул соединений
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе: %w", err)
	}

	// Делаем Ping, чтобы убедиться, что база реально отвечает, а не просто приняла конфиг
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("база данных не отвечает на ping: %w", err)
	}

	return pool, nil
}
