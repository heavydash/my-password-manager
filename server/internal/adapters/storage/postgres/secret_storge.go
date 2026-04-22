package postgres

import (
	"context"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/ports"
)

type secretStorage struct {
	secrets map[string]*ports.Secret
}

func NewSecretStorage() ports.SecretStoragePort {
	return &secretStorage{
		secrets: make(map[string]*ports.Secret),
	}
}

func (s *secretStorage) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	if secret == nil {
		return nil, domain.ErrInvalidInput
	}

	s.secrets[secret.ID] = secret
	return secret, nil
}

func (s *secretStorage) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {
	var result []*ports.Secret

	for _, secret := range s.secrets {
		if secret.UserID == userID {
			result = append(result, secret)
		}
	}

	return result, nil

}

func (s *secretStorage) Delete(ctx context.Context, userID, secretID string) error {
	secret, exists := s.secrets[secretID]
	if !exists || secret.UserID != userID {
		return domain.ErrNotFound
	}

	delete(s.secrets, secretID)
	return nil
}
