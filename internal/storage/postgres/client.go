package postgres

import (
	"context"
	"fmt"
	"github.com/ilindan-dev/delayed-notifier/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// NewPool creates and returns a new connection pool for PostgreSQL using pgx.
// It also registers lifecycle hooks with Fx to handle startup and shutdown.
func NewPool(lc fx.Lifecycle, cfg *config.Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.Postgres.MasterDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.Postgres.Pool.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.Postgres.Pool.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.Postgres.Pool.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return pool.Ping(ctx)
		},
		OnStop: func(ctx context.Context) error {
			pool.Close()
			return nil
		},
	})

	return pool, nil
}
