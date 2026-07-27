// Package rest настраивает HTTP-роуты для GophKeeper.
//
// Содержит:
//   - Публичные роуты: регистрация, логин
//   - OAuth роуты: получение URL и callback
//   - Защищённые роуты: profile, CRUD секретов (требуют JWT)
package rest

import (
	"github.com/gin-gonic/gin"
	"gophkeeper/server/internal/adapters/protocol/rest/handler"
	"gophkeeper/server/internal/adapters/protocol/rest/middleware"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
)

// NewRouter создаёт и настраивает маршрутизатор Gin.
//
// Алгоритм:
//  1. Создаёт экземпляр gin.Engine
//  2. Добавляет глобальные middleware (Recovery)
//  3. Регистрирует публичные эндпоинты (регистрация, логин)
//  4. Регистрирует OAuth эндпоинты
//  5. Создаёт группу защищённых эндпоинтов с AuthMiddleware
//  6. Регистрирует защищённые эндпоинты (profile, secrets CRUD)
//
// Параметры:
//   - authUC: бизнес-логика аутентификации
//   - secretUC: бизнес-логика управления секретами
//   - oauthHandler: обработчик OAuth-запросов
//   - jwtValidator: функция валидации JWT-токена
//   - log: логгер для записи событий
//
// Возвращает настроенный gin.Engine.
func NewRouter(
	authUC *domain.AuthUseCase,
	secretUC *domain.SecretUseCase,
	oauthHandler *handler.OAuthHandler,
	jwtValidator middleware.JWTValidator,
	log logger.Logger,
) *gin.Engine {

	// Создание роутера с базовыми настройками
	r := gin.New()

	// Глобальные middleware
	r.Use(gin.Recovery())

	// Публичные эндпоинты (не требуют аутентификации)
	r.POST("/register", handler.Register(authUC, log))
	r.POST("/login", handler.Login(authUC, log))

	// OAuth эндпоинты
	r.GET("/auth/:provider", oauthHandler.OAuthLogin)
	r.GET("/auth/:provider/callback", oauthHandler.OAuthCallback)

	// Группа защищённых эндпоинтов (требуют JWT)
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtValidator, log))

	// Защищённые эндпоинты
	protected.GET("/profile", handler.Profile(log))
	protected.POST("/secrets", handler.CreateSecret(secretUC, log))
	protected.GET("/secrets", handler.GetSecrets(secretUC, log))
	protected.GET("/secrets/:id", handler.GetSecret(secretUC, log))
	protected.DELETE("/secrets/:id", handler.DeleteSecret(secretUC, log))

	return r
}
