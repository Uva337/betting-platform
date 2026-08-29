package handlers

import (
	"encoding/json"
	"log" // <-- Добавили забытый пакет log
	"net/http"

	"github.com/uva337/betting-platform/betting-service/internal/service"
)

type PlaceBetRequest struct {
	UserID  int     `json:"user_id"`
	MatchID int     `json:"match_id"`
	Amount  float64 `json:"amount"`
}

type BetHandler struct {
	betService *service.BetService
}

func NewBetHandler(betService *service.BetService) *BetHandler {
	return &BetHandler{betService: betService}
}

// PlaceBet теперь является методом структуры BetHandler
func (h *BetHandler) PlaceBet(w http.ResponseWriter, r *http.Request) {
	var req PlaceBetRequest

	// 1. Распаковываем JSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
		return
	}

	// 2. Валидация (должна быть строго ДО обращения к сервису)
	if req.Amount <= 0 {
		http.Error(w, `{"error": "amount must be greater than zero"}`, http.StatusBadRequest)
		return
	}
	if req.UserID == 0 || req.MatchID == 0 {
		http.Error(w, `{"error": "user_id and match_id are required"}`, http.StatusBadRequest)
		return
	}

	// 3. Вызываем слой бизнес-логики ОДИН РАЗ
	bet, err := h.betService.ProcessBet(r.Context(), req.UserID, req.MatchID, req.Amount)
	if err != nil {
		// Вот наш лог для отладки
		log.Printf("Ошибка при обработке ставки: %v", err)
		http.Error(w, `{"error": "failed to process bet"}`, http.StatusInternalServerError)
		return
	}

	// 4. Отправляем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   bet,
	})
}
