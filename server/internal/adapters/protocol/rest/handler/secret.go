package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"net/http"
)

// CreateSecret — создание нового секрета
func CreateSecret(uc *domain.SecretUseCase, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Получение UserID из middleware
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			log.Warn("unauthorized request to create secret")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "user not authenticated",
			})
		}

		userID, ok := userIDInterface.(string)
		if !ok || userID == "" {
			log.Warn("invalid user_id in context")
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
			log.Error("failed to bind request", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Info("creating secret",
			zap.String("user_id", userID),
			zap.String("type", req.Type),
			zap.String("title", req.Title),
		)

		// Конвертация string → SecretType
		secretType := domain.SecretType(req.Type)

		// Создаём секрет
		secret, err := uc.CreateSecret(
			c.Request.Context(),
			userID,
			secretType,
			req.Title,
			req.Data,
		)
		if err != nil {
			log.Error("failed to create secret",
				zap.String("user_id", userID),
				zap.Error(err),
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Info("secret created successfully",
			zap.String("secret_id", secret.ID),
			zap.String("user_id", userID),
		)

		c.JSON(http.StatusCreated, gin.H{
			"message":   "secret created successfully",
			"secret_id": secret.ID,
		})
	}
}

// GetSecrets — получение списка секретов пользователя
func GetSecrets(uc *domain.SecretUseCase, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			log.Warn("unauthorized attempt to get secrets")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
			return
		}

		userID, ok := userIDInterface.(string)
		if !ok || userID == "" {
			log.Warn("invalid user_id in context")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in context"})
			return
		}

		log.Info("fetching secrets for user", zap.String("user_id", userID))

		secrets, err := uc.GetSecrets(c.Request.Context(), userID)
		if err != nil {
			log.Error("failed to get secrets", zap.String("user_id", userID),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "internal server error"})
			return
		}

		log.Info("secrets retrieved successfully",
			zap.String("user_id", userID),
			zap.Int("count", len(secrets)),
		)

		c.JSON(http.StatusOK, gin.H{
			"secrets": secrets,
		})
	}
}

func GetSecret(uc *domain.SecretUseCase, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetString("user_id")

		if id == "" {
			log.Warn("get secret called without id")
			c.JSON(http.StatusBadRequest, gin.H{"error": "secret id is required"})
			return
		}

		log.Info("fetching secret", zap.String("secret_id", id), zap.String("user_id", userID))

		secret, err := uc.GetSecret(c.Request.Context(), id, userID)
		if err != nil {
			log.Error("failed to get secret",
				zap.String("secret_id", id),
				zap.String("user_id", userID),
				zap.Error(err),
			)
			c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
			return
		}

		log.Info("secret retrieved",
			zap.String("secret_id", id),
			zap.String("user_id", userID),
		)

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

// DeleteSecret - Удаление секрета
func DeleteSecret(uc *domain.SecretUseCase, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetString("user_id")

		if id == "" || userID == "" {
			log.Warn("delete secret called with missing parameters",
				zap.String("secret_id", id),
				zap.String("user_id", userID),
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id or user"})
			return
		}

		log.Info("deleting secret",
			zap.String("secret_id", id),
			zap.String("user_id", userID),
		)

		err := uc.DeleteSecret(c.Request.Context(), userID, id)
		if err != nil {
			log.Error("failed to delete secret",
				zap.String("secret_id", id),
				zap.String("user_id", userID),
				zap.Error(err),
			)
			if err == domain.ErrNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		log.Info("secret deleted successfully",
			zap.String("secret_id", id),
			zap.String("user_id", userID),
		)

		c.JSON(http.StatusOK, gin.H{
			"message": "secret deleted successfully",
		})
	}
}
