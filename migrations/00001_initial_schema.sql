-- +goose Up
-- This migration sets up the initial database schema for the notifier service.

-- Step 1: Enable necessary extensions.
-- pgcrypto is used for the gen_random_uuid() function.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Step 2: Define custom types for data integrity.
-- Using ENUMs prevents invalid values from being inserted into these columns.
CREATE TYPE channel_type AS ENUM ('email', 'telegram');
CREATE TYPE notification_status AS ENUM ('scheduled', 'sent', 'failed', 'cancelled');

-- Step 3: Create the main "parent" partitioned table.
-- The table is partitioned by `scheduled_at` for efficient time-based queries.
-- The actual data will live in the child partition tables.
CREATE TABLE notifications (
                               id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                               subject TEXT NOT NULL,
                               message TEXT NOT NULL,
                               author_id TEXT,

    -- Recipient-specific fields
                               email_to TEXT,
                               telegram_chat_id BIGINT,

    -- Core notification fields
                               channel channel_type NOT NULL,
                               status notification_status NOT NULL DEFAULT 'scheduled',
                               attempts SMALLINT NOT NULL DEFAULT 0,

    -- Timestamps
                               scheduled_at TIMESTAMPTZ NOT NULL,
                               sent_at TIMESTAMPTZ,
                               created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
                               updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- This CHECK constraint ensures that recipient data matches the selected channel.
                               CONSTRAINT chk_recipient_channel
                                   CHECK (
                                       (channel = 'email' AND email_to IS NOT NULL AND telegram_chat_id IS NULL) OR
                                       (channel = 'telegram' AND telegram_chat_id IS NOT NULL AND email_to IS NULL)
                                       )
) PARTITION BY RANGE (scheduled_at);

-- Step 4: Create the first partition(s) manually.
-- This makes the table immediately usable after migration.
-- Subsequent partitions will be created automatically by pg_partman or a cron job.
CREATE TABLE notifications_2025_09 PARTITION OF notifications
    FOR VALUES FROM ('2025-09-01 00:00:00Z') TO ('2025-10-01 00:00:00Z');

-- Step 5: Create indexes for performance.
-- This index is critical for the worker to find jobs to process quickly.
CREATE INDEX idx_notifications_status_scheduled_at ON notifications (status, scheduled_at);

-- Step 6: Create a trigger to automatically update the 'updated_at' column.
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_timestamp
    BEFORE UPDATE ON notifications
    FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

-- +goose Down
-- Reverses the setup in the opposite order of creation.
DROP TABLE IF EXISTS notifications;
DROP TRIGGER IF EXISTS set_timestamp ON notifications;
DROP FUNCTION IF EXISTS trigger_set_timestamp();
DROP TYPE IF EXISTS notification_status;
DROP TYPE IF EXISTS channel_type;
DROP EXTENSION IF EXISTS "pgcrypto";