package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/ilindan-dev/delayed-notifier/internal/domain/model"
	repo "github.com/ilindan-dev/delayed-notifier/internal/domain/repository"
	"github.com/ilindan-dev/delayed-notifier/internal/storage/postgres/db"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// Ensure NotificationRepository implements the interface
var _ repo.NotificationRepository = (*NotificationRepository)(nil)

// NotificationRepository implements the domain.repository.NotificationRepository interface
// using PostgreSQL as a backend.
type NotificationRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	logger  zerolog.Logger
}

// NewNotificationRepository creates a new instance of the NotificationRepository
func NewNotificationRepository(pool *pgxpool.Pool, logger *zerolog.Logger) *NotificationRepository {
	return &NotificationRepository{
		pool:    pool,
		queries: db.New(pool),
		logger:  logger.With().Str("layer", "postgres_repository").Logger(),
	}
}

// Save persists a new notification and returns the created object with DB-generated fields.
func (r *NotificationRepository) Save(ctx context.Context, n *model.Notification) (*model.Notification, error) {
	params, err := toDBCreateParams(n)
	if err != nil {
		r.logger.Error().Err(err).Msg("failed to map domain model to db params")
		return nil, err
	}

	createdDB, err := r.queries.CreateNotification(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return nil, repo.ErrDuplicateRecord
		}
		r.logger.Err(err).Msg("cannot create notification")
		return nil, fmt.Errorf("postgres: CreateNotification failed: %w", err)
	}

	return toDomainModel(&createdDB)
}

// GetByID retrieves a notification by its unique ID.
func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	pgUUID := pgtype.UUID{Bytes: id, Valid: true}

	dbNotification, err := r.queries.GetNotificationByID(ctx, pgUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn().Stringer("id", id).Msg("notification not found by id")
			return nil, repo.ErrNotFound
		}
		r.logger.Err(err).Str("method", "GetByID").Msg("cannot get notification")
		return nil, fmt.Errorf("postgres: GetNotificationByID failed: %w", err)
	}

	return toDomainModel(&dbNotification)
}

// Update updates the mutable fields of a notification.
func (r *NotificationRepository) Update(ctx context.Context, n *model.Notification) error {
	params, err := toDBUpdateParams(n)
	if err != nil {
		r.logger.Error().Err(err).Stringer("id", n.ID).Msg("failed to map domain model to update db params")
		return err
	}

	_, err = r.queries.UpdateNotificationStatus(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn().Stringer("id", n.ID).Msg("tried to update non-existent notification")
			return repo.ErrNotFound
		}
		r.logger.Err(err).Stringer("id", n.ID).Msg("cannot update notification")
		return fmt.Errorf("postgres: UpdateNotificationStatus failed: %w", err)
	}
	return nil
}

// Delete performs a "soft delete" on a notification by setting its status to 'cancelled'.
func (r *NotificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	pgUUID := pgtype.UUID{Bytes: id, Valid: true}
	_, err := r.queries.CancelNotification(ctx, pgUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			r.logger.Warn().Stringer("id", id).Msg("tried to cancel non-existent notification")
			return repo.ErrNotFound
		}
		r.logger.Err(err).Stringer("id", id).Msg("cannot cancel notification")
		return fmt.Errorf("postgres: CancelNotification failed: %w", err)
	}
	return nil
}

// === Mapper Functions ===

// toDBCreateParams safely converts a domain model to sqlc create parameters.
func toDBCreateParams(n *model.Notification) (db.CreateNotificationParams, error) {
	params := db.CreateNotificationParams{
		Subject:     n.Subject,
		Message:     n.Message,
		Channel:     db.ChannelType(n.Channel),
		Status:      db.NotificationStatus(n.Status),
		Attempts:    int16(n.Attempts),
		ScheduledAt: pgtype.Timestamptz{Time: n.ScheduledAt, Valid: true},
	}
	if n.AuthorID != nil {
		params.AuthorID = pgtype.Text{String: *n.AuthorID, Valid: true}
	}
	switch n.Channel {
	case model.ChannelEmail:
		if n.Email == nil || n.Email.To == "" {
			return db.CreateNotificationParams{}, errors.New("email recipient is required for email channel")
		}
		params.EmailTo = pgtype.Text{String: n.Email.To, Valid: true}
	case model.ChannelTelegram:
		if n.Telegram == nil {
			return db.CreateNotificationParams{}, errors.New("telegram recipient is required for telegram channel")
		}
		params.TelegramChatID = pgtype.Int8{Int64: n.Telegram.ChatID, Valid: true}
	default:
		return db.CreateNotificationParams{}, fmt.Errorf("unsupported channel type: %s", n.Channel)
	}
	return params, nil
}

// toDBUpdateParams converts a domain model to the sqlc-generated parameters for updating.
func toDBUpdateParams(n *model.Notification) (db.UpdateNotificationStatusParams, error) {
	params := db.UpdateNotificationStatusParams{
		ID:       pgtype.UUID{Bytes: n.ID, Valid: true},
		Status:   db.NotificationStatus(n.Status),
		Attempts: int16(n.Attempts),
	}
	if n.SentAt != nil {
		params.SentAt = pgtype.Timestamptz{Time: *n.SentAt, Valid: true}
	} else {
		params.SentAt = pgtype.Timestamptz{Valid: false}
	}
	return params, nil
}

// toDomainModel safely converts a database model to a domain model.
func toDomainModel(dbn *db.Notification) (*model.Notification, error) {
	if dbn == nil {
		return nil, errors.New("cannot convert nil db notification")
	}
	domainModel := &model.Notification{
		ID:          dbn.ID.Bytes,
		Subject:     dbn.Subject,
		Message:     dbn.Message,
		Channel:     model.Channel(dbn.Channel),
		Status:      model.Status(dbn.Status),
		Attempts:    int(dbn.Attempts),
		ScheduledAt: dbn.ScheduledAt.Time,
		CreatedAt:   dbn.CreatedAt.Time,
		UpdatedAt:   dbn.UpdatedAt.Time,
	}
	if dbn.AuthorID.Valid {
		domainModel.AuthorID = &dbn.AuthorID.String
	}
	if dbn.SentAt.Valid {
		domainModel.SentAt = &dbn.SentAt.Time
	}
	switch domainModel.Channel {
	case model.ChannelEmail:
		if dbn.EmailTo.Valid {
			domainModel.Email = &model.EmailDetails{To: dbn.EmailTo.String}
		}
	case model.ChannelTelegram:
		if dbn.TelegramChatID.Valid {
			domainModel.Telegram = &model.TelegramDetails{ChatID: dbn.TelegramChatID.Int64}
		}
	}
	return domainModel, nil
}
