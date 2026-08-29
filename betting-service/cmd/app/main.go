package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/uva337/betting-platform/betting-service/internal/repository"
)

func main() {
	log.Println("Starting Betting Service...")

	// 1. Формируем строку подключения (DSN)
	// Мы стучимся на localhost, потому что Go-код запущен там же, где и Докер
	// И подключаемся именно к базе betting_service_db
	connString := "postgres://betting_admin:betting_secret_pass@localhost:5432/betting_service_db"

	// 2. Подключаемся к PostgreSQL
	// context.Background() дает нам базовый контекст, который живет всё время работы приложения
	pool, err := repository.NewPostgresDB(context.Background(), connString)
	if err != nil {
		// Если база недоступна, log.Fatalf выведет ошибку и мгновенно убьет процесс.
		// Нет смысла запускать сервис ставок, если он не может сохранять ставки.
		log.Fatalf("Failed to initialize database: %v", err)
	}
	// defer гарантирует, что пул соединений закроется корректно, когда приложение будет остановлено
	defer pool.Close()

	log.Println("Successfully connected to PostgreSQL (betting_service_db)!")

	// 3. Инициализируем роутер Chi
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
		})
	})

	// 4. Запуск HTTP-сервера
	log.Println("Server is listening on port 8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
