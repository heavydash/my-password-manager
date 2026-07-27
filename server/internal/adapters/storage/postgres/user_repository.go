// Package postgres реализует PostgreSQL хранилище пользователей.
//
// Содержит:
//   - storage: реализация ports.StoragePort
//   - Поддержка контекста и graceful shutdown
package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"strings"
)

// storage реализует хранилище пользователей в PostgreSQL.
//
// Реализует ports.StoragePort.
type storage struct {
	pool   ports.DBPool
	logger logger.Logger
}

// NewUserRepository создаёт новое PostgreSQL хранилище пользователей.
//
// Параметры:
//   - pool: пул соединений с PostgreSQL (реализует ports.DBPool)
//   - log: логгер для записи событий
//
// Паникует, если pool или log == nil.
//
// Возвращает реализацию ports.StoragePort.
func NewUserRepository(pool ports.DBPool, log logger.Logger) ports.StoragePort {
	if log == nil {
		panic("logger is required for user storage")
	}
	if pool == nil {
		panic("database connection is required")
	}

	return &storage{pool: pool,
		logger: log}
}

// CreateUser создаёт нового пользователя в БД.
//
// Алгоритм:
//  1. Выполняет INSERT
//  2. Возвращает сгенерированный userID
//
// Возвращает userID или ошибку.
func (s *storage) CreateUser(ctx context.Context, user ports.User) (string, error) {
	const query = `
	INSERT INTO users (email, password_hash, provider, provider_id) 
	VALUES ($1, $2, $3, $4)
	RETURNING id;
`

	s.logger.Info("creating new user in database",
		zap.String("email", user.Email),
		zap.String("provider", user.Provider),
	)

	var userID string
	err := s.pool.QueryRow(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Provider,
		user.ProviderID,
	).Scan(&userID)

	if err != nil {
		s.logger.Error("failed to create user in database",
			zap.String("email", user.Email),
			zap.Error(err),
		)
		return "", err
	}

	s.logger.Info("user created in database successfully",
		zap.String("user_id", userID),
		zap.String("email", user.Email),
	)

	return userID, nil
}

// GetUserByEmail возвращает пользователя по email.
//
// Алгоритм:
//  1. Выполняет SELECT по email
//  2. При отсутствии строки возвращает domain.ErrUserNotFound
func (s *storage) GetUserByEmail(ctx context.Context, email string) (ports.User, error) {
	const query = `
	SELECT id, email, password_hash, provider, provider_id
	FROM users
	WHERE email = $1;
`
	s.logger.Debug("fetching user by email", zap.String("email", email))

	var u ports.User
	err := s.pool.QueryRow(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Provider,
		&u.ProviderID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			s.logger.Debug("postgres: user not found by email", zap.String("email", email))
			return ports.User{}, domain.ErrUserNotFound
		}
		s.logger.Error("postgres: failed to get user by email", zap.Error(err))
		return ports.User{}, err
	}

	s.logger.Debug("user found by email",
		zap.String("user_id", u.ID),
		zap.String("email", u.Email),
	)

	return u, nil
}

// GetUserByID возвращает пользователя по ID.
//
// Алгоритм:
//  1. Выполняет SELECT по id
//  2. При отсутствии строки возвращает domain.ErrUserNotFound
func (s *storage) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	const query = `
	SELECT id, email, password_hash, provider, provider_id
	FROM users
	WHERE id = $1;
`
	s.logger.Debug("fetching user by id", zap.String("user_id", id))

	var u ports.User
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Provider,
		&u.ProviderID,
	)

	if err != nil {
		if err == pgx.ErrNoRows || strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			s.logger.Debug("postgres: user not found by id", zap.String("user_id", id))
			return ports.User{}, domain.ErrUserNotFound
		}
		s.logger.Error("postgres: failed to get user by id", zap.Error(err))
		return ports.User{}, err
	}

	s.logger.Debug("user found by id",
		zap.String("user_id", u.ID),
		zap.String("email", u.Email),
	)

	return u, nil
}
