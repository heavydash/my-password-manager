package domain

import (
	"context"
	"gophkeeper/server/internal/ports"
)

// User - основная сущность пользователя
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Provider     string
	ProviderID   string
}

// AuthUseCase - вся бизнес логика аутентификации
type AuthUseCase struct {
	authPort ports.AuthPort
}

func NewAuthUseCase(authPort ports.AuthPort) *AuthUseCase {
	return &AuthUseCase{authPort: authPort}
}

func (u *AuthUseCase) Register(ctx context.Context, email, password string) (string, error) {
	user := User{
		Email:    email,
		Provider: "password",
	}
	if password != "" {
		user.PasswordHash = password
	}

	// Преобразуем domain.User в ports.User
	portUser := ports.User{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Provider:     user.Provider,
		ProviderID:   user.ProviderID,
	}

	return u.authPort.CreateUser(ctx, portUser)
}

func (u *AuthUseCase) LoginPassword(ctx context.Context, email, password string) (string, error) {
	return u.authPort.AuthenticatePassword(ctx, email, password)
}

func (u *AuthUseCase) LoginOAuth(ctx context.Context, provider, code string) (string, error) {
	return u.authPort.AuthenticateOAuth(ctx, provider, code)
}
