CREATE TABLE IF NOT EXISTS feedbacks (
    id          UUID        PRIMARY KEY DEFAULT uuidv7(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMPTZ,
    platform    VARCHAR(10) NOT NULL,
    platform_id VARCHAR(20) NOT NULL,
    text        TEXT        NOT NULL
);
CREATE INDEX IF NOT EXISTS feedbacks_platform_id_idx ON feedbacks (platform_id);
CREATE INDEX IF NOT EXISTS feedbacks_created_at_idx  ON feedbacks USING BRIN (created_at);
