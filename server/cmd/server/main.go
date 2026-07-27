// Package main implements the GophKeeper server entry point.
//
// Выполняет:
//   - Загрузку и валидацию конфигурации
//   - Инициализацию логгера
//   - Подключение к PostgreSQL и запуск миграций
//   - Инициализацию репозиториев, адаптеров и usecases
//   - Запуск HTTP (Gin), gRPC и pprof серверов
//   - Graceful shutdown при получении SIGINT/SIGTERM
package main

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"gophkeeper/server/internal/adapters/auth"
	"gophkeeper/server/internal/adapters/protocol/grpc"
	"gophkeeper/server/internal/adapters/protocol/rest"
	"gophkeeper/server/internal/adapters/protocol/rest/handler"
	"gophkeeper/server/internal/adapters/secret"
	"gophkeeper/server/internal/adapters/storage"
	"gophkeeper/server/internal/adapters/storage/postgres"
	"gophkeeper/server/internal/config"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/migrations"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Переменные версии заполняются при сборке через ldflags.
//
// Пример сборки:
//
//	go build -ldflags "-X main.buildVersion=1.0.0 -X main.buildDate=$(date -u +%Y-%m-%d) -X main.buildCommit=$(git rev-parse --short HEAD)" -o gophkeeper ./cmd/server
var (
	buildVersion string // Версия сборки
	buildDate    string // Дата сборки
	buildCommit  string // Хэш коммита
)

func main() {
	// Вывод информации о сборке
	fmt.Printf("Build version: %s\n", valueOrNA(buildVersion))
	fmt.Printf("Build date:    %s\n", valueOrNA(buildDate))
	fmt.Printf("Build commit:  %s\n", valueOrNA(buildCommit))
	fmt.Println("---")

	// Загрузка и валидация конфигурации из флагов, env, JSON
	cfg, err := config.New()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Инициализация логгера с уровнем из конфига
	log, err := logger.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create logger: %v", zap.Error(err))
		os.Exit(1)
	}
	defer log.Sync()

	log.Info("GophKeeper Server starting...",
		zap.String("version", buildVersion),
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
	)

	// Создание контекста с таймаутом для инициализации БД
	// Таймаут берётся из конфига (InitTimeout)
	initCtx, initCancel := context.WithTimeout(context.Background(), cfg.Server.InitTimeout)
	defer initCancel()

	// Создание пула соединений с PostgreSQL
	dbPool, err := postgres.NewPool(initCtx, cfg, log)
	if err != nil {
		log.Error("Failed to create connection pool", zap.Error(err))
		os.Exit(1)
	}
	defer dbPool.Close()

	log.Info("Successfully connected to database (pool)")

	// Запуск миграций с поддержкой отмены через контекст
	log.Info("running database migrations...")
	if err := migrations.RunMigrations(initCtx, cfg); err != nil {
		log.Error("Failed to run migrations: %v", zap.Error(err))
		os.Exit(1)
	}
	log.Info("migrations completed")

	// Инициализация репозиториев с передачей контекста
	secretRepository, err := storage.NewSecretRepository(initCtx, cfg, log)
	if err != nil {
		log.Error("Failed to create secret repository", zap.Error(err))
		os.Exit(1)
	}
	userRepository, err := storage.NewUserRepository(initCtx, cfg, log)
	if err != nil {
		log.Error("Failed to create user repository", zap.Error(err))
		os.Exit(1)
	}

	log.Info("Repositories initialized successfully", zap.String("secret_repo_type", "auto (postgres/memory)"))

	// Создание адаптеров для аутентификации и работы с секретами
	passwordAdapter := auth.NewPasswordAdapter(userRepository, cfg, log)
	oauthAdapter := auth.NewOAuthAdapter(cfg.OAuth, dbPool.Pool, passwordAdapter, log)
	secretAdapter := secret.NewSecretAdapter(secretRepository, log)

	// Инициализация бизнес-логики (use cases)
	authUserCase := domain.NewAuthUseCase(oauthAdapter, log)
	if authUserCase == nil {
		log.Fatal("failed to create AuthUseCase")
	}
	secretUseCase := domain.NewSecretUseCase(secretAdapter, domain.JWTValidatorAdapter{
		ValidateFunc: passwordAdapter.ValidateJWT,
	})
	if secretUseCase == nil {
		log.Fatal("failed to create SecretUseCase")
	}

	log.Info("All layers initialized successfully")

	// Настройка режима Gin (debug/release)
	if cfg.Server.Debug {
		gin.SetMode(gin.DebugMode)
		log.Info("Running in DEBUG mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Настройка HTTP роутера с middleware и хендлерами
	r := rest.NewRouter(
		authUserCase,
		secretUseCase,
		handler.NewOAuthHandler(authUserCase, log),
		passwordAdapter.ValidateJWT,
		log,
	)

	// Конфигурация HTTP сервера
	srv := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        r,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Конфигурация pprof сервера (для профилирования)
	pprofSrv := &http.Server{
		Addr:    ":" + cfg.Pprof.Port,
		Handler: nil,
	}

	serverURL := fmt.Sprintf("http://%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Info("HTTP server started",
		zap.String("address", srv.Addr),
		zap.String("url", serverURL),
	)

	log.Info("pprof server started",
		zap.String("address", pprofSrv.Addr),
		zap.String("url", "http://"+cfg.Server.Host+":"+cfg.Pprof.Port+"/debug/pprof"),
	)

	// Запуск HTTP сервера в горутине
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", zap.Error(err))
		}
	}()

	// Запуск pprof сервера в горутине
	go func() {
		if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Warn("pprof server stopped", zap.Error(err))
		}
	}()

	grpcAddr := ":" + cfg.Server.GRPCPort
	grpcServer := grpc.NewServer(authUserCase, secretUseCase, log)

	// Запуск gRPC сервера
	go func() {
		if err := grpcServer.Start(grpcAddr); err != nil {
			log.Error("gRPC server failed", zap.Error(err))
		}
	}()

	log.Info("gRPC server started",
		zap.String("address", grpcAddr),
	)

	// Ожидание сигнала завершения \
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Shutting down server...")

	// Graceful shutdown с таймаутом из конфига
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Остановка gRPC сервера
	grpcServer.GracefulStop()

	// Остановка HTTP сервера
	if err := srv.Shutdown(shutdownCtx); err != nil {

		log.Error("HTTP shutdown failed", zap.Error(err))
	}

	// Остановка pprof сервера
	if err := pprofSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("Pprof Server forced to shutdown", zap.Error(err))
	}

	// Закрытие соединения с БД
	dbPool.Close()

	log.Info("Server exited gracefully")
}

// valueOrNA возвращает переданную строку или "N/A", если строка пустая.
//
// Используется для форматированного вывода информации о сборке.
// Если переменные версии не были заданы при сборке, выводится "N/A".
//
// Параметры:
//   - s: строка для проверки
//
// Возвращает:
//   - исходную строку, если она не пустая
//   - "N/A", если строка пустая
func valueOrNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}
