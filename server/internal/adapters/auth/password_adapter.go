package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"time"
)

type passwordAdapter struct {
	storage   ports.StoragePort
	jwtSecret []byte
	logger    logger.Logger
}

func NewPasswordAdapter(storage ports.StoragePort, jwtSecret string, log logger.Logger) ports.AuthPort {
	return &passwordAdapter{
		storage:   storage,
		jwtSecret: []byte(jwtSecret),
		logger:    log,
	}
}

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

func (a *passwordAdapter) AuthenticateOAuth(ctx context.Context, provider, code string) (string, error) {
	a.logger.Info("OAuth authentication started",
		zap.String("provider", provider),
	)

	a.logger.Warn("OAuth provider not implemented yet", zap.String("provider", provider))
	return "", nil
}

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

func (a *passwordAdapter) checkPassword(password, hashed string) bool {
	expectedHash := a.hashPassword(password)
	return hashed == expectedHash
}

func (a *passwordAdapter) generateJWT(userID string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"user_ID": userID,
		"exp":     now.Add(15 * time.Minute).Unix(), // Ограничение
		"iat":     now.Unix(),                       // issued at - когда выдан
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

func (a *passwordAdapter) hashPassword(password string) string {
	salt := []byte("gophkeeper-salt")
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return base64.URLEncoding.EncodeToString(hash)
}
