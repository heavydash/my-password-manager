// Package auth содержит адаптеры аутентификации для GophKeeper.
//
// Поддерживаются два провайдера:
//   - Password (email + пароль с Argon2 + JWT)
//   - OAuth2 (Google, Yandex)
//
// Все адаптеры реализуют интерфейс ports.AuthPort.
package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
	"gophkeeper/server/internal/config"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"time"
)

// passwordAdapter — адаптер парольной аутентификации.
//
// Реализует ports.AuthPort.
// Использует Argon2 для хеширования паролей и JWT (HS256) для токенов.
// Хранит пользователей через StoragePort.
type passwordAdapter struct {
	storage      ports.StoragePort
	jwtSecret    []byte
	jwtExpires   time.Duration
	argon2Salt   []byte
	argon2Iter   uint32
	argon2Mem    uint32
	argon2Par    uint8
	argon2KeyLen uint32
	logger       logger.Logger
}

// NewPasswordAdapter создаёт новый адаптер парольной аутентификации.
//
// Параметры:
//   - storage: порт для работы с хранилищем пользователей
//   - cfg: конфигурация JWT и Argon2
//   - log: логгер
//
// Возвращает объект, реализующий ports.AuthPort.
func NewPasswordAdapter(storage ports.StoragePort, cfg *config.Config, log logger.Logger) ports.AuthPort {
	return &passwordAdapter{
		storage:      storage,
		jwtSecret:    []byte(cfg.JWT.Secret),
		jwtExpires:   cfg.JWT.ExpiresIn,
		argon2Salt:   []byte(cfg.Argon2.Salt),
		argon2Iter:   cfg.Argon2.Iterations,
		argon2Mem:    cfg.Argon2.Memory,
		argon2Par:    cfg.Argon2.Parallelism,
		argon2KeyLen: cfg.Argon2.KeyLen,
		logger:       log,
	}
}

// CreateUser создаёт нового пользователя.
//
// Алгоритм:
//  1. Хеширует пароль через Argon2 (если передан)
//  2. Сохраняет пользователя через StoragePort
//
// Возвращает userID или domain.ErrUserAlreadyExists.
func (a *passwordAdapter) CreateUser(ctx context.Context, user ports.User) (string,
	error) {

	a.logger.Info("creating new user", zap.String("email", user.Email),
		zap.String("provider", user.Provider))

	// Преобразовывание ports.User в domain.User для StoragePort
	domainUser := domain.User{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Provider:     user.Provider,
		ProviderID:   user.ProviderID,
	}

	if domainUser.PasswordHash != "" {
		domainUser.PasswordHash = a.hashPassword(domainUser.PasswordHash)
	}

	userID, err := a.storage.CreateUser(ctx, ports.User{
		Email:        domainUser.Email,
		PasswordHash: domainUser.PasswordHash,
		Provider:     domainUser.Provider,
		ProviderID:   domainUser.ProviderID,
	})

	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			a.logger.Warn("user already exists", zap.String("email", user.Email))
			return "", domain.ErrUserAlreadyExists
		}
		a.logger.Error("failed to create user in storage", zap.String("email", user.Email), zap.Error(err))
		return "", fmt.Errorf("create user: %w", err)
	}

	a.logger.Info("user created successfully", zap.String("user_id", userID), zap.String("email", user.Email))
	return userID, nil
}

// AuthenticatePassword выполняет аутентификацию по email и паролю.
//
// 1. Находит пользователя по email
// 2. Проверяет пароль через Argon2
// 3. Генерирует JWT
// Возвращает token, userID или domain.ErrInvalidCredentials.
func (a *passwordAdapter) AuthenticatePassword(ctx context.Context, email, password string) (string, string, error) {
	a.logger.Info("password authentication attempt", zap.String("email", email))

	user, err := a.storage.GetUserByEmail(ctx, email)
	if err != nil {
		a.logger.Error("failed to get user by email", zap.String("email", email), zap.Error(err))
		return "", "", err
	}

	if !a.checkPassword(password, user.PasswordHash) {
		a.logger.Warn("invalid password", zap.String("email", email))
		return "", "", domain.ErrInvalidCredentials
	}

	token, err := a.generateJWT(user.ID)
	if err != nil {
		a.logger.Error("failed to generate JWT", zap.String("user_id", user.ID), zap.Error(err))
		return "", "", fmt.Errorf("generate token: %w", err)
	}

	a.logger.Info("password authentication successful",
		zap.String("user_id", user.ID),
		zap.String("email", email),
	)

	return token, user.ID, nil
}

// AuthenticateOAuth — заглушка для OAuth-аутентификации в passwordAdapter.
//
// В будущем может делегировать в OAuthAdapter.
// Сейчас возвращает токен для временного пользователя.
func (a *passwordAdapter) AuthenticateOAuth(ctx context.Context, oneTimeCode string) (token string, err error) {
	a.logger.Info("OAuth authentication started with oneTimeCode")

	token, err = a.generateJWT("temp-user-id")
	if err != nil {
		a.logger.Error("failed to generate JWT for OAuth", zap.Error(err))
		return "", err
	}

	a.logger.Info("OAuth authentication successful")
	return token, nil
}

