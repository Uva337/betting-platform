package models

import "time"

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type Match struct {
	ID        int     `json:"id"`
	Title     string  `json:"title"`
	TeamAodds float64 `json:"team_a_odds"`
	TeamBodds float64 `json:"team_b_odds"`
	Status    string  `json:"status"`
}

type Bet struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id"`
	MatchID    int       `json:"match_id"`
	Amount     float64   `json:"amount"`
	Odds       float64   `json:"odds"`
	Prediction string    `json:"prediction"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}
