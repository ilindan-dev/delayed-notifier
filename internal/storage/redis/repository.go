package redis

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	repo "github.com/ilindan-dev/delayed-notifier/internal/domain/repository"
	"github.com/rs/zerolog"
	"time"
)

// Ensure CachedNotificationRepository implements the interface
var _ repo.NotificationRepository = (*CachedNotificationRepository)(nil)

// CachedNotificationRepository is a decorator for a NotificationRepository
// that adds a caching layer using Redis.
type CachedNotificationRepository struct {
	primaryRepo repo.NotificationRepository
	cache       repo.NotificationCache
	logger      zerolog.Logger
	ttl         time.Duration
}

// NewCachedNotificationRepository creates a new instance of the cached repository.
// It takes the primary repository and the cache as dependencies.
func NewCachedNotificationRepository(
	primaryRepo repo.NotificationRepository,
	cache repo.NotificationCache,
	logger *zerolog.Logger,
) *CachedNotificationRepository {
	return &CachedNotificationRepository{
		primaryRepo: primaryRepo,
		cache:       cache,
		logger:      logger.With().Str("layer", "cached_repository").Logger(),
		ttl:         time.Hour * 24, // Default cache TTL of 24 hours
	}
}

// Save first persists the notification in the primary repository,
// then warms up the cache with the new data.
func (r *CachedNotificationRepository) Save(ctx context.Context, n *model.Notification) (*model.Notification, error) {
	created, err := r.primaryRepo.Save(ctx, n)
	if err != nil {
		return nil, err
	}

	if err := r.cache.Set(ctx, created, r.ttl); err != nil {
		r.logger.Error().Err(err).Stringer("id", created.ID).Msg("failed to cache notification after save")
	}

	return created, nil
}

// GetByID implements the cache-aside pattern.
// It first tries to fetch the data from the cache. If it's a miss,
// it fetches from the primary repository, caches the result, and then returns it.
func (r *CachedNotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	cached, err := r.cache.Get(ctx, id)
	if err == nil {
		r.logger.Info().Stringer("id", id).Msg("cache hit")
		return cached, nil
	}

	if !errors.Is(err, repo.ErrNotFound) {
		r.logger.Error().Err(err).Stringer("id", id).Msg("cache get error, falling back to primary repository")
	} else {
		r.logger.Info().Stringer("id", id).Msg("cache miss")
	}

	primary, err := r.primaryRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := r.cache.Set(ctx, primary, r.ttl); err != nil {
		r.logger.Error().Err(err).Stringer("id", primary.ID).Msg("failed to set cache after db fetch")
	}

	return primary, nil
}

// Update first updates the data in the primary repository,
// then invalidates the corresponding cache entry.
func (r *CachedNotificationRepository) Update(ctx context.Context, n *model.Notification) error {
	if err := r.primaryRepo.Update(ctx, n); err != nil {
		return err
	}

	if err := r.cache.Delete(ctx, n.ID); err != nil {
		r.logger.Error().Err(err).Stringer("id", n.ID).Msg("failed to invalidate cache after update")
	}

	return nil
}

// Delete first deletes the data from the primary repository,
// then invalidates the cache.
func (r *CachedNotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.primaryRepo.Delete(ctx, id); err != nil {
		return err
	}

	if err := r.cache.Delete(ctx, id); err != nil {
		r.logger.Error().Err(err).Stringer("id", id).Msg("failed to invalidate cache after delete")
	}

	return nil
}
