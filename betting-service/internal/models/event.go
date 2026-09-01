package models

import "time"

// MatchEvent описывает событие, произошедшее в игре
type MatchEvent struct {
	MatchID    int       `json:"match_id"`
	EventType  string    `json:"event_type"`
	TeamAScore int       `json:"team_a_score"`
	TeamBScore int       `json:"team_b_score"`
	Timestamp  time.Time `json:"timestamp"`
}
