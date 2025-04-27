package main

import (
	"fmt"
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
		return err
	}
	defer repo.DB.Close()

	// Инициализация сервисов
	shURLService := services.NewShURLService(repo)

	// Инициализация обработчиков
	shURLHandler := handlers.NewShURLHandler(shURLService, flagShURLBaseAddr)

	// Если адрес один - запускаем то и то на одном порту
	if flagRunAddr == flagShURLBaseAddr {
		r := chi.NewRouter()
		r.HandleFunc("/{token}", shURLHandler.GetFullURL)
		r.HandleFunc("/", shURLHandler.ShortenURL)
		return http.ListenAndServe(flagRunAddr, r)
	}

	// Если разные - разные сервера для разных хэндлеров
	fullURLGetter := chi.NewRouter()
	fullURLGetter.HandleFunc("/{token}", shURLHandler.GetFullURL)
	fmt.Println("Running short-to-long redirect server on", flagRunAddr)
	err = http.ListenAndServe(flagRunAddr, fullURLGetter)
	if err != nil {
		return err
	}

	shortener := chi.NewRouter()
	shortener.HandleFunc("/", shURLHandler.ShortenURL)
	fmt.Println("Running URL shortener on", flagRunAddr)
	err = http.ListenAndServe(flagRunAddr, shortener)
	if err != nil {
		return err
	}

	return nil
}
