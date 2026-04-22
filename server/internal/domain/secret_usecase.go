package domain

import (
	"context"
	"fmt"
	"gophkeeper/server/internal/ports"
)

type SecretUseCase struct {
	secretPort ports.SecretPort
}

func NewSecretUseCase(SecretPort ports.SecretPort) *SecretUseCase {
	if SecretPort == nil {
		panic("secret port is nil")
	}
	return &SecretUseCase{
		secretPort: SecretPort,
	}
}

func (uc *SecretUseCase) CreateSecret(ctx context.Context, userID string, SecretType string,
	title string, encryptedData []byte) (*ports.Secret, error) {

	if userID == "" || title == "" || len(encryptedData) == 0 {
		return nil, ErrInvalidInput
	}

	switch SecretType {
	case "password", "note", "card", "ssh_key", "custom":
	// разрешенные типы
	default:
		return nil, ErrInvalidInput
	}

	secret, err := NewSecret(userID, SecretType, title, encryptedData)
	if err != nil {
		return nil, err
	}

	createdSecret, err := uc.secretPort.Create(ctx, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	return createdSecret, nil
}

func (uc *SecretUseCase) GetSecrets(ctx context.Context, userID string) ([]*ports.Secret, error) {

	if userID == "" {
		return nil, ErrInvalidInput
	}

	secrets, err := uc.secretPort.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	}

	return secrets, nil
}

func (uc *SecretUseCase) DeleteSecret(ctx context.Context, userID, secretID string) error {

	if userID == "" || secretID == "" {
		return ErrInvalidInput
	}

	err := uc.secretPort.Delete(ctx, userID, secretID)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	return nil
}
