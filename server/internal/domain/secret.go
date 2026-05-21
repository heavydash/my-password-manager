// Package domain содержит чистую бизнес-логику приложения.
//
// Здесь находятся сущности, usecases и ошибки домена.
// Не зависит от БД, HTTP, фреймворков.
package domain

import (
	"fmt"
	"gophkeeper/server/internal/ports"
	"time"
)

// SecretType — тип секрета.
type SecretType = ports.SecretType

// Константы типов секретов
const (
	SecretTypePassword SecretType = "password"
	SecretTypeNote     SecretType = "note"
	SecretTypeCard     SecretType = "card"
	SecretTypeSSHKey   SecretType = "ssh_key"
	SecretTypeCustom   SecretType = "custom"
)

// TokenValidator — интерфейс для валидации токена. Используется в middleware
type TokenValidator interface {
	ValidateToken(token string) (string, error)
}

// JWTValidatorAdapter — адаптер для приведения, позволяющий легко подменять валидатор в тестах.
type JWTValidatorAdapter struct {
	ValidateFunc func(token string) (string, error)
}

// ValidateToken делегирует вызов функции-валидатора.
func (a JWTValidatorAdapter) ValidateToken(token string) (string, error) {
	return a.ValidateFunc(token)
}

// NewSecret создаёт новую доменную сущность секрета.
//
// Выполняет валидацию:
//   - userID, title, encryptedData не пустые
//   - secretType — один из разрешённых
//
// Возвращает готовый *Secret с заполненными CreatedAt/UpdatedAt и ID.
func NewSecret(userID string, secretType SecretType, title string, encryptedData string) (*ports.Secret, error) {
	if userID == "" {
		return nil, ErrInvalidInput
	}

	if title == "" {
		return nil, ErrInvalidInput
	}

	if len(encryptedData) == 0 {
		return nil, ErrInvalidInput
	}

	// Валидация типа
	switch secretType {
	case SecretTypePassword, SecretTypeNote, SecretTypeCard, SecretTypeSSHKey, SecretTypeCustom:
		// разрешённые типы
	default:
		return nil, fmt.Errorf("unknown secret type: %s", secretType)
	}

	now := time.Now()

	return &ports.Secret{
		ID:        generateID(),
		UserID:    userID,
		Type:      string(secretType),
		Title:     title,
		Data:      encryptedData,
		Metadata:  "",
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// generateID генерирует простой ID на основе текущего времени.
//
// Используется только внутри пакета. В production рекомендуется заменить на UUID.
func generateID() string {
	return time.Now().UTC().Format("20060102150405")
}
