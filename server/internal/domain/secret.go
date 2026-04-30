package domain

import (
	"fmt"
	"gophkeeper/server/internal/ports"
	"time"
)

type SecretType string

const (
	SecretTypePassword SecretType = "password"
	SecretTypeNote     SecretType = "note"
	SecretTypeCard     SecretType = "card"
	SecretTypeSSHKey   SecretType = "ssh_key"
	SecretTypeCustom   SecretType = "custom"
)

func NewSecret(userID string, secretType SecretType, title string, encryptedData string) (*ports.Secret, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	if title == "" {
		return nil, ErrInvalidInput
	}

	if len(encryptedData) == 0 {
		return nil, ErrInvalidInput
	}

	// Валидация типа
	switch secretType {
	case SecretTypePassword, SecretTypeNote, SecretTypeCard, SecretTypeSSHKey, SecretTypeCustom:
		// разрешённые типы
	default:
		return nil, fmt.Errorf("unknown secret type: %s", secretType)
	}

	now := time.Now()

	return &ports.Secret{
		ID:        generateID(),
		UserID:    userID,
		Type:      string(secretType),
		Title:     title,
		Data:      encryptedData,
		Metadata:  "",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func generateID() string {
	return "" + time.Now().UTC().Format("20060102150405")
}
