package main

import (
	"context"
	"gophkeeper/server/internal/adapters/auth"
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

	// Auth
	passwordAdapter := auth.NewPasswordAdapter()
	authUserCase := domain.NewAuthUseCase(passwordAdapter)

	_ = authUserCase

	ctx := context.Background()
	userID, err := authUserCase.Register(ctx, "test@example.com", "pass123")
	if err != nil {
		log.Printf("Failed to register user: %v", err)
	} else {
		log.Printf("User registered with ID: %v", userID)
	}

	log.Println("Auth User Case and password adapter initialized")

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
