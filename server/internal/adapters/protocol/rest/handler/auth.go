package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"net/http"
)

// Register — фабрика handler'а регистрации
func Register(uc *domain.AuthUseCase, log logger.Logger) gin.HandlerFunc {
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
				c.JSON(http.StatusConflict, gin.H{"error:": "user with this email already exists"})
				return
			}
			log.Error("registration failed", zap.String("email", req.Email),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Успешный ответ
		log.Info("user registered successfully",
			zap.String("user_id", userID),
			zap.String("email", req.Email),
		)

		c.JSON(http.StatusCreated, gin.H{
			"message": "user registered successfully",
			"userID":  userID,
		})
	}
}

// Login — фабрика handler'а логина
func Login(uc *domain.AuthUseCase, log logger.Logger) gin.HandlerFunc {
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
		log.Info("user logged in successfully",
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

// Profile — защищённый эндпоинт, возвращает информацию о текущем пользователе
func Profile(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем userID, который положил middleware
		userID, exist := c.Get("userID")
		if !exist {
			log.Warn("profile request without user_id in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "userID not found in context"})
			return
		}

		uid, _ := userID.(string)

		log.Info("profile accessed successfully", zap.String("user_id", uid))

		c.JSON(http.StatusOK, gin.H{
			"message": "protected route accessed successfully",
			"userID":  userID,
		})
	}
}
