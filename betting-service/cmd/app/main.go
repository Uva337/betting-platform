package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Unable to connect to Redis: %v", err)
	}

	log.Println("Successfully connected to Redis!")

	// ============================================
	// СБОРКА СЛОЕВ (Dependency Injection)
	// ============================================

	betRepo := repository.NewBetRepository(pool)
	betSvc := service.NewBetService(betRepo)
	betHandler := handlers.NewBetHandler(betSvc)

	userRepo := repository.NewUserRepository(pool)
	userSvc := service.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userSvc)

	redisRepo := repository.NewRedisRepository(rdb)
	matchRepo := repository.NewMatchRepository(pool)

	matchSvc := service.NewMatchService(matchRepo, betRepo, redisRepo)
	matchHandler := handlers.NewMatchHandler(matchSvc)
	wsHandler := handlers.NewWebSocketHandler(matchSvc)

	kafkaBroker := "localhost:9094"
	kafkaTopic := "match-events"

	// Инициализируем админский хэндлер
	adminHandler := handlers.NewAdminHandler(matchSvc, kafkaBroker, kafkaTopic)

	// ============================================
	// НАСТРОЙКА РОУТЕРА
	// ============================================
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
		})

		// Публичные маршруты
		r.Post("/bets", betHandler.PlaceBet)
		r.Get("/bets/{id}", betHandler.GetBet)
		r.Get("/users/{id}/balance", userHandler.GetBalance)
		r.Get("/users/{id}/bets", betHandler.GetUserBets)
		r.Get("/matches/{id}/odds", matchHandler.GetLiveOdds)
		r.Get("/matches/{id}/ws", wsHandler.StreamOdds)

		// Закрытые маршруты (Admin API)
		r.Route("/admin", func(r chi.Router) {
			r.Post("/matches", adminHandler.CreateMatchAPI)
			r.Post("/matches/{id}/start", adminHandler.StartMatchSimulation)
		})
	})

	// ============================================
	// ЗАПУСК ФОНОВЫХ ВОРКЕРОВ
	// ============================================
	// Запускаем только движок прослушивания (Consumer). Симулятор (Producer) теперь запускается только по API!
	oddsEngine := worker.NewOddsEngine(redisRepo, matchSvc, kafkaBroker, kafkaTopic)
	go oddsEngine.Start(context.Background())

	// ============================================
	// ЗАПУСК СЕРВЕРА
	// ============================================
	log.Println("Server is listening on port 8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
