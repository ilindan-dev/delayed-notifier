package app

import (
	"context"
	"github.com/ilindan-dev/delayed-notifier/internal/config"
	"github.com/ilindan-dev/delayed-notifier/internal/consumer"
	deliveryHTTP "github.com/ilindan-dev/delayed-notifier/internal/delivery/http"
	repo "github.com/ilindan-dev/delayed-notifier/internal/domain/repository"
	"github.com/ilindan-dev/delayed-notifier/internal/logger"
	"github.com/ilindan-dev/delayed-notifier/internal/notifiers"
	"github.com/ilindan-dev/delayed-notifier/internal/service"
	"github.com/ilindan-dev/delayed-notifier/internal/storage/postgres"
	"github.com/ilindan-dev/delayed-notifier/internal/storage/rabbitmq"
	"github.com/ilindan-dev/delayed-notifier/internal/storage/redis"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"net/http"
)

// CommonModule provides dependencies that are shared between the API and Worker applications.
var CommonModule = fx.Options(
	fx.Provide(
		// Core components
		config.NewConfig,
		logger.NewLogger,

		// Storage Layer - concrete implementations
		postgres.NewPool,
		redis.NewClient,
		rabbitmq.NewConnection,
		redis.NewNotificationCache,
		postgres.NewNotificationRepository,
		rabbitmq.NewRabbitMQQueue,

		// Service Layer
		service.NewNotificationService,
	),

	fx.Decorate(func(
		pgRepo *postgres.NotificationRepository,
		cache *redis.NotificationCache,
		logger *zerolog.Logger,
	) repo.NotificationRepository {
		return redis.NewCachedNotificationRepository(pgRepo, cache, logger)
	}),
)

// APIModule defines the Fx module for the HTTP API application.
var APIModule = fx.Options(
	CommonModule, // Include all shared components
	fx.Provide(
		// API-specific components
		deliveryHTTP.NewHandlers,
		deliveryHTTP.NewServer,
	),

	fx.Invoke(func(server *deliveryHTTP.Server, lc fx.Lifecycle) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
						panic(err)
					}
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return server.Shutdown(ctx)
			},
		})
	}),
)

// WorkerModule defines the Fx module for the background worker application.
var WorkerModule = fx.Options(
	CommonModule, // Include all shared components
	fx.Provide(
		// Worker-specific components
		notifiers.NewDispatcher,
		consumer.New,
	),
	fx.Invoke(func(consumer *consumer.Consumer, lc fx.Lifecycle) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go consumer.Start(ctx)
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return nil
			},
		})
	}),
)
