package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"net/http"
	"strings"
)

type OAuthHandler struct {
	useCase ports.AuthPort
	log     logger.Logger
}

func NewOAuthHandler(useCase ports.AuthPort, log logger.Logger) *OAuthHandler {
	return &OAuthHandler{
		useCase: useCase,
		log:     log,
	}
}

func (h *OAuthHandler) OAuthLogin(c *gin.Context) {
	provider := strings.ToLower(c.Param("provider"))

	switch provider {
	case domain.ProviderGoogle, domain.ProviderYandex:
	default:
		h.log.Info("unsupported oauth provider", zap.String("provider", provider))
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}

	url, _, err := h.useCase.GetOAuthURL(provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, url)
}

func (h *OAuthHandler) OAuthCallback(c *gin.Context) {
	provider := strings.ToLower(c.Param("provider"))
	code := c.Query("code")
	state := c.Query("state")

	oneTimeCode, err := h.useCase.HandleOAuthCallback(provider, code, state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Success! Copy this code to the app:",
		"code":    oneTimeCode,
	})
}
