package domain

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gophkeeper/server/internal/ports"
	"testing"
)

type mockSecretPort struct{ mock.Mock }

func (m *mockSecretPort) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	args := m.Called(ctx, secret)
	return args.Get(0).(*ports.Secret), args.Error(1)
}

func (m *mockSecretPort) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*ports.Secret), args.Error(1)
}

func (m *mockSecretPort) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	args := m.Called(ctx, id, userID)
	return args.Get(0).(*ports.Secret), args.Error(1)
}

func (m *mockSecretPort) Delete(ctx context.Context, userID, secretID string) error {
	args := m.Called(ctx, userID, secretID)
	return args.Error(0)
}

type mockTokenValidator struct{ mock.Mock }

func (m *mockTokenValidator) ValidateToken(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

func TestSecretUseCase_CreateSecret(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		secretType SecretType
		title      string
		data       string
		wantErr    error
	}{
		{
			name:       "success",
			userID:     "user-123",
			secretType: SecretTypePassword,
			title:      "Bank password",
			data:       "base64encrypteddata123==",
		},
		{
			name:    "empty userID",
			userID:  "",
			wantErr: ErrInvalidInput,
		},
		{
			name:       "empty title",
			userID:     "user-123",
			secretType: SecretTypeNote,
			title:      "",
			wantErr:    ErrInvalidInput,
		},
		{
			name:       "invalid secret type",
			userID:     "user-123",
			secretType: "unknown",
			wantErr:    ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPort := &mockSecretPort{}
			if tt.wantErr == nil {
				mockPort.On("Create", mock.Anything, mock.AnythingOfType("*ports.Secret")).
					Return(&ports.Secret{ID: "secret-456"}, nil)
			}

			uc := NewSecretUseCase(mockPort, &mockTokenValidator{})

			secret, err := uc.CreateSecret(context.Background(), tt.userID, tt.secretType, tt.title, tt.data)

			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil {
				assert.NotNil(t, secret)
			}
			mockPort.AssertExpectations(t)
		})
	}
}

func TestSecretUseCase_GetSecrets(t *testing.T) {
	mockPort := &mockSecretPort{}
	mockPort.On("GetByUserID", mock.Anything, "user-123").
		Return([]*ports.Secret{{ID: "s1"}, {ID: "s2"}}, nil)

	uc := NewSecretUseCase(mockPort, &mockTokenValidator{})

	secrets, err := uc.GetSecrets(context.Background(), "user-123")

	assert.NoError(t, err)
	assert.Len(t, secrets, 2)
	mockPort.AssertExpectations(t)
}
