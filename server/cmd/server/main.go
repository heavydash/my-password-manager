package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"gophkeeper/server/internal/adapters/auth"
	"gophkeeper/server/internal/adapters/protocol/rest/handler"
	"gophkeeper/server/internal/adapters/protocol/rest/middleware"
	"gophkeeper/server/internal/adapters/secret"
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
// Пример сборки:
//
//	go build -ldflags "-X main.buildVersion=1.0.0 -X main.buildDate=$(date -u +%Y-%m-%d) -X main.buildCommit=$(git rev-parse --short HEAD)" -o shortener ./cmd/shortener
var (
	buildVersion string // Версия сборки
	buildDate    string // Дата сборки
	buildCommit  string // Хэш коммита
)

func main() {
	// Информация о сборке
	fmt.Printf("Build version: %s\n", valueOrNA(buildVersion))
	fmt.Printf("Build date:    %s\n", valueOrNA(buildDate))
	fmt.Printf("Build commit:  %s\n", valueOrNA(buildCommit))
	fmt.Println("---")

	// Загружаем и валидируем конфиг
	cfg, err := config.New()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Создаём логгер
	log, err := logger.New(cfg)
	if err != nil {
		log.Error("Failed to create logger: %v", zap.Error(err))
		os.Exit(1)
	}
	defer func() {
		if err := log.Sync(); err != nil {
			panic(err)
		}
	}()

	log.Info("GophKeeper Server starting...",
		zap.String("version", buildVersion),
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
	)

	// Подключаемся к БД через конфиг
	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		log.Error("Failed to connect to database: %v", zap.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	// Проверка работоспособности
	if err := db.Ping(); err != nil {
		log.Error("Failed to ping database: %v", zap.Error(err))
		os.Exit(1)
	}
	log.Info("Successfully connected to database")

	// Запуск миграций
	log.Info("running database migrations...")
	if err := migrations.RunMigrations(cfg.Database.DSN); err != nil {
		log.Error("Failed to run migrations: %v", zap.Error(err))
		os.Exit(1)
	}
	log.Info("migrations completed")

	// Создаем Storage
	userRepository := postgres.NewUserRepository(db, log)
	secretRepository := postgres.NewSecretRepository(db, log)

	// Создаем адаптеры
	passwordAdapter := auth.NewPasswordAdapter(userRepository, cfg.JWT.Secret, log)
	secretAdapter := secret.NewSecretAdapter(secretRepository, log)

	// UseCases
	authUserCase := domain.NewAuthUseCase(passwordAdapter, log)
	secretUseCase := domain.NewSecretUseCase(secretAdapter)

	log.Info("All layers initialized successfully")

	// Настройка Gin
	if cfg.Server.Debug {
		gin.SetMode(gin.DebugMode)
		log.Info("Running in DEBUG mode")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Роуты
	r := gin.Default()

	// Публичные роуты
	pub := r.Group("/")
	pub.POST("/register", handler.Register(authUserCase, log))
	pub.POST("/login", handler.Login(authUserCase, log))

	// Защищенные роуты
	protect := r.Group("/")
	protect.Use(middleware.AuthMiddleware(passwordAdapter.ValidateJWT, log))
	protect.GET("/profile", handler.Profile(log))

	protect.POST("/secrets", handler.CreateSecret(secretUseCase, log))
	protect.GET("/secrets", handler.GetSecrets(secretUseCase, log))
	protect.GET("/secrets/:id", handler.GetSecret(secretUseCase, log))
	protect.DELETE("/secrets/:id", handler.DeleteSecret(secretUseCase, log))

	// HTTP + pprof
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	pprofSrv := &http.Server{
		Addr:    ":" + cfg.Pprof.Port,
		Handler: nil,
	}

	log.Info("HTTP server started",
		zap.String("address", srv.Addr),
		zap.String("url", "http://localhost"+srv.Addr),
	)

	log.Info("pprof server started",
		zap.String("address", pprofSrv.Addr),
		zap.String("url", "http://localhost"+pprofSrv.Addr+"/debug/pprof"),
	)

	// Запуск основного сервера
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("HTTP server failed", zap.Error(err))
		}
	}()

	// Запуск Pprof
	go func() {
		if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Warn("pprof server stopped", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {

		log.Error("Server forced to shutdown", zap.Error(err))
	}

	if err := pprofSrv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", zap.Error(err))
	}

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
