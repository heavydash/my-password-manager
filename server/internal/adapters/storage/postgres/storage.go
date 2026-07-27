package postgres

import (
	"context"
	"database/sql"
	_ "github.com/lib/pq"
	"gophkeeper/server/internal/ports"
)

type storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) ports.StoragePort {
	return &storage{db: db}
}

func (s *storage) CreateUser(ctx context.Context, user ports.User) (string, error) {
	const query = `
	INSERT INTO users (email, password_hash, provider, provider_id) 
	VALUES ($1, $2, $3, $4)
	RETURNING id;
`
	var userID string
	err := s.db.QueryRowContext(ctx, query,
		user.Email,
		user.PasswordHash,
		user.Provider,
		user.ProviderID,
	).Scan(&userID)

	if err != nil {
		return "", err
	}

	return userID, nil
}

func (s *storage) GetUserByEmail(ctx context.Context, email string) (ports.User, error) {
	const query = `
	SELECT id, email, password_hash, provider, provider_id
	FROM users
	WHERE email = $1;
`
	var u ports.User
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Provider,
		&u.ProviderID,
	)
	if err != nil {
		return ports.User{}, err
	}
	return u, nil
}

func (s *storage) GetUserByID(ctx context.Context, id string) (ports.User, error) {
	const query = `
	SELECT id, email, password_hash, provider, provider_id
	FROM users
	WHERE id = $1;
`
	var u ports.User
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Provider,
		&u.ProviderID,
	)

	if err != nil {
		return ports.User{}, err
	}
	return u, nil
}
