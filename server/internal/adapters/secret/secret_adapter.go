package secret

import (
	"context"
	"fmt"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
)

type SecretAdapter struct {
	storage ports.SecretStoragePort
	logger  logger.Logger
}

func NewSecretAdapter(storage ports.SecretStoragePort, log logger.Logger) *SecretAdapter {
	if storage == nil {
		panic("storage is nil")
	}
	if log == nil {
		panic("logger is nil")
	}
	return &SecretAdapter{
		storage: storage,
		logger:  log,
	}
}

func (a *SecretAdapter) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	if secret == nil {
		return nil, domain.ErrInvalidInput
	}

	created, err := a.storage.Create(ctx, secret)
	if err != nil {
		return nil, fmt.Errorf("error storage creating secret: %w", err)
	}
	return created, nil

}

func (a *SecretAdapter) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {

	if userID == "" {
		return nil, domain.ErrInvalidInput
	}

	secrets, err := a.storage.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("error storage getting secrets: %w", err)
	}

	return secrets, nil
}

func (a *SecretAdapter) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	return a.storage.GetSecret(ctx, id, userID)
}

func (a *SecretAdapter) Delete(ctx context.Context, userID, secretID string) error {
	if userID == "" || secretID == "" {
		return domain.ErrInvalidInput
	}

	err := a.storage.Delete(ctx, userID, secretID)
	if err != nil {
		return fmt.Errorf("error storage deleting secret: %w", err)
	}

	return nil
}
