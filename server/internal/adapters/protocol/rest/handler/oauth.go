package handler

import (
	"github.com/gin-gonic/gin"
	"gophkeeper/server/internal/domain"
	"net/http"
	"strings"
)

type OAuthHandler struct {
	useCase *domain.AuthUseCase
}

func NewOAuthHandler(useCase *domain.AuthUseCase) *OAuthHandler {
	return &OAuthHandler{useCase: useCase}
}

func (h *OAuthHandler) OAuthLogin(c *gin.Context) {
	provider := strings.ToLower(c.Param("provider"))
	if provider != "google" && provider != "yandex" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown provider"})
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
