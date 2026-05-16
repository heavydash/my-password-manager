package handler

import (
	"context"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
)

type MockAuthUseCase struct {
	mock.Mock
	*domain.AuthUseCase
}

type TestLogger struct{}

func (TestLogger) Debug(msg string, fields ...zap.Field)  {}
func (TestLogger) Info(msg string, fields ...zap.Field)   {}
func (TestLogger) Warn(msg string, fields ...zap.Field)   {}
func (TestLogger) Error(msg string, fields ...zap.Field)  {}
func (TestLogger) With(fields ...zap.Field) logger.Logger { return TestLogger{} }
func (TestLogger) Sync() error                            { return nil }

func NewMockAuthUseCase() *MockAuthUseCase {
	return &MockAuthUseCase{AuthUseCase: &domain.AuthUseCase{}}
}

func (m *MockAuthUseCase) Register(ctx context.Context, email, password string) (string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Error(1)
}

func (m *MockAuthUseCase) LoginPassword(ctx context.Context, email, password string) (string, string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockAuthUseCase) GetOAuthURL(provider string) (string, string, error) {
	args := m.Called(provider)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockAuthUseCase) HandleOAuthCallback(provider, code, state string) (string, error) {
	args := m.Called(provider, code, state)
	return args.String(0), args.Error(1)
}

type MockSecretUseCase struct {
	mock.Mock
	*domain.SecretUseCase
}

func NewMockSecretUseCase() *MockSecretUseCase {
	return &MockSecretUseCase{
		SecretUseCase: &domain.SecretUseCase{},
	}
}

func (m *MockSecretUseCase) CreateSecret(ctx context.Context, userID string, typ domain.SecretType, title, data string) (*ports.Secret, error) {
	args := m.Called(ctx, userID, typ, title, data)
	if sec := args.Get(0); sec != nil {
		return sec.(*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSecretUseCase) GetSecrets(ctx context.Context, userID string) ([]*ports.Secret, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*ports.Secret), args.Error(1)
}

func (m *MockSecretUseCase) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	args := m.Called(ctx, id, userID)
	if sec := args.Get(0); sec != nil {
		return sec.(*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockSecretUseCase) DeleteSecret(ctx context.Context, userID, id string) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}
