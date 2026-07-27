package postgres

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"gophkeeper/server/internal/config"
	"gophkeeper/server/internal/logger"
)

type Pool struct {
	*pgxpool.Pool
}

func NewPool(ctx context.Context, cfg *config.Config, log logger.Logger) (*Pool, error) {
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

	pool, err := pgxpool.NewWithConfig(ctx, pgxCfg)
	if err != nil {
		return nil, err
	}

	// Пинг при старте
	pingCtx, cancel := context.WithTimeout(ctx, cfg.Database.PingTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}

	log.Info("PostgreSQL connection pool initialized",
		zap.Int("max_conns", cfg.Database.MaxConns),
		zap.Int("min_conns", cfg.Database.MinConns),
		zap.Duration("max_lifetime", cfg.Database.MaxConnLifetime))

	return &Pool{Pool: pool}, nil
}
