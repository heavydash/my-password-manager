package domain

import (
	"context"
	"gophkeeper/server/internal/ports"
	"testing"
)

type mockAuthPort struct {
}

func (m *mockAuthPort) CreateUser(ctx context.Context, user ports.User) (string, error) {
	return "mock-user-id", nil
}

func (m *mockAuthPort) AuthenticatePassword(ctx context.Context, email, password string) (string, error) {
	return "mock-jwt-token", nil
}

func (m *mockAuthPort) AuthenticateOAuth(ctx context.Context, provider, code string) (string, error) {
	return "mock-oauth-userID", nil
}

func (m *mockAuthPort) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	return ports.User{ID: "mock-user-id"}, nil
}

func TestAuthUseCase_Register(t *testing.T) {
	port := &mockAuthPort{}
	useCase := NewAuthUseCase(port)

	userID, err := useCase.Register(context.Background(), "test@example.com", "pass123")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if userID != "mock-user-id" {
		t.Errorf("expected mock-user-id, got %s", userID)
	}
}

func TestAuthUseCase_LoginPassword(t *testing.T) {
	port := &mockAuthPort{}
	useCase := NewAuthUseCase(port)

	token, err := useCase.LoginPassword(context.Background(), "test@example.com", "strongpassword123")

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if token != "mock-jwt-token" {
		t.Errorf("expected mock-jwt-token, got %s", token)
	}
}
