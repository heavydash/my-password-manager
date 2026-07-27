// Package config отвечает за управление конфигурацией клиента GophKeeper.
//
// Поддерживает:
//   - YAML конфигурационный файл (~/.gophkeeper/config.yaml)
//   - Автоматическое создание файла с настройками по умолчанию
//   - Функции для получения настроек
//
// Конфигурация включает:
//   - server_url: адрес сервера (по умолчанию http://localhost:8080)
//   - log_level: уровень логирования (debug, info, warn, error)
package main

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
	"path/filepath"
)

// Config — глобальная структура конфигурации приложения.
//
// Поля:
//   - ServerURL: URL адрес сервера GophKeeper
//   - LogLevel: уровень логирования (debug, info, warn, error)
type Config struct {
	ServerURL string `yaml:"server_url"`
	LogLevel  string `yaml:"log_level"`
}

// cfg — глобальный экземпляр конфигурации (приватный).
var cfg Config

// LoadConfig загружает конфигурацию из файла ~/.gophkeeper/config.yaml.
//
// Алгоритм:
//  1. Определяет путь к конфигурационному файлу в домашней директории
//  2. Если файл не существует, создаёт директорию и записывает конфиг по умолчанию
//  3. Читает и парсит YAML файл
//  4. Сохраняет результат в глобальную переменную cfg
//
// Возвращает:
//   - error: ошибка при создании директории, записи файла или парсинге YAML
//
// Пример содержимого config.yaml:
//
//	server_url: https://gophkeeper.example.com:8080
//	log_level: info
func LoadConfig() error {
	// Получение домашней директории пользователя
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Формирование пути к конфигурационному файлу
	configDir := filepath.Join(home, ".gophkeeper")
	configPath := filepath.Join(configDir, "config.yaml")

	// Проверка существования конфигурационного файла
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Создание директории (если не существует)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}

		// Конфигурация по умолчанию
		defaultCfg := Config{
			ServerURL: "http://localhost:8080",
			LogLevel:  "info",
		}

		// Маршалинг в YAML
		data, _ := yaml.Marshal(defaultCfg)
		if err != nil {
			return fmt.Errorf("failed to marshal default config: %w", err)
		}

		// Запись файла конфигурации
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}

		fmt.Printf(" Created default configuration file at %s\n", configPath)
		fmt.Println("  Edit this file to change server URL or log level")
	}

	// Читаем конфиг
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// Парсинг YAML
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// GetServerURL возвращает URL сервера из конфигурации.
//
// Если URL не задан, возвращает значение по умолчанию "http://localhost:8080".
//
// Возвращает:
//   - string: URL сервера (схема + хост + порт)
func GetServerURL() string {
	if cfg.ServerURL == "" {
		return "http://localhost:8080"
	}
	return cfg.ServerURL
}

// GetLogLevel возвращает уровень логирования из конфигурации.
//
// Если уровень не задан, возвращает значение по умолчанию "info".
//
// Возвращает:
//   - string: уровень логирования (debug, info, warn, error)
func GetLogLevel() string {
	if cfg.LogLevel == "" {
		return "info"
	}
	return cfg.LogLevel
}
