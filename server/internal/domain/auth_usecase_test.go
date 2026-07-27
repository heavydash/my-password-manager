package domain

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"testing"
)

type mockAuthPort struct {
	mock.Mock
}

func (m *mockAuthPort) Register(ctx context.Context, email, password string) (string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Error(1)
}

func (m *mockAuthPort) LoginPassword(ctx context.Context, email, password string) (string, string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockAuthPort) CreateUser(ctx context.Context, user ports.User) (string, error) {
	args := m.Called(ctx, user)
	return args.String(0), args.Error(1)
}

func (m *mockAuthPort) AuthenticatePassword(ctx context.Context, email, password string) (string, string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockAuthPort) GetOAuthURL(provider string) (string, string, error) {
	args := m.Called(provider)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockAuthPort) HandleCallback(provider, code, state string) (string, error) {
	args := m.Called(provider, code, state)
	return args.String(0), args.Error(1)
}

func (m *mockAuthPort) HandleOAuthCallback(provider, code, state string) (string, error) {
	args := m.Called(provider, code, state)
	return args.String(0), args.Error(1)
}

func (m *mockAuthPort) AuthenticateOAuth(ctx context.Context, oneTimeCode string) (string, error) {
	args := m.Called(ctx, oneTimeCode)
	return args.String(0), args.Error(1)
}

func (m *mockAuthPort) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	return ports.User{}, nil
}

func (m *mockAuthPort) ValidateJWT(tokenString string) (string, error) {
	return "", nil
}

func (m *mockAuthPort) GenerateJWT(userID string) (string, error) {
	return "", nil
}

type mockLogger struct{ mock.Mock }

func (m *mockLogger) Info(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Warn(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Error(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Fatal(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Debug(msg string, fields ...zap.Field)  {}
func (m *mockLogger) With(fields ...zap.Field) logger.Logger { return m }
func (m *mockLogger) Sync() error                            { return nil }

func TestAuthUseCase_Register(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		mockErr  error
		wantErr  string
	}{
		{
			name:     "success",
			email:    "test@example.com",
			password: "pass123",
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: "email is required",
		},
		{
			name:    "user already exists",
			email:   "exists@example.com",
			mockErr: ErrUserAlreadyExists,
			wantErr: "user already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPort := &mockAuthPort{}

			if tt.mockErr != nil {
				mockPort.On("CreateUser", mock.Anything, mock.Anything).Return("", tt.mockErr)
			} else if tt.wantErr == "" {
				mockPort.On("CreateUser", mock.Anything, mock.Anything).Return("user-123", nil)
			}

			uc := NewAuthUseCase(mockPort, &mockLogger{})

			id, err := uc.Register(context.Background(), tt.email, tt.password)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "user-123", id)
			}

			mockPort.AssertExpectations(t)
		})
	}
}
func TestAuthUseCase_LoginPassword(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		mockErr  error
		wantErr  string
	}{
		{
			name:     "success",
			email:    "test@example.com",
			password: "correctpass",
		},
		{
			name:     "empty credentials",
			email:    "",
			password: "",
			wantErr:  "invalid input",
		},
		{
			name:     "invalid credentials",
			email:    "test@example.com",
			password: "wrong",
			mockErr:  ErrInvalidCredentials,
			wantErr:  "invalid credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPort := &mockAuthPort{}

			if tt.mockErr != nil {
				mockPort.On("AuthenticatePassword", mock.Anything, tt.email, tt.password).Return("", "", tt.mockErr)
			} else if tt.wantErr == "" {
				mockPort.On("AuthenticatePassword", mock.Anything, tt.email, tt.password).Return("jwt-token", "user-123", nil)
			}

			uc := NewAuthUseCase(mockPort, &mockLogger{})

			token, userID, err := uc.LoginPassword(context.Background(), tt.email, tt.password)

			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "jwt-token", token)
				assert.Equal(t, "user-123", userID)
			}

			mockPort.AssertExpectations(t)
		})
	}
}
