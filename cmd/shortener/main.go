package main

import (
	"log"
	"net/http"

	"github.com/JustScorpio/urlshortener/internal/handlers"
	"github.com/JustScorpio/urlshortener/internal/repository/postgres"
	"github.com/JustScorpio/urlshortener/internal/services"

	"github.com/gorilla/mux"
)

// функция main вызывается автоматически при запуске приложения
func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run() error {

	// Инициализация репозиториев (+ базы данных)
	repo, err := postgres.NewPostgresShURLRepository()
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Db.Close()

	// Инициализация сервисов
	shURLService := services.NewShURLService(repo)

	// Инициализация обработчиков
	shURLHandler := handlers.NewShURLHandler(shURLService)

	r := mux.NewRouter()
	r.HandleFunc("/", shURLHandler.GetFullURL).Methods("GET")
	r.HandleFunc("/", shURLHandler.ShortenURL).Methods("POST")

	return http.ListenAndServe(":8080", r)
}
