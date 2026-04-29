package postgres

import (
	"context"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/ports"
	"sync"
)

type secretStorage struct {
	secrets map[string]*ports.Secret
	mu      sync.RWMutex
}

func NewSecretStorage() ports.SecretStoragePort {
	return &secretStorage{
		secrets: make(map[string]*ports.Secret),
	}
}

func (s *secretStorage) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if secret == nil {
		return nil, domain.ErrInvalidInput
	}

	copyData := make([]byte, len(secret.Data))
	copy(copyData, secret.Data)

	copySecret := *secret
	if secret.Data != "" {
		copySecret.Data = secret.Data
	}

	s.secrets[copySecret.ID] = &copySecret

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

	return result, nil

}

func (s *secretStorage) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	secret, exists := s.secrets[id]
	if !exists || secret.UserID != userID {
		return nil, domain.ErrNotFound
	}

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
		return domain.ErrNotFound
	}

	delete(s.secrets, secretID)
	return nil
}
