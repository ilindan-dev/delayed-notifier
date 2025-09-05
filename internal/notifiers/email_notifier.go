package notifiers

import (
	"context"
	"fmt"
	"github.com/ilindan-dev/delayed-notifier/internal/config"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	"github.com/rs/zerolog"
	"gopkg.in/gomail.v2"
)

// EmailNotifier sends notifications via SMTP.
type EmailNotifier struct {
	dialer *gomail.Dialer
	from   string
	logger zerolog.Logger
}

// NewEmailNotifier creates a new instance of EmailNotifier.
func NewEmailNotifier(cfg config.EmailConfig, logger *zerolog.Logger) *EmailNotifier {
	d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	return &EmailNotifier{
		dialer: d,
		from:   cfg.From,
		logger: logger.With().Str("component", "email_notifier").Logger(),
	}
}

// Send implements the Notifier interface for email.
func (n *EmailNotifier) Send(_ context.Context, notification *model.Notification) error {
	if notification.Channel != model.ChannelEmail || notification.Email == nil {
		return fmt.Errorf("invalid notification for email channel")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", n.from)
	m.SetHeader("To", notification.Email.To)
	m.SetHeader("Subject", notification.Subject)
	m.SetBody("text/plain", notification.Message)

	// DialAndSend opens a connection, sends the email, and closes it.
	if err := n.dialer.DialAndSend(m); err != nil {
		n.logger.Error().Err(err).Stringer("notification_id", notification.ID).Msg("failed to send email")
		return err
	}

	n.logger.Info().Stringer("notification_id", notification.ID).Str("recipient", notification.Email.To).Msg("email sent successfully")
	return nil
}
