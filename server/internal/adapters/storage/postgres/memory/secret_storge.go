package memory

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"sync"
)

type secretStorage struct {
	secrets map[string]*ports.Secret
	logger  logger.Logger
	mu      sync.RWMutex
}

func NewSecretStorage(pool *pgxpool.Pool, log logger.Logger) ports.SecretStoragePort {
	if pool == nil {
		panic("database connection is required")
	}

	if log == nil {
		panic("logger is required for secretStorage")
	}

	return &secretStorage{
		secrets: make(map[string]*ports.Secret),
		logger:  log,
	}
}

func (s *secretStorage) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if secret == nil {
		s.logger.Warn("attempt to create nil secret")
		return nil, domain.ErrInvalidInput
	}

	copyData := make([]byte, len(secret.Data))
	copy(copyData, secret.Data)

	copySecret := *secret
	if secret.Data != "" {
		copySecret.Data = secret.Data
	}

	s.secrets[copySecret.ID] = &copySecret

	s.logger.Info("secret created",
		zap.String("secret_id", copySecret.ID),
		zap.String("user_id", copySecret.UserID),
		zap.String("type", copySecret.Type),
		zap.String("title", copySecret.Title),
	)

	return &copySecret, nil
}

func (s *secretStorage) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ports.Secret

	for _, secret := range s.secrets {
		if secret.UserID == userID {
			result = append(result, secret)
		}
	}

	s.logger.Info("secrets retrieved for user",
		zap.String("user_id", userID),
		zap.Int("count", len(result)),
	)

	return result, nil

}

func (s *secretStorage) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	secret, exists := s.secrets[id]
	if !exists || secret.UserID != userID {
		s.logger.Warn("secret not found",
			zap.String("secret_id", id),
			zap.String("user_id", userID),
		)
		return nil, domain.ErrNotFound
	}

	// Глубокая копия при чтении
	copyData := make([]byte, len(secret.Data))
	copy(copyData, secret.Data)

	// Копия при чтении
	copySecret := *secret
	if secret.Data != "" {
		copySecret.Data = secret.Data
	}

	return &copySecret, nil
}

func (s *secretStorage) Delete(ctx context.Context, userID, secretID string) error {
	secret, exists := s.secrets[secretID]
	if !exists || secret.UserID != userID {
		s.logger.Warn("delete failed: secret not found or access denied",
			zap.String("secret_id", secretID),
			zap.String("user_id", userID),
		)
		return domain.ErrNotFound
	}

	delete(s.secrets, secretID)

	s.logger.Info("secret deleted successfully",
		zap.String("secret_id", secretID),
		zap.String("user_id", userID),
	)

	return nil
}
