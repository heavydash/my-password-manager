package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gophkeeper/server/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuthLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := TestLogger{}

	tests := []struct {
		name         string
		provider     string
		setupMock    func(*MockAuthUseCase)
		wantStatus   int
		wantLocation string
		wantBody     string
	}{{
		name:     "google_success",
		provider: "google",
		setupMock: func(m *MockAuthUseCase) {
			m.On("GetOAuthURL", "google").Return("https://accounts.google.com/o/oauth2/auth?...", "state123", nil)
		},
		wantStatus:   http.StatusFound,
		wantLocation: "https://accounts.google.com",
	},
		{
			name:     "yandex_success",
			provider: "yandex",
			setupMock: func(m *MockAuthUseCase) {
				m.On("GetOAuthURL", "yandex").Return("https://oauth.yandex.ru/authorize?...", "state123", nil)
			},
			wantStatus:   http.StatusFound,
			wantLocation: "https://oauth.yandex.ru",
		},
		{
			name:       "unsupported_provider",
			provider:   "facebook",
			setupMock:  func(*MockAuthUseCase) {},
			wantStatus: http.StatusBadRequest,
			wantBody:   "unsupported provider",
		},
		{
			name:     "usecase_error",
			provider: "google",
			setupMock: func(m *MockAuthUseCase) {
				m.On("GetOAuthURL", "google").Return("", "", domain.ErrOAuthUnavailable)
			},
			wantStatus: http.StatusInternalServerError,
			wantBody:   "oauth unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := NewMockAuthUseCase()
			tt.setupMock(mockUC)

			h := NewOAuthHandler(mockUC, log)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			c.Request, _ = http.NewRequest(http.MethodGet, "/auth/"+tt.provider, nil)
			c.Params = gin.Params{{Key: "provider", Value: tt.provider}}

			h.OAuthLogin(c)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantLocation != "" {
				assert.Contains(t, w.Header().Get("Location"), tt.wantLocation)
			}
			if tt.wantBody != "" {
				assert.Contains(t, w.Body.String(), tt.wantBody)
			}
			mockUC.AssertExpectations(t)
		})
	}
}

func TestOAuthCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := TestLogger{}

	tests := []struct {
		name             string
		provider         string
		code             string
		state            string
		setupMock        func(*MockAuthUseCase)
		wantStatus       int
		wantBodyContains string
	}{
		{
			name:     "success",
			provider: "google",
			code:     "4/0A...valid",
			state:    "state-xyz",
			setupMock: func(m *MockAuthUseCase) {
				m.On("HandleOAuthCallback", "google", "4/0A...valid", "state-xyz").
					Return("one-time-code-abc123", nil)
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "one-time-code-abc123",
		},
		{
			name:     "yandex_success",
			provider: "yandex",
			code:     "yandex-code-999",
			state:    "state-ya",
			setupMock: func(m *MockAuthUseCase) {
				m.On("HandleOAuthCallback", "yandex", "yandex-code-999", "state-ya").
					Return("one-time-999", nil)
			},
			wantStatus:       http.StatusOK,
			wantBodyContains: "Success! Copy this code",
		},
		{
			name:     "error_from_usecase",
			provider: "google",
			code:     "bad-code",
			state:    "state-xyz",
			setupMock: func(m *MockAuthUseCase) {
				m.On("HandleOAuthCallback", "google", "bad-code", "state-xyz").
					Return("", domain.ErrInvalidOAuthState)
			},
			wantStatus:       http.StatusBadRequest,
			wantBodyContains: "invalid state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUC := NewMockAuthUseCase()
			tt.setupMock(mockUC)

			h := NewOAuthHandler(mockUC, log)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{{Key: "provider", Value: tt.provider}}
			c.Request, _ = http.NewRequest(http.MethodGet,
				"/callback?code="+tt.code+"&state="+tt.state, nil)

			h.OAuthCallback(c)

			assert.Equal(t, tt.wantStatus, w.Code)
			if tt.wantBodyContains != "" {
				assert.Contains(t, w.Body.String(), tt.wantBodyContains)
			}
			mockUC.AssertExpectations(t)
		})
	}
}
