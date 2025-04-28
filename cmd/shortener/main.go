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
	// обрабатываем аргументы командной строки
	parseFlags()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run() error {

	// Инициализация репозиториев с базой данных
	repo, err := sqlite.NewSQLiteShURLRepository()
	if err != nil {
		return err
	}
	defer repo.DB.Close()

	// Инициализация сервисов
	shURLService := services.NewShURLService(repo)

	// Инициализация обработчиков
	shURLHandler := handlers.NewShURLHandler(shURLService, flagShURLBaseAddr)

	// Если адрес один - запускаем то и то на одном порту
	// if flagShortenerAddr == flagShURLBaseAddr {
	r := chi.NewRouter()
	r.Get("/{token}", shURLHandler.GetFullURL)
	r.Post("/", shURLHandler.ShortenURL)
	return http.ListenAndServe(flagShortenerAddr, r)
	// }

	// // Если разные - разные сервера для разных хэндлеров
	// shURLBase := chi.NewRouter()
	// shURLBase.Get("/{token}", shURLHandler.GetFullURL)
	// fmt.Println("Running short-to-long redirect server on", flagShURLBaseAddr)
	// err = http.ListenAndServe(flagShURLBaseAddr, shURLBase)
	// if err != nil {
	// 	return err
	// }

	// shortener := chi.NewRouter()
	// shortener.Post("/", shURLHandler.ShortenURL)
	// fmt.Println("Running URL shortener on", flagShortenerAddr)
	// err = http.ListenAndServe(flagShortenerAddr, shortener)
	// if err != nil {
	// 	return err
	// }

	// return nil
}
