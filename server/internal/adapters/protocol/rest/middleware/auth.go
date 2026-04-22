package middleware

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gophkeeper/server/internal/domain"
	"net/http"
	"strings"
)

// JWTValidator — абстрактный контракт для валидации JWT.
type JWTValidator func(tokenString string) (string, error)

func AuthMiddleware(validateJWT JWTValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем заголовок Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "No Authorization header found, authorization header is required",
			})
			return
		}
		// Проверяем наличие и корректность префикса Bearer
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is not Bearer token, bearer token required",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		tokenString = strings.TrimSpace(tokenString)

		// Валидируем токен через переданную функцию
		userID, err := validateJWT(tokenString)
		if err != nil {
			if errors.Is(err, domain.ErrTokenExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Token is expired",
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Token is invalid",
			})
			return
		}
		// Успешная валидация — сохраняем user_id в контексте Gin
		// Теперь любой handler ниже по цепочке может достать его через c.Get("user_id")
		c.Set("user_id", userID)

		// Продолжаем выполнение следующего middleware / handler
		c.Next()
	}
}
