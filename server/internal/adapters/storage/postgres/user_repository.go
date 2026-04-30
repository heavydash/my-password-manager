package postgres

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
)

type storage struct {
	db     *sql.DB
	logger logger.Logger
}

func NewUserRepository(db *sql.DB, log logger.Logger) ports.StoragePort {
	if log == nil {
		panic("logger is required for user storage")
	}
	if db == nil {
		panic("database connection is required")
	}

	return &storage{db: db,
		logger: log}
}

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
	err := s.db.QueryRowContext(ctx, query,
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

func (s *storage) GetUserByEmail(ctx context.Context, email string) (ports.User, error) {
	const query = `
	SELECT id, email, password_hash, provider, provider_id
	FROM users
	WHERE email = $1;
`
	s.logger.Debug("fetching user by email", zap.String("email", email))

	var u ports.User
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Provider,
		&u.ProviderID,
	)
	if err != nil {
		s.logger.Warn("user not found by email",
			zap.String("email", email),
			zap.Error(err),
		)
		return ports.User{}, err
	}

	s.logger.Debug("user found by email",
		zap.String("user_id", u.ID),
		zap.String("email", u.Email),
	)

	return u, nil
}

func (s *storage) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	const query = `
	SELECT id, email, password_hash, provider, provider_id
	FROM users
	WHERE id = $1;
`
	s.logger.Debug("fetching user by id", zap.String("user_id", id))

	var u ports.User
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Provider,
		&u.ProviderID,
	)

	if err != nil {
		s.logger.Warn("user not found by id",
			zap.String("user_id", id),
			zap.Error(err),
		)
		return ports.User{}, err
	}

	s.logger.Debug("user found by id",
		zap.String("user_id", u.ID),
		zap.String("email", u.Email),
	)

	return u, nil
}
