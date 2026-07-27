package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Database DatabaseConfig
	JWT      JWTConfig
	Server   ServerConfig
	Pprof    PprofConfig
}

type DatabaseConfig struct {
	DSN string
}

type JWTConfig struct {
	Secret string
}

type ServerConfig struct {
	Port string
}

type PprofConfig struct {
	Port string
}

// Load точка загрузки конфига
func Load() *Config {
	log.Println("Config load start")

	// Абсолютный путь к файлу
	wd, _ := os.Getwd()
	projectRoot := filepath.Dir(wd) // server/ → my-password-manager/
	envPath := filepath.Join(projectRoot, ".env")

	// Основная попытка загрузки
	if err := godotenv.Load(envPath); err == nil {
		log.Printf(".env loaded from %s", envPath)
	} else {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Читаем DB_DSN
	dbDSN := os.Getenv("DB_DSN")
	jwtSecret := os.Getenv("JWT_SECRET")

	if dbDSN == "" {
		log.Fatal("DB_DSN environment variable is required")
	}

	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable not set, JWT_SECRET environment variable is required")
	}

	// Валидация минимальной длины секрета для безопасности (HS256)
	if len(jwtSecret) < 32 {
		log.Fatal("JWT secret length must be at least 32 bytes")
	}

	log.Println("Configuration loaded successfully")

	return &Config{
		Database: DatabaseConfig{
			DSN: dbDSN,
		},
		JWT: JWTConfig{
			Secret: jwtSecret,
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Pprof: PprofConfig{
			Port: getEnv("PPROF_PORT", "6060"),
		},
	}
}

// getEnv вспомогательная функция с деф значением
func getEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
