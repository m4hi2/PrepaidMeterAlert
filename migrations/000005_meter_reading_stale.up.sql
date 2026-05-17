ALTER TABLE meters
    ADD COLUMN IF NOT EXISTS provider_reading_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reading_stale_notified BOOLEAN NOT NULL DEFAULT false;
