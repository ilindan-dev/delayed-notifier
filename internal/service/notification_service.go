package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	repo "github.com/ilindan-dev/delayed-notifier/internal/domain/repository"
	"github.com/rs/zerolog"
	"net/mail"
	"strconv"
	"time"
)

// NotificationService encapsulates the business logic for managing notifications.
// It orchestrates the repository and the queue.
type NotificationService struct {
	repo   repo.NotificationRepository
	queue  repo.NotificationQueue
	logger zerolog.Logger
}

func NewNotificationService(
	repo repo.NotificationRepository,
	queue repo.NotificationQueue,
	logger zerolog.Logger,
) *NotificationService {
	return &NotificationService{
		repo:   repo,
		queue:  queue,
		logger: logger.With().Str("layer", "service").Logger(),
	}
}

// CreateNotification orchestrates the creation of a new notification.
// It validates input, saves the notification, and publishes it to the queue.
func (s *NotificationService) CreateNotification(ctx context.Context, recipient string, channel model.Channel, subject, message string, scheduledAt time.Time, authorID *string) (*model.Notification, error) {
	s.logger.Info().Str("channel", string(channel)).Msg("creating new notification")

	var notification *model.Notification

	switch channel {
	case model.ChannelEmail:
		if _, err := mail.ParseAddress(recipient); err != nil {
			s.logger.Warn().Err(err).Str("recipient", recipient).Msg("invalid recipient")
			return nil, fmt.Errorf("invalid email format: %w", err)
		}
		notification = model.NewEmailNotification(recipient, subject, message, scheduledAt, authorID)
	case model.ChannelTelegram:
		chatID, err := strconv.ParseInt(recipient, 10, 64)
		if err != nil {
			s.logger.Warn().Err(err).Str("recipient", recipient).Msg("invalid recipient")
			return nil, fmt.Errorf("invalid telegram chat id: %w", err)
		}
		notification = model.NewTelegramNotification(chatID, subject, message, scheduledAt, authorID)
	default:
		s.logger.Warn().Str("channel", string(channel)).Msg("invalid channel")
		return nil, fmt.Errorf("unknown channel: %s", channel)
	}

	createdNotification, err := s.repo.Save(ctx, notification)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to save notification")
		return nil, err
	}
	s.logger.Info().Stringer("id", createdNotification.ID).Msg("notification saved successfully")

	err = s.queue.Publish(ctx, createdNotification)
	if err != nil {
		s.logger.Error().Err(err).Stringer("id", createdNotification.ID).Msg("CRITICAL: failed to publish notification to queue after saving")
		return nil, fmt.Errorf("failed to schedule notification: %w", err)
	}
	s.logger.Info().Stringer("id", createdNotification.ID).Msg("notification published to queue")

	return createdNotification, nil
}

// GetNotificationByID retrieves a notification by its ID.
// The business logic is simple: just ask the repository.
// The repository decorator handles the cache-aside logic transparently.
func (s *NotificationService) GetNotificationByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	n, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Msgf("Failed to get notification by ID: %s", id)
		return nil, err
	}
	s.logger.Info().Msgf("Getting notification by ID: %s", id)
	return n, nil
}

// UpdateNotification is used by the consumer to update the status after a send attempt.
// The repository decorator will handle cache invalidation.
func (s *NotificationService) UpdateNotification(ctx context.Context, n *model.Notification) error {
	if err := s.repo.Update(ctx, n); err != nil {
		s.logger.Error().Err(err).Msgf("Failed to update notification: %s", n.ID)
		return err
	}
	return nil
}

// CancelNotification cancels a scheduled notification.
func (s *NotificationService) CancelNotification(ctx context.Context, id uuid.UUID) error {
	notification, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("notification_id", id.String()).Msg("can't get notification")
		return err
	}

	if notification.Status != model.StatusScheduled {
		s.logger.Warn().Str("notification_id", id.String()).Msg("can't cancel notification")
		return fmt.Errorf("cannot cancel notification with status: %s", notification.Status)
	}

	s.logger.Info().Str("notification_id", id.String()).Msg("cancel notification")
	return s.repo.Delete(ctx, id)
}
