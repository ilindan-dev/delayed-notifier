package notifiers

import (
	"context"
	"fmt"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	"github.com/rs/zerolog"
)

// LogNotifier is a mock notifier that implements the Notifier interface.
// It simply logs the notification details to the console instead of sending them
// through a real channel. This is extremely useful for development and testing.
type LogNotifier struct {
	logger zerolog.Logger
}

// NewLogNotifier creates a new instance of LogNotifier.
func NewLogNotifier(logger *zerolog.Logger) *LogNotifier {
	return &LogNotifier{
		logger: logger.With().Str("component", "log_notifier").Logger(),
	}
}

// Send implements the Notifier interface.
func (n *LogNotifier) Send(ctx context.Context, notification *model.Notification) error {
	var recipient string
	switch notification.Channel {
	case model.ChannelEmail:
		if notification.Email != nil {
			recipient = notification.Email.To
		}
	case model.ChannelTelegram:
		if notification.Telegram != nil {
			recipient = fmt.Sprintf("ChatID %d", notification.Telegram.ChatID)
		}
	}

	n.logger.Info().
		Stringer("notification_id", notification.ID).
		Str("channel", string(notification.Channel)).
		Str("recipient", recipient).
		Str("subject", notification.Subject).
		Msg(">>> MOCK SEND: Notification dispatched")

	return nil
}
