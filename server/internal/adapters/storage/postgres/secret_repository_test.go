// Package postgres содержит интеграционные тесты для PostgreSQL хранилища секретов.
//
// Тестируются:
//   - Create: создание секрета
//   - GetByUserID: получение списка секретов
//   - GetSecret: получение одного секрета
//   - Delete: удаление секрета
//
// Внимание: тесты используют реальную БД (требуют запущенный PostgreSQL)
package postgres

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/logger"
	"gophkeeper/server/internal/ports"
	"os"
	"testing"
	"time"
)

// testLogger — мок для logger.Logger
type testLogger struct{}

func (testLogger) Debug(msg string, fields ...zap.Field)  {}
func (testLogger) Info(msg string, fields ...zap.Field)   {}
func (testLogger) Warn(msg string, fields ...zap.Field)   {}
func (testLogger) Error(msg string, fields ...zap.Field)  {}
func (testLogger) Fatal(msg string, fields ...zap.Field)  {}
func (testLogger) With(fields ...zap.Field) logger.Logger { return testLogger{} }
func (testLogger) Sync() error                            { return nil }

// newTestPostgresRepo создаёт тестовое подключение к реальной БД.
// - Требует наличия запущенного PostgreSQL с тестовой БД
func newTestPostgresRepo(t *testing.T) (*secretRepository, string) {
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
	repo := NewSecretRepository(pool, log).(*secretRepository)

	// Очистка таблиц перед тестами
	_, _ = pool.Exec(ctx, "TRUNCATE TABLE secrets, users RESTART IDENTITY CASCADE")

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

// TestSecretRepository_Create тестирует создание секрета.
//
// Сценарии:
//   - success: успешное создание
//   - nil_secret: секрет nil → ошибка
func TestSecretRepository_Create(t *testing.T) {
	repo, userID := newTestPostgresRepo(t)

	tests := []struct {
		name    string
		secret  *ports.Secret
		wantErr bool
	}{
		{
			name: "success",
			secret: &ports.Secret{
				ID:        uuid.New().String(),
				UserID:    userID,
				Type:      "password",
				Title:     "Test Password",
				Data:      "encrypted-pass",
				Metadata:  "{}",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name:    "nil_secret",
			secret:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			created, err := repo.Create(context.Background(), tt.secret)

			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, domain.ErrInvalidInput)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, created.ID)
			assert.Equal(t, tt.secret.Title, created.Title)
		})
	}
}

// TestSecretRepository_GetByUserID тестирует получение списка секретов.
func TestSecretRepository_GetByUserID(t *testing.T) {
	repo, userID := newTestPostgresRepo(t)

	// Создание тестовых секретов
	_, err := repo.Create(context.Background(), &ports.Secret{
		ID: uuid.New().String(), UserID: userID, Type: "password", Title: "S1", Data: "d1",
	})
	require.NoError(t, err)

	_, _ = repo.Create(context.Background(), &ports.Secret{
		ID: uuid.New().String(), UserID: userID, Type: "note", Title: "S2", Data: "d2",
	})
	require.NoError(t, err)

	// Получение списка
	secrets, err := repo.GetByUserID(context.Background(), userID)
	require.NoError(t, err)
	assert.Len(t, secrets, 2)
}

// TestSecretRepository_GetSecret тестирует получение одного секрета.
func TestSecretRepository_GetSecret(t *testing.T) {
	repo, userID := newTestPostgresRepo(t)

	// Создание секрета
	secret := &ports.Secret{
		ID:     uuid.New().String(),
		UserID: userID,
		Type:   "card",
		Title:  "My Card",
		Data:   "card-data",
	}
	_, err := repo.Create(context.Background(), secret)
	require.NoError(t, err)

	// Успешное получение
	found, err := repo.GetSecret(context.Background(), secret.ID, secret.UserID)
	require.NoError(t, err)
	assert.Equal(t, "My Card", found.Title)

	// Отрицательный случай: неверный пользователь
	_, err = repo.GetSecret(context.Background(), secret.ID, "wrong-user")
	assert.Error(t, err)

}

