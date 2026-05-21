// Package config содержит тесты для конфигурации приложения.
//
// Тестируются:
//   - New: загрузка конфигурации (дефолты, env, флаги)
//   - Validate: валидация обязательных полей
package config

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

// TestConfig_New тестирует загрузку конфигурации.
//
// Сценарии:
//   - default_config: значения по умолчанию
//   - env_override: переопределение через переменные окружения
func TestConfig_New(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		// Загрузка конфигурации с тестовым FlagSet
		cfg, err := newWithFlags(flag.NewFlagSet("test", flag.ContinueOnError))
		assert.NoError(t, err)

		// Проверка значений по умолчанию
		assert.Equal(t, "8080", cfg.Server.Port)
		assert.Equal(t, "9090", cfg.Server.GRPCPort)
		assert.Equal(t, "development", cfg.Server.Env)
		assert.Equal(t, false, cfg.Server.Debug)
	})

	t.Run("env_override", func(t *testing.T) {
		// Сохранение старых значений переменных окружения
		oldPort := os.Getenv("SERVER_PORT")
		oldSecret := os.Getenv("JWT_SECRET")

		// Установка тестовых значений
		os.Setenv("SERVER_PORT", "3000")
		os.Setenv("JWT_SECRET", "my-super-secret-jwt-key-32-chars-minimum")

		// Восстановление после теста
		defer func() {
			os.Setenv("SERVER_PORT", oldPort)
			os.Setenv("JWT_SECRET", oldSecret)
		}()

		// Загрузка конфигурации
		cfg, err := newWithFlags(flag.NewFlagSet("test", flag.ContinueOnError))
		assert.NoError(t, err)

		// Проверка переопределённых значений
		assert.Equal(t, "3000", cfg.Server.Port)
	})
}

// TestConfig_Validate тестирует валидацию конфигурации.
//
// Сценарии:
//   - valid config: корректная конфигурация
//   - short jwt secret: слишком короткий JWT секрет
//   - missing DB_DSN: отсутствует DSN базы данных
//   - production without OAuth: production режим без OAuth credentials
func TestConfig_Validate(t *testing.T) {
	// Подготовка тестовых случаев
	tests := []struct {
		name    string  // имя теста
		cfg     *Config // конфигурация для валидации
		wantErr string  // ожидаемая подстрока ошибки (пусто — ошибки нет)
	}{
		{
			name: "valid config",
			cfg: &Config{
				JWT: JWTConfig{
					Secret:    "very-long-secret-key-at-least-32-chars",
					ExpiresIn: 15 * time.Minute,
				},
				Database: DatabaseConfig{
					DSN:      "postgres://user:pass@localhost/db",
					MaxConns: 25,
					MinConns: 5,
				},
				Server: ServerConfig{
					Port:     "8080",
					GRPCPort: "9090",
					Env:      "development",
				},
				OAuth: OAuthConfig{
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
				Argon2: Argon2Config{
					Salt:        "test-salt",
					Iterations:  1,
					Memory:      64 * 1024,
					Parallelism: 4,
					KeyLen:      32,
				},
			},
			wantErr: "",
		},
		{
			name: "short jwt secret",
			cfg: &Config{
				JWT: JWTConfig{
					Secret:    "short",
					ExpiresIn: 15 * time.Minute,
				},
				Database: DatabaseConfig{
					DSN:      "postgres://user:pass@localhost/db",
					MaxConns: 25,
					MinConns: 5,
				},
				Server: ServerConfig{
					Port:     "8080",
					GRPCPort: "9090",
					Env:      "development",
				},
			},
			wantErr: "JWT_SECRET must be at least 32 characters long",
		},
		{
			name: "missing DB_DSN",
			cfg: &Config{
				JWT: JWTConfig{
					Secret:    "very-long-secret-key-at-least-32-chars",
					ExpiresIn: 15 * time.Minute,
				},
				Database: DatabaseConfig{
					MaxConns: 25,
					MinConns: 5,
				},
				Server: ServerConfig{
					Port:     "8080",
					GRPCPort: "9090",
					Env:      "development",
				},
			},
			wantErr: "DB_DSN is required",
		},
		{
			name: "production without OAuth",
			cfg: &Config{
				JWT: JWTConfig{
					Secret:    "very-long-secret-key-at-least-32-chars",
					ExpiresIn: 15 * time.Minute,
				},
				Database: DatabaseConfig{
					DSN:      "postgres://user:pass@localhost/db",
					MaxConns: 25,
					MinConns: 5,
				},
				Server: ServerConfig{
					Port:     "8080",
					GRPCPort: "9090",
					Env:      "production",
				},
				OAuth: OAuthConfig{},
			},
			wantErr: "Google OAuth credentials required in production",
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
