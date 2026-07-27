package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gophkeeper/server/internal/domain"
	"net/http"
)

// CreateSecret — создание нового секрета
func CreateSecret(uc *domain.SecretUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получение UserID из middleware
		userIDInterface, exists := c.Get("user_id")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "user not authenticated",
			})
		}

		userID, ok := userIDInterface.(string)
		if !ok || userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
			return
		}

		// Валидация данных
		var req struct {
			Type     string `json:"type" binding:"required"`
			Title    string `json:"title" binding:"required"`
			Data     string `json:"data" binding:"required"` // зашифрованные данные в base64
			Metadata string `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error()})
			return
		}

		// Пока Data как строка base64, потом []byte
		encryptedData := []byte(req.Data)

		secret, err := uc.CreateSecret(
			c.Request.Context(),
			userID,
			req.Type,
			req.Title,
			encryptedData,
		)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidInput) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid input"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":   "secret created successfully",
			"secret_id": secret.ID,
		})
	}
}

// GetSecrets — получение списка секретов пользователя
func GetSecrets(uc *domain.SecretUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			return
		}

		userID, ok := userIDInterface.(string)
		if !ok || userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
			return
		}

		secrets, err := uc.GetSecrets(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"secrets": secrets,
		})
	}
}
