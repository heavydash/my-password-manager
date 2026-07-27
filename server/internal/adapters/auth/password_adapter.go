package auth

import (
	"context"
	"encoding/base64"
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

func (a *passwordAdapter) AuthenticatePassword(ctx context.Context, email, password string) (string, error) {

	user, err := a.storage.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	if !a.checkPassword(password, user.PasswordHash) {
		return "", domain.ErrInvalidCredentials
	}

	return a.generateJWT(user.ID)
}

func (a *passwordAdapter) AuthenticateOAuth(ctx context.Context, provider, code string) (string, error) {
	return "", nil
}

func (a *passwordAdapter) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	domainUser, err := a.GetUserByID(ctx, id)
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
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString(a.jwtSecret)
}

func (a *passwordAdapter) hashPassword(password string) string {
	salt := []byte("gophkeeper-salt")
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return base64.URLEncoding.EncodeToString(hash)
}
