// Package memory содержит тесты для in-memory хранилища пользователей.
//
// Тестируются:
//   - CreateUser: создание пользователя
//   - GetUserByEmail: получение пользователя по email
//   - GetUserByID: получение пользователя по ID
//   - Контекст: отмена операций
package memory

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gophkeeper/server/internal/domain"
	"gophkeeper/server/internal/ports"
	"testing"
	"time"
)

// TestUserStorage_CreateUser тестирует создание пользователя.
//
// Сценарии:
//   - success: успешное создание пользователя
//   - empty_email: пустой email → ошибка ErrInvalidInput
//   - duplicate_email: дубликат email → ошибка ErrUserAlreadyExists
func TestUserStorage_CreateUser(t *testing.T) {
	// Инициализация логгера и хранилища
	log := testLogger{}
	repo := NewUserRepository(log).(*UserStorage)

	// Подготовка тестовых случаев
	tests := []struct {
		name        string     // имя теста
		user        ports.User // пользователь для создания
		wantErr     bool       // ожидается ли ошибка
		wantErrType error      // тип ожидаемой ошибки
	}{
		{
			name: "success",
			user: ports.User{
				ID:           "user-123",
				Email:        "test@example.com",
				PasswordHash: "hash123",
			},
			wantErr: false,
		},
		{
			name: "empty_email",
			user: ports.User{
				ID:           "user-456",
				Email:        "",
				PasswordHash: "hash",
			},
			wantErr:     true,
			wantErrType: domain.ErrInvalidInput,
		},
		{
			name: "duplicate_email",
			user: ports.User{
				ID:           "user-789",
				Email:        "test@example.com",
				PasswordHash: "hash",
			},
			wantErr:     true,
			wantErrType: domain.ErrUserAlreadyExists,
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := repo.CreateUser(context.Background(), tt.user)

			// Проверка ошибок
			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				return
			}

			// Проверка успешного создания
			require.NoError(t, err)
			assert.NotEmpty(t, id)

			// Проверяем, что сохранилось
			stored, exists := repo.users[id]
			assert.True(t, exists)
			assert.Equal(t, tt.user.Email, stored.Email)
		})
	}
}

// TestUserStorage_GetUserByEmail тестирует получение пользователя по email.
//
// Сценарии:
//   - success: пользователь найден
//   - not_found: пользователь не найден
//   - empty_email: пустой email
func TestUserStorage_GetUserByEmail(t *testing.T) {
	// Инициализация логгера и хранилища
	log := testLogger{}
	repo := NewUserRepository(log).(*UserStorage)

	// Подготовка тестовых данных
	user := ports.User{
		ID:           "user-123",
		Email:        "test@example.com",
		PasswordHash: "hash123",
	}
	repo.users[user.ID] = user

	// Подготовка тестовых случаев
	tests := []struct {
		name      string // имя теста
		email     string // email для поиска
		wantErr   bool   // ожидается ли ошибка
		wantEmail string // ожидаемый email в ответе
	}{
		{"success", "test@example.com", false, "test@example.com"},
		{"not_found", "unknown@example.com", true, ""},
		{"empty_email", "", true, ""},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := repo.GetUserByEmail(context.Background(), tt.email)

			// Проверка ошибок
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			// Проверка успешного получения
			require.NoError(t, err)
			assert.Equal(t, tt.wantEmail, u.Email)
		})
	}
}

// TestUserStorage_GetUserByID тестирует получение пользователя по ID.
//
// Сценарии:
//   - success: пользователь найден
//   - not_found: пользователь не найден
func TestUserStorage_GetUserByID(t *testing.T) {
	// Инициализация логгера и хранилища
	log := testLogger{}
	repo := NewUserRepository(log).(*UserStorage)

	// Подготовка тестовых данных
	user := ports.User{
		ID:           "user-456",
		Email:        "another@example.com",
		PasswordHash: "hash456",
	}
	repo.users[user.ID] = user

	// Подготовка тестовых случаев
	tests := []struct {
		name      string // имя теста
		id        string // ID для поиска
		wantErr   bool   // ожидается ли ошибка
		wantEmail string // ожидаемый email в ответе
	}{
		{"success", "user-456", false, "another@example.com"},
		{"not_found", "user-999", true, ""},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := repo.GetUserByID(context.Background(), tt.id)

			// Проверка ошибок
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			// Проверка успешного получения
			require.NoError(t, err)
			assert.Equal(t, tt.wantEmail, u.Email)
		})
	}
}

// TestUserStorage_ContextCancellation тестирует отмену контекста.
//
// Сценарии:
//   - create_user_cancelled: отмена при создании
//   - get_by_email_cancelled: отмена при поиске по email
//   - get_by_id_cancelled: отмена при поиске по ID
func TestUserStorage_ContextCancellation(t *testing.T) {
	// Инициализация логгера и хранилища
	log := testLogger{}
	repo := NewUserRepository(log).(*UserStorage)

	// Подготовка тестовых случаев
	tests := []struct {
		name string                          // имя теста
		fn   func(ctx context.Context) error // функция для вызова
	}{
		{
			name: "create_user_cancelled",
			fn: func(ctx context.Context) error {
				_, err := repo.CreateUser(ctx, ports.User{Email: "test@example.com"})
				return err
			},
		},
		{
			name: "get_by_email_cancelled",
			fn: func(ctx context.Context) error {
				_, err := repo.GetUserByEmail(ctx, "test@example.com")
				return err
			},
		},
		{
			name: "get_by_id_cancelled",
			fn: func(ctx context.Context) error {
				_, err := repo.GetUserByID(ctx, "user-123")
				return err
			},
		},
	}

	// Выполнение тестов
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создание отменённого контекста
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			// Вызов функции
			err := tt.fn(ctx)

			// Проверка ошибки отмены
			assert.ErrorIs(t, err, context.Canceled)
		})
	}
}

// TestUserStorage_ContextTimeout тестирует таймаут контекста.
func TestUserStorage_ContextTimeout(t *testing.T) {
	// Инициализация логгера и хранилища
	log := testLogger{}
	repo := NewUserRepository(log).(*UserStorage)

	// Создание контекста с истекшим таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Ожидание истечения таймаута
	time.Sleep(1 * time.Millisecond)

	// Вызов метода
	_, err := repo.CreateUser(ctx, ports.User{Email: "test@example.com"})

	// Проверка ошибки таймаута
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
