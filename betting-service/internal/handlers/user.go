package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/uva337/betting-platform/betting-service/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetBalance возвращает текущий баланс пользователя
func (h *UserHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	// Вытаскиваем ID пользователя из URL (например, /users/1/balance)
	idStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(idStr)
	if err != nil || userID <= 0 {
		http.Error(w, `{"error": "invalid user id"}`, http.StatusBadRequest)
		return
	}

	// Идем за балансом в сервис
	balance, err := h.userService.GetBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error": "user not found"}`, http.StatusNotFound)
		return
	}

	// Отдаем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"user_id": userID,
			"balance": balance,
		},
	})
}
