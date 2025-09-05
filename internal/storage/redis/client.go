package redis

import (
	"context"
	"github.com/ilindan-dev/delayed-notifier/internal/config"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// NewClient creates and returns a new client for Redis.
// It also registers lifecycle hooks with Fx for graceful shutdown.
func NewClient(lc fx.Lifecycle, cfg *config.Config) (*goredis.Client, error) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return rdb.Ping(ctx).Err()
		},
		OnStop: func(ctx context.Context) error {

			return rdb.Close()
		},
	})

	return rdb, nil
}
