package domain

const (
	ProviderGoogle = "google"
	ProviderYandex = "yandex"
)

type OAuthUserInfo struct {
	Provider   string `json:"provider"`
	ProviderID string `json:"provider_id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
}

type AuthResponse struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
}
