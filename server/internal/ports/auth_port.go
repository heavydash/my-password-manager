package ports

import (
	"context"
)

type AuthPort interface {
	CreateUser(ctx context.Context, user User) (string, error)
	AuthenticatePassword(ctx context.Context, email, password string) (token string, userID string, err error)
	GetUserByID(ctx context.Context, id string) (User, error)
	ValidateJWT(tokenString string) (string, error)

	// Новые OAuth-методы
	GetOAuthURL(provider string) (authURL, state string, err error)
	HandleCallback(provider, code, state string) (oneTimeCode string, err error)
	AuthenticateOAuth(ctx context.Context, oneTimeCode string) (token string, err error)

	GenerateJWT(userID string) (string, error)
}
