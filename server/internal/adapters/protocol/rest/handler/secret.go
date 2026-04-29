package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gophkeeper/server/internal/domain"
	"net/http"
)

// CreateSecret — создание нового секрета
func CreateSecret(uc *domain.SecretUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("=== CREATE SECRET HANDLER CALLED ===")
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

		fmt.Printf("DEBUG: UserID = %s\n", userID)

		// Простой binding без строгой валидации
		var req struct {
			Type     string `json:"type"`
			Title    string `json:"title"`
			Data     string `json:"data"`
			Metadata string `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		encryptedDataBase64 := req.Data

		fmt.Printf("Data length = %d\n", len(encryptedDataBase64))

		// Создаём секрет
		secret, err := uc.CreateSecret(
			c.Request.Context(),
			userID,
			req.Type,
			req.Title,
			req.Data,
		)
		if err != nil {
			fmt.Printf("USECASE ERROR: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

func GetSecret(uc *domain.SecretUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetString("user_id")

		secret, err := uc.GetSecret(c.Request.Context(), id, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
			return
		}

		resp := map[string]interface{}{
			"id":         secret.ID,
			"user_id":    secret.UserID,
			"type":       secret.Type,
			"title":      secret.Title,
			"data":       secret.Data,
			"metadata":   secret.Metadata,
			"created_at": secret.CreatedAt,
			"updated_at": secret.UpdatedAt,
		}

		c.JSON(http.StatusOK, resp)
	}
}
