package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gophkeeper/server/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := TestLogger{}

	tests := []struct {
		name             string
		body             interface{}
		setupMock        func(*MockAuthUseCase)
		wantStatus       int
		wantBodyContains string
	}{
		{
			name: "success",
			body: map[string]string{"email": "test@example.com", "password": "supersecret123"},
			setupMock: func(m *MockAuthUseCase) {
				m.On("Register", mock.Anything, "test@example.com", "supersecret123").
					Return("user-123", nil)
			},
			wantStatus:       http.StatusCreated,
			wantBodyContains: "user registered successfully",
		},
		{
			name:             "binding_error",
			body:             map[string]string{"email": "invalid-email", "password": "123"},
			setupMock:        func(m *MockAuthUseCase) {},
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "Key: 'Email' Error:Field validation for 'Email' failed",
		},
		{
			name: "user_already_exists",
			body: map[string]string{"email": "exists@example.com", "password": "supersecret123"},
			setupMock: func(m *MockAuthUseCase) {
				m.On("Register", mock.Anything, mock.Anything, mock.Anything).
					Return("", domain.ErrUserAlreadyExists)
			},
			wantStatus:       http.StatusConflict,
			wantBodyContains: "user with this email already exists",
		},
		{
			name: "internal_error",
			body: map[string]string{"email": "test@example.com", "password": "supersecret123"},
			setupMock: func(m *MockAuthUseCase) {
				m.On("Register", mock.Anything, mock.Anything, mock.Anything).
					Return("", errors.New("db error"))
			},
			wantStatus:       http.StatusInternalServerError,
			wantBodyContains: "internal server error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := NewMockAuthUseCase()
			tt.setupMock(mockUC)

			h := Register(mockUC, log)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tt.body)
			c.Request, _ = http.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			h(c)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := TestLogger{}

	tests := []struct {
		name             string
		body             interface{}
		setupMock        func(*MockAuthUseCase)
		wantStatus       int
		wantBodyContains string
	}{
		{
			name: "success",
			body: map[string]string{"email": "test@example.com", "password": "pass123"},
			setupMock: func(m *MockAuthUseCase) {
				m.On("LoginPassword", mock.Anything, "test@example.com", "pass123").
					Return("jwt.token.123", "user-123", nil)
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "login successfully",
		},
		{
			name: "invalid_credentials",
			body: map[string]string{"email": "test@example.com", "password": "wrong"},
			setupMock: func(m *MockAuthUseCase) {
				m.On("LoginPassword", mock.Anything, mock.Anything, mock.Anything).
					Return("", "", domain.ErrInvalidCredentials)
			},
			wantStatus:       http.StatusUnauthorized,
			wantBodyContains: "invalid credentials",
		},
		// ... добавь binding_error и internal_error аналогично Register
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := NewMockAuthUseCase()
			tt.setupMock(mockUC)

			h := Login(mockUC, log)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tt.body)
			c.Request, _ = http.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			h(c)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			mockUC.AssertExpectations(t)
		})
	}
}

func TestProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := TestLogger{}

	tests := []struct {
		name        string
		userIDInCtx interface{}
		wantStatus  int
	}{
		{"success", "user-123", http.StatusOK},
		{"no_user_id", nil, http.StatusUnauthorized},
		{"wrong_type", 123, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := Profile(log)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.userIDInCtx != nil {
				c.Set("userID", tt.userIDInCtx)
			}

			h(c)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}
