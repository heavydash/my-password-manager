// Package memory содержит тесты для in-memory хранилища секретов.
//
// Тестируются:
//   - Create: создание секрета
//   - GetByUserID: получение списка секретов
//   - GetSecret: получение одного секрета
//   - Delete: удаление секрета
//   - Контекст: отмена операций
package memory

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"testing"
	"time"
)

// testLogger — мок для logger.Logger.
type testLogger struct{}

func (testLogger) Debug(msg string, fields ...zap.Field)  {}
func (testLogger) Info(msg string, fields ...zap.Field)   {}
func (testLogger) Warn(msg string, fields ...zap.Field)   {}
func (testLogger) Error(msg string, fields ...zap.Field)  {}
func (testLogger) Fatal(msg string, fields ...zap.Field)  {}
func (testLogger) With(fields ...zap.Field) logger.Logger { return testLogger{} }
func (testLogger) Sync() error                            { return nil }

// TestSecretStorage_Create тестирует создание секрета.
//
// Сценарии:
//   - success: успешное создание
//   - nil_secret: секрет равен nil
//   - empty_user_id: пустой UserID
func TestSecretStorage_Create(t *testing.T) {
	log := testLogger{}
	storage := NewSecretStorage(log).(*secretStorage)

	tests := []struct {
		name        string
		secret      *ports.Secret
		wantErr     bool
		wantErrType error
	}{
		{
			name: "success",
			secret: &ports.Secret{
				ID:     "sec-123",
				UserID: "user-456",
				Type:   "password",
				Title:  "My Bank",
				Data:   "encrypted-data",
			},
			wantErr: false,
		},
		{
			name:        "nil_secret",
			secret:      nil,
			wantErr:     true,
			wantErrType: domain.ErrInvalidInput,
		},
		{
			name: "empty_user_id",
			secret: &ports.Secret{
				ID:    "sec-789",
				Title: "Test",
				Data:  "data",
			},
			wantErr:     true,
			wantErrType: domain.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := storage.Create(context.Background(), tt.secret)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, created)
			assert.Equal(t, tt.secret.UserID, created.UserID)
			assert.Equal(t, tt.secret.Title, created.Title)

			// Проверяем, что сохранилось в map
			stored, exists := storage.secrets[created.ID]
			assert.True(t, exists)
			assert.Equal(t, created.ID, stored.ID)
		})
	}
}

// TestSecretStorage_GetByUserID тестирует получение списка секретов.
//
// Сценарии:
//   - user_with_secrets: пользователь с секретами
//   - user_without_secrets: пользователь без секретов
//   - empty_user_id: пустой userID (ошибка)
func TestSecretStorage_GetByUserID(t *testing.T) {
	log := testLogger{}
	storage := NewSecretStorage(log).(*secretStorage)

	// Подготовка данных
	s1 := &ports.Secret{ID: "s1", UserID: "user-1", Title: "Secret 1"}
	s2 := &ports.Secret{ID: "s2", UserID: "user-1", Title: "Secret 2"}
	s3 := &ports.Secret{ID: "s3", UserID: "user-2", Title: "Secret 3"}

	storage.secrets[s1.ID] = s1
	storage.secrets[s2.ID] = s2
	storage.secrets[s3.ID] = s3

	tests := []struct {
		name      string
		userID    string
		wantCount int
		wantErr   bool
	}{
		{"user_with_secrets", "user-1", 2, false},
		{"user_without_secrets", "user-999", 0, false},
		{"empty_user_id", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets, err := storage.GetByUserID(context.Background(), tt.userID)

			if tt.wantErr {
				assert.ErrorIs(t, err, domain.ErrInvalidInput)
				return
			}

			require.NoError(t, err)
			assert.Len(t, secrets, tt.wantCount)
		})
	}
}

