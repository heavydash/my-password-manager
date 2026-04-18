package domain

import (
	"context"
	"errors"
	"fmt"
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
	if authPort == nil {
		panic("auth cannot be nil")
	}
	return &AuthUseCase{authPort: authPort}
}

func (u *AuthUseCase) Register(ctx context.Context, email, password string) (string, error) {
	if email == "" {
		return "", errors.New("email is required")
	}
	// Создаем чистую доменную сущность
	domainUser := User{
		Email:    email,
		Provider: "password",
	}
	if password != "" {
		domainUser.PasswordHash = password
	}

	// Преобразуем domain.User в ports.User
	portUser := ports.User{
		Email:        domainUser.Email,
		PasswordHash: domainUser.PasswordHash,
		Provider:     domainUser.Provider,
		ProviderID:   domainUser.ProviderID,
	}

	userID, err := u.authPort.CreateUser(ctx, portUser)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			return "", ErrUserAlreadyExists
		}
		return "", fmt.Errorf("create user: %w", err)
	}
	return userID, nil
}

func (u *AuthUseCase) LoginPassword(ctx context.Context, email, password string) (string, error) {
	// Валидация входных данных
	if email == "" || password == "" {
		return "", errors.New("email or password is required")
	}

	// Генерация токена JWT
	token, err := u.authPort.AuthenticatePassword(ctx, email, password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) ||
			errors.Is(err, ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}

		return "", fmt.Errorf("authenticate password: %w", err)
	}
	return token, nil
}

func (u *AuthUseCase) LoginOAuth(ctx context.Context, provider, code string) (string, error) {
	return u.authPort.AuthenticateOAuth(ctx, provider, code)
}
