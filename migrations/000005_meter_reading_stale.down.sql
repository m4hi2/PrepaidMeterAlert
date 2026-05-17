ALTER TABLE meters
    DROP COLUMN IF EXISTS reading_stale_notified,
    DROP COLUMN IF EXISTS provider_reading_at;
