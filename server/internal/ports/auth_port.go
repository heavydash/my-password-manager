package ports

import (
	"context"
)

// User — минимальная копия доменной сущности, что нужно порту
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Provider     string
	ProviderID   string
}

type AuthPort interface {
	CreateUser(ctx context.Context, user User) (string, error)
	AuthenticatePassword(ctx context.Context, email, password string) (string, error)
	AuthenticateOAuth(ctx context.Context, provider, code string) (string, error)
	GetUserByID(ctx context.Context, id string) (User, error)
}
