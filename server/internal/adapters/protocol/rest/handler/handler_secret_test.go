// Package handler содержит тесты для HTTP-handlers управления секретами GophKeeper.
//
// Тестируются:
//   - CreateSecret: создание секрета
//   - GetSecrets: получение списка секретов
//   - GetSecret: получение одного секрета
//   - DeleteSecret: удаление секрета
//
// Используются моки из mocks.go (MockSecretUseCase, TestLogger).
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

// TestCreateSecret тестирует создание секрета.
//
// Сценарии:
//   - success: успешное создание секрета
//   - unauthorized_no_user_id: отсутствует user_id в контексте
//   - invalid_input: невалидные входные данные
func TestCreateSecret(t *testing.T) {
	// Включение тестового режима Gin
	gin.SetMode(gin.TestMode)
	log := TestLogger{}
	mockUC := NewMockSecretUseCase()

	h := CreateSecret(mockUC, log)

	// Подготовка тестовых случаев
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
				mockUC.On("CreateSecret", mock.Anything, "user-123", ports.SecretType("password"), "My Bank", "pass123").
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
			body:        map[string]string{"type": "valid_type", "title": "title", "data": "data"},
			setupMock: func() {
				mockUC.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, domain.ErrInvalidInput)
			},

			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid input",
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сброс мока перед каждым тестом
			mockUC.ExpectedCalls = nil
			tt.setupMock()

			// Подготовка HTTP запроса
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.userIDInCtx != "" {
				c.Set(ginUserIDKey, tt.userIDInCtx)
			}

			bodyBytes, _ := json.Marshal(tt.body)
			c.Request, _ = http.NewRequest(http.MethodPost, "/secrets", bytes.NewReader(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			// Вызов handler
			h(c)

			// Проверка результатов
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

// TestGetSecrets тестирует получение списка секретов.
//
// Сценарии:
//   - success: успешное получение списка
//   - unauthorized: отсутствует user_id в контексте
//   - internal_error: ошибка от useCase
func TestGetSecrets(t *testing.T) {
	// Включение тестового режима Gin
	gin.SetMode(gin.TestMode)
	log := TestLogger{}
	mockUC := NewMockSecretUseCase()

	h := GetSecrets(mockUC, log)

	// Подготовка тестовых случаев
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
		{
			name:        "internal_error",
			userIDInCtx: "user-123",
			setupMock: func() {
				mockUC.On("GetSecrets", mock.Anything, "user-123").Return(nil, domain.ErrInternal)
			},
			wantStatus:       http.StatusInternalServerError,
			wantBodyContains: "internal server error",
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сброс мока перед каждым тестом
			mockUC.ExpectedCalls = nil
			tt.setupMock()

			// Подготовка HTTP запроса
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.userIDInCtx != "" {
				c.Set(ginUserIDKey, tt.userIDInCtx)
			}

			c.Request, _ = http.NewRequest(http.MethodGet, "/secrets", nil)

			// Вызов handler
			h(c)

			// Проверка результатов
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

// TestGetSecret тестирует получение одного секрета по ID.
//
// Сценарии:
//   - success: успешное получение секрета
//   - no_id: не указан ID секрета
//   - not_found: секрет не найден
//   - unauthorized: отсутствует user_id в контексте
func TestGetSecret(t *testing.T) {
	// Включение тестового режима Gin
	gin.SetMode(gin.TestMode)
	log := TestLogger{}
	mockUC := NewMockSecretUseCase()

	h := GetSecret(mockUC, log)

	// Подготовка тестовых случаев
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
		{
			name:        "unauthorized",
			id:          "sec-456",
			userIDInCtx: "",
			setupMock:   func() {},
			wantStatus:  http.StatusUnauthorized,
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сброс мока перед каждым тестом
			mockUC.ExpectedCalls = nil
			tt.setupMock()

			// Подготовка HTTP запроса
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.userIDInCtx != "" {
				c.Set(ginUserIDKey, tt.userIDInCtx)
			}
			c.Params = gin.Params{{Key: "id", Value: tt.id}}
			c.Request, _ = http.NewRequest(http.MethodGet, "/secrets/"+tt.id, nil)

			// Вызов handler
			h(c)

			// Проверка результатов
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

// TestDeleteSecret тестирует удаление секрета.
//
// Сценарии:
//   - success: успешное удаление секрета
//   - missing_params: не указан ID секрета
//   - not_found: секрет не найден
//   - unauthorized: отсутствует user_id в контексте
func TestDeleteSecret(t *testing.T) {
	// Включение тестового режима Gin
	gin.SetMode(gin.TestMode)
	log := TestLogger{}
	mockUC := NewMockSecretUseCase()

	h := DeleteSecret(mockUC, log)

	// Подготовка тестовых случаев
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
		{
			name:        "unauthorized",
			id:          "sec-456",
			userIDInCtx: "",
			setupMock:   func() {},
			wantStatus:  http.StatusUnauthorized,
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сброс мока перед каждым тестом
			mockUC.ExpectedCalls = nil
			tt.setupMock()

			// Подготовка HTTP запроса
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.userIDInCtx != "" {
				c.Set(ginUserIDKey, tt.userIDInCtx)
			}
			c.Params = gin.Params{{Key: "id", Value: tt.id}}
			c.Request, _ = http.NewRequest(http.MethodDelete, "/secrets/"+tt.id, nil)

			// Вызов handler
			h(c)

			// Проверка результатов
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			}
			mockUC.AssertExpectations(t)
		})
	}
}
