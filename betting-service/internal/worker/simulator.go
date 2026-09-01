package worker

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/uva337/betting-platform/betting-service/internal/models"
)

type MatchSimulator struct {
	writer *kafka.Writer
}

func NewMatchSimulator(brokerURL, topic string) *MatchSimulator {
	// Настраиваем запись в Kafka
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokerURL),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &MatchSimulator{writer: writer}
}

// Start запускает симуляцию матча
func (s *MatchSimulator) Start(ctx context.Context, matchID int) {
	log.Printf("Simulator: начал симуляцию матча %d", matchID)
	defer s.writer.Close()

	scoreA, scoreB := 0, 0

	// 1. ОТПРАВЛЯЕМ СТАРТОВЫЕ НУЛИ СРАЗУ (без задержки)
	startEvent := models.MatchEvent{
		MatchID:    matchID,
		EventType:  "match_started",
		TeamAScore: scoreA,
		TeamBScore: scoreB,
		Timestamp:  time.Now(),
	}
	startBytes, _ := json.Marshal(startEvent)
	_ = s.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte("live-match"),
		Value: startBytes,
	})

	// 2. НАЧИНАЕМ ИГРАТЬ РАУНДЫ
	for scoreA < 13 && scoreB < 13 {
		// Ждем 10 секунд перед следующим раундом
		time.Sleep(10 * time.Second)

		if rand.Float32() < 0.5 {
			scoreA++
		} else {
			scoreB++
		}

		event := models.MatchEvent{
			MatchID:    matchID,
			EventType:  "round_finished",
			TeamAScore: scoreA,
			TeamBScore: scoreB,
			Timestamp:  time.Now(),
		}
		eventBytes, _ := json.Marshal(event)

		err := s.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte("live-match"),
			Value: eventBytes,
		})

		if err != nil {
			log.Printf("Simulator ошибка отправки в Kafka: %v", err)
		}
	}
	log.Printf("Simulator: матч %d завершен со счетом %d:%d", matchID, scoreA, scoreB)
}
