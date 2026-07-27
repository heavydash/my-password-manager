// Package memory реализует in-memory хранилище пользователей.
//
// Используется для:
//   - Unit и integration тестов
//   - Development окружения
//   - Fallback при недоступности БД
//
// Особенности:
//   - Потокобезопасность через sync.RWMutex
//   - Хранение данных в map[string]ports.User
package memory

import (
	"context"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"sync"
)

// UserStorage реализует in-memory хранилище пользователей.
//
// Реализует ports.StoragePort.
type UserStorage struct {
	users  map[string]ports.User
	logger logger.Logger
	mu     sync.RWMutex
}

// NewUserRepository создаёт новое in-memory хранилище пользователей.
//
// Параметры:
//   - log: логгер для записи событий
//
// Паникует, если log == nil.
//
// Возвращает реализацию ports.StoragePort.
func NewUserRepository(log logger.Logger) ports.StoragePort {
	if log == nil {
		panic("logger is required for userStorage")
	}

	return &UserStorage{
		users:  make(map[string]ports.User),
		logger: log,
	}
}

// CreateUser создаёт нового пользователя.
//
// Алгоритм:
//  1. Проверяет отмену контекста
//  2. Проверяет, что email не пустой
//  3. Проверяет, что email не занят
//  4. Генерирует ID если пустой
//  5. Сохраняет пользователя в map
//
// Возвращает userID или ошибку.
func (s *UserStorage) CreateUser(ctx context.Context, user ports.User) (string, error) {

	// Проверка отмены контекста
	select {
	case <-ctx.Done():
		s.logger.Warn("memory storage: create user cancelled", zap.Error(ctx.Err()))
		return "", ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Валидация входных данных
	if user.Email == "" {
		s.logger.Warn("memory storage: create user with empty email")
		return "", domain.ErrInvalidInput
	}

	// Проверка на дубликат email
	for _, u := range s.users {
		if u.Email == user.Email {
			s.logger.Warn("memory storage: user already exists", zap.String("email", user.Email))
			return "", domain.ErrUserAlreadyExists
		}
	}

	// Сохранение
	s.users[user.ID] = user

	s.logger.Info("user created",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email))

	return user.ID, nil
}

// GetUserByEmail возвращает пользователя по email.
//
// Алгоритм:
//  1. Проверяет отмену контекста
//  2. Перебирает всех пользователей в поиске email
func (s *UserStorage) GetUserByEmail(ctx context.Context, email string) (ports.User, error) {

	// Проверка отмены контекста
	select {
	case <-ctx.Done():
		s.logger.Warn("memory storage: get user by email cancelled", zap.Error(ctx.Err()))
		return ports.User{}, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Email == email {
			s.logger.Debug("memory storage: user found by email",
				zap.String("user_id", user.ID),
				zap.String("email", email),
			)
			return user, nil
		}
	}

	s.logger.Warn("memory storage: user not found by email", zap.String("email", email))
	return ports.User{}, domain.ErrUserNotFound
}

// GetUserByID возвращает пользователя по ID.
//
// Алгоритм:
//  1. Проверяет отмену контекста
//  2. Ищет пользователя в map по ID
func (s *UserStorage) GetUserByID(ctx context.Context, id string) (ports.User, error) {

	// Проверка отмены контекста
	select {
	case <-ctx.Done():
		s.logger.Warn("memory storage: get user by id cancelled", zap.Error(ctx.Err()))
		return ports.User{}, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		s.logger.Warn("memory storage: user not found by id", zap.String("user_id", id))
		return ports.User{}, domain.ErrUserNotFound
	}

	s.logger.Debug("memory storage: user found by id",
		zap.String("user_id", user.ID),
		zap.String("email", user.Email),
	)

	return user, nil
}
