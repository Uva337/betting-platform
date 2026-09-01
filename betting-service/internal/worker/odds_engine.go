package worker

import (
	"context"
	"encoding/json"
	"log"
	"math"

	"github.com/segmentio/kafka-go"
	"github.com/uva337/betting-platform/betting-service/internal/models"
	"github.com/uva337/betting-platform/betting-service/internal/repository"
	"github.com/uva337/betting-platform/betting-service/internal/service"
)

type OddsEngine struct {
	redisRepo *repository.RedisRepository
	matchSvc  *service.MatchService
	reader    *kafka.Reader
}

func NewOddsEngine(redisRepo *repository.RedisRepository, matchSvc *service.MatchService, brokerURL, topic string) *OddsEngine {
	// Настраиваем чтение из Kafka
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokerURL},
		Topic:   topic,
		GroupID: "odds-engine-group", // Группа консьюмеров для балансировки
	})

	return &OddsEngine{
		redisRepo: redisRepo,
		matchSvc:  matchSvc,
		reader:    reader,
	}
}

func (e *OddsEngine) Start(ctx context.Context) {
	log.Printf("OddsEngine: начал прослушивание Kafka...")
	defer e.reader.Close()

	for {
		msg, err := e.reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("OddsEngine ошибка чтения из Kafka: %v", err)
			continue
		}

		var event models.MatchEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("OddsEngine ошибка парсинга события: %v", err)
			continue
		}

		// АВТОМАТИЧЕСКИЙ РАСЧЕТ СТАВОК
		if event.EventType == "match_finished" {
			log.Printf("OddsEngine: получен сигнал о завершении матча %d. Запускаем расчет ставок! Победитель: %s", event.MatchID, event.Winner)

			// Вызываем нашу мощную транзакцию из БД!
			err := e.matchSvc.FinishMatch(ctx, event.MatchID, event.Winner)
			if err != nil {
				log.Printf("❌ Ошибка расчета ставок: %v", err)
			} else {
				log.Printf("✅ Ставки для матча %d успешно рассчитаны и деньги начислены!", event.MatchID)
			}
			continue // Выходим на следующий круг цикла, котировки обновлять уже не нужно
		}

		log.Printf("OddsEngine получил событие: %s, Счет: %d:%d", event.EventType, event.TeamAScore, event.TeamBScore)
		e.recalculateAndSave(ctx, event)
	}
}

// recalculateAndSave содержит математику букмекера
func (e *OddsEngine) recalculateAndSave(ctx context.Context, event models.MatchEvent) {
	// Базовая вероятность 50% (0.50).
	// За каждый выигранный раунд (относительно противника) вероятность растет на 4% (0.04)
	probA := 0.50 + float64(event.TeamAScore-event.TeamBScore)*0.04

	// Ограничиваем вероятности, чтобы они не ушли за пределы (5% - 95%)
	probA = math.Max(0.05, math.Min(0.95, probA))
	probB := 1.0 - probA

	// Формула коэффициента: 1 / вероятность.
	// Вычитаем маржу букмекера 5% (0.05) из итогового кэфа, чтобы контора была в плюсе
	oddsA := (1 / probA) * 0.95
	oddsB := (1 / probB) * 0.95

	// Округляем до двух знаков после запятой
	oddsA = math.Round(oddsA*100) / 100
	oddsB = math.Round(oddsB*100) / 100

	odds := models.MatchOdds{
		MatchID: event.MatchID,
		TeamA:   oddsA,
		TeamB:   oddsB,
	}

	if err := e.redisRepo.SetMatchOdds(ctx, odds); err != nil {
		log.Printf("OddsEngine ошибка записи в Redis: %v", err)
	} else {
		log.Printf("OddsEngine обновил котировки в Redis -> Team A: %.2f | Team B: %.2f", odds.TeamA, odds.TeamB)
	}
}
