package auth

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

type mockStoragePort struct{ mock.Mock }

func (m *mockStoragePort) CreateUser(ctx context.Context, user ports.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}

func (m *mockStoragePort) GetUserByEmail(ctx context.Context, email string) (ports.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(ports.User), args.Error(1)
}

func (m *mockStoragePort) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(ports.User), args.Error(1)
}

func (m *mockStoragePort) DeleteUser(ctx context.Context, id string) error       { return nil }
func (m *mockStoragePort) UpdateUser(ctx context.Context, user ports.User) error { return nil }
func (m *mockStoragePort) ListUsers(ctx context.Context) ([]domain.User, error)  { return nil, nil }

type mockLogger struct{ mock.Mock }

func (m *mockLogger) Info(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Warn(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Error(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Debug(msg string, fields ...zap.Field)  {}
func (m *mockLogger) With(fields ...zap.Field) logger.Logger { return m }
func (m *mockLogger) Sync() error                            { return nil }

func TestPasswordAdapter_CreateUser(t *testing.T) {
	tests := []struct {
		name     string
		user     ports.User
		mockResp string
		mockErr  error
		wantErr  error
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

			adapter := NewPasswordAdapter(mockStorage, "super-secret-jwt-key-32-chars-min", &mockLogger{})

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
	// Вспомогательная функция для вычисления хеша
	computeHash := func(pass string) string {
		adapter := NewPasswordAdapter(nil, "", nil).(*passwordAdapter)
		return adapter.hashPassword(pass)
	}

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
			mockUser: ports.User{ID: "user-123", PasswordHash: "somehash"},
			wantErr:  domain.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &mockStoragePort{}
			mockStorage.On("GetUserByEmail", mock.Anything, tt.email).
				Return(tt.mockUser, tt.mockErr)

			adapter := NewPasswordAdapter(mockStorage, "super-secret-jwt-key-32-chars-min", &mockLogger{})

			token, userID, err := adapter.AuthenticatePassword(context.Background(), tt.email, tt.password)

			assert.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil {
				assert.NotEmpty(t, token)
				assert.Equal(t, tt.mockUser.ID, userID)
			}
		})
	}
}
