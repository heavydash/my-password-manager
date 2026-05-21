// Package ports содержит контракты (интерфейсы) между слоями приложения.
//
// Все зависимости идут от domain/usecases → ports → adapters.
package ports

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBPool — интерфейс пула соединений с PostgreSQL.
//
// Реализуется:
//   - pgxpool.Pool (production)
//   - pgxmock.Pool (тесты)
//
// Используется в репозиториях для выполнения запросов к БД.
//
// Методы:
//   - Exec: выполнение запроса без возврата строк (INSERT, UPDATE, DELETE)
//   - Query: выполнение запроса с возвратом строк (SELECT)
//   - QueryRow: выполнение запроса с возвратом одной строки
//   - Close: закрытие пула соединений
type DBPool interface {
	// Exec выполняет SQL-запрос, который не возвращает строки.
	// Возвращает CommandTag с информацией о количестве затронутых строк.
	//
	// Пример:
	//   tag, err := pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	//   rowsAffected := tag.RowsAffected()
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)

	// Query выполняет SQL-запрос, возвращающий строки.
	// Возвращает Rows, которые нужно закрыть после использования.
	//
	// Пример:
	//   rows, err := pool.Query(ctx, "SELECT id, email FROM users WHERE id = $1", userID)
	//   defer rows.Close()
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)

	// QueryRow выполняет SQL-запрос, возвращающий одну строку.
	// Возвращает Row, которая сканируется в переменные.
	//
	// Пример:
	//   var id string
	//   err := pool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&id)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row

	// Close закрывает пул соединений и освобождает ресурсы.
	// Должен вызываться при завершении работы приложения.
	Close()
}
