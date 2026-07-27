// Package handler содержит тесты для HTTP-handlers (Gin) GophKeeper.
//
// Тестируются:
//   - Register: регистрация пользователя
//   - Login: вход по email/паролю
//   - Profile: защищённый эндпоинт
//
// Используются моки из mocks.go (MockAuthUseCase, TestLogger).
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

// TestRegister тестирует handler регистрации пользователя.
//
// Сценарии:
//   - success: успешная регистрация, возвращает 201 Created
//   - binding_error_invalid_email: неверный формат email, возвращает 400 Bad Request
//   - user_already_exists: пользователь уже существует, возвращает 409 Conflict
//   - internal_error: ошибка БД, возвращает 500 Internal Server Error
func TestRegister(t *testing.T) {
	// Включение тестового режима Gin для стабильного вывода ошибок
	gin.SetMode(gin.TestMode)
	log := TestLogger{}

	// Подготовка тестовых случаев
	tests := []struct {
		name             string                 // имя теста
		body             interface{}            // тело запроса
		setupMock        func(*MockAuthUseCase) // настройка мока
		wantStatus       int                    // ожидаемый HTTP статус
		wantBodyContains string                 // ожидаемая подстрока в ответе
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
	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создание и настройка мока
			mockUC := NewMockAuthUseCase()
			if tt.setupMock != nil {
				tt.setupMock(mockUC)
			}

			// Создание handler
			h := Register(mockUC, log)

			// Подготовка HTTP запроса
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tt.body)
			c.Request, _ = http.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
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

// TestLogin тестирует handler входа пользователя.
//
// Сценарии:
//   - success: успешный вход, возвращает 200 OK с токеном
//   - invalid_credentials: неверный пароль, возвращает 401 Unauthorized
//   - binding_error: невалидные данные, возвращает 400 Bad Request
//   - internal_error: ошибка БД, возвращает 500 Internal Server Error
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
		{
			name:             "binding_error",
			body:             map[string]string{"password": "pass123"},
			setupMock:        func(m *MockAuthUseCase) {},
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "Email",
		},
		{
			name: "internal_error",
			body: map[string]string{"email": "test@example.com", "password": "pass123"},
			setupMock: func(m *MockAuthUseCase) {
				m.On("LoginPassword", mock.Anything, "test@example.com", "pass123").
					Return("", "", errors.New("db error"))
			},
			wantStatus:       http.StatusInternalServerError,
			wantBodyContains: "internal server error",
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создание и настройка мока
			mockUC := NewMockAuthUseCase()
			tt.setupMock(mockUC)

			// Создание handler
			h := Login(mockUC, log)

			// Подготовка HTTP запроса
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tt.body)
			c.Request, _ = http.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			// Вызов handler
			h(c)

			// Проверка результатов
			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			mockUC.AssertExpectations(t)
		})
	}
}

// TestProfile тестирует защищённый эндпоинт Profile.
//
// Сценарии:
//   - success: userID есть в контексте, возвращает 200 OK
//   - no_user_id: userID отсутствует в контексте, возвращает 401 Unauthorized
//   - wrong_type: userID имеет неверный тип, возвращает 401 Unauthorized
func TestProfile(t *testing.T) {
	// Включение тестового режима Gin
	gin.SetMode(gin.TestMode)
	log := TestLogger{}

	// Подготовка тестовых случаев
	tests := []struct {
		name        string      // имя теста
		userIDInCtx interface{} // значение userID в контексте
		wantStatus  int         // ожидаемый HTTP статус
	}{
		{"success", "user-123", http.StatusOK},
		{"no_user_id", nil, http.StatusUnauthorized},
		{"wrong_type", 123, http.StatusUnauthorized},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создание handler
			h := Profile(log)

			// Подготовка HTTP запроса с контекстом
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			if tt.userIDInCtx != nil {
				c.Set("userID", tt.userIDInCtx)
			}

			// Вызов handler
			h(c)

			// Проверка результатов
			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}
