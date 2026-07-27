// Package postgres реализует PostgreSQL хранилище секретов.
//
// Содержит:
//   - secretRepository: реализация ports.SecretStoragePort
//   - Поддержка контекста и graceful shutdown
package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"time"
)

// secretRepository реализует хранилище секретов в PostgreSQL.
//
// Реализует ports.SecretStoragePort.
type secretRepository struct {
	pool   ports.DBPool
	logger logger.Logger
}

// NewSecretRepository создаёт новое PostgreSQL хранилище секретов.
//
// Параметры:
//   - pool: пул соединений с PostgreSQL (реализует ports.DBPool)
//   - log: логгер для записи событий
//
// Паникует, если pool или log == nil.
//
// Возвращает реализацию ports.SecretStoragePort.
func NewSecretRepository(pool ports.DBPool, log logger.Logger) ports.SecretStoragePort {
	if log == nil {
		panic("logger is required")
	}
	if pool == nil {
		panic("database pool is required")
	}
	return &secretRepository{
		pool:   pool,
		logger: log,
	}
}

// Create сохраняет новый секрет в БД.
//
// Алгоритм:
//  1. Проверяет secret на nil
//  2. Валидирует обязательные поля
//  3. Генерирует ID и временные метки при необходимости
//  4. Выполняет INSERT
//
// Возвращает созданный секрет или ошибку.
func (r *secretRepository) Create(ctx context.Context, secret *ports.Secret) (*ports.Secret, error) {

	if secret == nil {
		r.logger.Warn("attempt to create nil secret")
		return nil, domain.ErrInvalidInput
	}
	if secret.Title == "" {
		r.logger.Warn("postgres: empty title")
		return nil, domain.ErrInvalidInput
	}
	if secret.Data == "" {
		r.logger.Warn("postgres: empty data")
		return nil, domain.ErrInvalidInput
	}
	if secret.Type == "" {
		r.logger.Warn("postgres: empty type")
		return nil, domain.ErrInvalidInput
	}
	// Генерация ID если пустой
	if secret.ID == "" {
		secret.ID = uuid.New().String()
	}

	// Установка временных меток
	now := time.Now()
	if secret.CreatedAt.IsZero() {
		secret.CreatedAt = now
	}
	if secret.UpdatedAt.IsZero() {
		secret.UpdatedAt = now
	}

	const query = `
		INSERT INTO secrets (id, user_id, type, title, data, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at;
	`

	r.logger.Debug("saving secret to database",
		zap.String("secret_id", secret.ID),
		zap.String("user_id", secret.UserID),
		zap.String("title", secret.Title),
	)

	// Использует контекст
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

// GetByUserID возвращает все секреты пользователя.
//
// Алгоритм:
//  1. Выполняет SELECT по user_id
//  2. Возвращает слайс секретов
func (r *secretRepository) GetByUserID(ctx context.Context, userID string) ([]*ports.Secret, error) {

	const query = `
		SELECT id, user_id, type, title, data, metadata, created_at, updated_at
		FROM secrets 
		WHERE user_id = $1 
		ORDER BY created_at DESC;
	`

	r.logger.Debug("GetByUserID called", zap.String("user_id", userID), zap.String("query", query))

	// Использует контекст
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
			r.logger.Error("postgres: scan failed", zap.Error(err))
			return nil, err
		}
		secrets = append(secrets, &s)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("GetByUserID rows error", zap.Error(err))
		return nil, err
	}

	r.logger.Debug("GetByUserID success", zap.Int("count", len(secrets)))
	return secrets, nil
}

// GetSecret возвращает один секрет по ID и userID.
//
// Алгоритм:
//  1. Выполняет SELECT с фильтром по id и user_id
//  2. При отсутствии строки возвращает domain.ErrNotFound
func (r *secretRepository) GetSecret(ctx context.Context, id, userID string) (*ports.Secret, error) {
	const query = `
		SELECT id, user_id, type, title, data, metadata, created_at, updated_at
		FROM secrets 
		WHERE id = $1 AND user_id = $2;
	`

	r.logger.Debug("GetSecret called", zap.String("id", id),
		zap.String("user_id", userID))

	var s ports.Secret
	// Использует контекст
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

	// Корректная обработка ErrNoRows
	if err == pgx.ErrNoRows {
		r.logger.Debug("GetSecret not found", zap.String("id", id), zap.String("user_id", userID))
		return nil, domain.ErrNotFound
	}
	if err != nil {
		r.logger.Error("GetSecret query failed", zap.Error(err))
		return nil, err
	}

	r.logger.Debug("GetSecret success", zap.String("id", s.ID), zap.String("title", s.Title))
	return &s, nil
}

// Delete удаляет секрет пользователя.
//
// Алгоритм:
//  1. Выполняет DELETE с фильтром по id и user_id
//  2. Если ничего не удалено, возвращает domain.ErrNotFound
func (r *secretRepository) Delete(ctx context.Context, userID, secretID string) error {
	const query = `DELETE FROM secrets WHERE id = $1 AND user_id = $2`

	r.logger.Debug("Delete called", zap.String("secret_id", secretID), zap.String("user_id", userID))

	result, err := r.pool.Exec(ctx, query, secretID, userID)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	r.logger.Debug("Delete result", zap.Int64("rows_affected", rowsAffected))

	// Проверка, что удалили хотя бы одну строку
	if result.RowsAffected() == 0 {
		r.logger.Debug("Delete not found", zap.String("secret_id", secretID), zap.String("user_id", userID))
		return domain.ErrNotFound
	}

	r.logger.Info("postgres: secret deleted", zap.String("secret_id", secretID))
	return nil
}
