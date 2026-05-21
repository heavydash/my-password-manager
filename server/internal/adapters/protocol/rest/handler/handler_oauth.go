// Package handler содержит HTTP-handlers (Gin) для GophKeeper.
//
// Содержит OAuth handlers:
//   - OAuthLogin: редирект на страницу авторизации провайдера
//   - OAuthCallback: обработка callback от провайдера, возвращает code для обмена на JWT
package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"net/http"
	"strings"
)

// OAuthHandler обрабатывает OAuth2-аутентификацию через внешних провайдеров.
//
// Поддерживаемые провайдеры:
//   - google
//   - yandex
type OAuthHandler struct {
	useCase ports.AuthPort
	log     logger.Logger
}

// NewOAuthHandler создаёт новый OAuth handler.
//
// Параметры:
//   - useCase: порт аутентификации (реализует OAuth методы)
//   - log: логгер для записи событий
//
// Возвращает готовый OAuthHandler.
func NewOAuthHandler(useCase ports.AuthPort, log logger.Logger) *OAuthHandler {
	return &OAuthHandler{
		useCase: useCase,
		log:     log,
	}
}

// OAuthLogin перенаправляет пользователя на страницу авторизации OAuth-провайдера.
//
// Алгоритм:
//  1. Извлекает provider из URL параметра
//  2. Проверяет, что провайдер поддерживается
//  3. Получает URL авторизации от useCase
//  4. Выполняет редирект (302 Found)
//
// Параметры URL:
//   - provider: google или yandex
//
// Пример запроса:
//
//	GET /api/v1/auth/oauth/login/google
func (h *OAuthHandler) OAuthLogin(c *gin.Context) {
	// Извлечение и нормализация имени провайдера
	provider := strings.ToLower(c.Param("provider"))

	// Проверка поддержки провайдера
	switch provider {
	case domain.ProviderGoogle, domain.ProviderYandex:
		// поддерживается, продолжаем
	default:
		h.log.Info("unsupported oauth provider", zap.String("provider", provider))
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	// Получение URL авторизации
	// state игнорируется, так как уже сохранён в БД внутри useCase
	url, _, err := h.useCase.GetOAuthURL(provider)
	if err != nil {
		h.log.Error("failed to get oauth url",
			zap.String("provider", provider), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.log.Info("redirecting to oauth provider", zap.String("provider", provider), zap.String("url", url))
	c.Redirect(http.StatusFound, url)
}

// OAuthCallback обрабатывает callback от OAuth-провайдера после авторизации пользователя.
//
// Алгоритм:
//  1. Извлекает provider из URL параметра
//  2. Извлекает code и state из query параметров
//  3. Валидирует обязательность параметров
//  4. Вызывает useCase для обмена code на временный код
//  5. Возвращает временный код клиенту (для последующего обмена на JWT)
//
// Параметры запроса:
//   - code: авторизационный код от провайдера
//   - state: CSRF-токен
//
// Пример запроса:
//
//	GET /api/v1/auth/oauth/callback/google?code=xxx&state=yyy
//
// Ответ:
//
//	{
//	    "message": "success! copy this code to the app:",
//	    "code": "temp_code_xxx"
//	}
func (h *OAuthHandler) OAuthCallback(c *gin.Context) {
	// Извлечение параметров
	provider := strings.ToLower(c.Param("provider"))
	code := c.Query("code")
	state := c.Query("state")

	// Валидация входных данных
	if provider == "" {
		h.log.Warn("oauth callback: empty provider")
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider is required"})
		return
	}
	if code == "" {
		h.log.Warn("oauth callback: empty code", zap.String("provider", provider))
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}
	if state == "" {
		h.log.Warn("oauth callback: empty state", zap.String("provider", provider))
		c.JSON(http.StatusBadRequest, gin.H{"error": "state is required"})
		return
	}

	h.log.Info("processing oauth callback",
		zap.String("provider", provider),
	)

	// Вызов useCase для получения временного кода
	tempCode, err := h.useCase.HandleOAuthCallback(provider, code, state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("oauth callback successful",
		zap.String("provider", provider),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Success! Copy this code to the app:",
		"code":    tempCode,
	})
}
