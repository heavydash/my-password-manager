// Package secret содержит адаптер для работы с секретами.
//
// Реализует ports.SecretPort, делегируя операции хранения в SecretStoragePort.
// Добавляет логирование и базовую валидацию.
package secret

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
)

// SecretAdapter — адаптер для работы с секретами.
//
// Реализует ports.SecretPort.
// Делегирует все операции хранения в SecretStoragePort и добавляет логирование.
type SecretAdapter struct {
	storage ports.SecretStoragePort
	logger  logger.Logger
}

// NewSecretAdapter создаёт новый адаптер секретов.
//
// Параметры:
//   - storage: хранилище секретов (реализует ports.SecretStoragePort)
//   - log: логгер для записи событий
//
// Паникует, если storage или logger == nil.
func NewSecretAdapter(storage ports.SecretStoragePort, log logger.Logger) *SecretAdapter {
	if storage == nil {
		panic("storage is nil")
	}
	if log == nil {
		panic("logger is nil")
	}
	return &SecretAdapter{
		storage: storage,
		logger:  log,
	}
}

// Create создаёт новый секрет.
//
// Алгоритм:
//  1. Проверяет, что secret не nil
//  2. Валидирует обязательные поля
//  3. Делегирует создание в storage
//
// Возвращает созданный секрет с заполненным ID или ошибку.
func (a *SecretAdapter) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	// Валидация входных данных
	if secret == nil {
		a.logger.Warn("secret adapter create: secret is nil")
		return nil, domain.ErrInvalidInput
	}
	if secret.Title == "" {
		a.logger.Warn("secret adapter create: empty title")
		return nil, domain.ErrInvalidInput
	}
	if secret.Data == "" {
		a.logger.Warn("secret adapter create: empty data")
		return nil, domain.ErrInvalidInput
	}
	if secret.Type == "" {
		a.logger.Warn("secret adapter create: empty type")
		return nil, domain.ErrInvalidInput
	}

	a.logger.Debug("secret adapter create",
		zap.String("title", secret.Title),
		zap.String("user_id", secret.UserID),
	)

	created, err := a.storage.Create(ctx, secret)
	if err != nil {
		a.logger.Error("secret adapter create failed",
			zap.String("title", secret.Title),
			zap.String("user_id", secret.UserID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("error storage creating secret: %w", err)
	}
	a.logger.Info("secret adapter create success",
		zap.String("secret_id", created.ID),
		zap.String("user_id", created.UserID),
	)
	return created, nil

}

// GetByUserID возвращает все секреты пользователя.
//
// Алгоритм:
//  1. Проверяет, что userID не пустой
//  2. Делегирует получение в storage
//
// Возвращает список секретов или ошибку.
func (a *SecretAdapter) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {
	// Валидация входных данных
	if userID == "" {
		a.logger.Warn("secret adapter get by user id: empty user_id")
		return nil, domain.ErrInvalidInput
	}

	a.logger.Debug("secret adapter get by user id", zap.String("user_id", userID))

	secrets, err := a.storage.GetByUserID(ctx, userID)
	if err != nil {
		a.logger.Error("secret adapter get by user id failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("error storage getting secrets: %w", err)
	}

	a.logger.Info("secret adapter get by user id success",
		zap.String("user_id", userID),
		zap.Int("count", len(secrets)),
	)
	return secrets, nil
}

// GetSecret возвращает один секрет по ID.
//
// Алгоритм:
//  1. Проверяет, что id и userID не пустые
//  2. Делегирует получение в storage
//
// Возвращает секрет или ошибку (domain.ErrNotFound если не найден).
func (a *SecretAdapter) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	// Валидация входных данных
	if id == "" {
		a.logger.Warn("secret adapter get secret: empty id", zap.String("user_id", userID))
		return nil, domain.ErrInvalidInput
	}
	if userID == "" {
		a.logger.Warn("secret adapter get secret: empty user_id", zap.String("secret_id", id))
		return nil, domain.ErrInvalidInput
	}

	a.logger.Debug("secret adapter get secret",
		zap.String("secret_id", id),
		zap.String("user_id", userID),
	)
	secret, err := a.storage.GetSecret(ctx, id, userID)
	if err != nil {
		a.logger.Error("secret adapter get secret failed",
			zap.String("secret_id", id),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return nil, err
	}

	a.logger.Info("secret adapter get secret success",
		zap.String("secret_id", id),
		zap.String("user_id", userID),
	)
	return secret, nil
}

// Delete удаляет секрет пользователя.
//
// Алгоритм:
//  1. Проверяет, что userID и secretID не пустые
//  2. Делегирует удаление в storage
//
// Возвращает ошибку, если удаление не удалось.
func (a *SecretAdapter) Delete(ctx context.Context, userID, secretID string) error {
	// Валидация входных данных
	if userID == "" {
		a.logger.Warn("secret adapter delete: empty user_id", zap.String("secret_id", secretID))
		return domain.ErrInvalidInput
	}
	if secretID == "" {
		a.logger.Warn("secret adapter delete: empty secret_id", zap.String("user_id", userID))
		return domain.ErrInvalidInput
	}

	a.logger.Info("secret adapter delete",
		zap.String("secret_id", secretID),
		zap.String("user_id", userID),
	)

	err := a.storage.Delete(ctx, userID, secretID)
	if err != nil {
		a.logger.Error("secret adapter delete failed",
			zap.String("secret_id", secretID),
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return fmt.Errorf("error storage deleting secret: %w", err)
	}

	a.logger.Info("secret adapter delete success",
		zap.String("secret_id", secretID),
		zap.String("user_id", userID),
	)
	return nil
}
