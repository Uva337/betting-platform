package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/uva337/betting-platform/betting-service/internal/handlers"
	"github.com/uva337/betting-platform/betting-service/internal/repository"
	"github.com/uva337/betting-platform/betting-service/internal/service"
	"github.com/uva337/betting-platform/betting-service/internal/worker"
)

func main() {
	log.Println("Starting Betting Service...")

	// 1. Читаем конфиг / строку подключения
	dsn := "postgres://betting_admin:betting_secret_pass@localhost:5432/betting_service_db?sslmode=disable"

	// 2. Создаем пул соединений с базой данных
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("Successfully connected to PostgreSQL (betting_service_db)!")

	// ==========================================
	// Подключение к Redis
	// ==========================================
	redisHost := os.Getenv("REDIS_PORT")
	if redisHost == "" {
		redisHost = "6379"
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:" + redisHost,
		Password: "", // пароля нет
		DB:       0,  // дефолтная база
	})

	// Проверяем соединение с Redis
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v", err)
	}
	// defer rdb.Close() // ВАЖНО: Закрывать соединение в main не нужно, иначе приложение не сможет с ним работать после старта

	log.Println("Successfully connected to Redis!")

	// ============================================
	// СБОРКА СЛОЕВ (Dependency Injection)
	// ============================================

	betRepo := repository.NewBetRepository(pool)
	betSvc := service.NewBetService(betRepo)
	betHandler := handlers.NewBetHandler(betSvc)

	// Домен пользователей
	userRepo := repository.NewUserRepository(pool)
	userSvc := service.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userSvc)

	// Домен матчей + Redis
	redisRepo := repository.NewRedisRepository(rdb)
	matchRepo := repository.NewMatchRepository(pool)

	// Теперь мы передаем redisRepo внутрь сервиса матчей!
	matchSvc := service.NewMatchService(matchRepo, betRepo, redisRepo)
	matchHandler := handlers.NewMatchHandler(matchSvc)

	wsHandler := handlers.NewWebSocketHandler(matchSvc)

	// ============================================
	// НАСТРОЙКА РОУТЕРА
	// ============================================
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
		})

		// Маршруты ставок
		r.Post("/bets", betHandler.PlaceBet)
		r.Get("/bets/{id}", betHandler.GetBet)

		// Маршруты пользователей
		r.Get("/users/{id}/balance", userHandler.GetBalance)
		r.Get("/users/{id}/bets", betHandler.GetUserBets)

		// Маршруты матчей
		r.Post("/matches/{id}/finish", matchHandler.FinishMatch)
		r.Get("/matches/{id}/odds", matchHandler.GetLiveOdds)
		r.Get("/matches/{id}/ws", wsHandler.StreamOdds)
	})

	kafkaBroker := "localhost:9094" // Адрес из нашего docker-compose (External порт)
	kafkaTopic := "match-events"

	// 1. Запускаем торговый движок
	oddsEngine := worker.NewOddsEngine(redisRepo, kafkaBroker, kafkaTopic)
	go oddsEngine.Start(context.Background())

	// 2. Запускаем симулятор с УНИКАЛЬНЫМ ID матча
	// Берем последние 4 цифры от текущего unix-времени, чтобы ID каждый раз был новым (например, 5823)
	liveMatchID := int(time.Now().Unix() % 10000)

	simulator := worker.NewMatchSimulator(kafkaBroker, kafkaTopic)
	go simulator.Start(context.Background(), liveMatchID)

	// ============================================
	// ЗАПУСК СЕРВЕРА
	// ============================================
	log.Println("Server is listening on port 8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
