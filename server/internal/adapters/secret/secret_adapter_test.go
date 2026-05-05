package secret

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"testing"
)

type mockSecretStoragePort struct{ mock.Mock }

func (m *mockSecretStoragePort) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	args := m.Called(ctx, secret)
	return args.Get(0).(*ports.Secret), args.Error(1)
}

func (m *mockSecretStoragePort) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*ports.Secret), args.Error(1)
}

func (m *mockSecretStoragePort) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	args := m.Called(ctx, id, userID)
	return args.Get(0).(*ports.Secret), args.Error(1)
}

func (m *mockSecretStoragePort) Delete(ctx context.Context, userID, secretID string) error {
	args := m.Called(ctx, userID, secretID)
	return args.Error(0)
}

type mockLogger struct{ mock.Mock }

func (m *mockLogger) Info(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Warn(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Error(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Debug(msg string, fields ...zap.Field)  {}
func (m *mockLogger) With(fields ...zap.Field) logger.Logger { return m }
func (m *mockLogger) Sync() error                            { return nil }

func TestSecretAdapter_Create(t *testing.T) {
	mockStorage := &mockSecretStoragePort{}
	mockStorage.On("Create", mock.Anything, mock.AnythingOfType("*ports.Secret")).
		Return(&ports.Secret{ID: "secret-456"}, nil)

	adapter := NewSecretAdapter(mockStorage, &mockLogger{})

	secret := &ports.Secret{
		UserID: "user-123",
		Title:  "Test secret",
		Data:   "encrypteddata",
		Type:   "password",
	}

	created, err := adapter.Create(context.Background(), secret)

	assert.NoError(t, err)
	assert.Equal(t, "secret-456", created.ID)
	mockStorage.AssertExpectations(t)
}

func TestSecretAdapter_Create_NilSecret(t *testing.T) {
	adapter := NewSecretAdapter(&mockSecretStoragePort{}, &mockLogger{})

	_, err := adapter.Create(context.Background(), nil)

	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestSecretAdapter_GetByUserID(t *testing.T) {
	mockStorage := &mockSecretStoragePort{}
	mockStorage.On("GetByUserID", mock.Anything, "user-123").
		Return([]*ports.Secret{{ID: "s1"}, {ID: "s2"}}, nil)

	adapter := NewSecretAdapter(mockStorage, &mockLogger{})

	secrets, err := adapter.GetByUserID(context.Background(), "user-123")

	assert.NoError(t, err)
	assert.Len(t, secrets, 2)
	mockStorage.AssertExpectations(t)
}

func TestSecretAdapter_Delete_EmptyParams(t *testing.T) {
	adapter := NewSecretAdapter(&mockSecretStoragePort{}, &mockLogger{})

	err := adapter.Delete(context.Background(), "", "secret-1")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	err = adapter.Delete(context.Background(), "user-123", "")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}
