package http

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	repo "github.com/ilindan-dev/delayed-notifier/internal/domain/repository"
	"github.com/ilindan-dev/delayed-notifier/internal/service"
	"github.com/rs/zerolog"
	"net/http"
)

type Handlers struct {
	service *service.NotificationService
	logger  zerolog.Logger
}

// NewHandlers creates a new instance of Handlers.
func NewHandlers(service *service.NotificationService, logger *zerolog.Logger) *Handlers {
	return &Handlers{
		service: service,
		logger:  logger.With().Str("layer", "http_handler").Logger(),
	}
}

// RegisterRoutes sets up the routing for the notification API.
func (h *Handlers) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.POST("/notifications", h.CreateNotification)
		api.GET("/notifications/:id", h.GetNotificationByID)
		api.DELETE("/notifications/:id", h.CancelNotification)
	}
}

// CreateNotification handles the HTTP request for creating a new notification.
func (h *Handlers) CreateNotification(c *gin.Context) {
	var req CreateNotificationRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn().Err(err).Msg("invalid request body")
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	notification, err := h.service.CreateNotification(
		c.Request.Context(),
		req.Recipient,
		model.Channel(req.Channel),
		req.Subject,
		req.Message,
		req.ScheduledAt,
		req.AuthorID,
	)
	if err != nil {
		if errors.Is(err, repo.ErrDuplicateRecord) {
			c.JSON(http.StatusConflict, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error().Err(err).Msg("failed to create notification")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create notification"})
		return
	}

	c.JSON(http.StatusCreated, toNotificationResponse(notification))
}

// GetNotificationByID handles the HTTP request to retrieve a notification.
func (h *Handlers) GetNotificationByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid notification ID format"})
		return
	}

	notification, err := h.service.GetNotificationByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}
		h.logger.Error().Err(err).Stringer("id", id).Msg("failed to get notification by id")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to retrieve notification"})
		return
	}

	c.JSON(http.StatusOK, toNotificationResponse(notification))
}

// CancelNotification handles the HTTP request to cancel a notification.
func (h *Handlers) CancelNotification(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid notification ID format"})
		return
	}

	err = h.service.CancelNotification(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
			return
		}

		h.logger.Error().Err(err).Stringer("id", id).Msg("failed to cancel notification")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to cancel notification"})
		return
	}

	c.Status(http.StatusNoContent)
}

// toNotificationResponse is a helper function to map the domain model to the DTO.
func toNotificationResponse(n *model.Notification) NotificationResponse {
	return NotificationResponse{
		ID:          n.ID,
		Status:      string(n.Status),
		Channel:     string(n.Channel),
		Subject:     n.Subject,
		ScheduledAt: n.ScheduledAt,
		CreatedAt:   n.CreatedAt,
	}
}
