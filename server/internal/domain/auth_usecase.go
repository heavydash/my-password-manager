package domain

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gophkeeper/server/internal/logger"
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
	logger   logger.Logger
}

func NewAuthUseCase(authPort ports.AuthPort, log logger.Logger) *AuthUseCase {
	if log == nil {
		panic("logger is required")
	}
	if authPort == nil {
		panic("auth cannot be nil")
	}
	return &AuthUseCase{authPort: authPort,
		logger: log}
}

func (u *AuthUseCase) Register(ctx context.Context, email, password string) (string, error) {
	if email == "" {
		u.logger.Warn("registration failed: email is required")
		return "", errors.New("email is required")
	}

	u.logger.Info("attempting to register user", zap.String("email", email))

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
			u.logger.Warn("registration failed: user already exists",
				zap.String("email", email))
			return "", ErrUserAlreadyExists
		}
		u.logger.Error("failed to create user", zap.String("email", email),
			zap.Error(err))
		return "", fmt.Errorf("create user: %w", err)
	}

	u.logger.Info("user registered successfully", zap.String("user_id", userID),
		zap.String("email", email))
	return userID, nil
}

func (u *AuthUseCase) LoginPassword(ctx context.Context, email, password string) (string, string, error) {
	// Валидация входных данных
	if email == "" || password == "" {
		u.logger.Warn("login failed: email or password is empty")
		return "", "", errors.New("email or password is required")
	}

	u.logger.Info("attempting login", zap.String("email", email))

	// Генерация токена JWT
	token, userID, err := u.authPort.AuthenticatePassword(ctx, email, password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) ||
			errors.Is(err, ErrUserNotFound) {
			u.logger.Warn("login failed: invalid credentials", zap.String("email", email))
			return "", "", ErrInvalidCredentials
		}

		u.logger.Error("authentication error", zap.String("email", email), zap.Error(err))
		return "", "", fmt.Errorf("authenticate password: %w", err)
	}

	u.logger.Info("user logged in successfully",
		zap.String("user_id", userID),
		zap.String("email", email),
	)

	return token, userID, nil
}

func (u *AuthUseCase) GetOAuthURL(provider string) (string, string, error) {
	u.logger.Info("generating OAuth URL", zap.String("provider", provider))
	return u.authPort.GetOAuthURL(provider)
}

func (u *AuthUseCase) HandleOAuthCallback(provider, code, state string) (string, error) {
	u.logger.Info("handling OAuth callback", zap.String("provider", provider))
	return u.authPort.HandleCallback(provider, code, state)
}

func (u *AuthUseCase) LoginOAuth(ctx context.Context, oneTimeCode string) (string, error) {
	u.logger.Info("completing OAuth login")
	token, err := u.authPort.AuthenticateOAuth(ctx, oneTimeCode)
	if err != nil {
		u.logger.Error("OAuth login failed", zap.Error(err))
		return "", err
	}
	return token, nil
}
