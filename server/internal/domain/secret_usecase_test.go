// Package domain содержит тесты для бизнес-логики управления секретами.
//
// Тестируются:
//   - CreateSecret: создание секрета
//   - GetSecrets: получение списка секретов
//   - GetSecret: получение одного секрета
//   - DeleteSecret: удаление секрета
//   - GetTokenValidator: получение валидатора токена
package domain

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gophkeeper/server/internal/ports"
	"testing"
)

// mockSecretPort — мок для интерфейса SecretPort.
type mockSecretPort struct{ mock.Mock }

// Create мокирует создание секрета.
func (m *mockSecretPort) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	args := m.Called(ctx, secret)
	if sec := args.Get(0); sec != nil {
		return sec.(*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetByUserID мокирует получение списка секретов пользователя.
func (m *mockSecretPort) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {
	args := m.Called(ctx, userID)
	if secrets := args.Get(0); secrets != nil {
		return secrets.([]*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetSecret мокирует получение одного секрета по ID.
func (m *mockSecretPort) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	args := m.Called(ctx, id, userID)
	if sec := args.Get(0); sec != nil {
		return sec.(*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

// Delete мокирует удаление секрета.
func (m *mockSecretPort) Delete(ctx context.Context, userID, secretID string) error {
	args := m.Called(ctx, userID, secretID)
	return args.Error(0)
}

// mockTokenValidator — мок для интерфейса TokenValidator.
type mockTokenValidator struct{ mock.Mock }

// ValidateToken мокирует валидацию JWT-токена.
func (m *mockTokenValidator) ValidateToken(token string) (string, error) {
	args := m.Called(token)
	return args.String(0), args.Error(1)
}

// TestSecretUseCase_CreateSecret тестирует создание секрета.
//
// Сценарии:
//   - success: успешное создание секрета
//   - empty userID: пустой userID → ошибка ErrInvalidInput
//   - empty title: пустой заголовок → ошибка ErrInvalidInput
//   - empty data: пустые данные → ошибка ErrInvalidInput
//   - invalid secret type: неизвестный тип секрета → ошибка ErrInvalidInput
func TestSecretUseCase_CreateSecret(t *testing.T) {
	tests := []struct {
		name       string     // имя теста
		userID     string     // ID пользователя
		secretType SecretType // тип секрета
		title      string     // заголовок
		data       string     // зашифрованные данные
		wantErr    error      // ожидаемая ошибка
	}{
		{
			name:       "success",
			userID:     "user-123",
			secretType: SecretTypePassword,
			title:      "Bank password",
			data:       "base64encrypteddata123==",
			wantErr:    nil,
		},
		{
			name:       "empty userID",
			userID:     "",
			secretType: SecretTypeNote,
			title:      "Note",
			data:       "data",
			wantErr:    ErrInvalidInput,
		},
		{
			name:       "empty title",
			userID:     "user-123",
			secretType: SecretTypeNote,
			title:      "",
			data:       "data",
			wantErr:    ErrInvalidInput,
		},
		{
			name:       "empty data",
			userID:     "user-123",
			secretType: SecretTypeCard,
			title:      "Card",
			data:       "",
			wantErr:    ErrInvalidInput,
		},
		{
			name:       "invalid secret type",
			userID:     "user-123",
			secretType: SecretType("unknown"),
			title:      "Test",
			data:       "data",
			wantErr:    ErrInvalidInput,
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создание и настройка мока
			mockPort := &mockSecretPort{}
			if tt.wantErr == nil {
				mockPort.On("Create", mock.Anything, mock.AnythingOfType("*ports.Secret")).
					Return(&ports.Secret{ID: "secret-456"}, nil)
			}

			// Создание usecase
			uc := NewSecretUseCase(mockPort, &mockTokenValidator{})

			// Вызов тестируемого метода
			secret, err := uc.CreateSecret(context.Background(), tt.userID, tt.secretType, tt.title, tt.data)

			// Проверка результатов
			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil {
				assert.NotNil(t, secret)
				assert.Equal(t, "secret-456", secret.ID)
			}
			mockPort.AssertExpectations(t)
		})
	}
}

// TestSecretUseCase_GetSecrets тестирует получение списка секретов.
//
// Сценарии:
//   - success: успешное получение списка
//   - empty userID: пустой userID → ошибка ErrInvalidInput
func TestSecretUseCase_GetSecrets(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Создание и настройка мока
		mockPort := &mockSecretPort{}
		mockPort.On("GetByUserID", mock.Anything, "user-123").
			Return([]*ports.Secret{{ID: "s1"}, {ID: "s2"}}, nil)

		// Создание usecase
		uc := NewSecretUseCase(mockPort, &mockTokenValidator{})

		// Вызов тестируемого метода
		secrets, err := uc.GetSecrets(context.Background(), "user-123")

		// Проверка результатов
		assert.NoError(t, err)
		assert.Len(t, secrets, 2)
		mockPort.AssertExpectations(t)
	})

	t.Run("empty userID", func(t *testing.T) {
		// Создание мока (не должен вызываться)
		mockPort := &mockSecretPort{}

		// Создание usecase
		uc := NewSecretUseCase(mockPort, &mockTokenValidator{})

		// Вызов тестируемого метода с пустым userID
		secrets, err := uc.GetSecrets(context.Background(), "")

		// Проверка результатов
		assert.ErrorIs(t, err, ErrInvalidInput)
		assert.Nil(t, secrets)
		mockPort.AssertExpectations(t)
	})
}

// TestSecretUseCase_GetSecret тестирует получение одного секрета по ID.
//
// Сценарии:
//   - success: успешное получение секрета
//   - empty id: пустой ID секрета → ошибка ErrInvalidInput
//   - empty userID: пустой userID → ошибка ErrInvalidInput
func TestSecretUseCase_GetSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Создание и настройка мока
		mockPort := &mockSecretPort{}
		mockPort.On("GetSecret", mock.Anything, "secret-123", "user-123").
			Return(&ports.Secret{ID: "secret-123", Title: "My Secret"}, nil)

		// Создание usecase
		uc := NewSecretUseCase(mockPort, &mockTokenValidator{})

		// Вызов тестируемого метода
		secret, err := uc.GetSecret(context.Background(), "secret-123", "user-123")

		// Проверка результатов
		assert.NoError(t, err)
		assert.Equal(t, "secret-123", secret.ID)
		mockPort.AssertExpectations(t)
	})

	t.Run("empty params", func(t *testing.T) {
		// Создание мока (не должен вызываться)
		mockPort := &mockSecretPort{}

		// Создание usecase
		uc := NewSecretUseCase(mockPort, &mockTokenValidator{})

		// Вызов с пустым id
		_, err := uc.GetSecret(context.Background(), "", "user-123")
		assert.ErrorIs(t, err, ErrInvalidInput)

		// Вызов с пустым userID
		_, err = uc.GetSecret(context.Background(), "secret-123", "")
		assert.ErrorIs(t, err, ErrInvalidInput)

		mockPort.AssertExpectations(t)
	})
}

// TestSecretUseCase_DeleteSecret тестирует удаление секрета.
//
// Сценарии:
//   - success: успешное удаление секрета
//   - empty userID: пустой userID → ошибка ErrInvalidInput
//   - empty secretID: пустой ID секрета → ошибка ErrInvalidInput
func TestSecretUseCase_DeleteSecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Создание и настройка мока
		mockPort := &mockSecretPort{}
		mockPort.On("Delete", mock.Anything, "user-123", "secret-456").Return(nil)

		// Создание usecase
		uc := NewSecretUseCase(mockPort, &mockTokenValidator{})

		// Вызов тестируемого метода
		err := uc.DeleteSecret(context.Background(), "user-123", "secret-456")

		// Проверка результатов
		assert.NoError(t, err)
		mockPort.AssertExpectations(t)
	})

	t.Run("empty params", func(t *testing.T) {
		// Создание мока (не должен вызываться)
		mockPort := &mockSecretPort{}

		// Создание usecase
		uc := NewSecretUseCase(mockPort, &mockTokenValidator{})

		// Вызов с пустым userID
		err := uc.DeleteSecret(context.Background(), "", "secret-456")
		assert.ErrorIs(t, err, ErrInvalidInput)

		// Вызов с пустым secretID
		err = uc.DeleteSecret(context.Background(), "user-123", "")
		assert.ErrorIs(t, err, ErrInvalidInput)

		mockPort.AssertExpectations(t)
	})
}

// TestSecretUseCase_GetTokenValidator тестирует получение валидатора токена.
func TestSecretUseCase_GetTokenValidator(t *testing.T) {
	// Создание мока валидатора
	mockValidator := &mockTokenValidator{}

	// Создание usecase
	uc := NewSecretUseCase(&mockSecretPort{}, mockValidator)

	// Получение валидатора
	validator := uc.GetTokenValidator()

	// Проверка результатов
	assert.Equal(t, mockValidator, validator)
}
