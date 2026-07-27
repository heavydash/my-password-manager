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

// JWTValidator — абстрактный контракт для валидации JWT.
type JWTValidator func(tokenString string) (string, error)

func AuthMiddleware(validateJWT JWTValidator, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем заголовок Authorization
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			log.Warn("request without Authorization header",
				zap.String("path", c.FullPath()),
				zap.String("method", c.Request.Method),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
			return
		}
		// Проверяем наличие и корректность префикса Bearer
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn("invalid authorization header format",
				zap.String("path", c.FullPath()),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is not Bearer token, bearer token required",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		log.Debug("validating JWT token",
			zap.String("path", c.FullPath()),
		)

		// Валидируем токен через переданную функцию
		userID, err := validateJWT(tokenString)
		if err != nil {
			log.Warn("JWT validation failed",
				zap.Error(err),
				zap.String("path", c.FullPath()),
			)

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

		// Успешная валидация — сохраняем user_id в контексте Gin
		c.Set("user_id", userID)
		c.Next()
	}
}
