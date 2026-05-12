// Package config отвечает за загрузку и валидацию конфигурации приложения.
//
// Поддерживает (по приоритету):
// 1. Переменные окружения
// 2. .env файл
// 3. JSON-файл (флаг -c)
// 4. Значения по умолчанию
package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Config — корневая структура конфигурации GophKeeper.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Pprof    PprofConfig
	OAuth    OAuth
	Debug    bool
}

type ServerConfig struct {
	Port     string
	GRPCPort string
	Env      string
	Debug    bool

	// Таймауты HTTP-сервера
	ReadTimeout     time.Duration `json:"read_timeout"`
	WriteTimeout    time.Duration `json:"write_timeout"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

type DatabaseConfig struct {
	DSN               string        `json:"dsn"`
	MaxConns          int           `json:"max_conns"`
	MinConns          int           `json:"min_conns"`
	MaxConnLifetime   time.Duration `json:"max_conn_lifetime"`
	MaxConnIdleTime   time.Duration `json:"max_conn_idle_time"`
	HealthCheckPeriod time.Duration `json:"health_check_period"`
	PingTimeout       time.Duration `json:"ping_timeout"`
}

type OAuth struct {
	Google struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURL  string `json:"redirect_url"`
	}
	Yandex struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RedirectURL  string `json:"redirect_url"`
	}
}

type JWTConfig struct {
	Secret string
}

type PprofConfig struct {
	Port string
}

// New — основная функция для продакшена.
func New() (*Config, error) {
	return newWithFlags(flag.CommandLine)
}

// newWithFlags — версия для тестов (принимает свой FlagSet).
func newWithFlags(fs *flag.FlagSet) (*Config, error) {
	cfg := defaultConfig()

	// Флаги командной строки
	configFile := fs.String("c", "", "path to config file")
	if err := fs.Parse(os.Args[1:]); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			log.Printf("flag parse warning: %v", err)
		}
	}

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

	if cfg.Server.Debug {
		log.Printf("OAuth Google enabled: %t (ClientID present: %t)",
			cfg.OAuth.Google.ClientID != "", cfg.OAuth.Google.ClientID != "")
		log.Printf("OAuth Yandex enabled: %t", cfg.OAuth.Yandex.ClientID != "")

	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	log.Println("Configuration loaded successfully")
	return cfg, nil
}

// defaultConfig возвращает конфигурацию со значениями по умолчанию.
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            "8080",
			GRPCPort:        "9090",
			Env:             "development",
			Debug:           false,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		Database: DatabaseConfig{
			DSN:               "postgres://postgres:supersecretpassword123@localhost:5433/gophkeeper?sslmode=disable",
			MaxConns:          25,
			MinConns:          5,
			MaxConnLifetime:   1 * time.Hour,
			MaxConnIdleTime:   30 * time.Minute,
			HealthCheckPeriod: 1 * time.Minute,
			PingTimeout:       5 * time.Second,
		},
		JWT: JWTConfig{
			Secret: "change-me-in-production-very-long-random-string-2026-gophkeeper",
		},
		Pprof: PprofConfig{
			Port: "6060",
		},
		OAuth: OAuth{
			Google: struct {
				ClientID     string `json:"client_id"`
				ClientSecret string `json:"client_secret"`
				RedirectURL  string `json:"redirect_url"`
			}{},
			Yandex: struct {
				ClientID     string `json:"client_id"`
				ClientSecret string `json:"client_secret"`
				RedirectURL  string `json:"redirect_url"`
			}{},
		},
	}
}

// loadDotEnv пытается загрузить .env файл из корня проекта.
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

// overwriteFromEnv перезаписывает конфигурацию значениями из переменных окружения.
func overwriteFromEnv(cfg *Config) {
	// Server
	if v := os.Getenv("SERVER_PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("GRPC_PORT"); v != "" {
		cfg.Server.GRPCPort = v
	}
	if v := os.Getenv("ENV"); v != "" {
		cfg.Server.Env = v
	}
	if v := os.Getenv("DEBUG"); v != "" {
		cfg.Server.Debug = v == "true" || v == "1"
	}
	if v := os.Getenv("PPROF_PORT"); v != "" {
		cfg.Pprof.Port = v
	}
	if v := os.Getenv("SERVER_READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if v := os.Getenv("SERVER_WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}
	if v := os.Getenv("SERVER_IDLE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.IdleTimeout = d
		}
	}
	if v := os.Getenv("SERVER_SHUTDOWN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ShutdownTimeout = d
		}
	}
	// DB
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Database.MaxConns)
	}
	if v := os.Getenv("DB_MIN_CONNS"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Database.MinConns)
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}

	// OAuth
	if v := os.Getenv("OAUTH_GOOGLE_CLIENT_ID"); v != "" {
		cfg.OAuth.Google.ClientID = v
	}
	if v := os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET"); v != "" {
		cfg.OAuth.Google.ClientSecret = v
	}
	if v := os.Getenv("OAUTH_GOOGLE_REDIRECT_URL"); v != "" {
		cfg.OAuth.Google.RedirectURL = v
	}
	if v := os.Getenv("OAUTH_YANDEX_CLIENT_ID"); v != "" {
		cfg.OAuth.Yandex.ClientID = v
	}
	if v := os.Getenv("OAUTH_YANDEX_CLIENT_SECRET"); v != "" {
		cfg.OAuth.Yandex.ClientSecret = v
	}
	if v := os.Getenv("OAUTH_YANDEX_REDIRECT_URL"); v != "" {
		cfg.OAuth.Yandex.RedirectURL = v
	}
}

// loadFromJSON загружает конфигурацию из JSON-файла.
func loadFromJSON(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

// Validate проверяет обязательные поля и корректность значений.
//
// Проверяемые условия:
//   - JWT.Secret ≥ 32 символа
//   - Database.DSN обязателен
//   - Порты и окружение обязательны
//   - В production режиме требуются OAuth credentials
func (c *Config) Validate() error {
	if c.JWT.Secret == "" || len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("DB_DSN is required")
	}
	if c.Database.MaxConns < 1 {
		return fmt.Errorf("DB_MAX_CONNS must be at least 1")
	}
	if c.Database.MinConns < 1 {
		return fmt.Errorf("DB_MIN_CONNS must be at least 1")
	}
	if c.Server.Port == "" {
		return fmt.Errorf("Server.Port is required")
	}
	if c.Server.GRPCPort == "" {
		return fmt.Errorf("Server.GRPCPort is required")
	}
	if c.Server.Env == "" {
		return fmt.Errorf("Server.Env is required")
	}
	if c.Server.Env == "production" {
		if c.OAuth.Google.ClientID == "" || c.OAuth.Google.ClientSecret == "" ||
			c.OAuth.Yandex.ClientID == "" || c.OAuth.Yandex.ClientSecret == "" {
			return fmt.Errorf("OAuth credentials required in production")
		}
	}
	if c.OAuth.Google.RedirectURL == "" {
		c.OAuth.Google.RedirectURL = "http://localhost:" + c.Server.Port + "/auth/google/callback"
	}
	return nil
}
