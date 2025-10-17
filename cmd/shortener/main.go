// Пакет Main
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

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

	//ЗАполняем параметры из конфига (но приоритет всё равно за переменными окружения)
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
		fmt.Println("Running server on", flagShortenerRouterAddr)

		if flagEnableHTTPS {
			return http.ListenAndServeTLS(flagShortenerRouterAddr, string(cert), string(privateKey), r)
		} else {
			return http.ListenAndServe(flagShortenerRouterAddr, r)
		}
	}

	// Если разные - разные сервера для разных хэндлеров в разных горутинах
	redirectRouter := chi.NewRouter()
	redirectRouter.Use(auth.AuthMiddleware()) //Нужно при обращении к /api/user/urls (GET и DELETE)
	redirectRouter.Use(logger.LoggingMiddleware(zapLogger))
	redirectRouter.Use(gzipencoder.GZIPEncodingMiddleware())
	redirectRouter.Get("/ping", pingFunc) //Дублируется в обоих роутерах
	redirectRouter.Get("/api/user/urls", shURLHandler.GetShURLsByUserID)
	redirectRouter.Delete("/api/user/urls", shURLHandler.DeleteMany)
	redirectRouter.Get("/{token}", shURLHandler.GetFullURL)

	shortenerRouter := chi.NewRouter()
	shortenerRouter.Use(logger.LoggingMiddleware(zapLogger))
	shortenerRouter.Use(gzipencoder.GZIPEncodingMiddleware())
	shortenerRouter.Post("/api/shorten", shURLHandler.ShortenURL)
	shortenerRouter.Post("/api/shorten/batch", shURLHandler.ShortenURLsBatch)
	redirectRouter.Get("/ping", pingFunc) //Дублируется в обоих роутерах
	shortenerRouter.Post("/", shURLHandler.ShortenURL)

	errCh := make(chan error)

	go func() {
		fmt.Println("Running short-to-long redirect server on", flagRedirectRouterAddr)
		if flagEnableHTTPS {
			errCh <- http.ListenAndServeTLS(flagRedirectRouterAddr, string(cert), string(privateKey), redirectRouter)
		} else {
			errCh <- http.ListenAndServe(flagRedirectRouterAddr, redirectRouter)
		}
	}()

	go func() {
		fmt.Println("Running URL shortener on", flagShortenerRouterAddr)
		if flagEnableHTTPS {
			errCh <- http.ListenAndServeTLS(flagShortenerRouterAddr, string(cert), string(privateKey), shortenerRouter)
		} else {
			errCh <- http.ListenAndServe(flagShortenerRouterAddr, shortenerRouter)
		}
	}()

	// Блокируем основную горутину и обрабатываем ошибки
	return <-errCh
}
