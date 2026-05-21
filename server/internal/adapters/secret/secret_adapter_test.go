// Package secret содержит тесты для адаптера работы с секретами.
//
// Тестируются:
//   - Create: создание секрета
//   - GetByUserID: получение списка секретов
//   - GetSecret: получение одного секрета
//   - Delete: удаление секрета
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

// mockSecretStoragePort — мок для ports.SecretStoragePort.
type mockSecretStoragePort struct{ mock.Mock }

// Create мокирует создание секрета.
func (m *mockSecretStoragePort) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	args := m.Called(ctx, secret)
	if sec := args.Get(0); sec != nil {
		return sec.(*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetByUserID мокирует получение списка секретов.
func (m *mockSecretStoragePort) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {
	args := m.Called(ctx, userID)
	if secrets := args.Get(0); secrets != nil {
		return secrets.([]*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetSecret мокирует получение одного секрета.
func (m *mockSecretStoragePort) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	args := m.Called(ctx, id, userID)
	if sec := args.Get(0); sec != nil {
		return sec.(*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

// Delete мокирует удаление секрета.
func (m *mockSecretStoragePort) Delete(ctx context.Context, userID, secretID string) error {
	args := m.Called(ctx, userID, secretID)
	return args.Error(0)
}

// mockLogger — мок для logger.Logger.
type mockLogger struct{ mock.Mock }

func (m *mockLogger) Info(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Warn(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Error(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Fatal(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Debug(msg string, fields ...zap.Field)  {}
func (m *mockLogger) With(fields ...zap.Field) logger.Logger { return m }
func (m *mockLogger) Sync() error                            { return nil }

// TestSecretAdapter_Create тестирует успешное создание секрета.
func TestSecretAdapter_Create(t *testing.T) {
	// Создание мока хранилища
	mockStorage := &mockSecretStoragePort{}
	mockStorage.On("Create", mock.Anything, mock.AnythingOfType("*ports.Secret")).
		Return(&ports.Secret{ID: "secret-456"}, nil)

	// Создание адаптера
	adapter := NewSecretAdapter(mockStorage, &mockLogger{})

	// Подготовка секрета
	secret := &ports.Secret{
		UserID: "user-123",
		Title:  "Test secret",
		Data:   "encrypteddata",
		Type:   "password",
	}

	// Вызов метода
	created, err := adapter.Create(context.Background(), secret)

	// Проверка результатов
	assert.NoError(t, err)
	assert.Equal(t, "secret-456", created.ID)
	mockStorage.AssertExpectations(t)
}

// TestSecretAdapter_Create_NilSecret тестирует создание с nil секретом.
func TestSecretAdapter_Create_NilSecret(t *testing.T) {
	// Создание адаптера
	adapter := NewSecretAdapter(&mockSecretStoragePort{}, &mockLogger{})

	// Вызов метода с nil
	_, err := adapter.Create(context.Background(), nil)

	// Проверка результатов
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

// TestSecretAdapter_Create_EmptyFields тестирует создание с пустыми полями.
func TestSecretAdapter_Create_EmptyFields(t *testing.T) {
	// Создание адаптера
	adapter := NewSecretAdapter(&mockSecretStoragePort{}, &mockLogger{})

	// Пустой title
	secret := &ports.Secret{
		UserID: "user-123",
		Title:  "",
		Data:   "data",
		Type:   "password",
	}
	_, err := adapter.Create(context.Background(), secret)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	// Пустая data
	secret.Title = "title"
	secret.Data = ""
	_, err = adapter.Create(context.Background(), secret)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	// Пустой type
	secret.Data = "data"
	secret.Type = ""
	_, err = adapter.Create(context.Background(), secret)
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

// TestSecretAdapter_Create_StorageError тестирует ошибку хранилища при создании.
func TestSecretAdapter_Create_StorageError(t *testing.T) {
	// Создание мока с ошибкой
	mockStorage := &mockSecretStoragePort{}
	mockStorage.On("Create", mock.Anything, mock.Anything).Return(nil, domain.ErrInternal)

	// Создание адаптера
	adapter := NewSecretAdapter(mockStorage, &mockLogger{})

	// Подготовка секрета
	secret := &ports.Secret{
		UserID: "user-123",
		Title:  "Test",
		Data:   "data",
		Type:   "password",
	}

	// Вызов метода
	_, err := adapter.Create(context.Background(), secret)

	// Проверка результатов
	assert.Error(t, err)
	mockStorage.AssertExpectations(t)
}

// TestSecretAdapter_GetByUserID тестирует успешное получение списка секретов.
func TestSecretAdapter_GetByUserID(t *testing.T) {
	// Создание мока хранилища
	mockStorage := &mockSecretStoragePort{}
	mockStorage.On("GetByUserID", mock.Anything, "user-123").
		Return([]*ports.Secret{{ID: "s1"}, {ID: "s2"}}, nil)

	// Создание адаптера
	adapter := NewSecretAdapter(mockStorage, &mockLogger{})

	// Вызов метода
	secrets, err := adapter.GetByUserID(context.Background(), "user-123")

	// Проверка результатов
	assert.NoError(t, err)
	assert.Len(t, secrets, 2)
	mockStorage.AssertExpectations(t)
}

// TestSecretAdapter_GetByUserID_EmptyUserID тестирует получение с пустым userID.
func TestSecretAdapter_GetByUserID_EmptyUserID(t *testing.T) {
	// Создание адаптера
	adapter := NewSecretAdapter(&mockSecretStoragePort{}, &mockLogger{})

	// Вызов метода с пустым userID
	_, err := adapter.GetByUserID(context.Background(), "")

	// Проверка результатов
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

// TestSecretAdapter_GetByUserID_StorageError тестирует ошибку хранилища.
func TestSecretAdapter_GetByUserID_StorageError(t *testing.T) {
	// Создание мока с ошибкой
	mockStorage := &mockSecretStoragePort{}
	mockStorage.On("GetByUserID", mock.Anything, "user-123").Return(nil, domain.ErrInternal)

	// Создание адаптера
	adapter := NewSecretAdapter(mockStorage, &mockLogger{})

	// Вызов метода
	_, err := adapter.GetByUserID(context.Background(), "user-123")

	// Проверка результатов
	assert.Error(t, err)
	mockStorage.AssertExpectations(t)
}

// TestSecretAdapter_GetSecret тестирует успешное получение одного секрета.
func TestSecretAdapter_GetSecret(t *testing.T) {
	// Создание мока хранилища
	mockStorage := &mockSecretStoragePort{}
	mockStorage.On("GetSecret", mock.Anything, "secret-123", "user-123").
		Return(&ports.Secret{ID: "secret-123", Title: "My Secret"}, nil)

	// Создание адаптера
	adapter := NewSecretAdapter(mockStorage, &mockLogger{})

	// Вызов метода
	secret, err := adapter.GetSecret(context.Background(), "secret-123", "user-123")

	// Проверка результатов
	assert.NoError(t, err)
	assert.Equal(t, "secret-123", secret.ID)
	assert.Equal(t, "My Secret", secret.Title)
	mockStorage.AssertExpectations(t)
}

// TestSecretAdapter_GetSecret_NotFound тестирует ситуацию, когда секрет не найден.
func TestSecretAdapter_GetSecret_NotFound(t *testing.T) {
	// Создание мока с ошибкой NotFound
	mockStorage := &mockSecretStoragePort{}
	mockStorage.On("GetSecret", mock.Anything, "secret-999", "user-123").
		Return(nil, domain.ErrNotFound)

	// Создание адаптера
	adapter := NewSecretAdapter(mockStorage, &mockLogger{})

	// Вызов метода
	_, err := adapter.GetSecret(context.Background(), "secret-999", "user-123")

	// Проверка результатов
	assert.ErrorIs(t, err, domain.ErrNotFound)
	mockStorage.AssertExpectations(t)
}

// TestSecretAdapter_Delete тестирует успешное удаление секрета.
func TestSecretAdapter_Delete(t *testing.T) {
	// Создание мока хранилища
	mockStorage := &mockSecretStoragePort{}
	mockStorage.On("Delete", mock.Anything, "user-123", "secret-456").Return(nil)

	// Создание адаптера
	adapter := NewSecretAdapter(mockStorage, &mockLogger{})

	// Вызов метода
	err := adapter.Delete(context.Background(), "user-123", "secret-456")

	// Проверка результатов
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

// TestSecretAdapter_Delete_EmptyParams тестирует удаление с пустыми параметрами.
func TestSecretAdapter_Delete_EmptyParams(t *testing.T) {
	// Создание адаптера
	adapter := NewSecretAdapter(&mockSecretStoragePort{}, &mockLogger{})

	// Пустой userID
	err := adapter.Delete(context.Background(), "", "secret-1")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)

	// Пустой secretID
	err = adapter.Delete(context.Background(), "user-123", "")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
}

// TestSecretAdapter_Delete_NotFound тестирует удаление несуществующего секрета.
func TestSecretAdapter_Delete_NotFound(t *testing.T) {
	// Создание мока с ошибкой NotFound
	mockStorage := &mockSecretStoragePort{}
	mockStorage.On("Delete", mock.Anything, "user-123", "secret-999").
		Return(domain.ErrNotFound)

	// Создание адаптера
	adapter := NewSecretAdapter(mockStorage, &mockLogger{})

	// Вызов метода
	err := adapter.Delete(context.Background(), "user-123", "secret-999")

	// Проверка результатов
	assert.ErrorIs(t, err, domain.ErrNotFound)
	mockStorage.AssertExpectations(t)
}
