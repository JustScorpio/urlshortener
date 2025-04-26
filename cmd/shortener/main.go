package main

import (
	"log"
	"net/http"

	"github.com/JustScorpio/urlshortener/internal/handlers"
	"github.com/JustScorpio/urlshortener/internal/repository/sqlite"
	"github.com/JustScorpio/urlshortener/internal/services"

	"github.com/go-chi/chi"
)

// функция main вызывается автоматически при запуске приложения
func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run() error {

	// Инициализация репозиториев с базой данных
	repo, err := sqlite.NewSQLiteShURLRepository()
	if err != nil {
		log.Fatal(err)
	}
	defer repo.DB.Close()

	// Инициализация сервисов
	shURLService := services.NewShURLService(repo)

	// Инициализация обработчиков
	shURLHandler := handlers.NewShURLHandler(shURLService)

	// r := mux.NewRouter()
	// r.HandleFunc("/{token}", shURLHandler.GetFullURL).Methods("GET")
	// r.HandleFunc("/", shURLHandler.ShortenURL).Methods("POST")

	// return http.ListenAndServe(":8080", r)

	r := chi.NewRouter()
	r.Get("/{token}", shURLHandler.GetFullURL)
	r.Post("/", shURLHandler.ShortenURL)
	return http.ListenAndServe(":8080", r)
}
