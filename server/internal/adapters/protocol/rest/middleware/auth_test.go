// Package middleware содержит тесты для Gin middleware GophKeeper.
//
// Тестируются:
//   - AuthMiddleware: проверка JWT-токена и добавление user_id в контекст
package middleware

import (
	"errors"
	"go.uber.org/zap"
	"gophkeeper/server/internal/logger"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"gophkeeper/server/internal/domain"
)

// TestAuthMiddleware тестирует middleware аутентификации.
//
// Сценарии:
//   - success: валидный токен, проходит аутентификацию
//   - missing_header: отсутствует заголовок Authorization
//   - invalid_format: неверный формат заголовка (нет Bearer)
//   - empty_token: пустой токен
//   - expired_token: истёкший токен
//   - invalid_token: невалидный токен
func TestAuthMiddleware(t *testing.T) {
	// Включение тестового режима Gin
	gin.SetMode(gin.TestMode)

	// Тестовые функции валидации JWT
	validValidator := func(token string) (string, error) {
		if token == "valid-token" {
			return "user-123", nil
		}
		return "", errors.New("invalid token")
	}

	expiredValidator := func(token string) (string, error) {
		if token == "expired-token" {
			return "", domain.ErrTokenExpired
		}
		return "", errors.New("invalid token")
	}

	// Подготовка тестовых случаев
	tests := []struct {
		name            string
		authHeader      string
		validator       JWTValidator
		wantStatus      int
		wantBody        string
		wantUserIDInCtx string
	}{
		{
			name:            "success",
			authHeader:      "Bearer valid-token",
			validator:       validValidator,
			wantStatus:      http.StatusOK,
			wantUserIDInCtx: "user-123",
		},
		{
			name:       "missing_header",
			authHeader: "",
			validator:  validValidator,
			wantStatus: http.StatusUnauthorized,
			wantBody:   "no authorization header",
		},
		{
			name:       "invalid_format",
			authHeader: "Basic token123",
			validator:  validValidator,
			wantStatus: http.StatusUnauthorized,
			wantBody:   "Authorization header is not Bearer token, bearer token required",
		},
		{
			name:       "empty_token",
			authHeader: "Bearer ",
			validator:  validValidator,
			wantStatus: http.StatusUnauthorized,
			wantBody:   "invalid token",
		},
		{
			name:       "expired_token",
			authHeader: "Bearer expired-token",
			validator:  expiredValidator,
			wantStatus: http.StatusUnauthorized,
			wantBody:   "token expired",
		},
		{
			name:       "invalid_token",
			authHeader: "Bearer wrong-token",
			validator:  validValidator,
			wantStatus: http.StatusUnauthorized,
			wantBody:   "invalid token",
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создание middleware
			middleware := AuthMiddleware(tt.validator, &testLogger{})

			// Создание тестового handler (просто возвращает 200 OK)
			testHandler := func(c *gin.Context) {
				userID, exists := c.Get(ginUserIDKey)
				if tt.wantUserIDInCtx != "" {
					assert.True(t, exists)
					assert.Equal(t, tt.wantUserIDInCtx, userID)
				}
				c.Status(http.StatusOK)
			}

			// Подготовка HTTP запроса
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				c.Request.Header.Set("Authorization", tt.authHeader)
			}

			// Вызов middleware + handler
			middleware(c)
			if c.IsAborted() {
				// Если middleware прервал, не вызываем handler
			} else {
				testHandler(c)
			}

			// Проверка результатов
			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBody != "" {
				assert.Contains(t, w.Body.String(), tt.wantBody)
			}
		})
	}
}

// TestAuthMiddleware_NextCalled проверяет, что при успешной аутентификации
// middleware вызывает следующий handler.
func TestAuthMiddleware_NextCalled(t *testing.T) {
	// Включение тестового режима Gin
	gin.SetMode(gin.TestMode)

	// Создание middleware с валидным валидатором
	validator := func(token string) (string, error) {
		return "user-123", nil
	}
	middleware := AuthMiddleware(validator, &testLogger{})

	// Флаг вызова next handler
	nextCalled := false
	testHandler := func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	}

	// Подготовка HTTP запроса
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer valid-token")

	// Вызов middleware
	middleware(c)
	if !c.IsAborted() {
		testHandler(c)
	}

	// Проверка
	assert.True(t, nextCalled, "next handler should be called")
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAuthMiddleware_AbortOnError проверяет, что при ошибке аутентификации
// middleware прерывает выполнение и не вызывает следующий handler.
func TestAuthMiddleware_AbortOnError(t *testing.T) {
	// Включение тестового режима Gin
	gin.SetMode(gin.TestMode)

	// Создание middleware с валидатором, который всегда возвращает ошибку
	validator := func(token string) (string, error) {
		return "", errors.New("invalid token")
	}
	middleware := AuthMiddleware(validator, &testLogger{})

	// Флаг вызова next handler
	nextCalled := false
	testHandler := func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	}

	// Подготовка HTTP запроса
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid-token")

	// Вызов middleware
	middleware(c)
	if !c.IsAborted() {
		testHandler(c)
	}

	// Проверка
	assert.False(t, nextCalled, "next handler should not be called")
	assert.True(t, c.IsAborted(), "request should be aborted")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// testLogger — тестовый логгер, реализующий logger.Logger.
type testLogger struct{}

func (l *testLogger) Debug(msg string, fields ...zap.Field) {}
func (l *testLogger) Info(msg string, fields ...zap.Field)  {}
func (l *testLogger) Warn(msg string, fields ...zap.Field)  {}
func (l *testLogger) Error(msg string, fields ...zap.Field) {}
func (l *testLogger) Fatal(msg string, fields ...zap.Field) {}
func (l *testLogger) With(fields ...zap.Field) logger.Logger {
	return l
}
func (l *testLogger) Sync() error { return nil }
