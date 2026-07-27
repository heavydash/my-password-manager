// Package handler содержит тесты для OAuth HTTP-handlers GophKeeper.
//
// Тестируются:
//   - OAuthLogin: редирект на страницу авторизации провайдера
//   - OAuthCallback: обработка callback от провайдера
//
// Используются моки из mocks.go (MockAuthUseCase, TestLogger).
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gophkeeper/server/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestOAuthLogin тестирует handler перенаправления на OAuth-провайдера.
//
// Сценарии:
//   - google_success: успешный редирект на Google
//   - yandex_success: успешный редирект на Yandex
//   - unsupported_provider: неподдерживаемый провайдер, возвращает 400 Bad Request
//   - usecase_error: ошибка от useCase, возвращает 500 Internal Server Error
func TestOAuthLogin(t *testing.T) {
	// Включение тестового режима Gin
	gin.SetMode(gin.TestMode)
	log := TestLogger{}

	// Подготовка тестовых случаев
	tests := []struct {
		name         string                 // имя теста
		provider     string                 // провайдер в URL
		setupMock    func(*MockAuthUseCase) // настройка мока
		wantStatus   int                    // ожидаемый HTTP статус
		wantLocation string                 // ожидаемый Location header (часть URL)
		wantBody     string                 // ожидаемая подстрока в теле ответа
	}{{
		name:     "google_success",
		provider: "google",
		setupMock: func(m *MockAuthUseCase) {
			m.On("GetOAuthURL", "google").Return("https://accounts.google.com/o/oauth2/auth?...", "state123", nil)
		},
		wantStatus:   http.StatusFound,
		wantLocation: "https://accounts.google.com",
	},
		{
			name:     "yandex_success",
			provider: "yandex",
			setupMock: func(m *MockAuthUseCase) {
				m.On("GetOAuthURL", "yandex").Return("https://oauth.yandex.ru/authorize?...", "state123", nil)
			},
			wantStatus:   http.StatusFound,
			wantLocation: "https://oauth.yandex.ru",
		},
		{
			name:       "unsupported_provider",
			provider:   "facebook",
			setupMock:  func(*MockAuthUseCase) {},
			wantStatus: http.StatusBadRequest,
			wantBody:   "unsupported provider",
		},
		{
			name:     "usecase_error",
			provider: "google",
			setupMock: func(m *MockAuthUseCase) {
				m.On("GetOAuthURL", "google").Return("", "", domain.ErrOAuthUnavailable)
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error",
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создание и настройка мока
			mockUC := NewMockAuthUseCase()
			tt.setupMock(mockUC)

			// Создание handler
			h := NewOAuthHandler(mockUC, log)

			// Подготовка HTTP запроса
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request, _ = http.NewRequest(http.MethodGet, "/auth/"+tt.provider, nil)
			c.Params = gin.Params{{Key: "provider", Value: tt.provider}}

			// Вызов handler
			h.OAuthLogin(c)

			// Проверка результатов
			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantLocation != "" {
				assert.Contains(t, w.Header().Get("Location"), tt.wantLocation)
			}
			if tt.wantBody != "" {
				assert.Contains(t, w.Body.String(), tt.wantBody)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

// TestOAuthCallback тестирует handler обработки callback от OAuth-провайдера.
//
// Сценарии:
//   - google_success: успешная обработка callback Google, возвращает 200 OK с code
//   - yandex_success: успешная обработка callback Yandex, возвращает 200 OK
//   - empty_code: пустой code, возвращает 400 Bad Request
//   - empty_state: пустой state, возвращает 400 Bad Request
//   - error_from_usecase: ошибка от useCase, возвращает 400 Bad Request
func TestOAuthCallback(t *testing.T) {
	// Включение тестового режима Gin
	gin.SetMode(gin.TestMode)
	log := TestLogger{}

	// Подготовка тестовых случаев
	tests := []struct {
		name             string                 // имя теста
		provider         string                 // провайдер в URL
		code             string                 // query параметр code
		state            string                 // query параметр state
		setupMock        func(*MockAuthUseCase) // настройка мока
		wantStatus       int                    // ожидаемый HTTP статус
		wantBodyContains string                 // ожидаемая подстрока в теле ответа
	}{
		{
			name:     "success",
			provider: "google",
			code:     "4/0A...valid",
			state:    "state-xyz",
			setupMock: func(m *MockAuthUseCase) {
				m.On("HandleOAuthCallback", "google", "4/0A...valid", "state-xyz").
					Return("one-time-code-abc123", nil)
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "one-time-code-abc123",
		},
		{
			name:     "yandex_success",
			provider: "yandex",
			code:     "yandex-code-999",
			state:    "state-ya",
			setupMock: func(m *MockAuthUseCase) {
				m.On("HandleOAuthCallback", "yandex", "yandex-code-999", "state-ya").
					Return("one-time-999", nil)
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "Success! Copy this code",
		},
		{
			name:             "empty_code",
			provider:         "google",
			code:             "",
			state:            "state-xyz",
			setupMock:        func(m *MockAuthUseCase) {},
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "code is required",
		},
		{
			name:     "error_from_usecase",
			provider: "google",
			code:     "bad-code",
			state:    "state-xyz",
			setupMock: func(m *MockAuthUseCase) {
				m.On("HandleOAuthCallback", "google", "bad-code", "state-xyz").
					Return("", domain.ErrInvalidOAuthState)
			},
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid state or expired",
		},
		{
			name:             "empty_state",
			provider:         "google",
			code:             "some-code",
			state:            "",
			setupMock:        func(m *MockAuthUseCase) {},
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "state is required",
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создание и настройка мока
			mockUC := NewMockAuthUseCase()
			tt.setupMock(mockUC)

			// Создание handler
			h := NewOAuthHandler(mockUC, log)

			// Подготовка HTTP запроса
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "provider", Value: tt.provider}}
			c.Request, _ = http.NewRequest(http.MethodGet,
				"/callback?code="+tt.code+"&state="+tt.state, nil)

			// Вызов handler
			h.OAuthCallback(c)

			// Проверка результатов
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			}
			mockUC.AssertExpectations(t)
		})
	}
}
