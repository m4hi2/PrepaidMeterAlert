-- From the ashes of defeat, knowledge rises

ALTER TABLE meters ADD COLUMN last_reading_at TIMESTAMPTZ;
ALTER TABLE meters ADD COLUMN stale_notification_status VARCHAR(10) NOT NULL DEFAULT 'not_needed' CHECK (stale_notification_status IN ('not_needed', 'pending', 'success', 'failed'));

ALTER TABLE notification_logs ADD COLUMN notification_type VARCHAR(20) NOT NULL DEFAULT 'low_balance' CHECK (notification_type IN ('low_balance', 'stale_reading'));