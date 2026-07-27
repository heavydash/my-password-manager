// Package auth содержит тесты для адаптеров аутентификации GophKeeper.
//
// Тестируются:
//   - PasswordAdapter (CreateUser, AuthenticatePassword, Register, LoginPassword)
//   - Хеширование паролей через Argon2
//   - Генерация и валидация JWT
//
// Используются моки для StoragePort и Logger.
package auth

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gophkeeper/server/internal/config"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"testing"
	"time"
)

// mockStoragePort — мок для ports.StoragePort.
// Используется для изоляции тестов от реальной БД.
type mockStoragePort struct{ mock.Mock }

// CreateUser имитирует создание пользователя в хранилище.
func (m *mockStoragePort) CreateUser(ctx context.Context, user ports.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}

// GetUserByEmail имитирует поиск пользователя по email.
func (m *mockStoragePort) GetUserByEmail(ctx context.Context, email string) (ports.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(ports.User), args.Error(1)
}

// GetUserByID имитирует поиск пользователя по ID.
func (m *mockStoragePort) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(ports.User), args.Error(1)
}

// Заглушки для неиспользуемых в тестах методов
func (m *mockStoragePort) DeleteUser(ctx context.Context, id string) error       { return nil }
func (m *mockStoragePort) UpdateUser(ctx context.Context, user ports.User) error { return nil }
func (m *mockStoragePort) ListUsers(ctx context.Context) ([]domain.User, error)  { return nil, nil }

// mockLogger — мок для logger.Logger.
// Игнорирует все вызовы логирования.
type mockLogger struct{ mock.Mock }

func (m *mockLogger) Info(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Warn(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Error(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Fatal(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Debug(msg string, fields ...zap.Field)  {}
func (m *mockLogger) With(fields ...zap.Field) logger.Logger { return m }
func (m *mockLogger) Sync() error                            { return nil }

// testConfig возвращает минимальную конфигурацию для тестов.
//
// Содержит:
//   - JWT секрет и время жизни токена
//   - Параметры Argon2 для хеширования паролей
func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			Secret:    "test-secret-key-for-unit-tests-only-32chars",
			ExpiresIn: 15 * time.Minute,
		},
		Argon2: config.Argon2Config{
			Salt:        "test-salt-for-argon2-16bytes",
			Iterations:  1,
			Memory:      64 * 1024,
			Parallelism: 4,
			KeyLen:      32,
		},
	}
}

// computeHash вычисляет хеш пароля для тестов.
//
// Создаёт временный адаптер с тестовой конфигурацией
// и вызывает hashPassword.
func computeHash(password string) string {
	adapter := NewPasswordAdapter(nil, testConfig(), &mockLogger{}).(*passwordAdapter)
	return adapter.hashPassword(password)
}

// TestPasswordAdapter_CreateUser тестирует создание пользователя.
//
// Сценарии:
//   - success: успешное создание пользователя
//   - already exists: попытка создать существующего пользователя
func TestPasswordAdapter_CreateUser(t *testing.T) {
	tests := []struct {
		name     string     // имя теста
		user     ports.User // входные данные пользователя
		mockResp string     // ожидаемый ID от мока
		mockErr  error      // ошибка от мока
		wantErr  error      // ожидаемая ошибка
	}{
		{
			name:     "success",
			user:     ports.User{Email: "test@example.com", PasswordHash: "password123"},
			mockResp: "user-123",
			mockErr:  nil,
		},
		{
			name:    "already exists",
			user:    ports.User{Email: "exists@example.com"},
			mockErr: domain.ErrUserAlreadyExists,
			wantErr: domain.ErrUserAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &mockStoragePort{}
			mockStorage.On("CreateUser", mock.Anything, mock.Anything).Return(tt.mockResp, tt.mockErr)

			adapter := NewPasswordAdapter(mockStorage, testConfig(), &mockLogger{})

			id, err := adapter.CreateUser(context.Background(), tt.user)

			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil {
				assert.Equal(t, tt.mockResp, id)
			}
			mockStorage.AssertExpectations(t)
		})
	}
}

func TestPasswordAdapter_AuthenticatePassword(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		mockUser ports.User
		mockErr  error
		wantErr  error
	}{
		{
			name:     "success",
			email:    "test@example.com",
			password: "correctpass",
			mockUser: ports.User{ID: "user-123",
				PasswordHash: computeHash("correctpass"),
			},
		},
		{
			name:     "invalid credentials",
			email:    "test@example.com",
			password: "wrongpass",
			mockUser: ports.User{ID: "user-123", PasswordHash: computeHash("correctpass")},
			wantErr:  domain.ErrInvalidCredentials,
		},
		{
			name:     "user not found",
			email:    "notfound@example.com",
			password: "anypass",
			mockUser: ports.User{},
			mockErr:  domain.ErrUserNotFound,
			wantErr:  domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &mockStoragePort{}
			mockStorage.On("GetUserByEmail", mock.Anything, tt.email).
				Return(tt.mockUser, tt.mockErr)

			adapter := NewPasswordAdapter(mockStorage, testConfig(), &mockLogger{})

			token, userID, err := adapter.AuthenticatePassword(context.Background(), tt.email, tt.password)

			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil {
				assert.NotEmpty(t, token)
				assert.Equal(t, tt.mockUser.ID, userID)
			}
			mockStorage.AssertExpectations(t)
		})
	}
}

// TestPasswordAdapter_Register тестирует обёртку Register.
//
// Проверяет, что Register корректно делегирует вызов CreateUser.
func TestPasswordAdapter_Register(t *testing.T) {
	mockStorage := &mockStoragePort{}
	mockStorage.On("CreateUser", mock.Anything, mock.Anything).Return("user-123", nil)

	adapter := NewPasswordAdapter(mockStorage, testConfig(), &mockLogger{})

	id, err := adapter.Register(context.Background(), "test@example.com", "password123")

	assert.NoError(t, err)
	assert.Equal(t, "user-123", id)
	mockStorage.AssertExpectations(t)
}

// TestPasswordAdapter_LoginPassword тестирует обёртку LoginPassword.
//
// Проверяет, что LoginPassword корректно делегирует вызов AuthenticatePassword.
func TestPasswordAdapter_LoginPassword(t *testing.T) {
	mockStorage := &mockStoragePort{}
	mockStorage.On("GetUserByEmail", mock.Anything, "test@example.com").
		Return(ports.User{ID: "user-123", PasswordHash: computeHash("correctpass")}, nil)

	adapter := NewPasswordAdapter(mockStorage, testConfig(), &mockLogger{})

	token, userID, err := adapter.LoginPassword(context.Background(), "test@example.com", "correctpass")

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, "user-123", userID)
	mockStorage.AssertExpectations(t)
}

// TestPasswordAdapter_GenerateJWT тестирует генерацию JWT-токена.
func TestPasswordAdapter_GenerateJWT(t *testing.T) {
	adapter := NewPasswordAdapter(nil, testConfig(), &mockLogger{})

	token, err := adapter.GenerateJWT("user-123")

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

// TestPasswordAdapter_ValidateJWT тестирует валидацию JWT-токена.
func TestPasswordAdapter_ValidateJWT(t *testing.T) {
	adapter := NewPasswordAdapter(nil, testConfig(), &mockLogger{})

	token, err := adapter.GenerateJWT("user-123")
	assert.NoError(t, err)

	userID, err := adapter.ValidateJWT(token)

	assert.NoError(t, err)
	assert.Equal(t, "user-123", userID)
}
