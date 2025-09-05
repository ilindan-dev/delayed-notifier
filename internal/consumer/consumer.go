package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ilindan-dev/delayed-notifier/internal/config"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	repo "github.com/ilindan-dev/delayed-notifier/internal/domain/repository"
	"github.com/ilindan-dev/delayed-notifier/internal/notifiers"
	"github.com/ilindan-dev/delayed-notifier/internal/service"
	"github.com/ilindan-dev/delayed-notifier/internal/storage/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"math"
	"sync"
	"time"
)

const (
	// maxRetries is the maximum number of send attempts for a notification.
	maxRetries = 5
	// defaultWorkerCount is the default number of worker goroutines in the pool.
	defaultWorkerCount = 5
)

// Consumer listens to a RabbitMQ queue and processes messages using a pool of workers.
type Consumer struct {
	cfg         *config.Config
	logger      zerolog.Logger
	conn        *amqp.Connection // Raw connection to create channels for each worker.
	service     *service.NotificationService
	queue       repo.NotificationQueue
	notifier    notifiers.Notifier
	workerCount int
}

// New creates a new instance of Consumer.
func New(
	cfg *config.Config,
	logger *zerolog.Logger,
	conn *amqp.Connection,
	service *service.NotificationService,
	queue repo.NotificationQueue,
	notifier notifiers.Notifier,
) *Consumer {
	return &Consumer{
		cfg:         cfg,
		logger:      logger.With().Str("component", "consumer").Logger(),
		conn:        conn,
		service:     service,
		queue:       queue,
		notifier:    notifier,
		workerCount: defaultWorkerCount,
	}
}

// Start launches the worker pool to process messages from the queue.
// This is a blocking method that will run until the context is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	c.logger.Info().Int("count", c.workerCount).Msg("Starting worker pool")
	var wg sync.WaitGroup

	for i := 0; i < c.workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.runWorker(ctx, workerID)
		}(i + 1)
	}

	wg.Wait()
	c.logger.Info().Msg("Consumer stopped")
}

// runWorker contains the main logic for a single worker goroutine.
func (c *Consumer) runWorker(ctx context.Context, workerID int) {
	logger := c.logger.With().Int("worker_id", workerID).Logger()
	logger.Info().Msg("Worker started")

	ch, err := c.conn.Channel()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to open channel for worker")
		return
	}
	defer ch.Close()

	if err := ch.Qos(1, 0, false); err != nil {
		logger.Error().Err(err).Msg("Failed to set QoS")
		return
	}

	msgs, err := ch.Consume(
		rabbitmq.NotificationsQueue,
		fmt.Sprintf("worker-%d", workerID), // A unique consumer tag.
		false,                              // autoAck: false. We will manually acknowledge messages.
		false,                              // exclusive
		false,                              // noLocal
		false,                              // noWait
		nil,                                // args
	)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to register a consumer")
		return
	}

	logger.Info().Msg("Worker is waiting for messages")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Worker stopping due to context cancellation")
			return
		case msg, ok := <-msgs:
			if !ok {
				logger.Warn().Msg("Message channel closed by RabbitMQ, worker stopping")
				return
			}
			c.handleMessage(ctx, msg, logger)
		}
	}
}

// handleMessage processes a single message from the queue.
func (c *Consumer) handleMessage(ctx context.Context, msg amqp.Delivery, logger zerolog.Logger) {
	var notification model.Notification
	if err := json.Unmarshal(msg.Body, &notification); err != nil {
		logger.Error().Err(err).Msg("Failed to unmarshal message, rejecting")
		_ = msg.Nack(false, false)
		return
	}

	log := logger.With().Stringer("notification_id", notification.ID).Logger()

	latest, err := c.service.GetNotificationByID(ctx, notification.ID)
	if err != nil || latest.Status != model.StatusScheduled {
		status := "unknown"
		if latest != nil {
			status = string(latest.Status)
		}
		log.Warn().Str("status", status).Msg("Notification is no longer scheduled, skipping")
		_ = msg.Ack(false)
		return
	}

	log.Info().Int("attempt", notification.Attempts+1).Msg("Processing notification")
	err = c.notifier.Send(ctx, &notification)
	if err != nil {
		c.handleSendError(ctx, &notification, err, msg, log)
		return
	}

	log.Info().Msg("Notification sent successfully")
	notification.Status = model.StatusSent
	now := time.Now().UTC()
	notification.SentAt = &now
	if err := c.service.UpdateNotification(ctx, &notification); err != nil {
		log.Error().Err(err).Msg("CRITICAL: failed to update notification status to 'sent' after successful send")
		_ = msg.Nack(false, true)
		return
	}
	_ = msg.Ack(false)
}

// handleSendError encapsulates the logic for processing failed sends.
func (c *Consumer) handleSendError(ctx context.Context, n *model.Notification, sendErr error, msg amqp.Delivery, log zerolog.Logger) {
	n.Attempts++

	if n.Attempts >= maxRetries {
		log.Error().Err(sendErr).Int("attempts", n.Attempts).Msg("Max retries reached, failing notification")
		n.Status = model.StatusFailed
		if err := c.service.UpdateNotification(ctx, n); err != nil {
			log.Error().Err(err).Msg("CRITICAL: failed to update notification status to 'failed'")
			_ = msg.Nack(false, true) // Requeue.
			return
		}
		_ = msg.Ack(false)
		return
	}

	backoffDuration := calculateExponentialBackoff(n.Attempts)
	log.Warn().
		Err(sendErr).
		Int("attempt", n.Attempts).
		Dur("backoff", backoffDuration).
		Msg("Send failed, scheduling retry")

	if err := c.queue.PublishRetry(ctx, n, backoffDuration); err != nil {
		log.Error().Err(err).Msg("CRITICAL: failed to publish message to retry queue")
		_ = msg.Nack(false, true)
		return
	}

	_ = msg.Ack(false)
}

// calculateExponentialBackoff implements the exponential backoff strategy.
// Formula: 5s * 2^(attempt)
func calculateExponentialBackoff(attempt int) time.Duration {
	baseDelay := 5.0
	delay := baseDelay * math.Pow(2, float64(attempt))
	return time.Duration(delay) * time.Second
}
