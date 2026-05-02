package domain

import (
	"context"
	"fmt"
	"gophkeeper/server/internal/ports"
)

type SecretUseCase struct {
	secretPort     ports.SecretPort
	TokenValidator TokenValidator
}

func NewSecretUseCase(SecretPort ports.SecretPort, validator TokenValidator) *SecretUseCase {
	if SecretPort == nil {
		panic("secret port is nil")
	}
	return &SecretUseCase{
		secretPort:     SecretPort,
		TokenValidator: validator,
	}
}

func (uc *SecretUseCase) CreateSecret(ctx context.Context, userID string, SecretType SecretType,
	title string, encryptedDataBase64 string) (*ports.Secret, error) {

	if userID == "" || title == "" || len(encryptedDataBase64) == 0 {
		return nil, ErrInvalidInput
	}

	switch SecretType {
	case "password", "note", "card", "ssh_key", "custom":
	default:
		return nil, ErrInvalidInput
	}

	secret, err := NewSecret(userID, SecretType, title, encryptedDataBase64)
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

func (uc *SecretUseCase) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	return uc.secretPort.GetSecret(ctx, id, userID)
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

func (uc *SecretUseCase) GetTokenValidator() TokenValidator {
	return uc.TokenValidator
}
