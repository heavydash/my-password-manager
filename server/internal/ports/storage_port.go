// Package ports содержит контракты (интерфейсы) между слоями приложения.
//
// Все зависимости идут от domain/usecases → ports → adapters.
package ports

import (
	"context"
)

// User — общая минимальная структура для всех портов (AuthPort и StoragePort).
//
// Используется для передачи данных пользователя между слоями.
//
// Поля:
//   - ID: уникальный идентификатор пользователя (UUID)
//   - Email: email пользователя (уникальный)
//   - PasswordHash: хеш пароля (для password-провайдера)
//   - Provider: провайдер аутентификации (password, google, yandex)
//   - ProviderID: ID пользователя у провайдера (для OAuth)
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Provider     string
	ProviderID   string
}

// StoragePort — интерфейс для всех операций с хранилищем пользователей.
//
// Реализуется адаптерами:
//   - memory.UserStorage (in-memory)
//   - postgres.userStorage (PostgreSQL)
//
// Используется в AuthPort для управления пользователями.
type StoragePort interface {
	// CreateUser создаёт нового пользователя в хранилище.
	// Возвращает сгенерированный ID или ошибку.
	CreateUser(ctx context.Context, user User) (string, error)

	// GetUserByEmail возвращает пользователя по email.
	// Возвращает ErrUserNotFound, если пользователь не найден.
	GetUserByEmail(ctx context.Context, email string) (User, error)

	// GetUserByID возвращает пользователя по ID.
	// Возвращает ErrUserNotFound, если пользователь не найден.
	GetUserByID(ctx context.Context, id string) (User, error)
}
