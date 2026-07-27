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
	"github.com/joho/godotenv"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Config — корневая структура конфигурации GophKeeper.
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	JWT      JWTConfig      `json:"jwt"`
	Pprof    PprofConfig    `json:"pprof"`
	OAuth    OAuthConfig    `json:"oauth"`
	Argon2   Argon2Config   `json:"argon2"`
}

// ServerConfig содержит настройки HTTP/gRPC серверов.
type ServerConfig struct {
	Port     string // HTTP порт
	GRPCPort string // gRPC порт
	Env      string // окружение
	Debug    bool   // режим отладки

	// Таймауты HTTP-сервера
	InitTimeout     time.Duration `json:"init_timeout"`     // таймаут инициализации
	ReadTimeout     time.Duration `json:"read_timeout"`     // таймаут чтения запроса
	WriteTimeout    time.Duration `json:"write_timeout"`    // таймаут записи ответа
	IdleTimeout     time.Duration `json:"idle_timeout"`     // таймаут idle соединения
	ShutdownTimeout time.Duration `json:"shutdown_timeout"` // таймаут graceful shutdown
	MaxHeaderBytes  int           `json:"max_header_bytes"` // максимальный размер заголовка
	Host            string        `json:"host"`             // хост для формирования URL
}

// DatabaseConfig содержит настройки подключения к PostgreSQL.
type DatabaseConfig struct {
	DSN               string        `json:"dsn"`                 // строка подключения
	MaxConns          int           `json:"max_conns"`           // максимум соединений
	MinConns          int           `json:"min_conns"`           // минимум соединений
	MaxConnLifetime   time.Duration `json:"max_conn_lifetime"`   // макс. время жизни соединения
	MaxConnIdleTime   time.Duration `json:"max_conn_idle_time"`  // макс. время idle
	HealthCheckPeriod time.Duration `json:"health_check_period"` // период проверки здоровья
	PingTimeout       time.Duration `json:"ping_timeout"`        // таймаут ping
	MigrationTimeout  time.Duration `json:"migration_timeout"`   // таймаут миграций
}

// JWTConfig содержит настройки JWT-токенов.
type JWTConfig struct {
	Secret    string        `json:"secret"`     // секрет для подписи (мин. 32 символа)
	ExpiresIn time.Duration `json:"expires_in"` // время жизни токена
}

// PprofConfig содержит настройки pprof сервера.
type PprofConfig struct {
	Port string `json:"port"` // порт для pprof
}

// OAuthConfig содержит настройки OAuth2-провайдеров.
type OAuthConfig struct {
	StateTTL       time.Duration `json:"state_ttl"`         // TTL для state
	OneTimeCodeTTL time.Duration `json:"one_time_code_ttl"` // TTL для одноразового кода
	JWTExpiresAt   time.Duration `json:"jwt_expires_in"`    // время жизни JWT для OAuth

	Google struct {
		ClientID     string `json:"client_id"`     // Client ID Google OAuth
		ClientSecret string `json:"client_secret"` // Client Secret Google OAuth
		RedirectURL  string `json:"redirect_url"`  // Redirect URL Google OAuth
	} `json:"google"`

	Yandex struct {
		ClientID     string `json:"client_id"`     // Client ID Yandex OAuth
		ClientSecret string `json:"client_secret"` // Client Secret Yandex OAuth
		RedirectURL  string `json:"redirect_url"`  // Redirect URL Yandex OAuth
	} `json:"yandex"`
}

// Argon2Config содержит параметры хеширования паролей Argon2.
type Argon2Config struct {
	Salt        string `json:"salt"`        // соль для хеширования
	Iterations  uint32 `json:"iterations"`  // количество итераций
	Memory      uint32 `json:"memory"`      // объём памяти (KB)
	Parallelism uint8  `json:"parallelism"` // количество потоков
	KeyLen      uint32 `json:"key_len"`     // длина ключа
}

// New — основная функция для продакшена.
func New() (*Config, error) {
	return newWithFlags(flag.CommandLine)
}

// NewWithFlags — версия для тестов (принимает свой FlagSet).
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
			log.Printf("Warning: failed to load JSON config: %v", err)
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

