package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/uva337/betting-platform/betting-service/internal/service"
)

// Настраиваем Upgrader, который превращает обычный HTTP-запрос в WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Разрешаем подключения с любых портов и доменов (для локальной разработки)
	},
}

type WebSocketHandler struct {
	matchService *service.MatchService
}

func NewWebSocketHandler(matchService *service.MatchService) *WebSocketHandler {
	return &WebSocketHandler{matchService: matchService}
}

// StreamOdds держит соединение открытым и шлет JSON с котировками
func (h *WebSocketHandler) StreamOdds(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	matchID, err := strconv.Atoi(idStr)
	if err != nil || matchID <= 0 {
		http.Error(w, "invalid match id", http.StatusBadRequest)
		return
	}

	// 1. Апгрейдим HTTP до WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Ошибка апгрейда до WebSocket: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("Клиент подключился к трансляции матча %d", matchID)

	// 2. Запускаем бесконечный цикл отправки данных
	for {
		odds, err := h.matchService.GetLiveOdds(r.Context(), matchID)
		if err == nil {
			// Если котировки есть в Redis, отправляем их прямо в браузер
			if err := conn.WriteJSON(odds); err != nil {
				log.Printf("Клиент отключился от матча %d", matchID)
				break // Выходим из цикла, если клиент закрыл вкладку
			}
		}

		// Спим 1 секунду, чтобы не спамить (в идеальной системе тут был бы Pub/Sub, но для старта это идеально)
		time.Sleep(1 * time.Second)
	}
}
