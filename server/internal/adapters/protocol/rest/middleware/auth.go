package middleware

import (
	"github.com/gin-gonic/gin"
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "no authorization header"})
			c.Abort()
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Успешная валидация — сохраняем user_id в контексте Gin
		c.Set("user_id", userID)
		c.Next()
	}
}
