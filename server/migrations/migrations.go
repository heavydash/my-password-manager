// Package migrations управляет миграциями базы данных GophKeeper.
//
// Использует библиотеку goose для версионирования схемы БД.
// Миграции хранятся в директории server/migrations/.
//
// Поддерживаемые типы миграций:
//   - SQL-файлы с UP/DOWN инструкциями
//   - Автоматическое применение при запуске сервера
//   - Graceful shutdown с поддержкой отмены через контекст
//
// Пример SQL миграции (server/migrations/20240101000000_init.sql):
//
//	-- +goose Up
//	CREATE TABLE users (id UUID PRIMARY KEY);
//
//	-- +goose Down
//	DROP TABLE users;
package migrations

import (
	"context"
	"github.com/pressly/goose/v3"
	"gophkeeper/server/internal/config"
	_ "gophkeeper/server/internal/config"
	"gophkeeper/server/internal/domain"
	"log"
	"strings"
)

// RunMigrations выполняет все UP миграции из директории server/migrations/.
//
// Алгоритм:
//  1. Открывает соединение с БД через драйвер "pgx"
//  2. Устанавливает таймаут на выполнение каждой миграции (5 минут)
//  3. Выполняет все новые миграции через goose.Up
//  4. Закрывает соединение с БД
//
// Особенности:
//   - Игнорирует ошибку "no next version found" (миграции не требуются)
//   - Использует goose.Up, не goose.UpWithContext (совместимость с версией)
//
// Параметры:
//   - ctx: контекст для отмены операции (таймаут, graceful shutdown)
//   - dsn: строка подключения к PostgreSQL (pgx://...)
//
// Возвращает:
//   - error: ошибка выполнения миграций (кроме "no next version found")
func RunMigrations(ctx context.Context, cfg *config.Config) error {
	// Открытие соединения с БД с использованием драйвера pgx
	db, err := goose.OpenDBWithDriver("pgx", cfg.Database.DSN)
	if err != nil {
		return err
	}
	// Гарантированное закрытие соединения
	defer func() {
		if err = db.Close(); err != nil {
			log.Printf("error closing db: %v", err)
		}
	}()

	// Установка таймаута на выполнение миграций
	timeoutCtx, cancel := context.WithTimeout(ctx, cfg.Database.MigrationTimeout)
	defer cancel()

	// Создание канала для отслеживания завершения миграций
	done := make(chan error, 1)

	// Запуск миграций в горутине (goose не поддерживает контекст напрямую)
	go func() {
		err := goose.Up(db, "migrations")
		done <- err
	}()

	// Ожидание завершения миграций или отмены контекста
	select {
	case err := <-done:
		// Миграции завершились (успешно или с ошибкой)
		if err != nil {
			// "no next version found" — не ошибка, просто нет новых миграций
			if strings.Contains(err.Error(), "no next version found") {
				return nil
			}
			return err
		}
		return nil
	case <-timeoutCtx.Done():
		// Таймаут или отмена контекста
		return domain.ErrMigrationTimeout
	}
}
