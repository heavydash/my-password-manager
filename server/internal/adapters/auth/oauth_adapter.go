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
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"gophkeeper/server/internal/config"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"io"
	"net/http"
	"time"
)

// OAuthAdapter — адаптер для OAuth2-провайдеров (Google, Yandex).
// Реализует ports.AuthPort.
//
// Особенности:
//   - Хранит состояние OAuth-flow в таблице oauth_states
//   - Для генерации JWT делегирует вызовы в passwordAdapter
//   - Все операции с БД используют переданный context.Context
//
// Используется в:
//   - REST-хендлерах для OAuth-авторизации
//   - gRPC-сервере для callback-обработки
type OAuthAdapter struct {
	cfg      config.OAuthConfig
	db       *pgxpool.Pool
	authPort ports.AuthPort
	log      logger.Logger
}

// NewOAuthAdapter создаёт новый OAuth-адаптер.
//
// Параметры:
//   - cfg: конфигурация OAuth-клиентов (ClientID, ClientSecret, RedirectURL)
//   - db: пул подключений к PostgreSQL для хранения состояний
//   - authPort: делегат для работы с пользователями и генерации JWT
//   - logger: логгер для записи событий (на английском)
//
// Возвращает объект, реализующий ports.AuthPort.
func NewOAuthAdapter(cfg config.OAuthConfig, db *pgxpool.Pool, authPort ports.AuthPort, log logger.Logger) ports.AuthPort {
	return &OAuthAdapter{
		cfg:      cfg,
		db:       db,
		authPort: authPort,
		log:      log,
	}
}

// GetOAuthURL возвращает URL для начала OAuth2 flow и сгенерированный state.
//
// Выполняет:
//  1. Генерацию криптостойкого state
//  2. Сохранение state в БД с TTL из конфига
//
// State защищает от CSRF-атак.
func (a *OAuthAdapter) GetOAuthURL(provider string) (string, string, error) {
	// Генерация случайного state
	state := a.generateState()

	// Сохранение state с TTL из конфига
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := a.db.Exec(ctx,
		`INSERT INTO oauth_states (state, provider, expires_at) VALUES ($1, $2, NOW
() + INTERVAL '15 minutes')`,
		state, provider, a.cfg.StateTTL)
	if err != nil {
		a.log.Error("Failed to save OAuth state", zap.Error(err))
		return "", "", err
	}

	cfg := a.getConfig(provider)
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline), state, nil
}

// HandleCallback обрабатывает callback от OAuth-провайдера.
//
// Алгоритм:
//  1. Проверка state
//  2. Обмен code на access token
//  3. Получение данных пользователя
//  4. Генерация one_time_code
//  5. Возврат кода клиенту
func (a *OAuthAdapter) HandleCallback(provider, code, state string) (string, error) {
	// Проверка state с использованием контекста
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if !a.validateState(ctx, state, provider) {
		a.log.Warn("Invalid OAuth state", zap.String("provider", provider))
		return "", fmt.Errorf("invalid state")
	}

	// Обмен code на token
	token, err := a.getConfig(provider).Exchange(context.Background(), code)
	if err != nil {
		a.log.Error("Failed to exchange OAuth code", zap.Error(err))
		return "", err
	}

	// Получение информации о пользователе
	info, err := a.fetchUserInfo(provider, token.AccessToken)
	if err != nil {
		a.log.Error("Failed to fetch user info", zap.Error(err))
		return "", err
	}

	userID := info.ProviderID
	oneTimeCode := a.generateState()

	// Сохранение one_time_code
	_, err = a.db.Exec(ctx,
		`INSERT INTO oauth_states (state, provider, one_time_code,  user_id, expires
    _at)
VALUES ($1, $2, $3, $4, NOW() + INTERVAL '5 minutes')`,
		oneTimeCode, provider, oneTimeCode, userID, a.cfg.OneTimeCodeTTL)
	if err != nil {
		a.log.Error("failed to save oneTimeCode", zap.Error(err))
		return "", err
	}

	a.log.Info("OAuth callback processed successfully",
		zap.String("provider", provider),
		zap.String("user_id", userID))

	return oneTimeCode, nil
}

