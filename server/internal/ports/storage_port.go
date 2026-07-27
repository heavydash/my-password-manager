package ports

import (
	"context"
)

// User — общая минимальная структура для всех портов (AuthPort и StoragePort)
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Provider     string
	ProviderID   string
}

// StoragePort - интерфейс для всех операций с хранилищем
type StoragePort interface {
	CreateUser(ctx context.Context, user User) (string, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, id string) (User, error)
}
