package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/uva337/betting-platform/betting-service/internal/service"
	"github.com/uva337/betting-platform/betting-service/internal/worker"
)

type AdminHandler struct {
	matchSvc    *service.MatchService
	kafkaBroker string
	kafkaTopic  string
}

func NewAdminHandler(matchSvc *service.MatchService, broker, topic string) *AdminHandler {
	return &AdminHandler{
		matchSvc:    matchSvc,
		kafkaBroker: broker,
		kafkaTopic:  topic,
	}
}

// CreateMatchAPI обрабатывает POST /admin/matches
func (h *AdminHandler) CreateMatchAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title     string  `json:"title"`
		TeamAOdds float64 `json:"team_a_odds"`
		TeamBOdds float64 `json:"team_b_odds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	matchID, err := h.matchSvc.CreateMatch(r.Context(), req.Title, req.TeamAOdds, req.TeamBOdds)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"match_id": matchID,
		"message":  "Match created successfully",
	})
}

// StartMatchSimulation запускает наш движок для конкретного матча
func (h *AdminHandler) StartMatchSimulation(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	matchID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid match id", http.StatusBadRequest)
		return
	}

	// Запускаем симулятор как отдельную горутину по запросу админа!
	simulator := worker.NewMatchSimulator(h.kafkaBroker, h.kafkaTopic)
	go simulator.Start(context.Background(), matchID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Simulation started for match " + idStr,
	})
}