// TestSecretRepository_Delete тестирует удаление секрета.
func TestSecretRepository_Delete(t *testing.T) {
	repo, userID := newTestPostgresRepo(t)

	// Создание секрета
	secret := &ports.Secret{
		ID:       uuid.New().String(),
		UserID:   userID,
		Type:     "password",
		Title:    "Test Delete",
		Data:     "data-to-delete",
		Metadata: "{}",
	}

	_, err := repo.Create(context.Background(), secret)
	require.NoError(t, err)

	// Успешное удаление
	err = repo.Delete(context.Background(), secret.UserID, secret.ID)
	assert.NoError(t, err)

	// Повторное удаление — должно вернуть ErrNotFound
	err = repo.Delete(context.Background(), secret.UserID, secret.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// Unit-тесты с pgxmock (без реальной БД)
// TestSecretRepositoryUnit_Create тестирует Create с моком
func TestSecretRepositoryUnit(t *testing.T) {
	log := testLogger{}

	t.Run("Create", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewSecretRepository(mock, log).(*secretRepository)

		secret := &ports.Secret{
			ID:        "sec-1",
			UserID:    "user-1",
			Type:      "password",
			Title:     "Test",
			Data:      "data",
			Metadata:  "{}",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Ожидание INSERT
		mock.ExpectExec("INSERT INTO secrets").
			WithArgs(
				secret.ID,
				secret.UserID,
				secret.Type,
				secret.Title,
				secret.Data,
				secret.Metadata,
				secret.CreatedAt,
				secret.UpdatedAt,
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		created, err := repo.Create(context.Background(), secret)
		require.NoError(t, err)
		assert.Equal(t, secret.ID, created.ID)

		// Проверка выполнения всех ожиданий
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	// Test GetByUserID тестирует GetByUserID с моком
	t.Run("GetByUserID", func(t *testing.T) {
		log := testLogger{}

		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewSecretRepository(mock, log).(*secretRepository)

		// Подготовка мок-строк
		rows := mock.NewRows([]string{"id", "user_id", "type", "title", "data", "metadata", "created_at", "updated_at"}).
			AddRow("sec-1", "user-1", "password", "S1", "d1", "{}", time.Now(), time.Now()).
			AddRow("sec-2", "user-1", "note", "S2", "d2", "{}", time.Now(), time.Now())

		mock.ExpectQuery(`SELECT .* FROM secrets WHERE user_id = \$1`).
			WithArgs("user-1").
			WillReturnRows(rows)

		secrets, err := repo.GetByUserID(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Len(t, secrets, 2)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	// Test GetSecret тестирует GetSecret с моком
	t.Run("GetSecret", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewSecretRepository(mock, log).(*secretRepository)

		row := mock.NewRows([]string{"id", "user_id", "type", "title", "data", "metadata", "created_at", "updated_at"}).
			AddRow("sec-42", "user-1", "card", "My Card", "data", "{}", time.Now(), time.Now())

		mock.ExpectQuery(`SELECT .* FROM secrets WHERE id = \$1 AND user_id = \$2`).
			WithArgs("sec-42", "user-1").
			WillReturnRows(row)

		secret, err := repo.GetSecret(context.Background(), "sec-42", "user-1")
		require.NoError(t, err)
		assert.Equal(t, "My Card", secret.Title)
	})

	t.Run("Delete", func(t *testing.T) {
		log := testLogger{}

		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := NewSecretRepository(mock, log).(*secretRepository)

		// Ожидание DELETE с возвратом 0 строк (not found)
		mock.ExpectExec(`DELETE FROM secrets WHERE id = \$1 AND user_id = \$2`).
			WithArgs("sec-99", "user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err = repo.Delete(context.Background(), "user-1", "sec-99")
		assert.ErrorIs(t, err, domain.ErrNotFound)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
