package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	repo "github.com/ilindan-dev/delayed-notifier/internal/domain/repository"
	"github.com/ilindan-dev/delayed-notifier/pkg/keybuilder"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"time"
)

// Ensure NotificationCache implements the interface
var _ repo.NotificationCache = (*NotificationCache)(nil)

// NotificationCache implements the domain.NotificationCache interface
// using the standard go-redis client.
type NotificationCache struct {
	redis  *goredis.Client
	logger zerolog.Logger
}

// NewNotificationCache creates a new instance of the NotificationCache.
func NewNotificationCache(logger *zerolog.Logger, redis *goredis.Client) *NotificationCache {
	return &NotificationCache{
		redis:  redis,
		logger: logger.With().Str("layer", "redis_cache").Logger(),
	}
}

// Get retrieves an item from the cache.
func (c *NotificationCache) Get(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	key := keybuilder.RedisNotificationKeyBuild(id)
	val, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			c.logger.Info().Str("key", key).Str("cache", "miss").Msg("notification not found in cache")
			return nil, repo.ErrNotFound
		}
		c.logger.Error().Err(err).Str("key", key).Msg("failed to get key from redis")
		return nil, err
	}

	var notification model.Notification
	if err := json.Unmarshal([]byte(val), &notification); err != nil {
		c.logger.Error().Err(err).Str("key", key).Msg("failed to unmarshal notification from cache")
		return nil, fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	c.logger.Info().Str("key", key).Str("cache", "hit").Msg("notification found in cache")
	return &notification, nil
}

// Set adds an item to the cache for a specified duration.
func (c *NotificationCache) Set(ctx context.Context, n *model.Notification, expiration time.Duration) error {
	key := keybuilder.RedisNotificationKeyBuild(n.ID)
	nBytes, err := json.Marshal(n)
	if err != nil {
		c.logger.Error().Err(err).Stringer("id", n.ID).Msg("failed to marshal notification for cache")
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	if err := c.redis.Set(ctx, key, nBytes, expiration).Err(); err != nil {
		c.logger.Error().Err(err).Str("key", key).Msg("failed to set key in redis")
		return err
	}

	c.logger.Info().Str("key", key).Msg("notification successfully set in cache")
	return nil
}

// Delete removes an item from the cache.
func (c *NotificationCache) Delete(ctx context.Context, id uuid.UUID) error {
	key := keybuilder.RedisNotificationKeyBuild(id)
	if err := c.redis.Del(ctx, key).Err(); err != nil {
		c.logger.Error().Err(err).Str("key", key).Msg("failed to delete key from redis")
		return err
	}

	c.logger.Info().Str("key", key).Msg("successfully deleted key from redis")
	return nil
}
