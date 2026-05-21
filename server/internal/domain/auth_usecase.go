// Package domain содержит бизнес-логику GophKeeper.
//
// Содержит:
//   - AuthUseCase: аутентификация и регистрация
//   - SecretUseCase: управление секретами
//   - Ошибки домена (ErrInvalidCredentials, ErrUserAlreadyExists и т.д.)
package domain

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
)

// User — основная доменная сущность пользователя.
//
// Используется во всех usecases.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Provider     string
	ProviderID   string
}

// AuthUseCase — бизнес-логика аутентификации.
//
// Содержит все правила регистрации, логина (password + OAuth).
// Не зависит от БД и HTTP — только от ports.AuthPort.
type AuthUseCase struct {
	authPort ports.AuthPort
	logger   logger.Logger
}

// NewAuthUseCase создаёт usecase аутентификации.
//
// Паникует, если переданы nil (fail-fast).
func NewAuthUseCase(authPort ports.AuthPort, log logger.Logger) *AuthUseCase {
	if log == nil {
		log = &logger.ZapLogger{}
	}
	if authPort == nil {
		log.Error("authPort is required")
		return nil
	}
	return &AuthUseCase{authPort: authPort,
		logger: log}
}

// Register регистрирует нового пользователя.
//
// Алгоритм:
//  1. Валидирует email
//  2. Создаёт доменную сущность
//  3. Делегирует создание в authPort
//
// Возвращает userID или ошибку.
func (u *AuthUseCase) Register(ctx context.Context, email, password string) (string, error) {
	// Валидация входных данных
	if email == "" {
		u.logger.Warn("registration failed: email is required")
		return "", ErrEmailRequired
	}

	u.logger.Info("attempting to register user", zap.String("email", email))

	// Создание пользователя
	portUser := ports.User{
		Email:    email,
		Provider: "password",
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

// LoginPassword логинит пользователя по email + паролю.
//
// Алгоритм:
//  1. Валидирует входные данные
//  2. Делегирует аутентификацию в authPort
//
// Возвращает JWT-токен и userID.
func (u *AuthUseCase) LoginPassword(ctx context.Context, email, password string) (string, string, error) {
	// Валидация входных данных
	if email == "" || password == "" {
		u.logger.Warn("login failed: email or password is empty")
		return "", "", ErrInvalidInput
	}

	u.logger.Info("attempting login", zap.String("email", email))

	// Аутентификация
	token, userID, err := u.authPort.AuthenticatePassword(ctx, email, password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) ||
			errors.Is(err, ErrUserNotFound) {
			u.logger.Warn("domain: login failed - invalid credentials",
				zap.String("email", email))
			return "", "", ErrInvalidCredentials
		}

		u.logger.Error("domain: authentication error",
			zap.String("email", email),
			zap.Error(err),
		)
		return "", "", fmt.Errorf("authenticate password: %w", err)
	}

	u.logger.Info("domain: user logged in successfully",
		zap.String("user_id", userID),
		zap.String("email", email),
	)

	return token, userID, nil
}

// GetOAuthURL возвращает URL для OAuth-провайдера.
func (u *AuthUseCase) GetOAuthURL(provider string) (string, string, error) {
	u.logger.Info("domain: generating OAuth URL", zap.String("provider", provider))
	return u.authPort.GetOAuthURL(provider)
}

// HandleOAuthCallback обрабатывает callback от OAuth-провайдера.
func (u *AuthUseCase) HandleOAuthCallback(provider, code, state string) (string, error) {
	u.logger.Info("domain: handling OAuth callback", zap.String("provider", provider))
	return u.authPort.HandleCallback(provider, code, state)
}

// LoginOAuth завершает OAuth-аутентификацию по authorization code.
//
// Алгоритм:
//  1. Принимает code от клиента (authorization code от провайдера)
//  2. Делегирует аутентификацию в authPort
//
// Возвращает JWT-токен или ошибку.
func (u *AuthUseCase) LoginOAuth(ctx context.Context, code string) (string, error) {
	u.logger.Info("domain: completing OAuth login")
	token, err := u.authPort.AuthenticateOAuth(ctx, code)
	if err != nil {
		u.logger.Error("domain: OAuth login failed", zap.Error(err))
		return "", err
	}
	return token, nil
}

// CreateUser — делегирует создание пользователя в authPort.
func (u *AuthUseCase) CreateUser(ctx context.Context, user ports.User) (string, error) {
	return u.authPort.CreateUser(ctx, user)
}

// AuthenticatePassword — делегирует парольную аутентификацию в authPort.
func (u *AuthUseCase) AuthenticatePassword(ctx context.Context, email, password string) (string, string, error) {
	return u.authPort.AuthenticatePassword(ctx, email, password)
}

// GetUserByID — делегирует получение пользователя в authPort.
func (u *AuthUseCase) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	return u.authPort.GetUserByID(ctx, id)
}

// ValidateJWT — делегирует валидацию JWT в authPort.
func (u *AuthUseCase) ValidateJWT(tokenString string) (string, error) {
	return u.authPort.ValidateJWT(tokenString)
}

// GenerateJWT — делегирует генерацию JWT в authPort.
func (u *AuthUseCase) GenerateJWT(userID string) (string, error) {
	return u.authPort.GenerateJWT(userID)
}

// AuthenticateOAuth — делегирует OAuth-аутентификацию в authPort.
func (u *AuthUseCase) AuthenticateOAuth(ctx context.Context, code string) (string, error) {
	return u.authPort.AuthenticateOAuth(ctx, code)
}

// HandleCallback — делегирует обработку OAuth callback в authPort.
func (u *AuthUseCase) HandleCallback(provider, code, state string) (string, error) {
	return u.authPort.HandleCallback(provider, code, state)
}