// defaultConfig возвращает конфигурацию со значениями по умолчанию.
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:     "8080",
			GRPCPort: "9090",
			Env:      "development",
			Debug:    false,

			InitTimeout:     30 * time.Second,
			ReadTimeout:     15 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 10 * time.Second,
			MaxHeaderBytes:  1 << 20,
			Host:            "localhost",
		},
		Database: DatabaseConfig{
			DSN:               "postgres://postgres:supersecretpassword123@localhost:5433/gophkeeper?sslmode=disable",
			MaxConns:          25,
			MinConns:          5,
			MaxConnLifetime:   1 * time.Hour,
			MaxConnIdleTime:   30 * time.Minute,
			HealthCheckPeriod: 1 * time.Minute,
			PingTimeout:       5 * time.Second,
			MigrationTimeout:  30 * time.Second,
		},
		JWT: JWTConfig{
			Secret:    "change-me-in-production-very-long-random-string-2026-gophkeeper",
			ExpiresIn: 15 * time.Minute,
		},
		Pprof: PprofConfig{
			Port: "6060",
		},
		OAuth: OAuthConfig{
			StateTTL:       15 * time.Minute,
			OneTimeCodeTTL: 5 * time.Minute,
			JWTExpiresAt:   15 * time.Minute,
		},
		Argon2: Argon2Config{
			Salt:        "gophkeeper-salt",
			Iterations:  1,
			Memory:      64 * 1024,
			Parallelism: 4,
			KeyLen:      32,
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
	if v := os.Getenv("SERVER_INIT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.InitTimeout = d
		}
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
	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SERVER_MAX_HEADER_BYTES"); v != "" {
		var val int
		fmt.Sscanf(v, "%d", &val)
		if val > 0 {
			cfg.Server.MaxHeaderBytes = val
		}
	}
	// Database
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("DB_MAX_CONNS"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Database.MaxConns)
	}
	if v := os.Getenv("DB_MIN_CONNS"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Database.MinConns)
	}
	if v := os.Getenv("DB_MAX_CONN_LIFETIME"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Database.MaxConnLifetime = d
		}
	}
	if v := os.Getenv("DB_MAX_CONN_IDLE_TIME"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Database.MaxConnIdleTime = d
		}
	}
	if v := os.Getenv("DB_HEALTH_CHECK_PERIOD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Database.HealthCheckPeriod = d
		}
	}
	if v := os.Getenv("DB_PING_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Database.PingTimeout = d
		}
	}
	if v := os.Getenv("DB_MIGRATION_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Database.MigrationTimeout = d
		}
	}

	// JWT
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("JWT_EXPIRES_IN"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.JWT.ExpiresIn = d
		}
	}

	// Argon2
	if v := os.Getenv("ARGON2_SALT"); v != "" {
		cfg.Argon2.Salt = v
	}
	if v := os.Getenv("ARGON2_ITERATIONS"); v != "" {
		var val uint32
		fmt.Sscanf(v, "%d", &val)
		if val > 0 {
			cfg.Argon2.Iterations = val
		}
	}
	if v := os.Getenv("ARGON2_MEMORY"); v != "" {
		var val uint32
		fmt.Sscanf(v, "%d", &val)
		if val > 0 {
			cfg.Argon2.Memory = val
		}
	}
	if v := os.Getenv("ARGON2_PARALLELISM"); v != "" {
		var val uint8
		fmt.Sscanf(v, "%d", &val)
		if val > 0 {
			cfg.Argon2.Parallelism = val
		}
	}
	if v := os.Getenv("ARGON2_KEY_LEN"); v != "" {
		var val uint32
		fmt.Sscanf(v, "%d", &val)
		if val > 0 {
			cfg.Argon2.KeyLen = val
		}
	}
	// OAuth
	if v := os.Getenv("OAUTH_STATE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.OAuth.StateTTL = d
		}
	}
	if v := os.Getenv("OAUTH_ONETIME_CODE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.OAuth.OneTimeCodeTTL = d
		}
	}
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
	if v := os.Getenv("OAUTH_JWT_EXPIRES_IN"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.OAuth.JWTExpiresAt = d
		}
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
//   - JWT.ExpiresIn > 0
//   - Database.DSN обязателен
//   - Database.MaxConns ≥ 1
//   - Database.MinConns ≥ 1
//   - Server.Port, GRPCPort, Env обязательны
//   - В production режиме требуются OAuth credentials
//   - Argon2 параметры имеют дефолтные значения
func (c *Config) Validate() error {
	// JWT валидация
	if c.JWT.Secret == "" || len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}
	if c.JWT.ExpiresIn <= 0 {
		return fmt.Errorf("JWT_EXPIRES_IN must be greater than 0")
	}
	// Database валидация
	if c.Database.DSN == "" {
		return fmt.Errorf("DB_DSN is required")
	}
	if c.Database.MaxConns < 1 {
		return fmt.Errorf("DB_MAX_CONNS must be at least 1")
	}
	if c.Database.MinConns < 1 {
		return fmt.Errorf("DB_MIN_CONNS must be at least 1")
	}
	// Server валидация
	if c.Server.Port == "" {
		return fmt.Errorf("Server.Port is required")
	}
	if c.Server.GRPCPort == "" {
		return fmt.Errorf("Server.GRPCPort is required")
	}
	if c.Server.Env == "" {
		return fmt.Errorf("Server.Env is required")
	}

	// Production требует OAuth
	if c.Server.Env == "production" {
		if c.OAuth.Google.ClientID == "" || c.OAuth.Google.ClientSecret == "" {
			return fmt.Errorf("Google OAuth credentials required in production")
		}
		if c.OAuth.Yandex.ClientID == "" || c.OAuth.Yandex.ClientSecret == "" {
			return fmt.Errorf("Yandex OAuth credentials required in production")
		}
	}

	// Google RedirectURL по умолчанию
	if c.OAuth.Google.RedirectURL == "" {
		c.OAuth.Google.RedirectURL = "http://" + c.Server.Host + ":" + c.Server.Port + "/auth/google/callback"
	}

	// Argon2 дефолтные значения
	if c.Argon2.Iterations == 0 {
		c.Argon2.Iterations = 1
	}
	if c.Argon2.Memory == 0 {
		c.Argon2.Memory = 64 * 1024
	}
	if c.Argon2.Parallelism == 0 {
		c.Argon2.Parallelism = 4
	}
	if c.Argon2.KeyLen == 0 {
		c.Argon2.KeyLen = 32
	}
	if c.Argon2.Salt == "" {
		c.Argon2.Salt = "gophkeeper-salt"
	}
	return nil
}
