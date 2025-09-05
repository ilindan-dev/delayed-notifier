package repository

import (
	"context"
	"github.com/google/uuid"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	"time"
)

// NotificationRepository defines the contract for notification persistence (e.g., a database).
type NotificationRepository interface {
	// Save persists a new notification.
	Save(ctx context.Context, n *model.Notification) (*model.Notification, error)

	// GetByID retrieves a notification by its unique ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)

	// Update updates the mutable fields of a notification, primarily its status and attempts count.
	Update(ctx context.Context, n *model.Notification) error

	// Delete cancels a scheduled notification.
	Delete(ctx context.Context, id uuid.UUID) error
}

// NotificationCache defines the contract for a caching layer.
type NotificationCache interface {
	// Get retrieves an item from the cache.
	Get(ctx context.Context, id uuid.UUID) (*model.Notification, error)

	// Set adds an item to the cache for a specified duration
	Set(ctx context.Context, n *model.Notification, expiration time.Duration) error

	// Delete removes an item from the cache.
	Delete(ctx context.Context, id uuid.UUID) error
}

// NotificationQueue defines the contract for interacting with a delayed job queue.
// This provides an abstraction over a system like RabbitMQ.
type NotificationQueue interface {
	// Publish schedules a notification for delayed processing.
	Publish(ctx context.Context, n *model.Notification) error

	// PublishRetry schedules a notification for a retry attempt with a specific delay.
	PublishRetry(ctx context.Context, n *model.Notification, retryDelay time.Duration) error
}
