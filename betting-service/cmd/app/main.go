package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/uva337/betting-platform/betting-service/internal/handlers"
	"github.com/uva337/betting-platform/betting-service/internal/repository"
	"github.com/uva337/betting-platform/betting-service/internal/service"
)

func main() {
	log.Println("Starting Betting Service...")

	// 1. Читаем конфиг / строку подключения (используем стандартный адрес из твоего docker-compose)
	dsn := "postgres://betting_admin:betting_secret_pass@localhost:5432/betting_service_db?sslmode=disable"

	// 2. Создаем пул соединений с базой данных
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("Successfully connected to PostgreSQL (betting_service_db)!")

	// ============================================
	// СБОРКА СЛОЕВ (Dependency Injection)
	// ============================================
	// 1. Подключение к базе данных (предполагаем, что пул у тебя уже настроен выше)

	betRepo := repository.NewBetRepository(pool)
	betSvc := service.NewBetService(betRepo)
	betHandler := handlers.NewBetHandler(betSvc)

	// Домен пользователей
	userRepo := repository.NewUserRepository(pool)
	userSvc := service.NewUserService(userRepo)
	userHandler := handlers.NewUserHandler(userSvc)

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
	})

	// ============================================
	// ЗАПУСК СЕРВЕРА
	// ============================================
	log.Println("Server is listening on port 8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
