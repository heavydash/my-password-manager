package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"gophkeeper/server/internal/config"
	"gophkeeper/server/internal/logger"
)

// Pool — обёртка над pgxpool.Pool.
//
// Добавляет типизацию и может быть расширен дополнительными методами.
type Pool struct {
	*pgxpool.Pool
}

// NewPool создаёт новый пул соединений с PostgreSQL.
//
// Алгоритм:
//  1. Проверяет обязательные параметры (cfg, log)
//  2. Парсит DSN из конфига
//  3. Настраивает пул (MaxConns, MinConns, MaxConnLifetime, MaxConnIdleTime, HealthCheckPeriod)
//  4. Создаёт пул соединений
//  5. Выполняет Ping для проверки доступности БД
//  6. При ошибке закрывает пул и возвращает ошибку
//
// Параметры:
//   - ctx: контекст для операций с БД (таймаут, отмена)
//   - cfg: конфигурация сервера (содержит Database DSN и настройки пула)
//   - log: логгер для записи событий
//
// Возвращает:
//   - *Pool: готовый к использованию пул соединений
//   - error: ошибка при подключении или настройке
func NewPool(ctx context.Context, cfg *config.Config, log logger.Logger) (*Pool, error) {
	// Проверка обязательных параметров
	if cfg == nil {
		log.Fatal("postgres pool: config is nil")
		return nil, nil // никогда не выполнится из-за Fatal, но для компиляции
	}
	if log == nil {
		panic("postgres pool: logger is nil")
	}

	// Парсинг конфигурации из DSN
	pgxCfg, err := pgxpool.ParseConfig(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	// Настройки пула из конфига
	pgxCfg.MaxConns = int32(cfg.Database.MaxConns)
	pgxCfg.MinConns = int32(cfg.Database.MinConns)
	pgxCfg.MaxConnLifetime = cfg.Database.MaxConnLifetime
	pgxCfg.MaxConnIdleTime = cfg.Database.MaxConnIdleTime
	pgxCfg.HealthCheckPeriod = cfg.Database.HealthCheckPeriod

	log.Debug("postgres pool: config parsed",
		zap.Int("max_conns", cfg.Database.MaxConns),
		zap.Int("min_conns", cfg.Database.MinConns),
		zap.Duration("max_lifetime", cfg.Database.MaxConnLifetime),
		zap.Duration("max_idle_time", cfg.Database.MaxConnIdleTime),
	)

	// Создание пула соединений
	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, err
	}

	// Проверка соединения (Ping)
	pingCtx, cancel := context.WithTimeout(ctx, cfg.Database.PingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		log.Error("postgres pool: ping failed", zap.Error(err))
		pool.Close()
		return nil, err
	}

	log.Info("postgres pool: connection pool initialized successfully",
		zap.Int("max_conns", cfg.Database.MaxConns),
		zap.Int("min_conns", cfg.Database.MinConns),
		zap.Duration("max_lifetime", cfg.Database.MaxConnLifetime),
		zap.Duration("ping_timeout", cfg.Database.PingTimeout),
	)
	return &Pool{Pool: pool}, nil
}
