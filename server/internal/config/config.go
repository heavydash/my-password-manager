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
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Pprof    PprofConfig
	Debug    bool
}

type ServerConfig struct {
	Port     string
	GRPCPort string
	Env      string
	Debug    bool
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

type JWTConfig struct {
	Secret string
}

type PprofConfig struct {
	Port string
}

// New — основная функция загрузки конфига
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
			Port:     "8080",
			GRPCPort: "9090",
			Env:      "development",
			Debug:    false,
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