// GetUserByID возвращает пользователя по ID.
//
// Делегирует вызов в StoragePort и преобразует domain.User обратно в ports.User.
func (a *passwordAdapter) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	a.logger.Debug("getting user by id", zap.String("user_id", id))

	domainUser, err := a.storage.GetUserByID(ctx, id)
	if err != nil {
		a.logger.Error("failed to get user by id", zap.String("user_id", id), zap.Error(err))
		return ports.User{}, err
	}

	return ports.User{
		ID:           domainUser.ID,
		Email:        domainUser.Email,
		PasswordHash: domainUser.PasswordHash,
		Provider:     domainUser.Provider,
		ProviderID:   domainUser.ProviderID,
	}, nil
}

// checkPassword сравнивает введённый пароль с хешем.
//
// Использует Argon2. Неэкспортированный метод.
func (a *passwordAdapter) checkPassword(password, hashed string) bool {
	expectedHash := a.hashPassword(password)
	return hashed == expectedHash
}

// generateJWT создаёт JWT-токен для пользователя.
//
// Claims содержат user_ID, exp (15 минут), iat.
// Подпись HS256. Неэкспортированный метод.
func (a *passwordAdapter) generateJWT(userID string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"user_ID": userID,
		"exp":     now.Add(a.jwtExpires).Unix(), // Ограничение
		"iat":     now.Unix(),                   // issued at - когда выдан
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		a.logger.Error("failed to sign JWT", zap.String("user_id", userID),
			zap.Error(err))
		return "", fmt.Errorf("sign token: %w", err)
	}
	return tokenString, nil
}

// ValidateJWT проверяет JWT-токен и извлекает user_ID.
//
// Поддерживает несколько вариантов ключа в claims ("user_id", "userID" и т.д.).
// Возвращает userID или domain.ErrTokenExpired / domain.ErrTokenInvalid.
func (a *passwordAdapter) ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Защита для HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			a.logger.Warn("token expired", zap.Error(err))
			return "", domain.ErrTokenExpired
		}
		a.logger.Warn("invalid token", zap.Error(err))
		return "", domain.ErrTokenInvalid
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		for _, key := range []string{"user_id", "userID", "user_ID", "userid"} {
			if id, ok := claims[key].(string); ok && id != "" {
				return id, nil
			}
		}
	}
	a.logger.Warn("token claims invalid or missing user_id")
	return "", domain.ErrTokenInvalid
}

// hashPassword хэширует пароль с помощью Argon2id.
//
// Использует фиксированный salt "gophkeeper-salt" (в production рекомендуется рандомный).
// Неэкспортированный метод.
func (a *passwordAdapter) hashPassword(password string) string {

	hash := argon2.IDKey([]byte(password),
		a.argon2Salt,
		a.argon2Iter,
		a.argon2Mem,
		a.argon2Par,
		a.argon2KeyLen)
	return base64.URLEncoding.EncodeToString(hash)
}

// GetOAuthURL — заглушка для OAuth-метода в passwordAdapter.
//
// Возвращает ошибку, т.к. OAuth обрабатывается в OAuthAdapter.
func (a *passwordAdapter) GetOAuthURL(provider string) (string, string, error) {
	return "", "", fmt.Errorf("OAuth not fully implemented in passwordAdapter")
}

// HandleCallback — заглушка для OAuth-callback в passwordAdapter.
func (a *passwordAdapter) HandleCallback(provider, code, state string) (string, error) {
	return "", fmt.Errorf("OAuth callback not implemented")
}

// HandleOAuthCallback обрабатывает OAuth callback (алиас для HandleCallback).
func (a *passwordAdapter) HandleOAuthCallback(provider, state, code string) (string, error) {
	return "", fmt.Errorf("password adapter does not support OAuth")
}

// GenerateJWT делегирует генерацию JWT (экспортированная обёртка).
func (a *passwordAdapter) GenerateJWT(userID string) (string, error) {
	return a.generateJWT(userID)
}

// Register создаёт нового пользователя по email и паролю.
func (a *passwordAdapter) Register(ctx context.Context, email, password string) (string, error) {
	return a.CreateUser(ctx, ports.User{
		Email:        email,
		PasswordHash: password,
		Provider:     "password",
		ProviderID:   "",
	})
}

// ExchangeOneTimeCode обменивает one_time_code на AuthResponse.
//
// Для passwordAdapter возвращает ошибку (OAuth не поддерживается).
func (a *passwordAdapter) ExchangeOneTimeCode(ctx context.Context, oneTimeCode string) (*domain.AuthResponse, error) {
	return nil, domain.ErrOAuthUnavailable
}

// LoginPassword выполняет вход по email и паролю.
func (a *passwordAdapter) LoginPassword(ctx context.Context, email, password string) (string, string, error) {
	return a.AuthenticatePassword(ctx, email, password)
}
