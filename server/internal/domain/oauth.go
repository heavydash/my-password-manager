// Package domain содержит бизнес-логику GophKeeper.
//
// Содержит:
//   - OAuthUserInfo: данные пользователя от OAuth-провайдера
//   - AuthResponse: ответ аутентификации с JWT-токеном
package domain

// Константы провайдеров аутентификации
const (
	ProviderGoogle = "google"
	ProviderYandex = "yandex"
)

// OAuthUserInfo содержит информацию о пользователе от OAuth-провайдера.
type OAuthUserInfo struct {
	Provider   string `json:"provider"`    // название провайдера (google, yandex)
	ProviderID string `json:"provider_id"` // ID пользователя у провайдера
	Email      string `json:"email"`       // email пользователя
	Name       string `json:"name"`        // имя пользователя
}

// AuthResponse содержит результат успешной аутентификации.
type AuthResponse struct {
	Token     string `json:"token"`      // JWT-токен
	UserID    string `json:"user_id"`    // идентификатор пользователя
	ExpiresAt int64  `json:"expires_at"` // timestamp истечения токена
}
