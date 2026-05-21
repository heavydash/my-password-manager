// Package handler содержит HTTP-handlers (Gin) для GophKeeper.
//
// Все handlers являются фабриками (возвращают gin.HandlerFunc), чтобы можно было
// легко внедрять зависимости (UseCase, Logger).
//
// Содержит:
//   - Register: регистрация пользователя
//   - Login: вход по email/паролю
//   - Profile: защищённый эндпоинт для проверки аутентификации
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

// Константы для ключей контекста Gin
const (
	ginUserIDKey = "userID"
)

// Register возвращает handler регистрации пользователя.
//
// Принимает email и password, валидирует их через gin binding,
// вызывает AuthUseCase.Register и возвращает userID.
//
// Пример запроса:
//
//	POST /api/v1/auth/register
//	{
//	    "email": "user@example.com",
//	    "password": "securepassword123"
//	}
//
// Ответы:
//   - 201 Created: успешная регистрация
//   - 400 Bad Request: невалидные данные
//   - 409 Conflict: пользователь уже существует
//   - 500 Internal Server Error: внутренняя ошибка
func Register(uc ports.AuthPort, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=8"`
		}

		// Валидация входных данных
		if err := c.ShouldBind(&req); err != nil {
			log.Warn("registration request binding failed", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Info("user registration attempt", zap.String("email", req.Email))

		// Вызываем UseCase
		userID, err := uc.Register(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, domain.ErrUserAlreadyExists) {
				log.Warn("registration failed: user already exists",
					zap.String("email", req.Email))
				c.JSON(http.StatusConflict, gin.H{"error": "user with this email already exists"})
				return
			}
			log.Error("registration failed", zap.String("email", req.Email),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Успешный ответ
		log.Debug("user registered successfully",
			zap.String("user_id", userID),
			zap.String("email", req.Email),
		)

		c.JSON(http.StatusCreated, gin.H{
			"message": "user registered successfully",
			"userID":  userID,
		})
	}
}

// Login возвращает handler авторизации по email + password.
//
// При успешном логине возвращает JWT-токен.
//
// Пример запроса:
//
//	POST /api/v1/auth/login
//	{
//	    "email": "user@example.com",
//	    "password": "securepassword123"
//	}
//
// Ответы:
//   - 200 OK: успешный вход, возвращает token и user_id
//   - 400 Bad Request: невалидные данные
//   - 401 Unauthorized: неверные учётные данные
//   - 500 Internal Server Error: внутренняя ошибка
func Login(uc ports.AuthPort, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBind(&req); err != nil {
			log.Warn("login request binding failed", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Info("login attempt", zap.String("email", req.Email))

		// Вызываем UseCase
		token, userID, err := uc.LoginPassword(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidCredentials) {
				log.Warn("login failed: invalid credentials",
					zap.String("email", req.Email))
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}

			log.Error("login error", zap.String("email", req.Email),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Успешный логин
		log.Debug("user logged in successfully",
			zap.String("user_id", userID),
			zap.String("email", req.Email),
		)

		c.JSON(http.StatusOK, gin.H{
			"message": "login successfully",
			"token":   token,
			"userID":  userID,
		})
	}
}

// Profile — защищённый эндпоинт для проверки аутентификации.
//
// Ожидает, что middleware ранее положил `userID` в контекст Gin.
// Используется для проверки работоспособности JWT-аутентификации.
//
// Ответы:
//   - 200 OK: пользователь аутентифицирован
//   - 401 Unauthorized: user_id отсутствует в контексте
func Profile(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем userID, который положил middleware
		userIDInterface, exist := c.Get(ginUserIDKey)
		if !exist {
			log.Warn("profile request without user_id in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "userID not found in context"})
			return
		}

		userID, ok := userIDInterface.(string)
		if !ok || userID == "" {
			log.Warn("invalid user id in context", zap.Any("got", userIDInterface))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "userID not found in context",
			})
			return
		}

		log.Info("profile accessed successfully", zap.String("user_id", userID))

		c.JSON(http.StatusOK, gin.H{
			"message": "protected route accessed successfully",
			"userID":  userID,
		})
	}
}
