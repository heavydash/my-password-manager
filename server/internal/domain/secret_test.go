package domain

import (
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
			name:          "invalid secret type",
			userID:        "user-123",
			secretType:    "unknown",
			title:         "Test",
			encryptedData: "data",
			wantErr:       nil, // проверяем через assert.Contains ниже
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := NewSecret(tt.userID, tt.secretType, tt.title, tt.encryptedData)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			if tt.secretType == "unknown" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown secret type")
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
	adapter := JWTValidatorAdapter{
		ValidateFunc: func(token string) (string, error) {
			if token == "valid" {
				return "user-123", nil
			}
			return "", ErrTokenInvalid
		},
	}

	userID, err := adapter.ValidateToken("valid")
	assert.NoError(t, err)
	assert.Equal(t, "user-123", userID)

	_, err = adapter.ValidateToken("invalid")
	assert.ErrorIs(t, err, ErrTokenInvalid)
}
