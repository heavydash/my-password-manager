// Package ports содержит контракты (интерфейсы) между слоями приложения.
//
// Все зависимости идут от domain/usecases → ports → adapters.
package ports

import (
	"context"
)

// AuthPort — основной порт аутентификации.
//
// Реализуется адаптерами: passwordAdapter и OAuthAdapter.
type AuthPort interface {
	// CreateUser создаёт пользователя в хранилище.
	CreateUser(ctx context.Context, user User) (string, error)
	// AuthenticatePassword выполняет проверку email + пароль.
	AuthenticatePassword(ctx context.Context, email, password string) (token string, userID string, err error)
	// GetUserByID возвращает пользователя по ID.
	GetUserByID(ctx context.Context, id string) (User, error)
	// ValidateJWT проверяет JWT-токен и возвращает userID.
	ValidateJWT(tokenString string) (string, error)
	// GenerateJWT генерирует новый JWT-токен.
	GenerateJWT(userID string) (string, error)

	// Новые OAuth-методы

	// GetOAuthURL возвращает URL для редиректа на провайдера и state.
	GetOAuthURL(provider string) (authURL, state string, err error)

	// HandleCallback обрабатывает callback от OAuth-провайдера.
	HandleCallback(provider, code, state string) (oneTimeCode string, err error)

	// AuthenticateOAuth завершает OAuth-flow по one-time code.
	AuthenticateOAuth(ctx context.Context, oneTimeCode string) (token string, err error)
}
