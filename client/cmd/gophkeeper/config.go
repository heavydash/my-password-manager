package main

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"os"
	"path/filepath"
)

// Config — глобальная конфигурация приложения
type Config struct {
	ServerURL string `yaml:"server_url"`
	LogLevel  string `yaml:"log_level"`
}

var cfg Config

// LoadConfig загружает конфиг из ~/.gophkeeper/config.yaml
func LoadConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".gophkeeper")
	configPath := filepath.Join(configDir, "config.yaml")

	// Создаём папку и дефолтный конфиг, если его нет
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}

		defaultCfg := Config{
			ServerURL: "http://localhost:8080",
			LogLevel:  "info",
		}

		data, _ := yaml.Marshal(defaultCfg)
		os.WriteFile(configPath, data, 0644)
		fmt.Printf("Created default config at %s\n", configPath)
	}

	// Читаем конфиг
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, &cfg)
}

// GetServerURL возвращает URL сервера из конфига
func GetServerURL() string {
	if cfg.ServerURL == "" {
		return "http://localhost:8080"
	}
	return cfg.ServerURL
}
