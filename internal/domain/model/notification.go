package model

import (
	"github.com/google/uuid"
	"time"
)

// Channel represents the notification delivery channel (e.g., email, telegram).
type Channel string

const (
	ChannelEmail    Channel = "email"
	ChannelTelegram Channel = "telegram"
)

// Status represents the current state of a notification.
type Status string

const (
	StatusScheduled Status = "scheduled" // The notification is scheduled for a future time.
	StatusSent      Status = "sent"      // The notification has been successfully sent.
	StatusFailed    Status = "failed"    // The notification failed to send after all retry attempts.
	StatusCancelled Status = "cancelled" // The notification was cancelled by a user request.
)

// EmailDetails contains recipient information specific to the email channel.
type EmailDetails struct {
	To string // The recipient's email address.
}

// TelegramDetails contains recipient information specific to the telegram channel.
type TelegramDetails struct {
	ChatID int64 // The recipient's Telegram Chat ID.
}

// Notification is the core business entity of the application.
// It is technology-agnostic and does not contain any DB or JSON tags.
type Notification struct {
	ID       uuid.UUID
	Subject  string // The subject or title of the notification.
	Message  string // The main content/body of the notification.
	Channel  Channel
	Status   Status
	Attempts int
	AuthorID *string // Optional: ID of the user or system that created the notification.

	// Recipient details are mutually exclusive based on the Channel.
	Email    *EmailDetails
	Telegram *TelegramDetails

	ScheduledAt time.Time
	SentAt      *time.Time // Pointer to allow null value.
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewEmailNotification is a factory function to create a new notification for the email channel.
func NewEmailNotification(recipientEmail, subject, message string, scheduledAt time.Time, authorID *string) *Notification {
	return &Notification{
		ID:          uuid.New(),
		Subject:     subject,
		Message:     message,
		Channel:     ChannelEmail,
		Status:      StatusScheduled,
		Attempts:    0,
		AuthorID:    authorID,
		Email:       &EmailDetails{To: recipientEmail},
		ScheduledAt: scheduledAt,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

// NewTelegramNotification is a factory function to create a new notification for the telegram channel.
func NewTelegramNotification(chatID int64, subject, message string, scheduledAt time.Time, authorID *string) *Notification {
	return &Notification{
		ID:          uuid.New(),
		Subject:     subject,
		Message:     message,
		Channel:     ChannelTelegram,
		Status:      StatusScheduled,
		Attempts:    0,
		AuthorID:    authorID,
		Telegram:    &TelegramDetails{ChatID: chatID},
		ScheduledAt: scheduledAt,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}
