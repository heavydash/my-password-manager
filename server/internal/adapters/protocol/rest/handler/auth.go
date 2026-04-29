package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gophkeeper/server/internal/domain"
	"log"
	"net/http"
)

// Register — фабрика handler'а регистрации
func Register(uc *domain.AuthUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=8"`
		}

		// Валидация входных данных
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("[REGISTER] Attempt for email: %s", req.Email)

		// Вызываем UseCase
		userID, err := uc.Register(c.Request.Context(), req.Email, req.Password)
		if err != nil {

			log.Printf("Registration failed: %v", err)

			if errors.Is(err, domain.ErrUserAlreadyExists) {
				c.JSON(http.StatusConflict, gin.H{"error:": "user with this email already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		log.Printf("[REGISTER] Success for email: %s", req.Email)
		// Успешный ответ
		c.JSON(http.StatusCreated, gin.H{
			"message": "user registered successfully",
			"userID":  userID,
		})
	}
}

// Login — фабрика handler'а логина
func Login(uc *domain.AuthUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Вызываем UseCase
		token, userID, err := uc.LoginPassword(c.Request.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidCredentials) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		// Успешный логин
		c.JSON(http.StatusOK, gin.H{
			"message": "login successfully",
			"token":   token,
			"userID":  userID,
		})
	}
}

// Profile — — защищённый эндпоинт, возвращает информацию о текущем пользователе
func Profile() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Извлекаем userID, который положил middleware
		userID, exist := c.Get("userID")
		if !exist {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "userID not found in context"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "protected route accessed successfully",
			"userID":  userID,
		})
	}
}
