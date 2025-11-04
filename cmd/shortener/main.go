// Пакет Main
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/JustScorpio/urlshortener/internal/grpc/gen"
	grpchandlers "github.com/JustScorpio/urlshortener/internal/grpc/handlers"
	grpcauth "github.com/JustScorpio/urlshortener/internal/grpc/middleware/auth"
	grpclogger "github.com/JustScorpio/urlshortener/internal/grpc/middleware/logger"
	grpcwhitelist "github.com/JustScorpio/urlshortener/internal/grpc/middleware/whitelist"
	"google.golang.org/grpc"

	"github.com/JustScorpio/urlshortener/internal/http/handlers"
	"github.com/JustScorpio/urlshortener/internal/http/middleware/auth"
	"github.com/JustScorpio/urlshortener/internal/http/middleware/gzipencoder"
	"github.com/JustScorpio/urlshortener/internal/http/middleware/logger"
	"github.com/JustScorpio/urlshortener/internal/http/middleware/whitelist"
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
	//Проверяем указан ли конфигурационный файл.
	if envConfigPath, hasEnv := os.LookupEnv("CONFIG"); hasEnv {
		flagConfigPath = envConfigPath
	}

	//Заполняем параметры из конфига (но приоритет всё равно за переменными окружения)
	if flagConfigPath != "" {
		err := parseAppConfig(flagConfigPath)
		if err != nil {
			log.Fatal(err)
		}
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

	//При наличии переменной окружения или наличии флага - запускаем на HTTPS.
	if _, hasEnv := os.LookupEnv("ENABLE_HTTPS"); hasEnv {
		flagEnableHTTPS = true
	}

	//Сертификат для HTTPS (общий при разных flagShortenerRouterAddr и flagRedirectRouterAddr)
	var tlsConfig *tls.Config
	if flagEnableHTTPS {
		tlsConfig, err = GetTestTLSConfig()
		if err != nil {
			return err
		}
	}

	// Инициализация обработчиков
	shURLHandler := handlers.NewShURLHandler(shURLService, flagRedirectRouterAddr, flagEnableHTTPS)

	// Инициализация gRPC обработчиков
	grpcHandler := grpchandlers.NewGRPCHandler(shURLService, flagRedirectRouterAddr)

	// Инициализация логгера
	zapLogger, err := logger.NewLogger("Info", true)
	if err != nil {
		return err
	}
	defer zapLogger.Sync()

	//Инициализация subnet whitelist
	if trustedSubnet, hasEnv := os.LookupEnv("TRUSTED_SUBNET"); hasEnv {
		flagTrustedSubnet = trustedSubnet
	}
	cidrWhiteList, err := whitelist.NewCIDRWhitelistMiddleware(flagTrustedSubnet)
	if err != nil {
		return err
	}
	// Инициализация gRPC whitelist middleware
	grpcCIDRWhiteList, err := grpcwhitelist.NewCIDRWhitelistMiddleware(flagTrustedSubnet)
	if err != nil {
		return err
	}

	// Берём адрес сервера из переменной окружения. Иначе - из аргумента
	if envServerAddr, hasEnv := os.LookupEnv("SERVER_ADDRESS"); hasEnv {
		flagShortenerRouterAddr = normalizeHTTPAddress(envServerAddr)
	}

	// Берём адрес gRPC сервера из переменной окружения
	if envGRPCAddr, hasEnv := os.LookupEnv("GRPC_SERVER_ADDRESS"); hasEnv {
		flagGRPCRouterAddr = normalizeGRPCAddress(envGRPCAddr)
	}

	// Проверка подключения к БД
	pingFunc := func(w http.ResponseWriter, r *http.Request) {
		if repo.PingDB() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}

	// Канал для получения сигналов ОС
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)

	// Инициализация gRPC сервера с middleware
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpclogger.GRPCLoggingMiddleware(zapLogger),
			grpcauth.GRPCAuthMiddleware(),
			grpcCIDRWhiteList.CIDRWhitelistMiddleware("/urlshortener.URLShortener/GetStats"),
		),
	)

	// Регистрируем gRPC сервис
	gen.RegisterURLShortenerServer(grpcServer, grpcHandler)

	// Запуск gRPC сервера
	grpcListener, err := net.Listen("tcp", flagGRPCRouterAddr)
	if err != nil {
		return fmt.Errorf("failed to listen gRPC: %w", err)
	}
	defer grpcListener.Close()

	// Запускаем gRPC сервер в горутине
	grpcServerErr := make(chan error, 1)
	go func() {
		fmt.Printf("Running gRPC server on %s\n", flagGRPCRouterAddr)
		if err := grpcServer.Serve(grpcListener); err != nil && err != grpc.ErrServerStopped {
			grpcServerErr <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

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
		r.With(cidrWhiteList.CIDRWhitelistMiddleware()).Get("/api/internal/stats", shURLHandler.GetStats)

		server := createHttpServer(flagShortenerRouterAddr, r, tlsConfig)

		// Запуск сервера в горутине
		serverErr := make(chan error, 1)
		go func() {
			serverErr <- runHttpServer(server)
		}()

		// Ожидание сигнала остановки или ошибки сервера
		select {
		case <-stop:
			fmt.Println("Received shutdown signal")
		case err := <-serverErr:
			fmt.Printf("Server error: %v\n", err)
		case err := <-grpcServerErr:
			fmt.Printf("gRPC server error: %v\n", err)
		}

		return gracefulShutdown(shURLService, server, grpcServer)
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
	shortenerRouter.With(cidrWhiteList.CIDRWhitelistMiddleware()).Get("/api/internal/stats", shURLHandler.GetStats)

	// Создаем серверы
	redirectServer := createHttpServer(flagRedirectRouterAddr, redirectRouter, tlsConfig)
	shortenerServer := createHttpServer(flagShortenerRouterAddr, shortenerRouter, tlsConfig)

	// Запуск серверов в горутинах
	serverErr := make(chan error, 2)

	go func() {
		serverErr <- runHttpServer(redirectServer)
	}()

	go func() {
		serverErr <- runHttpServer(shortenerServer)
	}()

	// Ожидание сигнала остановки или ошибки сервера
	select {
	case <-stop:
		fmt.Println("Received shutdown signal")
	case err := <-serverErr:
		fmt.Printf("Server error: %v\n", err)
	case err := <-grpcServerErr:
		fmt.Printf("gRPC server error: %v\n", err)
	}

	return gracefulShutdown(shURLService, redirectServer, shortenerServer, grpcServer)
}

// createServer - создает и настраивает HTTP сервер
func createHttpServer(addr string, handler http.Handler, tlsConfig *tls.Config) *http.Server {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	if tlsConfig != nil {
		server.TLSConfig = tlsConfig
	}

	return server
}

// runServer - запускает сервер в горутине и возвращает канал с ошибкой
func runHttpServer(server *http.Server) error {
	fmt.Printf("Running server on %s\n", server.Addr)
	return server.ListenAndServe()
}

// gracefulShutdown - graceful shutdown приложения
func gracefulShutdown(service *services.ShURLService, servers ...interface{}) error {
	fmt.Println("Starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем серверы
	for i, server := range servers {
		switch s := server.(type) {
		case *http.Server:
			if s != nil {
				if err := s.Shutdown(ctx); err != nil {
					fmt.Printf("HTTP server %d shutdown error: %v\n", i, err)
				} else {
					fmt.Printf("HTTP server %d stopped\n", i)
				}
			}
		case *grpc.Server:
			if s != nil {
				fmt.Printf("Stopping gRPC server...\n")
				s.GracefulStop()
				fmt.Printf("gRPC server stopped\n")
			}
		}
	}

	// Останавливаем сервис
	service.Shutdown()
	fmt.Println("Service shutdown completed")

	fmt.Println("Graceful shutdown finished")
	return nil
}
