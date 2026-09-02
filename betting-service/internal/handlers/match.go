package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/uva337/betting-platform/betting-service/internal/service"
)

type MatchHandler struct {
	matchService *service.MatchService
}

func NewMatchHandler(matchService *service.MatchService) *MatchHandler {
	return &MatchHandler{matchService: matchService}
}

// Структура для приема JSON от клиента
type FinishMatchRequest struct {
	Winner string `json:"winner"`
}

// FinishMatch обрабатывает запрос на завершение матча
func (h *MatchHandler) FinishMatch(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	matchID, err := strconv.Atoi(idStr)
	if err != nil || matchID <= 0 {
		http.Error(w, `{"error": "invalid match id"}`, http.StatusBadRequest)
		return
	}

	var req FinishMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Winner == "" {
		http.Error(w, `{"error": "invalid request, 'winner' is required"}`, http.StatusBadRequest)
		return
	}

	// Запускаем бизнес-логику оркестратора
	err = h.matchService.FinishMatch(r.Context(), matchID, req.Winner)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "match finished and bets settled",
	})
}

// GetLiveOdds возвращает текущие котировки матча
func (h *MatchHandler) GetLiveOdds(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	matchID, err := strconv.Atoi(idStr)
	if err != nil || matchID <= 0 {
		http.Error(w, `{"error": "invalid match id"}`, http.StatusBadRequest)
		return
	}

	// Идем в Redis через сервис
	odds, err := h.matchService.GetLiveOdds(r.Context(), matchID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   odds,
	})
}

func (h *MatchHandler) GetActiveMatches(w http.ResponseWriter, r *http.Request) {
	matches, err := h.matchService.GetActiveMatches(r.Context())
	if err != nil {
		http.Error(w, "не удалось получить матчи", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}
