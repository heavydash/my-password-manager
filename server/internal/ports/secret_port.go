// Package ports содержит контракты (интерфейсы) между слоями приложения.
//
// Все зависимости идут от domain/usecases → ports → adapters.
package ports

import (
	"context"
	"time"
)

// SecretType — тип секрета (пароль, заметка, карта, SSH-ключ и т.д.).
type SecretType string

// Secret — структура секрета, используется для передачи данных между слоями.
//
// Содержит:
//   - ID: уникальный идентификатор секрета
//   - UserID: идентификатор владельца
//   - Type: тип секрета (password, note, card, ssh_key, custom)
//   - Title: заголовок/название секрета
//   - Data: зашифрованные данные секрета
//   - Metadata: дополнительные метаданные (JSON)
//   - CreatedAt: время создания
//   - UpdatedAt: время последнего обновления
type Secret struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Data      string    `json:"data"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SecretPort — порт для бизнес-логики секретов.
//
// Используется SecretUseCase для работы с хранилищем.
// Реализуется адаптерами: memory.SecretStorage, postgres.secretRepository.
type SecretPort interface {
	// Create сохраняет новый секрет в хранилище.
	Create(ctx context.Context, secret *Secret) (*Secret, error)

	// GetByUserID возвращает все секреты пользователя.
	GetByUserID(ctx context.Context, userID string) ([]*Secret, error)

	// GetSecret возвращает один секрет по ID и userID.
	GetSecret(ctx context.Context, id, userID string) (*Secret, error)

	// Delete удаляет секрет пользователя.
	Delete(ctx context.Context, userID, secretID string) error
}

// SecretStoragePort — порт для работы с хранилищем.
//
// Может совпадать с SecretPort, но разделён для ясности.
// Используется в адаптерах для абстракции конкретного хранилища.
type SecretStoragePort interface {
	// Create сохраняет новый секрет в хранилище.
	Create(ctx context.Context, secret *Secret) (*Secret, error)

	// GetByUserID возвращает все секреты пользователя.
	GetByUserID(ctx context.Context, userID string) ([]*Secret, error)

	// GetSecret возвращает один секрет по ID и userID.
	GetSecret(ctx context.Context, id, userID string) (*Secret, error)

	// Delete удаляет секрет пользователя.
	Delete(ctx context.Context, userID, secretID string) error
}

// SecretUseCase — интерфейс бизнес-логики управления секретами.
//
// Реализуется domain.SecretUseCase.
// Используется HTTP/gRPC handlers для выполнения операций с секретами.
type SecretUseCase interface {
	// CreateSecret создаёт новый секрет.
	CreateSecret(ctx context.Context, userID string, typ SecretType, title, data string) (*Secret, error)

	// GetSecrets возвращает все секреты пользователя.
	GetSecrets(ctx context.Context, userID string) ([]*Secret, error)

	// GetSecret возвращает один секрет по ID.
	GetSecret(ctx context.Context, id, userID string) (*Secret, error)

	// DeleteSecret удаляет секрет.
	DeleteSecret(ctx context.Context, userID, id string) error
}
