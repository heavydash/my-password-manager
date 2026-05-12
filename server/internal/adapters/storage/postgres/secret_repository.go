package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
)

type secretRepository struct {
	pool   *pgxpool.Pool
	logger logger.Logger
}

func NewSecretRepository(pool *pgxpool.Pool, log logger.Logger) ports.SecretStoragePort {
	if log == nil {
		panic("logger is required")
	}
	return &secretRepository{
		pool:   pool,
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

	_, err := r.pool.Exec(ctx, query,
		secret.ID,
		secret.UserID,
		secret.Type,
		secret.Title,
		secret.Data,
		secret.Metadata,
		secret.CreatedAt,
		secret.UpdatedAt,
	)

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

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []*ports.Secret
	for rows.Next() {
		var s ports.Secret
		if err := rows.Scan(&s.ID,
			&s.UserID,
			&s.Type,
			&s.Title,
			&s.Data,
			&s.Metadata,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		secrets = append(secrets, &s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
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
	err := r.pool.QueryRow(ctx, query, id, userID).Scan(
		&s.ID,
		&s.UserID,
		&s.Type,
		&s.Title,
		&s.Data,
		&s.Metadata,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func (r *secretRepository) Delete(ctx context.Context, userID, secretID string) error {
	const query = `DELETE FROM secrets WHERE id = $1 AND user_id = $2`

	result, err := r.pool.Exec(ctx, query, secretID, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}
