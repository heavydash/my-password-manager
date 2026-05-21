// Package domain содержит тесты для бизнес-логики GophKeeper.
//
// Тестируются:
//   - NewSecret: создание доменной сущности секрета
//   - JWTValidatorAdapter: адаптер для валидации JWT
package domain

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewSecret(t *testing.T) {
	tests := []struct {
		name          string
		userID        string
		secretType    SecretType
		title         string
		encryptedData string
		wantErr       error
	}{
		{
			name:          "success password",
			userID:        "user-123",
			secretType:    SecretTypePassword,
			title:         "Main password",
			encryptedData: "base64data123==",
		},
		{
			name:          "success note",
			userID:        "user-123",
			secretType:    SecretTypeNote,
			title:         "My note",
			encryptedData: "note data",
			wantErr:       nil,
		},
		{
			name:       "empty userID",
			userID:     "",
			secretType: SecretTypeNote,
			title:      "Note",
			wantErr:    ErrInvalidInput,
		},
		{
			name:          "empty title",
			userID:        "user-123",
			secretType:    SecretTypeCard,
			title:         "",
			encryptedData: "data",
			wantErr:       ErrInvalidInput,
		},
		{
			name:          "empty data",
			userID:        "user-123",
			secretType:    SecretTypeCard,
			title:         "Card",
			encryptedData: "",
			wantErr:       ErrInvalidInput,
		},
		{
			name:          "invalid secret type",
			userID:        "user-123",
			secretType:    SecretType("unknown"),
			title:         "Test",
			encryptedData: "data",
			wantErr:       fmt.Errorf("unknown secret type: unknown"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := NewSecret(tt.userID, tt.secretType, tt.title, tt.encryptedData)

			if tt.wantErr != nil {
				assert.Error(t, err)
				if tt.wantErr.Error() != "" {
					assert.Contains(t, err.Error(), tt.wantErr.Error())
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, secret)
			assert.NotEmpty(t, secret.ID)
			assert.Equal(t, tt.userID, secret.UserID)
			assert.Equal(t, string(tt.secretType), secret.Type)
			assert.Equal(t, tt.title, secret.Title)
			assert.Equal(t, tt.encryptedData, secret.Data)
		})
	}
}

func TestJWTValidatorAdapter(t *testing.T) {
	// Создание адаптера с тестовой функцией валидации
	adapter := JWTValidatorAdapter{
		ValidateFunc: func(token string) (string, error) {
			if token == "valid" {
				return "user-123", nil
			}
			return "", ErrTokenInvalid
		},
	}

	// Успешная валидация
	userID, err := adapter.ValidateToken("valid")
	assert.NoError(t, err)
	assert.Equal(t, "user-123", userID)

	// Неуспешная валидация
	_, err = adapter.ValidateToken("invalid")
	assert.ErrorIs(t, err, ErrTokenInvalid)
}
