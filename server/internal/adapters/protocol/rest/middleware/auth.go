// Package middleware содержит Gin middleware для GophKeeper.
//
// Содержит:
//   - AuthMiddleware: проверка JWT-токена и добавление user_id в контекст
package middleware

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"net/http"
	"strings"
)

// ginUserIDKey — ключ для хранения user_id в контексте Gin.
const ginUserIDKey = "user_id"

// JWTValidator — контракт для валидации JWT-токена.
//
// Принимает строку токена, возвращает user_id или ошибку.
type JWTValidator func(tokenString string) (string, error)

// AuthMiddleware создаёт middleware для аутентификации через JWT.
//
// Алгоритм:
//  1. Извлекает заголовок Authorization
//  2. Проверяет наличие и корректность префикса "Bearer "
//  3. Валидирует JWT-токен через переданную функцию
//  4. В случае успеха сохраняет user_id в контексте Gin
//  5. В случае ошибки возвращает 401 Unauthorized
//
// Параметры:
//   - validateJWT: функция валидации JWT-токена
//   - log: логгер для записи событий
//
// Возвращает gin.HandlerFunc для использования в роутере.
func AuthMiddleware(validateJWT JWTValidator, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем заголовок Authorization
		authHeader := c.GetHeader("Authorization")

		// Проверка наличия заголовка
		if authHeader == "" {
			log.Warn("request without Authorization header",
				zap.String("path", c.FullPath()),
				zap.String("method", c.Request.Method),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}
		// Проверка формата заголовка (должен начинаться с "Bearer ")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn("invalid authorization header format",
				zap.String("path", c.FullPath()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is not Bearer token, bearer token required",
			})
			return
		}

		// Извлечение токена
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		log.Debug("validating JWT token",
			zap.String("path", c.FullPath()),
		)

		// Валидация токена
		userID, err := validateJWT(tokenString)
		if err != nil {
			log.Warn("JWT validation failed",
				zap.Error(err),
				zap.String("path", c.FullPath()),
			)

			// Различные сообщения для разных ошибок
			if errors.Is(err, domain.ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			}
			c.Abort()
			return
		}

		log.Info("JWT validated successfully",
			zap.String("user_id", userID),
			zap.String("path", c.FullPath()),
		)

		// Сохранение user_id в контексте Gin для последующих handlers
		c.Set(ginUserIDKey, userID)
		c.Next()
	}
}
