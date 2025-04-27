package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/JustScorpio/urlshortener/internal/handlers"
	"github.com/JustScorpio/urlshortener/internal/repository/sqlite"
	"github.com/JustScorpio/urlshortener/internal/services"

	"github.com/go-chi/chi"
)

// функция main вызывается автоматически при запуске приложения
func main() {
	// обрабатываем аргументы командной строки
	parseFlags()

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
	shURLHandler := handlers.NewShURLHandler(shURLService, flagShURLBaseAddr)

	r := chi.NewRouter()
	r.Get("/{token}", shURLHandler.GetFullURL)
	r.Post("/", shURLHandler.ShortenURL)

	fmt.Println("Running server on", flagRunAddr)
	return http.ListenAndServe(flagRunAddr, r)
}
