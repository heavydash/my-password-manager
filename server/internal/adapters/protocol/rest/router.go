package rest

import (
	"github.com/gin-gonic/gin"
	"gophkeeper/server/internal/adapters/protocol/rest/handler"
	"gophkeeper/server/internal/adapters/protocol/rest/middleware"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
)

// NewRouter настраивает все routes
func NewRouter(
	authUC *domain.AuthUseCase,
	secretUC *domain.SecretUseCase,
	oauthHandler *handler.OAuthHandler,
	jwtValidator middleware.JWTValidator,
	log logger.Logger,
) *gin.Engine {

	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())

	// Публичные роуты
	r.POST("/register", handler.Register(authUC, log))
	r.POST("/login", handler.Login(authUC, log))

	// ОAuth роуты
	r.GET("/auth/:provider", oauthHandler.OAuthLogin)
	r.GET("/auth/:provider/callback", oauthHandler.OAuthCallback)

	// Защищенные роуты
	protected := r.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtValidator, log))

	protected.GET("/profile", handler.Profile(log))
	protected.POST("/secrets", handler.CreateSecret(secretUC, log))
	protected.GET("/secrets", handler.GetSecrets(secretUC, log))
	protected.GET("/secrets/:id", handler.GetSecret(secretUC, log))
	protected.DELETE("/secrets/:id", handler.DeleteSecret(secretUC, log))

	return r
}
