package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
)

type secretRepository struct {
	db     *sql.DB
	logger logger.Logger
}

func NewSecretRepository(db *sql.DB, log logger.Logger) ports.SecretStoragePort {
	if log == nil {
		panic("logger is required")
	}
	return &secretRepository{
		db:     db,
		logger: log,
	}
}

func (r *secretRepository) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {
	const query = `
		INSERT INTO secrets (id, user_id, type, title, data, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at;
	`

	r.logger.Info("saving secret to database",
		zap.String("secret_id", secret.ID),
		zap.String("user_id", secret.UserID),
		zap.String("title", secret.Title),
	)

	err := r.db.QueryRowContext(ctx, query,
		secret.ID,
		secret.UserID,
		secret.Type,
		secret.Title,
		secret.Data,
		secret.Metadata,
		secret.CreatedAt,
		secret.UpdatedAt,
	).Scan(&secret.CreatedAt, &secret.UpdatedAt)

	if err != nil {
		r.logger.Error("failed to save secret", zap.Error(err))
		return nil, fmt.Errorf("create secret: %w", err)
	}

	r.logger.Info("secret saved successfully", zap.String("secret_id", secret.ID))
	return secret, nil
}

func (r *secretRepository) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {
	const query = `
		SELECT id, user_id, type, title, data, metadata, created_at, updated_at
		FROM secrets 
		WHERE user_id = $1 
		ORDER BY created_at DESC;
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*ports.Secret
	for rows.Next() {
		var s ports.Secret
		if err := rows.Scan(&s.ID, &s.UserID, &s.Type, &s.Title, &s.Data, &s.Metadata, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		secrets = append(secrets, &s)
	}

	return secrets, nil
}

func (r *secretRepository) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	const query = `
		SELECT id, user_id, type, title, data, metadata, created_at, updated_at
		FROM secrets 
		WHERE id = $1 AND user_id = $2;
	`

	var s ports.Secret
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(
		&s.ID, &s.UserID, &s.Type, &s.Title, &s.Data, &s.Metadata, &s.CreatedAt, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *secretRepository) Delete(ctx context.Context, userID, secretID string) error {
	const query = `DELETE FROM secrets WHERE id = $1 AND user_id = $2`

	result, err := r.db.ExecContext(ctx, query, secretID, userID)
	if err != nil {
		return err
	}

	if rows, _ := result.RowsAffected(); rows == 0 {
		return domain.ErrNotFound
	}

	return nil
}
