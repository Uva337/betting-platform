package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/uva337/betting-platform/betting-service/internal/models"
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
		log.Printf("Ошибка при обработке ставки: %v", err)

		// Проверяем, является ли ошибка бизнес-логической (нехватка денег)
		if strings.Contains(err.Error(), "недостаточно средств") {
			// Отдаем статус 400 Bad Request и понятное сообщение клиенту
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "insufficient funds"}`))
			return
		}

		// Если это какая-то другая системная ошибка (например, отвалилась БД), возвращаем 500
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

// GetBet возвращает информацию о ставке по её ID
func (h *BetHandler) GetBet(w http.ResponseWriter, r *http.Request) {
	// 1. Вытаскиваем ID из URL (например, из /bets/4 достаем "4")
	idStr := chi.URLParam(r, "id")

	// Конвертируем строку "4" в число 4
	betID, err := strconv.Atoi(idStr)
	if err != nil || betID <= 0 {
		http.Error(w, `{"error": "invalid bet id"}`, http.StatusBadRequest)
		return
	}

	// 2. Идем за данными в слой бизнес-логики (его мы сейчас напишем)
	bet, err := h.betService.GetBet(r.Context(), betID)
	if err != nil {
		// Если ставка не найдена или случилась ошибка БД
		http.Error(w, `{"error": "bet not found"}`, http.StatusNotFound)
		return
	}

	// 3. Возвращаем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // 200 OK

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   bet,
	})
}

// GetUserBets возвращает историю ставок конкретного пользователя
func (h *BetHandler) GetUserBets(w http.ResponseWriter, r *http.Request) {
	// 1. Вытаскиваем ID пользователя из URL (например, /users/1/bets)
	idStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(idStr)
	if err != nil || userID <= 0 {
		http.Error(w, `{"error": "invalid user id"}`, http.StatusBadRequest)
		return
	}

	// 2. Идем за ставками в слой бизнес-логики
	bets, err := h.betService.GetUserBets(r.Context(), userID)
	if err != nil {
		log.Printf("Ошибка получения истории ставок: %v", err)
		http.Error(w, `{"error": "failed to fetch bets"}`, http.StatusInternalServerError)
		return
	}

	// 3. Небольшой трюк: если ставок нет, срез bets будет равен nil.
	// Чтобы JSON красиво выдал пустой массив [], а не null, делаем проверку:
	if bets == nil {
		bets = []models.Bet{} // инициализируем пустой срез
	}

	// 4. Отдаем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   bets,
	})
}
