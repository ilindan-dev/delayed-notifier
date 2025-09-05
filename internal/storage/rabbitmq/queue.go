package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	repo "github.com/ilindan-dev/delayed-notifier/internal/domain/repository"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"strconv"
	"time"
)

// Ensure RabbitMQQueue implements the repository interface at compile time.
var _ repo.NotificationQueue = (*RabbitMQQueue)(nil)

// Constants for our RabbitMQ topology.
const (
	WaitExchange          = "wait.exchange"
	RetryExchange         = "retry.exchange"
	NotificationsExchange = "notifications.exchange"

	NotificationsQueue = "notifications.queue.process"
	WaitQueue          = "wait.queue.delay"
	RetryQueue         = "retry.queue.delay"

	Direct = "direct"
)

// RabbitMQQueue implements the NotificationQueue interface. It acts as a PUBLISHER.
// It uses the low-level amqp091-go library directly for reliability.
type RabbitMQQueue struct {
	conn   *amqp.Connection
	ch     *amqp.Channel
	logger zerolog.Logger
}

// NewRabbitMQQueue creates a new instance of the RabbitMQQueue publisher.
// It receives a shared amqp.Connection to create its own channel.
func NewRabbitMQQueue(conn *amqp.Connection, logger *zerolog.Logger) (*RabbitMQQueue, error) {
	channel, err := conn.Channel()
	if err != nil {
		// Log the error before returning
		logger.Error().Err(err).Msg("storage: rabbitMQ: New: Failed to open a channel")
		return nil, fmt.Errorf("storage: rabbitMQ: New: Failed to open a channel: %w", err)
	}

	queue := &RabbitMQQueue{
		conn:   conn,
		ch:     channel,
		logger: logger.With().Str("component", "rabbitmq_publisher").Logger(),
	}

	if err = queue.setupTopology(); err != nil {
		// Log the error before returning
		queue.logger.Error().Err(err).Msg("storage: rabbitMQ: New: Failed to setup topology")
		return nil, fmt.Errorf("storage: rabbitMQ: New: Failed to setup topology: %w", err)
	}

	return queue, nil
}

// setupTopology declares all necessary exchanges and queues.
func (q *RabbitMQQueue) setupTopology() error {
	q.logger.Info().Msg("setting up rabbitmq topology")

	// Declare Exchanges
	exchangesToDeclare := []struct {
		name string
		kind string
	}{
		{NotificationsExchange, Direct},
		{WaitExchange, Direct},
		{RetryExchange, Direct},
	}
	for _, exInfo := range exchangesToDeclare {
		if err := q.ch.ExchangeDeclare(exInfo.name, exInfo.kind, true, false, false, false, nil); err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", exInfo.name, err)
		}
	}

	// Declare Queues
	if _, err := q.ch.QueueDeclare(NotificationsQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", NotificationsQueue, err)
	}
	waitQueueArgs := amqp.Table{"x-dead-letter-exchange": NotificationsExchange}
	if _, err := q.ch.QueueDeclare(WaitQueue, true, false, false, false, waitQueueArgs); err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", WaitQueue, err)
	}
	retryQueueArgs := amqp.Table{"x-dead-letter-exchange": NotificationsExchange}
	if _, err := q.ch.QueueDeclare(RetryQueue, true, false, false, false, retryQueueArgs); err != nil {
		return fmt.Errorf("failed to declare queue %s: %w", RetryQueue, err)
	}

	// Bind Queues
	if err := q.ch.QueueBind(NotificationsQueue, "", NotificationsExchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange %s: %w", NotificationsQueue, NotificationsExchange, err)
	}
	if err := q.ch.QueueBind(WaitQueue, "", WaitExchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange %s: %w", WaitQueue, WaitExchange, err)
	}
	if err := q.ch.QueueBind(RetryQueue, "", RetryExchange, false, nil); err != nil {
		return fmt.Errorf("failed to bind queue %s to exchange %s: %w", RetryQueue, RetryExchange, err)
	}

	q.logger.Info().Msg("rabbitmq topology setup successful")
	return nil
}

// Publish schedules a notification for delayed processing.
func (q *RabbitMQQueue) Publish(ctx context.Context, n *model.Notification) error {
	body, err := json.Marshal(n)
	if err != nil {
		q.logger.Error().Err(err).Stringer("id", n.ID).Msg("failed to marshal notification")
		return fmt.Errorf("failed to marshal notification: %w", err)
	}
	delay := time.Until(n.ScheduledAt)
	if delay < 0 {
		delay = 0
	}

	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Expiration:   strconv.FormatInt(delay.Milliseconds(), 10),
	}

	return q.ch.PublishWithContext(ctx, WaitExchange, "", false, false, msg)
}

// PublishRetry schedules a notification for a retry attempt.
func (q *RabbitMQQueue) PublishRetry(ctx context.Context, n *model.Notification, retryDelay time.Duration) error {
	body, err := json.Marshal(n)
	if err != nil {
		q.logger.Error().Err(err).Stringer("id", n.ID).Msg("failed to marshal notification for retry")
		return fmt.Errorf("failed to marshal notification for retry: %w", err)
	}

	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent,
		Expiration:   fmt.Sprintf("%d", retryDelay.Milliseconds()),
	}

	return q.ch.PublishWithContext(ctx, RetryExchange, "", false, false, msg)
}

// Close gracefully shuts down the channel. The connection is managed by Fx.
func (q *RabbitMQQueue) Close() error {
	if q.ch != nil {
		return q.ch.Close()
	}
	return nil
}
