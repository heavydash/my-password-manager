package handler

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/ports"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := TestLogger{}
	mockUC := NewMockSecretUseCase()

	h := CreateSecret(mockUC, log)

	tests := []struct {
		name             string
		userIDInCtx      string
		body             interface{}
		setupMock        func()
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:        "success",
			userIDInCtx: "user-123",
			body:        map[string]string{"type": "password", "title": "My Bank", "data": "pass123"},
			setupMock: func() {
				secret := &ports.Secret{ID: "sec-456"}
				mockUC.On("CreateSecret", mock.Anything, "user-123", domain.SecretType("password"), "My Bank", "pass123").
					Return(secret, nil)
			},
			wantStatus:       http.StatusCreated,
			wantBodyContains: "secret created successfully",
		},
		{
			name:        "unauthorized_no_user_id",
			userIDInCtx: "",
			body:        map[string]string{"type": "password", "title": "Test"},
			setupMock:   func() {},
			wantStatus:  http.StatusUnauthorized,
		},
		{
			name:        "invalid_input",
			userIDInCtx: "user-123",
			body:        map[string]string{"type": "", "title": ""},
			setupMock: func() {
				mockUC.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, domain.ErrInvalidInput)
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.userIDInCtx != "" {
				c.Set("user_id", tt.userIDInCtx) // ← проверь ключ! ("user_id" или "userID")
			}

			bodyBytes, _ := json.Marshal(tt.body)
			c.Request, _ = http.NewRequest(http.MethodPost, "/secrets", bytes.NewReader(bodyBytes))
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

func TestGetSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := TestLogger{}
	mockUC := NewMockSecretUseCase()

	h := GetSecrets(mockUC, log)

	tests := []struct {
		name             string
		userIDInCtx      string
		setupMock        func()
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:        "success",
			userIDInCtx: "user-123",
			setupMock: func() {
				secrets := []*ports.Secret{
					{ID: "sec1", Title: "Bank"},
					{ID: "sec2", Title: "Note"},
				}
				mockUC.On("GetSecrets", mock.Anything, "user-123").Return(secrets, nil)
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: `"secrets"`,
		},
		{
			name:        "unauthorized",
			userIDInCtx: "",
			setupMock:   func() {},
			wantStatus:  http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.userIDInCtx != "" {
				c.Set("user_id", tt.userIDInCtx)
			}

			c.Request, _ = http.NewRequest(http.MethodGet, "/secrets", nil)

			h(c)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

func TestGetSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := TestLogger{}
	mockUC := NewMockSecretUseCase()

	h := GetSecret(mockUC, log)

	tests := []struct {
		name             string
		id               string
		userIDInCtx      string
		setupMock        func()
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:        "success",
			id:          "sec-456",
			userIDInCtx: "user-123",
			setupMock: func() {
				secret := &ports.Secret{
					ID:    "sec-456",
					Title: "My Bank",
					Type:  "password",
				}
				mockUC.On("GetSecret", mock.Anything, "sec-456", "user-123").Return(secret, nil)
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: `"title":"My Bank"`,
		},
		{
			name:        "no_id",
			id:          "",
			userIDInCtx: "user-123",
			setupMock:   func() {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "not_found",
			id:          "sec-999",
			userIDInCtx: "user-123",
			setupMock: func() {
				mockUC.On("GetSecret", mock.Anything, "sec-999", "user-123").
					Return(nil, domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.userIDInCtx != "" {
				c.Set("user_id", tt.userIDInCtx)
			}
			c.Params = gin.Params{{Key: "id", Value: tt.id}}
			c.Request, _ = http.NewRequest(http.MethodGet, "/secrets/"+tt.id, nil)

			h(c)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

func TestDeleteSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := TestLogger{}
	mockUC := NewMockSecretUseCase()

	h := DeleteSecret(mockUC, log)

	tests := []struct {
		name             string
		id               string
		userIDInCtx      string
		setupMock        func()
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:        "success",
			id:          "sec-456",
			userIDInCtx: "user-123",
			setupMock: func() {
				mockUC.On("DeleteSecret", mock.Anything, "user-123", "sec-456").Return(nil)
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "secret deleted successfully",
		},
		{
			name:        "missing_params",
			id:          "",
			userIDInCtx: "user-123",
			setupMock:   func() {},
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "not_found",
			id:          "sec-999",
			userIDInCtx: "user-123",
			setupMock: func() {
				mockUC.On("DeleteSecret", mock.Anything, "user-123", "sec-999").
					Return(domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.userIDInCtx != "" {
				c.Set("user_id", tt.userIDInCtx)
			}
			c.Params = gin.Params{{Key: "id", Value: tt.id}}
			c.Request, _ = http.NewRequest(http.MethodDelete, "/secrets/"+tt.id, nil)

			h(c)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			}
			mockUC.AssertExpectations(t)
		})
	}
}