// TestSecretStorage_GetSecret тестирует получение одного секрета.
//
// Сценарии:
//   - success: успешное получение
//   - not_found_wrong_user: неверный userID
//   - not_found_wrong_id: неверный ID секрета
func TestSecretStorage_GetSecret(t *testing.T) {
	log := testLogger{}
	storage := NewSecretStorage(log).(*secretStorage)

	secret := &ports.Secret{
		ID:     "sec-123",
		UserID: "user-456",
		Title:  "My Bank",
		Type:   "password",
		Data:   "encrypted",
	}
	storage.secrets[secret.ID] = secret

	tests := []struct {
		name      string
		id        string
		userID    string
		wantErr   bool
		wantTitle string
	}{
		{"success", "sec-123", "user-456", false, "My Bank"},
		{"not_found_wrong_user", "sec-123", "user-999", true, ""},
		{"not_found_wrong_id", "sec-999", "user-456", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec, err := storage.GetSecret(context.Background(), tt.id, tt.userID)

			if tt.wantErr {
				assert.ErrorIs(t, err, domain.ErrNotFound)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantTitle, sec.Title)
		})
	}
}

// TestSecretStorage_Delete тестирует удаление секрета.
//
// Сценарии:
//   - success: успешное удаление
//   - not_found_wrong_user: неверный userID
//   - not_found_wrong_id: неверный ID секрета
func TestSecretStorage_Delete(t *testing.T) {
	log := testLogger{}
	storage := NewSecretStorage(log).(*secretStorage)

	secret := &ports.Secret{ID: "sec-123", UserID: "user-456"}
	storage.secrets[secret.ID] = secret

	tests := []struct {
		name    string
		userID  string
		id      string
		wantErr bool
	}{
		{"success", "user-456", "sec-123", false},
		{"not_found_wrong_user", "user-999", "sec-123", true},
		{"not_found_wrong_id", "user-456", "sec-999", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Delete(context.Background(), tt.userID, tt.id)

			if tt.wantErr {
				assert.ErrorIs(t, err, domain.ErrNotFound)
				return
			}

			require.NoError(t, err)
			_, exists := storage.secrets[tt.id]
			assert.False(t, exists)
		})
	}
}

// TestSecretStorage_ContextCancellation тестирует отмену контекста.
//
// Сценарии:
//   - create_cancelled: отмена контекста при создании
//   - get_by_user_id_cancelled: отмена контекста при получении списка
//   - get_secret_cancelled: отмена контекста при получении одного секрета
//   - delete_cancelled: отмена контекста при удалении
func TestSecretStorage_ContextCancellation(t *testing.T) {
	log := testLogger{}
	storage := NewSecretStorage(log).(*secretStorage)

	// Подготовка данных
	secret := &ports.Secret{
		UserID: "user-123",
		Title:  "Test",
		Data:   "data",
		Type:   "password",
	}
	created, err := storage.Create(context.Background(), secret)
	require.NoError(t, err)

	tests := []struct {
		name string
		fn   func(ctx context.Context) error
	}{
		{
			name: "create_cancelled",
			fn: func(ctx context.Context) error {
				_, err := storage.Create(ctx, secret)
				return err
			},
		},
		{
			name: "get_by_user_id_cancelled",
			fn: func(ctx context.Context) error {
				_, err := storage.GetByUserID(ctx, "user-123")
				return err
			},
		},
		{
			name: "get_secret_cancelled",
			fn: func(ctx context.Context) error {
				_, err := storage.GetSecret(ctx, created.ID, "user-123")
				return err
			},
		},
		{
			name: "delete_cancelled",
			fn: func(ctx context.Context) error {
				return storage.Delete(ctx, "user-123", created.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Немедленная отмена

			err := tt.fn(ctx)
			assert.ErrorIs(t, err, context.Canceled)
		})
	}
}

// TestSecretStorage_ContextTimeout тестирует таймаут контекста.
func TestSecretStorage_ContextTimeout(t *testing.T) {
	log := testLogger{}
	storage := NewSecretStorage(log).(*secretStorage)

	secret := &ports.Secret{
		UserID: "user-123",
		Title:  "Test",
		Data:   "data",
		Type:   "password",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Ждём истечения таймаута
	time.Sleep(1 * time.Millisecond)

	_, err := storage.Create(ctx, secret)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
