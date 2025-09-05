package notifiers

import (
	"context"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
)

// Notifier defines the interface for any notification sending service.
// This allows us to easily swap or add new notification channels (e.g., SMS, Slack).
type Notifier interface {
	// Send dispatches the notification.
	Send(ctx context.Context, n *model.Notification) error
}
