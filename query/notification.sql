-- name: CreateNotification :one
-- This query inserts a new notification into the database.
INSERT INTO notifications (
                           subject,
                           message,
                           author_id,
                           email_to,
                           telegram_chat_id,
                           channel,
                           status,
                           attempts,
                           scheduled_at
) VALUES (
          $1, $2, $3, $4, $5, $6, $7, $8, $9
         )
RETURNING *;

-- name: GetNotificationByID :one
-- This query retrieves a single notification by its unique UUID.
SELECT * FROM notifications
WHERE id = $1;


-- name: UpdateNotificationStatus :one
-- This query updates the status, attempts count, and sent_at timestamp of a notification.
UPDATE notifications
SET
    status = $2,
    attempts = $3,
    sent_at = $4
WHERE
    id = $1
RETURNING *;

-- name: CancelNotification :one
-- This query performs a "soft delete" by changing the status to 'cancelled'.
-- We never truly delete data, we just change its state.
UPDATE notifications
SET
    status = 'cancelled'
WHERE
    id = $1
RETURNING *;