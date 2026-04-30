package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"log"
	"os"
	"path/filepath"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Pprof    PprofConfig
	Debug    bool
}

type ServerConfig struct {
	Port  string
	Env   string
	Debug bool
}

type DatabaseConfig struct {
	DSN string
}

type JWTConfig struct {
	Secret string
}

type PprofConfig struct {
	Port string
}

// Load точка загрузки конфига
func New() (*Config, error) {

	cfg := defaultConfig()

	// Флаги командной строки
	configFile := flag.String("c", "", "path to config file")
	flag.Parse()

	// JSON-файл
	if *configFile != "" {
		if err := loadFromJSON(*configFile, cfg); err != nil {
			logger.Error("Warning: failed to load JSON config: %v", zap.Error(err))
		}
	}

	// .env файл
	loadDotEnv()

	// Переменные окружения (высший приоритет)
	overwriteFromEnv(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	log.Println("Configuration loaded successfully")
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:  "8080",
			Env:   "development",
			Debug: false,
		},
		Database: DatabaseConfig{
			DSN: "postgres://postgres:supersecretpassword123@localhost:5433/gophkeeper?sslmode=disable",
		},
		JWT: JWTConfig{
			Secret: "change-me-in-production-very-long-random-string-2026-gophkeeper",
		},
		Pprof: PprofConfig{
			Port: "6060",
		},
	}
}

func loadDotEnv() {
	wd, _ := os.Getwd()
	projectRoot := filepath.Dir(wd)
	envPath := filepath.Join(projectRoot, ".env")

	if err := godotenv.Load(envPath); err == nil {
		log.Printf(".env loaded from %s", envPath)
	} else {
		log.Println("Warning: .env file not found")
	}
}

func overwriteFromEnv(cfg *Config) {
	if v := os.Getenv("SERVER_PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("ENV"); v != "" {
		cfg.Server.Env = v
	}
	if v := os.Getenv("DEBUG"); v != "" {
		cfg.Server.Debug = v == "true" || v == "1"
	}
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("PPROF_PORT"); v != "" {
		cfg.Pprof.Port = v
	}
}

func loadFromJSON(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

func (c *Config) Validate() error {
	if c.JWT.Secret == "" || len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("DB_DSN is required")
	}
	return nil
}

/*

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

*/
