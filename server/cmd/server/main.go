package main

import (
	"context"
	"database/sql"
	"github.com/joho/godotenv"
	"gophkeeper/server/internal/adapters/auth"
	"gophkeeper/server/internal/adapters/storage/postgres"
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

	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found, using system environment variables")
	}

	// Читаем DSN
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		log.Fatal("DB_DSN environment variable is not set. Check your .env file")
	}

	// Подключаемся к БД
	db, err := sql.Open("postgres", dbDSN)
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
	storage := postgres.NewStorage(db)

	// Auth
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "change-in-prod"
	}

	passwordAdapter := auth.NewPasswordAdapter(storage, jwtSecret)
	authUserCase := domain.NewAuthUseCase(passwordAdapter)

	// Тестовая регистрация
	ctx := context.Background()
	userID, err := authUserCase.Register(ctx, "newuser@example.com", "pass123")
	if err != nil {
		log.Printf("Failed to register user: %v", err)
	} else {
		log.Printf("User registered with ID: %v", userID)
	}

	log.Println("Auth User Case and password adapter initialized")

	// HTTP + pprof
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	pprofSrv := &http.Server{
		Addr:    ":6060",
		Handler: nil,
	}
	go func() {
		log.Println("REST listening on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	go func() {
		log.Println("pprof available at :6060/debug/pprof")
		if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
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
