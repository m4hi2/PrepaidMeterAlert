-- From the ashes of defeat, knowledge rises

ALTER TABLE notification_logs DROP COLUMN notification_type;
ALTER TABLE meters DROP COLUMN stale_notification_status;
ALTER TABLE meters DROP COLUMN last_reading_at;