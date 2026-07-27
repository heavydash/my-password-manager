package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bytedance/gopkg/util/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gophkeeper/server/internal/config"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/ports"
	"io"
	"net/http"
	"time"
)

type OAuthAdapter struct {
	cfg      config.OAuth
	db       *pgxpool.Pool
	authPort ports.AuthPort
}

func NewOAuthAdapter(cfg config.OAuth, db *pgxpool.Pool, authPort ports.AuthPort) ports.AuthPort {
	return &OAuthAdapter{cfg: cfg, db: db, authPort: authPort}
}

func (a *OAuthAdapter) GetOAuthURL(provider string) (string, string, error) {
	state := a.generateState()
	_, err := a.db.Exec(context.Background(),
		`INSERT INTO oauth_states (state, provider, expires_at) VALUES ($1, $2, NOW() + INTERVAL '15 minutes')`,
		state, provider)
	if err != nil {
		return "", "", err
	}
	cfg := a.getConfig(provider)
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline), state, nil
}

func (a *OAuthAdapter) HandleCallback(provider, code, state string) (string, error) {
	if !a.validateState(state, provider) {
		return "", fmt.Errorf("invalid state")
	}

	token, err := a.getConfig(provider).Exchange(context.Background(), code)
	if err != nil {
		return "", err
	}

	info, err := a.fetchUserInfo(provider, token.AccessToken)
	if err != nil {
		return "", err
	}

	userID := info.ProviderID

	oneTimeCode := a.generateState()

	_, err = a.db.Exec(context.Background(),
		`INSERT INTO oauth_states (state, provider, one_time_code,  user_id, expires_at) VALUES ($1,
            $2, $3, $4, NOW() + INTERVAL '5 minutes')`,
		oneTimeCode, provider, oneTimeCode, userID)
	if err != nil {
		logger.Error("failed to save oneTimeCode", zap.Error(err))
		return "", err
	}

	return oneTimeCode, nil
}

func (a *OAuthAdapter) AuthenticateOAuth(ctx context.Context, oneTimeCode string) (token string, err error) {
	// Находим user_id по oneTimeCode
	var userID string
	err = a.db.QueryRow(context.Background(),
		`SELECT user_id FROM oauth_states WHERE one_time_code = $1 AND expires_at > NOW()`,
		oneTimeCode).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("invalid or expired oneTimeCode")
	}
	// Удаляем использованный code
	a.db.Exec(context.Background(), "DELETE FROM oauth_states WHERE one_time_code = $1", oneTimeCode)

	// Генерируем JWT, используя существующий метод из password adapter
	token, err = a.authPort.GenerateJWT(userID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (a *OAuthAdapter) ExchangeOneTimeCode(ctx context.Context, oneTimeCode string) (*domain.AuthResponse, error) {
	token, err := a.AuthenticateOAuth(ctx, oneTimeCode)
	if err != nil {
		return nil, err
	}
	return &domain.AuthResponse{
		Token:     token,
		UserID:    "",
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}, nil
}

// getConfig - вспомогательный метод
func (a *OAuthAdapter) getConfig(provider string) *oauth2.Config {
	switch provider {
	case "google":
		return &oauth2.Config{
			ClientID:     a.cfg.Google.ClientID,
			ClientSecret: a.cfg.Google.ClientSecret,
			RedirectURL:  a.cfg.Google.RedirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
		}
	case "yandex":
		return &oauth2.Config{
			ClientID:     a.cfg.Yandex.ClientID,
			ClientSecret: a.cfg.Yandex.ClientSecret,
			RedirectURL:  a.cfg.Yandex.RedirectURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://oauth.yandex.ru/authorize",
				TokenURL: "https://oauth.yandex.ru/token",
			},
			Scopes: []string{"login:email", "login:info"},
		}
	}
	panic("unknown provider")
}

// fetchUserInfo - вспомогательный метод
func (a *OAuthAdapter) fetchUserInfo(provider, Token string) (*domain.OAuthUserInfo, error) {
	var url string

	switch provider {
	case "google":
		url = "https://www.googleapis.com/oauth2/v3/userinfo"
	case "yandex":
		url = fmt.Sprintf("https://login.yandex.ru/info?oauth_token=%s", Token)
	default:
		return nil, fmt.Errorf("unknown provider %s", provider)
	}

	req, _ := http.NewRequest("GET", url, nil)
	if provider == "google" {
		req.Header.Set("Authorization", "Bearer "+Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo status %d", resp.StatusCode)
	}

	data, _ := io.ReadAll(resp.Body)
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)

	info := domain.OAuthUserInfo{
		Provider: provider,
		Email:    getString(raw, "email"),
		Name:     getString(raw, "name"),
	}

	// Provider ID
	if sub, ok := raw["sub"].(string); ok {
		info.ProviderID = sub
	} else if id, ok := raw["id"].(string); ok {
		info.ProviderID = id
	}

	logger.Info("userinfo", zap.String("provider", provider), zap.String("raw", string(data)))

	return &info, nil
}

func (a *OAuthAdapter) saveState(state, provider string) error {
	_, err := a.db.Exec(context.Background(),
		`INSERT INTO oauth_states (state, provider, expires_at) VALUES ($1, $2, NOW() + INTERVAL '15 minutes')`,
		state, provider)
	return err
}

func (a *OAuthAdapter) validateState(state, provider string) bool {
	var exists bool
	a.db.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM oauth_states WHERE state = $1 AND provider = $2 AND expires_at > NOW())`,
		state, provider).Scan(&exists)
	a.db.Exec(context.Background(), "DELETE FROM oauth_states WHERE state = $1", state)
	return exists
}

func (a *OAuthAdapter) saveOneTimeCode(code string, userID string) error { // UUID as string or []byte
	_, err := a.db.Exec(context.Background(),
		`UPDATE oauth_states SET one_time_code = $1, user_id = $2 WHERE state = $1`,
		code, userID)
	return err
}

// Старые методы

func (a *OAuthAdapter) CreateUser(ctx context.Context, user ports.User) (string, error) {
	return a.authPort.CreateUser(ctx, user)
}

func (a *OAuthAdapter) AuthenticatePassword(ctx context.Context, email, password string) (string, string, error) {
	return a.authPort.AuthenticatePassword(ctx, email, password)
}

func (a *OAuthAdapter) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	return a.authPort.GetUserByID(ctx, id)
}

func (a *OAuthAdapter) ValidateJWT(tokenString string) (string, error) {
	return a.authPort.ValidateJWT(tokenString)
}

func (a *OAuthAdapter) GenerateJWT(userID string) (string, error) {
	return a.authPort.GenerateJWT(userID)
}

// getString - вспомогательная функция
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// generateState
func (a *OAuthAdapter) generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
