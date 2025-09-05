package http

import (
	"github.com/google/uuid"
	"time"
)

// CreateNotificationRequest defines the structure for a new notification request.
// It uses `json` tags for unmarshalling and `binding` for validation with Gin.
type CreateNotificationRequest struct {
	Recipient   string    `json:"recipient" binding:"required"`
	Channel     string    `json:"channel" binding:"required"`
	Subject     string    `json:"subject" binding:"required"`
	Message     string    `json:"message"`
	ScheduledAt time.Time `json:"scheduled_at" binding:"required"`
	AuthorID    *string   `json:"author_id,omitempty"`
}

// NotificationResponse defines the structure for a standard notification response.
// We don't expose all internal fields to the client.
type NotificationResponse struct {
	ID          uuid.UUID `json:"id"`
	Status      string    `json:"status"`
	Channel     string    `json:"channel"`
	Subject     string    `json:"subject"`
	ScheduledAt time.Time `json:"scheduled_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// ErrorResponse defines a standard structure for API error responses.
type ErrorResponse struct {
	Error string `json:"error"`
}
