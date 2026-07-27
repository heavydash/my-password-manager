package domain

import (
	"context"
	"go.uber.org/zap"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"

	"testing"
)

type mockLogger struct{}

func (m *mockLogger) Info(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Warn(msg string, fields ...zap.Field)   {}
func (m *mockLogger) Error(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Debug(msg string, fields ...zap.Field)  {}
func (m *mockLogger) With(fields ...zap.Field) logger.Logger { return m }
func (m *mockLogger) Sync() error                            { return nil }

type mockAuthPort struct{}

func (m *mockAuthPort) CreateUser(ctx context.Context, user ports.User) (string, error) {
	return "mock-user-id", nil
}

func (m *mockAuthPort) AuthenticatePassword(ctx context.Context, email, password string) (string, string, error) {
	return "mock-jwt-token", "mock-user-id", nil
}

func (m *mockAuthPort) GetOAuthURL(provider string) (string, string, error) {
	return "https://example.com", "mock-state", nil
}

func (m *mockAuthPort) HandleCallback(provider, code, state string) (string, error) {
	return "mock-one-time-code", nil
}

func (m *mockAuthPort) AuthenticateOAuth(ctx context.Context, oneTimeCode string) (string, error) {
	return "mock-jwt-token", nil
}

func (m *mockAuthPort) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	return ports.User{ID: "mock-user-id"}, nil
}

func (m *mockAuthPort) ValidateJWT(tokenString string) (string, error) {
	return "mock-user-id", nil
}

func (m *mockAuthPort) GenerateJWT(userID string) (string, error) {
	return "mock-jwt-token", nil
}

func TestAuthUseCase_Register(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantID   string
		wantErr  bool
	}{
		{"valid", "test@example.com", "pass123", "mock-user-id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := &mockAuthPort{}
			useCase := NewAuthUseCase(port, &mockLogger{})

			userID, err := useCase.Register(context.Background(), tt.email, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if userID != tt.wantID {
				t.Errorf("Register() = %v, want %v", userID, tt.wantID)
			}
		})
	}
}

func TestAuthUseCase_LoginPassword(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		password   string
		wantToken  string
		wantUserID string
		wantErr    bool
	}{
		{"valid", "test@example.com", "strongpassword123", "mock-jwt-token", "mock-user-id", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := &mockAuthPort{}
			useCase := NewAuthUseCase(port, &mockLogger{})

			token, userID, err := useCase.LoginPassword(context.Background(), tt.email, tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoginPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if token != tt.wantToken {
				t.Errorf("LoginPassword() token = %v, want %v", token, tt.wantToken)
			}
			if userID != tt.wantUserID {
				t.Errorf("LoginPassword() userID = %v, want %v", userID, tt.wantUserID)
			}
		})
	}
}
