// Package auth содержит адаптеры аутентификации для GophKeeper.
//
// Поддерживаются два провайдера:
//   - Password (email + пароль с Argon2 + JWT)
//   - OAuth2 (Google, Yandex)
//
// Все адаптеры реализуют интерфейс ports.AuthPort и используются
// в handlers и usecases.
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

// OAuthAdapter — адаптер для OAuth2-провайдеров (Google, Yandex).
// Реализует ports.AuthPort.
//
// Хранит состояние OAuth-flow в таблице oauth_states (state, one_time_code).
// Для генерации JWT делегирует вызовы в authPort (passwordAdapter).
type OAuthAdapter struct {
	cfg      config.OAuth
	db       *pgxpool.Pool
	authPort ports.AuthPort
}

// NewOAuthAdapter создаёт новый OAuth-адаптер.
//
// Параметры:
//   - cfg — конфигурация OAuth-клиентов (ClientID, ClientSecret, RedirectURL)
//   - db — пул подключений к PostgreSQL для хранения состояний flow
//   - authPort — делегат для работы с пользователями и генерации JWT
//
// Возвращает объект, реализующий ports.AuthPort.
func NewOAuthAdapter(cfg config.OAuth, db *pgxpool.Pool, authPort ports.AuthPort) ports.AuthPort {
	return &OAuthAdapter{cfg: cfg, db: db, authPort: authPort}
}

// GetOAuthURL возвращает URL для начала OAuth2 flow и сгенерированный state.
//
// State сохраняется в БД с TTL 15 минут для защиты от CSRF.
// Используется на фронтенде/клиенте для редиректа пользователя на провайдера.
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

// HandleCallback обрабатывает callback от OAuth-провайдера.
//
// Проверяет state, обменивает code на token, получает данные пользователя,
// создаёт one_time_code и возвращает его клиенту для последующей авторизации.
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

// AuthenticateOAuth завершает OAuth-flow по one_time_code.
//
// Находит user_id по one_time_code, удаляет использованный код,
// генерирует JWT через authPort и возвращает токен.
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

// ExchangeOneTimeCode — высокоуровневый метод для обмена one_time_code на AuthResponse.
//
// Используется в handlers. Возвращает готовый domain.AuthResponse с токеном.
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

// getConfig возвращает *oauth2.Config для указанного провайдера.
//
// Поддерживает Google (стандартный endpoint) и Yandex (кастомный).
// Метод неэкспортированный, используется только внутри адаптера.
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

// fetchUserInfo запрашивает данные пользователя у OAuth-провайдера по access token.
//
// Для Google использует /userinfo, для Yandex — /info.
// Возвращает унифицированную структуру domain.OAuthUserInfo.
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

// saveState сохраняет state в БД (вспомогательный метод).
func (a *OAuthAdapter) saveState(state, provider string) error {
	_, err := a.db.Exec(context.Background(),
		`INSERT INTO oauth_states (state, provider, expires_at) VALUES ($1, $2, NOW() + INTERVAL '15 minutes')`,
		state, provider)
	return err
}

// validateState проверяет существование state и удаляет его после использования.
//
// Возвращает true, если state валиден и не истёк.
func (a *OAuthAdapter) validateState(state, provider string) bool {
	var exists bool
	a.db.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM oauth_states WHERE state = $1 AND provider = $2 AND expires_at > NOW())`,
		state, provider).Scan(&exists)
	a.db.Exec(context.Background(), "DELETE FROM oauth_states WHERE state = $1", state)
	return exists
}

// saveOneTimeCode сохраняет one_time_code и user_id (вспомогательный).
func (a *OAuthAdapter) saveOneTimeCode(code string, userID string) error { // UUID as string or []byte
	_, err := a.db.Exec(context.Background(),
		`UPDATE oauth_states SET one_time_code = $1, user_id = $2 WHERE state = $1`,
		code, userID)
	return err
}

// Старые методы

// CreateUser делегирует создание пользователя в passwordAdapter.
func (a *OAuthAdapter) CreateUser(ctx context.Context, user ports.User) (string, error) {
	return a.authPort.CreateUser(ctx, user)
}

// AuthenticatePassword делегирует парольную аутентификацию.
func (a *OAuthAdapter) AuthenticatePassword(ctx context.Context, email, password string) (string, string, error) {
	return a.authPort.AuthenticatePassword(ctx, email, password)
}

// GetUserByID делегирует получение пользователя по ID.
func (a *OAuthAdapter) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	return a.authPort.GetUserByID(ctx, id)
}

// HandleOAuthCallback
func (a *OAuthAdapter) HandleOAuthCallback(provider, code, state string) (string, error) {
	return a.HandleOAuthCallback(provider, code, state)
}

// ValidateJWT делегирует валидацию JWT.
func (a *OAuthAdapter) ValidateJWT(tokenString string) (string, error) {
	return a.authPort.ValidateJWT(tokenString)
}

// GenerateJWT делегирует генерацию JWT.
func (a *OAuthAdapter) GenerateJWT(userID string) (string, error) {
	return a.authPort.GenerateJWT(userID)
}

// Register делегирует регистрацию
func (a *OAuthAdapter) Register(ctx context.Context, email, password string) (string, error) {
	return a.Register(ctx, email, password)
}

// LoginPassword делегирует логгирование по паролю
func (a *OAuthAdapter) LoginPassword(ctx context.Context, email, password string) (string, string, error) {
	return a.LoginPassword(ctx, email, password)
}

// getString — вспомогательная функция для безопасного извлечения строки из map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// generateState генерирует криптографически стойкий state..
func (a *OAuthAdapter) generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
