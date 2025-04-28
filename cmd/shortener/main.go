package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

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
	shURLHandler := handlers.NewShURLHandler(shURLService, flagRedirectRouterAddr)

	// Если адрес один - запускаем то и то на одном порту
	if normalizeAddress(flagShortenerRouterAddr) == normalizeAddress(flagRedirectRouterAddr) {
		r := chi.NewRouter()
		r.Get("/{token}", shURLHandler.GetFullURL)
		r.Post("/", shURLHandler.ShortenURL)
		fmt.Println("Running server on", flagShortenerRouterAddr)
		return http.ListenAndServe(flagShortenerRouterAddr, r)
	}

	// Если разные - разные сервера для разных хэндлеров в разных горутинах
	redirectRouter := chi.NewRouter()
	redirectRouter.Get("/{token}", shURLHandler.GetFullURL)
	redirectServer := &http.Server{
		Addr:    flagRedirectRouterAddr,
		Handler: redirectRouter,
	}

	shortenerRouter := chi.NewRouter()
	shortenerRouter.Post("/", shURLHandler.ShortenURL)
	shortenerServer := &http.Server{
		Addr:    flagShortenerRouterAddr,
		Handler: shortenerRouter,
	}

	errCh := make(chan error)

	go func() {
		fmt.Println("Running short-to-long redirect server on", flagRedirectRouterAddr)
		errCh <- redirectServer.ListenAndServe()
	}()

	go func() {
		fmt.Println("Running URL shortener on", flagShortenerRouterAddr)
		errCh <- shortenerServer.ListenAndServe()
	}()

	// Блокируем основную горутину и обрабатываем ошибки
	return <-errCh
}

// Нормализация адресов
func normalizeAddress(addr string) string {

	// Добавляем порт, если его нет
	if !strings.Contains(addr, ":") {
		addr += ":8080"
	}

	// Убираем 127.0.0.1 и localhost
	if strings.HasPrefix(addr, "127.0.0.1:") {
		addr = strings.Replace(addr, "127.0.0.1", "", 1)
	}
	if strings.HasPrefix(addr, "localhost:") {
		addr = strings.Replace(addr, "localhost", "", 1)
	}

	return addr
}
