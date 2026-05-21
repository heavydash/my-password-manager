// Package storage предоставляет фабрики для создания репозиториев с автоматическим fallback.
//
// Содержит:
//   - NewSecretRepository: создание хранилища секретов (PostgreSQL → in-memory)
//   - NewUserRepository: создание хранилища пользователей (PostgreSQL → in-memory)
//
// Приоритет:
//  1. PostgreSQL — если DSN указан и подключение успешно
//  2. In-memory — fallback при недоступности БД или отсутствии DSN
package storage

import (
	"context"
	"go.uber.org/zap"
	"gophkeeper/server/internal/adapters/storage/memory"
	"gophkeeper/server/internal/adapters/storage/postgres"
	"gophkeeper/server/internal/config"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
)

// NewSecretRepository возвращает SecretStoragePort с автоматическим fallback.
//
// Алгоритм:
//  1. Если Database.DSN указан, пытается подключиться к PostgreSQL
//  2. При успехе возвращает PostgreSQL репозиторий
//  3. При ошибке логирует предупреждение и переходит к in-memory
//  4. Если DSN не указан или подключение не удалось — использует in-memory
//
// Параметры:
//   - ctx: контекст для операций с БД
//   - cfg: конфигурация сервера (содержит Database.DSN)
//   - log: логгер для записи событий
//
// Возвращает реализацию ports.SecretStoragePort.
func NewSecretRepository(ctx context.Context, cfg *config.Config, log logger.Logger) (ports.SecretStoragePort, error) {
	if cfg.Database.DSN != "" {
		// Попытка подключения к PostgreSQL
		pool, err := postgres.NewPool(ctx, cfg, log)
		if err == nil {
			log.Info("using PostgreSQL secret repository")
			// Передаём pool (реализует ports.DBPool)
			return postgres.NewSecretRepository(pool, log), nil
		}
		log.Warn("PostgreSQL unavailable, falling back to in-memory", zap.Error(err))
	}

	// In-memory fallback
	log.Info("using in-memory secret repository")
	return memory.NewSecretStorage(log), nil
}

// NewUserRepository возвращает StoragePort с автоматическим fallback.
//
// Алгоритм:
//  1. Если Database.DSN указан, пытается подключиться к PostgreSQL
//  2. При успехе возвращает PostgreSQL репозиторий
//  3. При ошибке логирует предупреждение и переходит к in-memory
//  4. Если DSN не указан или подключение не удалось — использует in-memory
//
// Параметры:
//   - ctx: контекст для операций с БД
//   - cfg: конфигурация сервера (содержит Database.DSN)
//   - log: логгер для записи событий
//
// Возвращает реализацию ports.StoragePort.
func NewUserRepository(ctx context.Context, cfg *config.Config, log logger.Logger) (ports.StoragePort, error) {
	// Попытка подключения к PostgreSQL
	if cfg.Database.DSN != "" {
		pool, err := postgres.NewPool(ctx, cfg, log)
		if err == nil {
			log.Info("using PostgreSQL user repository")
			// Передаём pool (реализует ports.DBPool)
			return postgres.NewUserRepository(pool, log), nil
		}
		log.Warn("PostgreSQL unavailable, falling back to in-memory user repo")
	}
	// Fallback на in-memory
	log.Info("using in-memory user repository")
	return memory.NewUserRepository(log), nil
}
