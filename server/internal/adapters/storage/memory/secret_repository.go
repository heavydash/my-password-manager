// Package memory реализует in-memory хранилище секретов.
//
// Используется для:
//   - Unit и integration тестов
//   - Development окружения
//   - Fallback при недоступности БД
//
// Особенности:
//   - Потокобезопасность через sync.RWMutex
//   - Хранение данных в map[string]*ports.Secret
package memory

import (
	"context"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"sync"
)

// secretStorage реализует in-memory хранилище секретов.
//
// Реализует ports.SecretStoragePort.
type secretStorage struct {
	secrets map[string]*ports.Secret
	logger  logger.Logger
	mu      sync.RWMutex
}

// NewSecretStorage создаёт новое in-memory хранилище секретов.
//
// Параметры:
//   - log: логгер для записи событий
//
// Паникует, если log == nil.
//
// Возвращает реализацию ports.SecretStoragePort.
func NewSecretStorage(log logger.Logger) ports.SecretStoragePort {
	if log == nil {
		panic("logger is required for secretStorage")
	}

	return &secretStorage{
		secrets: make(map[string]*ports.Secret),
		logger:  log,
	}
}

// Create сохраняет новый секрет в памяти.
//
// Алгоритм:
//  1. Проверяет, что secret не nil
//  2. Создаёт domain.Secret с валидацией через domain.NewSecret
//  3. Сохраняет секрет в map (потокобезопасно)
//
// Возвращает созданный секрет с заполненным ID или ошибку.
func (s *secretStorage) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {

	// Проверка отмены контекста
	select {
	case <-ctx.Done():
		s.logger.Warn("memory storage: create cancelled", zap.Error(ctx.Err()))
		return nil, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Валидация входных данных
	if secret == nil {
		s.logger.Warn("memory storage: attempt to create nil secret")
		return nil, domain.ErrInvalidInput
	}

	// Создание доменной модели с валидацией и генерацией ID
	created, err := domain.NewSecret(
		secret.UserID,
		domain.SecretType(secret.Type),
		secret.Title,
		secret.Data,
	)
	if err != nil {
		s.logger.Warn("domain validation failed", zap.Error(err))
		return nil, err
	}

	// Конвертация domain.Secret → ports.Secret
	result := &ports.Secret{
		ID:        created.ID,
		UserID:    created.UserID,
		Type:      created.Type,
		Title:     created.Title,
		Data:      created.Data,
		Metadata:  created.Metadata,
		CreatedAt: created.CreatedAt,
		UpdatedAt: created.UpdatedAt,
	}

	s.secrets[result.ID] = result

	s.logger.Info("secret created",
		zap.String("secret_id", result.ID),
		zap.String("user_id", result.UserID),
		zap.String("type", result.Type),
		zap.String("title", result.Title),
	)

	return result, nil
}

// GetByUserID возвращает все секреты пользователя.
//
// Алгоритм:
//  1. Проходит по всем секретам в map
//  2. Фильтрует по UserID
//  3. Возвращает копии секретов
func (s *secretStorage) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {

	// Проверка отмены контекста
	select {
	case <-ctx.Done():
		s.logger.Warn("memory storage: get by user id cancelled", zap.Error(ctx.Err()))
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if userID == "" {
		return nil, domain.ErrInvalidInput
	}

	var result []*ports.Secret

	for _, secret := range s.secrets {
		if secret.UserID == userID {
			// Возвращаем копию для безопасности
			copySecret := *secret
			result = append(result, &copySecret)
		}
	}

	s.logger.Info("secrets retrieved for user",
		zap.String("user_id", userID),
		zap.Int("count", len(result)),
	)

	return result, nil

}

// GetSecret возвращает один секрет по ID и userID.
//
// Алгоритм:
//  1. Проверяет существование секрета в map
//  2. Проверяет принадлежность пользователю
//  3. Возвращает копию секрета
func (s *secretStorage) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {

	// Проверка отмены контекста
	select {
	case <-ctx.Done():
		s.logger.Warn("memory storage: get secret cancelled", zap.Error(ctx.Err()))
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	secret, exists := s.secrets[id]
	if !exists {
		s.logger.Warn("memory storage: secret not found",
			zap.String("secret_id", id),
			zap.String("user_id", userID),
		)
		return nil, domain.ErrNotFound
	}

	if secret.UserID != userID {
		s.logger.Warn("memory storage: access denied to secret",
			zap.String("secret_id", id),
			zap.String("user_id", userID),
			zap.String("owner_id", secret.UserID),
		)
		return nil, domain.ErrNotFound
	}

	// Возвращаем копию для безопасности
	copySecret := *secret
	return &copySecret, nil
}

// Delete удаляет секрет пользователя.
//
// Алгоритм:
//  1. Проверяет существование секрета в map
//  2. Проверяет принадлежность пользователю
//  3. Удаляет секрет из map
func (s *secretStorage) Delete(ctx context.Context, userID, secretID string) error {

	// Проверка отмены контекста
	select {
	case <-ctx.Done():
		s.logger.Warn("memory storage: delete cancelled", zap.Error(ctx.Err()))
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	secret, exists := s.secrets[secretID]
	if !exists {
		s.logger.Warn("memory storage: delete failed - secret not found",
			zap.String("secret_id", secretID),
			zap.String("user_id", userID),
		)
		return domain.ErrNotFound
	}

	if secret.UserID != userID {
		s.logger.Warn("memory storage: delete failed - access denied",
			zap.String("secret_id", secretID),
			zap.String("user_id", userID),
			zap.String("owner_id", secret.UserID),
		)
		return domain.ErrNotFound
	}

	delete(s.secrets, secretID)

	s.logger.Info("secret deleted successfully",
		zap.String("secret_id", secretID),
		zap.String("user_id", userID),
	)

	return nil
}
