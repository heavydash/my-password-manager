// Package handler содержит HTTP-handlers (Gin) для управления секретами GophKeeper.
//
// Содержит handlers:
//   - CreateSecret: создание нового секрета
//   - GetSecrets: получение списка всех секретов пользователя
//   - GetSecret: получение одного секрета по ID
//   - DeleteSecret: удаление секрета
//
// Все handlers требуют аутентификации (user_id в контексте).
package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"net/http"
)

// CreateSecret — создание нового секрета.
//
// Ожидает аутентифицированный запрос (user_id в контексте).
//
// Пример запроса:
//
//	POST /api/v1/secrets
//	{
//	    "type": "credential",
//	    "title": "My Password",
//	    "data": "{\"login\":\"user\",\"password\":\"pass\"}",
//	    "metadata": "optional metadata"
//	}
//
// Ответы:
//   - 201 Created: секрет создан
//   - 400 Bad Request: невалидные данные
//   - 401 Unauthorized: не аутентифицирован
//   - 500 Internal Server Error: внутренняя ошибка
func CreateSecret(uc ports.SecretUseCase, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Извлечение user_id из контекста
		userID := getUserIDFromContext(c, log)
		if userID == "" {
			return
		}

		log.Debug("creating secret for user",
			zap.String("user_id", userID))

		// Парсинг тела запроса
		var req struct {
			Type     string `json:"type"`
			Title    string `json:"title"`
			Data     string `json:"data"`
			Metadata string `json:"metadata"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			log.Error("failed to bind request", zap.String("user_id", userID),
				zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Валидация обязательных полей
		if req.Type == "" {
			log.Warn("create secret: empty type", zap.String("user_id", userID))
			c.JSON(http.StatusBadRequest, gin.H{"error": "type is required"})
			return
		}
		if req.Title == "" {
			log.Warn("create secret: empty title", zap.String("user_id", userID))
			c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
			return
		}
		if req.Data == "" {
			log.Warn("create secret: empty data", zap.String("user_id", userID))
			c.JSON(http.StatusBadRequest, gin.H{"error": "data is required"})
			return
		}

		log.Info("creating secret",
			zap.String("user_id", userID),
			zap.String("type", req.Type),
			zap.String("title", req.Title),
		)

		// Конвертация string → SecretType
		secretType := ports.SecretType(req.Type)

		// Создаём секрет
		secret, err := uc.CreateSecret(
			c.Request.Context(),
			userID,
			secretType,
			req.Title,
			req.Data,
		)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidInput) {
				log.Warn("invalid secret input", zap.String("user_id", userID),
					zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
				return
			}

			log.Error("failed to create secret", zap.String("user_id", userID),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
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

// GetSecrets — получение списка всех секретов пользователя.
//
// Ожидает аутентифицированный запрос (user_id в контексте).
//
// Ответы:
//   - 200 OK: возвращает список секретов
//   - 401 Unauthorized: не аутентифицирован
//   - 500 Internal Server Error: внутренняя ошибка
func GetSecrets(uc ports.SecretUseCase, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлечение user_id из контекста
		userID := getUserIDFromContext(c, log)
		if userID == "" {
			return
		}

		log.Info("fetching secrets for user", zap.String("user_id", userID))

		secrets, err := uc.GetSecrets(c.Request.Context(), userID)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidInput) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
				return
			}
			log.Error("failed to get secrets", zap.String("user_id", userID),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
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

// GetSecret — получение одного секрета по ID.
//
// Ожидает аутентифицированный запрос (user_id в контексте).
//
// Параметры URL:
//   - id: идентификатор секрета
//
// Ответы:
//   - 200 OK: возвращает секрет
//   - 400 Bad Request: не указан ID секрета
//   - 401 Unauthorized: не аутентифицирован
//   - 404 Not Found: секрет не найден
//   - 500 Internal Server Error: внутренняя ошибка
func GetSecret(uc ports.SecretUseCase, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлечение user_id из контекста
		userID := getUserIDFromContext(c, log)
		if userID == "" {
			return
		}

		// Извлечение ID секрета из URL
		id := c.Param("id")
		if id == "" {
			log.Warn("get secret called without id",
				zap.String("user_id", userID))
			c.JSON(http.StatusBadRequest, gin.H{"error": "secret id is required"})
			return
		}

		log.Info("fetching secret",
			zap.String("secret_id", id),
			zap.String("user_id", userID))

		secret, err := uc.GetSecret(c.Request.Context(), id, userID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				log.Warn("secret not found",
					zap.String("secret_id", id),
					zap.String("user_id", userID),
				)
				c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
				return
			}
			log.Error("failed to get secret",
				zap.String("secret_id", id),
				zap.String("user_id", userID),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
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

// DeleteSecret — удаление секрета по ID.
//
// Ожидает аутентифицированный запрос (user_id в контексте).
//
// Параметры URL:
//   - id: идентификатор секрета
//
// Ответы:
//   - 200 OK: секрет удалён
//   - 400 Bad Request: не указан ID секрета
//   - 401 Unauthorized: не аутентифицирован
//   - 404 Not Found: секрет не найден
//   - 500 Internal Server Error: внутренняя ошибка
func DeleteSecret(uc ports.SecretUseCase, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлечение user_id из контекста
		userID := getUserIDFromContext(c, log)
		if userID == "" {
			return
		}

		id := c.Param("id")
		if id == "" {
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
			if errors.Is(err, domain.ErrNotFound) {
				log.Warn("secret not found for deletion",
					zap.String("secret_id", id),
					zap.String("user_id", userID),
				)
				c.JSON(http.StatusNotFound, gin.H{"error": "secret not found"})
				return
			}
			log.Error("failed to delete secret",
				zap.String("secret_id", id),
				zap.String("user_id", userID),
				zap.Error(err),
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
		log.Info("secret deleted successfully",
			zap.String("secret_id", id),
			zap.String("user_id", userID),
		)

		c.JSON(http.StatusOK, gin.H{"message": "secret deleted successfully"})
	}
}

// getUserIDFromContext извлекает user_id из контекста Gin.
//
// Если user_id отсутствует или невалиден, возвращает пустую строку
// и отправляет ответ 401 Unauthorized.
//
// Используется во всех защищённых handlers.
func getUserIDFromContext(c *gin.Context, log logger.Logger) string {
	userIDInterface, exists := c.Get(ginUserIDKey)
	if !exists {
		log.Warn("unauthorized request - no user_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		c.Abort()
		return ""
	}

	userID, ok := userIDInterface.(string)
	if !ok || userID == "" {
		log.Warn("invalid user_id in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		c.Abort()
		return ""
	}

	return userID
}
