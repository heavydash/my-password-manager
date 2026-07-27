// Package handler содержит моки для тестирования HTTP-handlers GophKeeper.
//
// Содержит:
//   - MockAuthUseCase: мок для ports.AuthPort
//   - MockSecretUseCase: мок для ports.SecretUseCase
//   - TestLogger: мок для logger.Logger
package handler

import (
	"context"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
)

// MockAuthUseCase — мок для ports.AuthPort.
type MockAuthUseCase struct {
	mock.Mock
	*domain.AuthUseCase
}

// TestLogger — мок для logger.Logger.
// Игнорирует все вызовы логирования.
type TestLogger struct{}

func (TestLogger) Debug(msg string, fields ...zap.Field)  {}
func (TestLogger) Info(msg string, fields ...zap.Field)   {}
func (TestLogger) Warn(msg string, fields ...zap.Field)   {}
func (TestLogger) Error(msg string, fields ...zap.Field)  {}
func (TestLogger) Fatal(msg string, fields ...zap.Field)  {}
func (TestLogger) With(fields ...zap.Field) logger.Logger { return TestLogger{} }
func (TestLogger) Sync() error                            { return nil }

// NewMockAuthUseCase создаёт новый мок для AuthUseCase.
func NewMockAuthUseCase() *MockAuthUseCase {
	return &MockAuthUseCase{AuthUseCase: &domain.AuthUseCase{}}
}

// Register мокирует регистрацию пользователя.
func (m *MockAuthUseCase) Register(ctx context.Context, email, password string) (string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Error(1)
}

// LoginPassword мокирует вход по паролю.
func (m *MockAuthUseCase) LoginPassword(ctx context.Context, email, password string) (string, string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.String(1), args.Error(2)
}

// GetOAuthURL мокирует получение URL для OAuth-провайдера.
func (m *MockAuthUseCase) GetOAuthURL(provider string) (string, string, error) {
	args := m.Called(provider)
	return args.String(0), args.String(1), args.Error(2)
}

// HandleOAuthCallback мокирует обработку callback от OAuth-провайдера.
func (m *MockAuthUseCase) HandleOAuthCallback(provider, code, state string) (string, error) {
	args := m.Called(provider, code, state)
	return args.String(0), args.Error(1)
}

// MockSecretUseCase — мок для ports.SecretUseCase.
type MockSecretUseCase struct {
	mock.Mock
	*domain.SecretUseCase
}

// NewMockSecretUseCase создаёт новый мок для SecretUseCase.
func NewMockSecretUseCase() *MockSecretUseCase {
	return &MockSecretUseCase{
		SecretUseCase: &domain.SecretUseCase{},
	}
}

// CreateSecret мокирует создание секрета.
func (m *MockSecretUseCase) CreateSecret(ctx context.Context, userID string, typ ports.SecretType, title, data string) (*ports.Secret, error) {
	args := m.Called(ctx, userID, typ, title, data)
	if sec := args.Get(0); sec != nil {
		return sec.(*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetSecrets мокирует получение списка секретов пользователя.
func (m *MockSecretUseCase) GetSecrets(ctx context.Context, userID string) ([]*ports.Secret, error) {
	args := m.Called(ctx, userID)
	if secrets := args.Get(0); secrets != nil {
		return secrets.([]*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

// GetSecret мокирует получение одного секрета по ID.
func (m *MockSecretUseCase) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	args := m.Called(ctx, id, userID)
	if sec := args.Get(0); sec != nil {
		return sec.(*ports.Secret), args.Error(1)
	}
	return nil, args.Error(1)
}

// DeleteSecret мокирует удаление секрета.
func (m *MockSecretUseCase) DeleteSecret(ctx context.Context, userID, id string) error {
	args := m.Called(ctx, userID, id)
	return args.Error(0)
}
