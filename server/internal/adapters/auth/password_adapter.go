package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/ports"
	"time"
)

type passwordAdapter struct {
	storage   ports.StoragePort
	jwtSecret []byte
}

func NewPasswordAdapter(storage ports.StoragePort, jwtSecret string) ports.AuthPort {
	return &passwordAdapter{
		storage:   storage,
		jwtSecret: []byte(jwtSecret),
	}
}

func (a *passwordAdapter) CreateUser(ctx context.Context, user ports.User) (string,
	error) {
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
	// Сохраняем в БД через StoragePort
	return a.storage.CreateUser(ctx, ports.User{
		Email:        domainUser.Email,
		PasswordHash: domainUser.PasswordHash,
		Provider:     domainUser.Provider,
		ProviderID:   domainUser.ProviderID,
	})
}

func (a *passwordAdapter) AuthenticatePassword(ctx context.Context, email, password string) (string, string, error) {

	user, err := a.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return "", "", err
	}

	if !a.checkPassword(password, user.PasswordHash) {
		return "", "", domain.ErrInvalidCredentials
	}

	token, err := a.generateJWT(user.ID)
	if err != nil {
		return "", "", err
	}

	return token, user.ID, nil
}

func (a *passwordAdapter) AuthenticateOAuth(ctx context.Context, provider, code string) (string, error) {
	return "", nil
}

func (a *passwordAdapter) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	domainUser, err := a.storage.GetUserByID(ctx, id)
	if err != nil {
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
			return "", domain.ErrTokenExpired
		}
		return "", domain.ErrTokenInvalid
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		for _, key := range []string{"user_id", "userID", "user_ID", "userid"} {
			if id, ok := claims[key].(string); ok && id != "" {
				return id, nil
			}
		}
	}

	return "", domain.ErrTokenInvalid
}

func (a *passwordAdapter) hashPassword(password string) string {
	salt := []byte("gophkeeper-salt")
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return base64.URLEncoding.EncodeToString(hash)
}
