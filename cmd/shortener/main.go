package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/JustScorpio/urlshortener/internal/handlers"
	"github.com/JustScorpio/urlshortener/internal/middleware/jsonpacker"
	"github.com/JustScorpio/urlshortener/internal/middleware/logger"
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
	shURLHandler := handlers.NewShURLHandler(shURLService, flagRedirectRouterAddr)

	//Инициализация логгера
	zapLogger, err := logger.NewLogger("Info", true)
	if err != nil {
		return err
	}
	defer zapLogger.Sync()

	// Берём адрес сервера из переменной окружения. Иначе - из аргумента
	if envServerAddr, hasEnv := os.LookupEnv("SERVER_ADDRESS"); hasEnv {
		flagShortenerRouterAddr = normalizeAddress(envServerAddr)
	}

	// Сравниваем нормализованные адреса. Если адрес один - запускаем то и то на одном порту
	if flagShortenerRouterAddr == flagRedirectRouterAddr {
		r := chi.NewRouter()
		r.Use(logger.LoggingMiddleware(zapLogger))
		r.Get("/{token}", shURLHandler.GetFullURL)
		r.With(jsonpacker.JsonPackingMiddleware()).Post("/api/shortener", shURLHandler.ShortenURL)
		r.Post("/", shURLHandler.ShortenURL)
		fmt.Println("Running server on", flagShortenerRouterAddr)
		return http.ListenAndServe(flagShortenerRouterAddr, r)
	}

	// Если разные - разные сервера для разных хэндлеров в разных горутинах
	redirectRouter := chi.NewRouter()
	redirectRouter.Get("/{token}", shURLHandler.GetFullURL)

	shortenerRouter := chi.NewRouter()
	shortenerRouter.Post("/", shURLHandler.ShortenURL)

	errCh := make(chan error)

	go func() {
		fmt.Println("Running short-to-long redirect server on", flagRedirectRouterAddr)
		errCh <- http.ListenAndServe(flagRedirectRouterAddr, redirectRouter)
	}()

	go func() {
		fmt.Println("Running URL shortener on", flagShortenerRouterAddr)
		errCh <- http.ListenAndServe(flagShortenerRouterAddr, shortenerRouter)
	}()

	// Блокируем основную горутину и обрабатываем ошибки
	return <-errCh
}
