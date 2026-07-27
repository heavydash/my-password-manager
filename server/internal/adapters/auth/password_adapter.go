package auth

import (
	"context"
	"golang.org/x/crypto/argon2"
	"gophkeeper/server/internal/ports"
)

type passwordAdapter struct{}

func NewPasswordAdapter() ports.AuthPort {
	return &passwordAdapter{}
}

func (a *passwordAdapter) CreateUser(ctx context.Context, user ports.User) (string, error) {
	if user.PasswordHash != "" {
		user.PasswordHash = a.hashPassword(user.PasswordHash)
	}
	return "userID" + user.Email, nil
}

func (a *passwordAdapter) AuthenticatePassword(ctx context.Context, email, password string) (string, error) {
	return "jwt-token", nil
}

func (a *passwordAdapter) AuthenticateOAuth(ctx context.Context, provider, code string) (string, error) {
	return "", nil
}

func (a *passwordAdapter) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	return ports.User{}, nil
}

func (a *passwordAdapter) hashPassword(password string) string {
	salt := []byte("goph-salt")
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return string(hash)
}
