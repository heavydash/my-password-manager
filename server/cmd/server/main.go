package main

import (
	"context"
	"database/sql"
	"github.com/gin-gonic/gin"
	"gophkeeper/server/internal/adapters/auth"
	"gophkeeper/server/internal/adapters/protocol/rest/handler"
	"gophkeeper/server/internal/adapters/protocol/rest/middleware"
	"gophkeeper/server/internal/adapters/secret"
	"gophkeeper/server/internal/adapters/storage/postgres"
	"gophkeeper/server/internal/config"
	"gophkeeper/server/internal/domain"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("GophKeeper Server starting...")

	// Загружаем и валидируем конфиг
	cfg := config.Load()

	// Подключаемся к БД через конфиг
	db, err := sql.Open("postgres", cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Проверка работоспособности
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to database")

	// Создаем Storage
	userStorage := postgres.NewStorage(db)
	secretStorage := postgres.NewSecretStorage()

	// Создаем адаптеры
	passwordAdapter := auth.NewPasswordAdapter(userStorage, cfg.JWT.Secret)
	secretAdapter := secret.NewSecretAdapter(secretStorage)

	// UseCases
	authUserCase := domain.NewAuthUseCase(passwordAdapter)
	secretUseCase := domain.NewSecretUseCase(secretAdapter)

	_ = secretUseCase

	log.Println("Auth User Case and password adapter initialized")
	log.Println("Secret Use Case and secret adapter initialized")

	// Настройка Gin
	if gin.Mode() == gin.DebugMode {
		log.Println("Running in DEBUG mode. Use GIN_MODE=release for production")
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Роуты
	r := gin.Default()

	// Публичные роуты
	pub := r.Group("/")
	pub.POST("/register", handler.Register(authUserCase))
	pub.POST("/login", handler.Login(authUserCase))

	// Защищенные роуты
	protect := r.Group("/")
	protect.Use(middleware.AuthMiddleware(passwordAdapter.ValidateJWT))
	protect.GET("/profile", handler.Profile())

	protect.POST("/secrets", handler.CreateSecret(secretUseCase))
	protect.GET("/secrets", handler.GetSecrets(secretUseCase))

	// HTTP + pprof
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	pprofSrv := &http.Server{
		Addr:    ":" + cfg.Pprof.Port,
		Handler: nil,
	}

	log.Printf("REST listening on: http://localhost:%s", cfg.Server.Port)
	log.Printf("pprof available at http://localhost:%s/debug/pprof", cfg.Pprof.Port)

	// Запуск основного сервера
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Запуск Pprof
	go func() {
		if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("pprof server error: %s", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Println("Server forced to shutdown:", err)
	}

	if err := pprofSrv.Shutdown(ctx); err != nil {
		log.Println("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
