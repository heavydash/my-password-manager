package ports

import (
	"context"
	"time"
)

type Secret struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Data      string    `json:"data"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SecretPort — порт, через который UseCase общается с адаптером
type SecretPort interface {
	Create(ctx context.Context, secret *Secret) (*Secret, error)
	GetByUserID(ctx context.Context, userID string) ([]*Secret, error)
	GetSecret(ctx context.Context, id, userID string) (*Secret, error)
	Delete(ctx context.Context, userID, secretID string) error
}

// SecretStoragePort — порт для работы с хранилищем
type SecretStoragePort interface {
	Create(ctx context.Context, secret *Secret) (*Secret, error)
	GetByUserID(ctx context.Context, userID string) ([]*Secret, error)
	GetSecret(ctx context.Context, id, userID string) (*Secret, error)
	Delete(ctx context.Context, userID, secretID string) error
}
