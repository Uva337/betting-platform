package models

// MatchOdds представляет текущие коэффициенты матча для кэша
type MatchOdds struct {
	MatchID int     `json:"match_id"`
	TeamA   float64 `json:"team_a"` // Коэффициент на победу Команды А
	TeamB   float64 `json:"team_b"` // Коэффициент на победу Команды Б
}
