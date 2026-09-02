package worker

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/uva337/betting-platform/betting-service/internal/service"
)

// MatchGenerator отвечает за автоматическое создание матчей по расписанию
type MatchGenerator struct {
	matchSvc    *service.MatchService
	kafkaBroker string
	kafkaTopic  string
	ticker      *time.Ticker
}

// NewMatchGenerator создает новый экземпляр генератора
func NewMatchGenerator(matchSvc *service.MatchService, broker, topic string, interval time.Duration) *MatchGenerator {
	return &MatchGenerator{
		matchSvc:    matchSvc,
		kafkaBroker: broker,
		kafkaTopic:  topic,
		ticker:      time.NewTicker(interval),
	}
}

// Start запускает бесконечный цикл генерации
func (g *MatchGenerator) Start(ctx context.Context) {
	log.Println("MatchGenerator: запущен автоматический генератор матчей...")

	// Запускаем самый первый матч сразу, чтобы не ждать первого "тика" таймера
	g.generateAndStartMatch(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("MatchGenerator: остановка...")
			g.ticker.Stop()
			return
		case <-g.ticker.C:
			// Срабатывает каждый раз по истечении интервала
			g.generateAndStartMatch(ctx)
		}
	}
}

func (g *MatchGenerator) generateAndStartMatch(ctx context.Context) {
	// Пул топовых команд
	teams := []string{"NAVI", "FaZe Clan", "Team Spirit", "Vitality", "G2 Esports", "MOUZ", "Heroic", "Virtus.pro"}

	// Выбираем две случайные разные команды
	teamA := teams[rand.Intn(len(teams))]
	teamB := teams[rand.Intn(len(teams))]
	for teamB == teamA {
		teamB = teams[rand.Intn(len(teams))]
	}

	title := fmt.Sprintf("%s vs %s", teamA, teamB)

	// Генерируем случайные стартовые коэффициенты от 1.50 до 2.50
	oddsA := 1.50 + rand.Float64()
	oddsB := 1.50 + rand.Float64()

	// 1. Создаем матч в базе данных (через наш существующий сервис)
	matchID, err := g.matchSvc.CreateMatch(ctx, title, oddsA, oddsB)
	if err != nil {
		log.Printf("⚠️ MatchGenerator: ошибка создания матча: %v", err)
		return
	}
	log.Printf("🎯 MatchGenerator: создан новый матч [%d] %s (Коэф: %.2f / %.2f)", matchID, title, oddsA, oddsB)

	// 2. Запускаем симулятор для этого матча
	simulator := NewMatchSimulator(g.kafkaBroker, g.kafkaTopic)
	go simulator.Start(ctx, matchID)
}
