package notifiers

import (
	"context"
	"fmt"
	"github.com/ilindan-dev/delayed-notifier/internal/config"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	"github.com/rs/zerolog"
)

// Dispatcher is a composite notifier that routes notifications to the correct channel-specific notifier.
// It implements the Notifier interface itself.
type Dispatcher struct {
	notifiers map[model.Channel]Notifier
	logger    zerolog.Logger
}

// NewDispatcher creates a new Dispatcher and initializes channel-specific notifiers
// based on the application's configuration mode.
func NewDispatcher(cfg *config.Config, logger *zerolog.Logger) (*Dispatcher, error) {
	log := logger.With().Str("component", "dispatcher").Logger()
	log.Info().Str("mode", cfg.Notifiers.Mode).Msg("initializing notifiers")

	notifiersMap := make(map[model.Channel]Notifier)
	// Create the LogNotifier once to use as a fallback.
	logNotifier := NewLogNotifier(logger)

	// Set LogNotifier as the default for all channels.
	notifiersMap[model.ChannelEmail] = logNotifier
	notifiersMap[model.ChannelTelegram] = logNotifier

	// If in "production" mode, try to override the defaults with real notifiers.
	if cfg.Notifiers.Mode == "production" {
		if cfg.Notifiers.Email.Host != "" {
			notifiersMap[model.ChannelEmail] = NewEmailNotifier(cfg.Notifiers.Email, logger)
			log.Info().Msg("email notifier enabled")
		}
		if cfg.Notifiers.Telegram.BotToken != "" {
			tgNotifier, err := NewTelegramNotifier(cfg.Notifiers.Telegram, logger)
			if err != nil {
				return nil, fmt.Errorf("failed to initialize telegram notifier: %w", err)
			}
			notifiersMap[model.ChannelTelegram] = tgNotifier
			log.Info().Msg("telegram notifier enabled")
		}
	}

	return &Dispatcher{
		notifiers: notifiersMap,
		logger:    log,
	}, nil
}

// Send implements the Notifier interface. It finds the correct notifier for the
// notification's channel and delegates the send operation to it.
func (d *Dispatcher) Send(ctx context.Context, n *model.Notification) error {
	notifier, ok := d.notifiers[n.Channel]
	if !ok {
		d.logger.Error().Str("channel", string(n.Channel)).Msg("no notifier found for channel")
		return fmt.Errorf("notifier for channel %s not found", n.Channel)
	}

	return notifier.Send(ctx, n)
}
