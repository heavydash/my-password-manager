// Package postgres содержит тесты для PostgreSQL хранилища пользователей.
//
// Тестируются:
//   - CreateUser: создание пользователя
//   - GetUserByEmail: получение пользователя по email
//   - GetUserByID: получение пользователя по ID
//
// Включает:
//   - Unit-тесты с pgxmock
//   - Интеграционные тесты с реальной БД (требуют запущенный PostgreSQL)
package postgres

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/ports"
	"os"
	"testing"
	"time"
)

func TestUserRepository(t *testing.T) {
	log := testLogger{}

	t.Run("CreateUser", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewUserRepository(mock, log).(*storage)

		user := ports.User{
			Email:        "test@example.com",
			PasswordHash: "$2a$12$...",
			Provider:     "",
			ProviderID:   "",
		}

		mock.ExpectQuery(`INSERT INTO users \(email, password_hash, provider, provider_id\) VALUES \(\$1, \$2, \$3, \$4\) RETURNING id`).
			WithArgs(user.Email, user.PasswordHash, user.Provider, user.ProviderID).
			WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("user-uuid-123"))

		id, err := repo.CreateUser(context.Background(), user)
		require.NoError(t, err)
		assert.Equal(t, "user-uuid-123", id)
	})

	t.Run("GetUserByEmail_success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewUserRepository(mock, log).(*storage)

		expectedUser := ports.User{
			ID:           "user-uuid-123",
			Email:        "test@example.com",
			PasswordHash: "$2a$12$...",
			Provider:     "google",
			ProviderID:   "google-123",
		}

		rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "provider", "provider_id"}).
			AddRow(expectedUser.ID, expectedUser.Email, expectedUser.PasswordHash,
				expectedUser.Provider, expectedUser.ProviderID)

		mock.ExpectQuery(`SELECT id, email, password_hash, provider, provider_id FROM users WHERE email = \$1`).
			WithArgs("test@example.com").
			WillReturnRows(rows)

		u, err := repo.GetUserByEmail(context.Background(), "test@example.com")
		require.NoError(t, err)
		assert.Equal(t, expectedUser, u)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetUserByEmail_not_found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewUserRepository(mock, log).(*storage)

		mock.ExpectQuery(`SELECT id, email, password_hash, provider, provider_id FROM users WHERE email = \$1`).
			WithArgs("notfound@example.com").
			WillReturnError(pgx.ErrNoRows)

		_, err = repo.GetUserByEmail(context.Background(), "notfound@example.com")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetUserByID_success", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewUserRepository(mock, log).(*storage)

		expectedUser := ports.User{
			ID:           "user-uuid-456",
			Email:        "another@example.com",
			PasswordHash: "$2a$12$...",
			Provider:     "",
			ProviderID:   "",
		}

		rows := pgxmock.NewRows([]string{"id", "email", "password_hash", "provider", "provider_id"}).
			AddRow(expectedUser.ID, expectedUser.Email, expectedUser.PasswordHash,
				expectedUser.Provider, expectedUser.ProviderID)

		mock.ExpectQuery(`SELECT id, email, password_hash, provider, provider_id FROM users WHERE id = \$1`).
			WithArgs("user-uuid-456").
			WillReturnRows(rows)

		u, err := repo.GetUserByID(context.Background(), "user-uuid-456")
		require.NoError(t, err)
		assert.Equal(t, expectedUser, u)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetUserByID_not_found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewUserRepository(mock, log).(*storage)

		mock.ExpectQuery(`SELECT id, email, password_hash, provider, provider_id FROM users WHERE id = \$1`).
			WithArgs("nonexistent").
			WillReturnError(pgx.ErrNoRows)

		_, err = repo.GetUserByID(context.Background(), "nonexistent")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// Интеграционные тесты с реальной БД (требуют запущенный PostgreSQL)
// newTestUserRepo создаёт тестовое подключение к реальной БД.
func newTestUserRepo(t *testing.T) (*storage, string) {
	t.Helper()

	// Загрузка .env файла
	_ = godotenv.Load("../../.env")

	// Получение DSN из переменных окружения
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_DSN")
	}
	if dsn == "" {
		t.Skip("DB_DSN or DATABASE_DSN environment variable not set")
	}

	// Подключение к БД
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "failed to connect to test database")

	log := testLogger{}
	repo := NewUserRepository(pool, log).(*storage)

	// Очистка таблицы перед тестами
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE users RESTART IDENTITY CASCADE")

	// Создание тестового пользователя
	testUserID := uuid.New().String()
	_, _ = pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, provider, provider_id)
		VALUES ($1, $2, $3, $4, $5)`,
		testUserID, "test@example.com", "hash", "", "")

	// Закрытие пула после тестов
	t.Cleanup(func() { pool.Close() })

	return repo, testUserID
}

// TestUserRepositoryIntegration_CreateUser тестирует создание пользователя.
func TestUserRepositoryIntegration_CreateUser(t *testing.T) {
	repo, _ := newTestUserRepo(t)

	user := ports.User{
		Email:        "new@example.com",
		PasswordHash: "hash123",
		Provider:     "",
		ProviderID:   "",
	}

	id, err := repo.CreateUser(context.Background(), user)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	// Проверка, что пользователь действительно создался
	created, err := repo.GetUserByID(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, user.Email, created.Email)
}

// TestUserRepositoryIntegration_GetUserByEmail тестирует получение по email.
func TestUserRepositoryIntegration_GetUserByEmail(t *testing.T) {
	repo, _ := newTestUserRepo(t)

	user, err := repo.GetUserByEmail(context.Background(), "test@example.com")
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", user.Email)

	// Отрицательный случай
	_, err = repo.GetUserByEmail(context.Background(), "notfound@example.com")
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}

// TestUserRepositoryIntegration_GetUserByID тестирует получение по ID.
func TestUserRepositoryIntegration_GetUserByID(t *testing.T) {
	repo, userID := newTestUserRepo(t)

	user, err := repo.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, userID, user.ID)

	// Отрицательный случай
	_, err = repo.GetUserByID(context.Background(), "nonexistent-id")
	assert.ErrorIs(t, err, domain.ErrUserNotFound)
}