// AuthenticateOAuth завершает OAuth-flow по one_time_code.
//
// Выполняет:
//  1. Поиск user_id по one_time_code
//  2. Удаление использованного кода
//  3. Генерацию JWT через authPort
//
// Все операции выполняются в одной транзакции.
func (a *OAuthAdapter) AuthenticateOAuth(ctx context.Context, code string) (token string, err error) {
	// Начало транзакции
	tx, err := a.db.Begin(ctx)
	if err != nil {
		a.log.Error("Failed to begin transaction", zap.Error(err))
		return "", err
	}
	defer tx.Rollback(ctx)

	// Находим user_id по oneTimeCode
	var userID string
	err = tx.QueryRow(ctx,
		`SELECT user_id FROM oauth_states
               WHERE one_time_code = $1 AND expires_at > NOW()`,
		code).Scan(&userID)
	if err != nil {
		a.log.Warn("Invalid or expired oneTimeCode", zap.Error(err))
		return "", fmt.Errorf("invalid or expired oneTimeCode")
	}

	// Удаляем использованный code
	_, err = tx.Exec(ctx, "DELETE FROM oauth_states WHERE one_time_code = $1", code)
	if err != nil {
		a.log.Error("Failed to delete oneTimeCode", zap.Error(err))
		return "", err
	}

	// Фиксация транзакции
	if err := tx.Commit(ctx); err != nil {
		a.log.Error("Failed to commit transaction", zap.Error(err))
		return "", err
	}

	// Генерируем JWT, используя существующий метод из password adapter
	token, err = a.authPort.GenerateJWT(userID)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ExchangeOneTimeCode — высокоуровневый метод для обмена one_time_code на AuthResponse.
//
// Используется в REST-хендлерах.
func (a *OAuthAdapter) ExchangeOneTimeCode(ctx context.Context, oneTimeCode string) (*domain.AuthResponse, error) {
	token, err := a.AuthenticateOAuth(ctx, oneTimeCode)
	if err != nil {
		a.log.Error("failed to exchange one_time_code", zap.Error(err))
		return nil, err
	}

	expiresAt := time.Now().Add(a.cfg.JWTExpiresAt)

	return &domain.AuthResponse{
		Token:     token,
		UserID:    "",
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

// Делегирующие методы (реализация ports.AuthPort)

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

// HandleOAuthCallback обрабатывает OAuth callback (адаптер для Handler).
func (a *OAuthAdapter) HandleOAuthCallback(provider, code, state string) (string, error) {
	return a.HandleCallback(provider, code, state)
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
	return a.authPort.Register(ctx, email, password)
}

// LoginPassword делегирует вход по паролю
func (a *OAuthAdapter) LoginPassword(ctx context.Context, email, password string) (string, string, error) {
	return a.authPort.LoginPassword(ctx, email, password)
}

// Вспомогательные методы (неэкспортируемые)

// getConfig возвращает oauth2.Config для указанного провайдера.
//
// Поддерживает:
//   - google — стандартные endpoint'ы Google
//   - yandex — кастомные URL из конфига
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
	default:
		a.log.Fatal("Unknown OAuth provider", zap.String("provider", provider))
		return nil
	}
}

// fetchUserInfo запрашивает данные пользователя у OAuth-провайдера.
//
// Для Google использует /userinfo, для Yandex — /info.
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

	// Создание запроса с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if provider == "google" {
		req.Header.Set("Authorization", "Bearer "+Token)
	}

	// Выполнение запроса
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo status %d", resp.StatusCode)
	}

	// Чтение и парсинг ответа
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Формирование результата
	info := domain.OAuthUserInfo{
		Provider: provider,
		Email:    getString(raw, "email"),
		Name:     getString(raw, "name"),
	}

	// Определение ProviderID (sub для Google, id для Yandex)
	if sub, ok := raw["sub"].(string); ok {
		info.ProviderID = sub
	} else if id, ok := raw["id"].(string); ok {
		info.ProviderID = id
	}

	a.log.Debug("Fetched user info",
		zap.String("provider", provider),
		zap.String("email", info.Email))

	return &info, nil
}

// saveState сохраняет state в БД (вспомогательный метод).
func (a *OAuthAdapter) saveState(state, provider string) error {
	_, err := a.db.Exec(context.Background(),
		`INSERT INTO oauth_states (state, provider, expires_at) VALUES ($1, $2,
        NOW() + INTERVAL '15 minutes')`,
		state, provider)
	return err
}

// validateState проверяет существование state и удаляет его после использования.
//
// Возвращает true, если state валиден и не истёк.
// Выполняет DELETE в любом случае (однократное использование).
func (a *OAuthAdapter) validateState(ctx context.Context, state, provider string) bool {
	// Проверка существования
	var exists bool
	err := a.db.QueryRow(ctx,
		`SELECT EXISTS(
		SELECT 1 FROM oauth_states 
         WHERE state = $1 AND provider = $2 AND expires_at > NOW())
         `,
		state, provider).Scan(&exists)
	if err != nil {
		a.log.Error("Failed to validate state", zap.Error(err))
		return false
	}

	// Удаление state (однократное использование)
	_, err = a.db.Exec(ctx, "DELETE FROM oauth_states WHERE state = $1", state)
	if err != nil {
		a.log.Error("Failed to delete state", zap.Error(err))
	}
	return exists
}

// saveOneTimeCode сохраняет one_time_code и user_id (вспомогательный).
func (a *OAuthAdapter) saveOneTimeCode(code string, userID string) error { // UUID as string or []byte
	_, err := a.db.Exec(context.Background(),
		`UPDATE oauth_states SET one_time_code = $1, user_id = $2 WHERE state = $1`,
		code, userID)
	return err
}

// getString безопасно извлекает строку из map[string]interface{}.
//
// Возвращает пустую строку, если ключа нет или значение не строка.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// generateState генерирует криптографически стойкий state.
//
// Использует crypto/rand для генерации 32 байт,
// которые кодируются в hex-строку (64 символа).
func (a *OAuthAdapter) generateState() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		a.log.Fatal("Failed to generate random state", zap.Error(err))
	}
	return hex.EncodeToString(b)
}
