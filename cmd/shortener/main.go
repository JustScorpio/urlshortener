// Пакет Main
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JustScorpio/urlshortener/internal/handlers"
	"github.com/JustScorpio/urlshortener/internal/middleware/auth"
	"github.com/JustScorpio/urlshortener/internal/middleware/gzipencoder"
	"github.com/JustScorpio/urlshortener/internal/middleware/logger"
	"github.com/JustScorpio/urlshortener/internal/models/entities"
	"github.com/JustScorpio/urlshortener/internal/repository"
	"github.com/JustScorpio/urlshortener/internal/repository/jsonfile"
	"github.com/JustScorpio/urlshortener/internal/repository/postgres"
	"github.com/JustScorpio/urlshortener/internal/services"

	_ "net/http/pprof"

	"github.com/go-chi/chi"
)

var (
	// build-переменные заполняемые с помощью ldflags -X
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

// main - вызывается автоматически при запуске приложения
func main() {
	// вывести аргументы
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)

	// обрабатываем аргументы командной строки
	parseFlags()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run - функция полезна при инициализации зависимостей сервера перед запуском
// Приоритет конфигурации: Переменные окружения > Конфиг > Флаги
func run() error {
	//Проверям указан ли конфигурационный файл.
	if envConfigPath, hasEnv := os.LookupEnv("CONFIG"); hasEnv {
		flagConfigPath = envConfigPath
	}

	//Заполняем параметры из конфига (но приоритет всё равно за переменными окружения)
	if flagConfigPath != "" {
		parseAppConfig(flagConfigPath)
	}

	//Для jsonfile-базы данных берём расположение файла БД из переменной окружения. Иначе - из аргумента
	if envDBAddr, hasEnv := os.LookupEnv("FILE_STORAGE_PATH"); hasEnv {
		flagDBFilePath = envDBAddr
	}

	//Для postgresql-базы данных берём строку подключения к БД из переменной окружения. Иначе - из аргумента.
	//Если и то и то пусто - берём базу на основе json-файла
	if envDBConnStr, hasEnv := os.LookupEnv("DATABASE_DSN"); hasEnv {
		flagDBConnStr = envDBConnStr
	}

	// Инициализация репозиториев с базой данных
	var repo repository.IRepository[entities.ShURL]
	var err error
	if flagDBConnStr != "" {
		repo, err = postgres.NewPostgresShURLRepository(flagDBConnStr)
	} else {
		repo, err = jsonfile.NewJSONFileShURLRepository(flagDBFilePath)
	}

	if err != nil {
		return err
	}

	defer repo.CloseConnection()

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

	// Проверка подключения к БД
	pingFunc := func(w http.ResponseWriter, r *http.Request) {
		if repo.PingDB() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}

	//При наличии переменной окружения или наличии флага - запускаем на HTTPS.
	if _, hasEnv := os.LookupEnv("ENABLE_HTTPS"); hasEnv {
		flagEnableHTTPS = true
	}

	//Сертификат для HTTPS (общий при разных flagShortenerRouterAddr и flagRedirectRouterAddr)
	var cert, privateKey []byte
	if flagEnableHTTPS {
		cert, privateKey, err = GetTestCert()
		if err != nil {
			return err
		}
	}

	// Канал для получения сигналов ОС
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// Сравниваем нормализованные адреса. Если адрес один - запускаем то и то на одном порту
	if flagShortenerRouterAddr == flagRedirectRouterAddr {
		r := chi.NewRouter()
		r.Use(auth.AuthMiddleware())
		r.Use(logger.LoggingMiddleware(zapLogger))
		r.Use(gzipencoder.GZIPEncodingMiddleware())
		r.Get("/ping", pingFunc)
		r.Get("/api/user/urls", shURLHandler.GetShURLsByUserID)
		r.Delete("/api/user/urls", shURLHandler.DeleteMany)
		r.Get("/{token}", shURLHandler.GetFullURL)
		r.Post("/api/shorten", shURLHandler.ShortenURL)
		r.Post("/api/shorten/batch", shURLHandler.ShortenURLsBatch)
		r.Post("/", shURLHandler.ShortenURL)

		server := &http.Server{
			Addr:    flagShortenerRouterAddr,
			Handler: r,
		}

		// Запуск сервера в горутине
		serverErr := make(chan error, 1)
		go func() {
			fmt.Println("Running server on", flagShortenerRouterAddr)
			if flagEnableHTTPS {
				serverErr <- server.ListenAndServeTLS(string(cert), string(privateKey))
			} else {
				serverErr <- server.ListenAndServe()
			}
		}()

		// Ожидание сигнала остановки или ошибки сервера
		select {
		case <-stop:
			fmt.Println("Received shutdown signal")
		case err := <-serverErr:
			fmt.Printf("Server error: %v\n", err)
		}

		return gracefulShutdown(shURLService, server)
	}

	// Если разные - разные сервера для разных хэндлеров в разных горутинах
	redirectRouter := chi.NewRouter()
	redirectRouter.Use(auth.AuthMiddleware()) //Нужно при обращении к /api/user/urls (GET и DELETE)
	redirectRouter.Use(logger.LoggingMiddleware(zapLogger))
	redirectRouter.Use(gzipencoder.GZIPEncodingMiddleware())
	redirectRouter.Get("/ping", pingFunc)
	redirectRouter.Get("/api/user/urls", shURLHandler.GetShURLsByUserID)
	redirectRouter.Delete("/api/user/urls", shURLHandler.DeleteMany)
	redirectRouter.Get("/{token}", shURLHandler.GetFullURL)

	shortenerRouter := chi.NewRouter()
	shortenerRouter.Use(logger.LoggingMiddleware(zapLogger))
	shortenerRouter.Use(gzipencoder.GZIPEncodingMiddleware())
	shortenerRouter.Post("/api/shorten", shURLHandler.ShortenURL)
	shortenerRouter.Post("/api/shorten/batch", shURLHandler.ShortenURLsBatch)
	shortenerRouter.Get("/ping", pingFunc)
	shortenerRouter.Post("/", shURLHandler.ShortenURL)

	// Создаем серверы
	redirectServer := &http.Server{
		Addr:    flagRedirectRouterAddr,
		Handler: redirectRouter,
	}

	shortenerServer := &http.Server{
		Addr:    flagShortenerRouterAddr,
		Handler: shortenerRouter,
	}

	// Запуск серверов в горутинах
	serverErr := make(chan error, 2)

	go func() {
		fmt.Println("Running short-to-long redirect server on", flagRedirectRouterAddr)
		if flagEnableHTTPS {
			serverErr <- redirectServer.ListenAndServeTLS(string(cert), string(privateKey))
		} else {
			serverErr <- redirectServer.ListenAndServe()
		}
	}()

	go func() {
		fmt.Println("Running URL shortener on", flagShortenerRouterAddr)
		if flagEnableHTTPS {
			serverErr <- shortenerServer.ListenAndServeTLS(string(cert), string(privateKey))
		} else {
			serverErr <- shortenerServer.ListenAndServe()
		}
	}()

	// Ожидание сигнала остановки или ошибки сервера
	select {
	case <-stop:
		fmt.Println("Received shutdown signal")
	case err := <-serverErr:
		fmt.Printf("Server error: %v\n", err)
	}

	return gracefulShutdown(shURLService, redirectServer, shortenerServer)
}

// gracefulShutdown - graceful shutdown для одного сервера
func gracefulShutdown(service *services.ShURLService, servers ...*http.Server) error {
	fmt.Println("Starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем HTTP сервера
	for i, server := range servers {
		if server != nil {
			if err := server.Shutdown(ctx); err != nil {
				fmt.Printf("Server %d shutdown error: %v\n", i, err)
			} else {
				fmt.Printf("Server %d stopped\n", i)
			}
		}
	}

	// Останавливаем сервис
	service.Shutdown()
	fmt.Println("Service shutdown completed")

	fmt.Println("Graceful shutdown finished")
	return nil
}
