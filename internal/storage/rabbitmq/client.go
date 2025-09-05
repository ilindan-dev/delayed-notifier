package rabbitmq

import (
	"fmt"
	"github.com/ilindan-dev/delayed-notifier/internal/config"
	amqp "github.com/rabbitmq/amqp091-go"
)

// NewConnection creates and returns a raw amqp.Connection.
// This single connection will be shared across the application (producer and consumer).
func NewConnection(cfg *config.Config) (*amqp.Connection, error) {
	conn, err := amqp.Dial(cfg.RabbitMQ.DSN)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq: failed to connect: %w", err)
	}
	return conn, nil
}
