// Package secret содержит адаптер для работы с секретами.
//
// Реализует ports.SecretPort, делегируя операции хранения в SecretStoragePort.
// Добавляет логирование и базовую валидацию.
package secret

import (
	"context"
	"fmt"
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
// Паникует, если storage или logger == nil
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
// Выполняет минимальную валидацию и делегирует в storage.
func (a *SecretAdapter) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	if secret == nil {
		return nil, domain.ErrInvalidInput
	}

	created, err := a.storage.Create(ctx, secret)
	if err != nil {
		return nil, fmt.Errorf("error storage creating secret: %w", err)
	}
	return created, nil

}

// GetByUserID возвращает все секреты пользователя.
func (a *SecretAdapter) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {

	if userID == "" {
		return nil, domain.ErrInvalidInput
	}

	secrets, err := a.storage.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error storage getting secrets: %w", err)
	}

	return secrets, nil
}

// GetSecret возвращает один секрет по ID.
func (a *SecretAdapter) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	return a.storage.GetSecret(ctx, id, userID)
}

// Delete удаляет секрет пользователя.
func (a *SecretAdapter) Delete(ctx context.Context, userID, secretID string) error {
	if userID == "" || secretID == "" {
		return domain.ErrInvalidInput
	}

	err := a.storage.Delete(ctx, userID, secretID)
	if err != nil {
		return fmt.Errorf("error storage deleting secret: %w", err)
	}

	return nil
}
