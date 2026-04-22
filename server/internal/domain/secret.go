package domain

import (
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

func NewSecret(userID string,
	secretType string,
	title string,
	encryptedData []byte) (*ports.Secret, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	if title == "" {
		return nil, ErrInvalidInput
	}

	if len(encryptedData) == 0 {
		return nil, ErrInvalidInput
	}

	now := time.Now()

	return &ports.Secret{
		ID:        generateID(),
		UserID:    userID,
		Type:      secretType,
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
