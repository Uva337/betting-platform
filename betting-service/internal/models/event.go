package models

import "time"

// MatchEvent описывает событие, произошедшее в игре

type MatchEvent struct {
	MatchID    int       `json:"match_id"`
	EventType  string    `json:"event_type"` // "match_started", "round_finished", "match_finished"
	TeamAScore int       `json:"team_a_score"`
	TeamBScore int       `json:"team_b_score"`
	Winner     string    `json:"winner,omitempty"` // Добавлено: Победитель матча (заполняется только в конце)
	Timestamp  time.Time `json:"timestamp"`
}
