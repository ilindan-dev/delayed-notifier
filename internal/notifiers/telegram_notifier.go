package notifiers

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ilindan-dev/delayed-notifier/internal/config"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	"github.com/rs/zerolog"
)

// TelegramNotifier sends notifications via a Telegram bot.
type TelegramNotifier struct {
	bot    *tgbotapi.BotAPI
	logger zerolog.Logger
}

// NewTelegramNotifier creates a new instance of TelegramNotifier.
func NewTelegramNotifier(cfg config.TelegramConfig, logger *zerolog.Logger) (*TelegramNotifier, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot api: %w", err)
	}
	return &TelegramNotifier{
		bot:    bot,
		logger: logger.With().Str("component", "telegram_notifier").Logger(),
	}, nil
}

// Send implements the Notifier interface for Telegram.
func (n *TelegramNotifier) Send(_ context.Context, notification *model.Notification) error {
	if notification.Channel != model.ChannelTelegram || notification.Telegram == nil {
		return fmt.Errorf("invalid notification for telegram channel")
	}

	fullMessage := fmt.Sprintf("*%s*\n\n%s", notification.Subject, notification.Message)

	msg := tgbotapi.NewMessage(notification.Telegram.ChatID, fullMessage)
	msg.ParseMode = tgbotapi.ModeMarkdown

	if _, err := n.bot.Send(msg); err != nil {
		n.logger.Error().Err(err).Stringer("notification_id", notification.ID).Msg("failed to send telegram message")
		return err
	}

	n.logger.Info().Stringer("notification_id", notification.ID).Int64("chat_id", notification.Telegram.ChatID).Msg("telegram message sent successfully")
	return nil
}
