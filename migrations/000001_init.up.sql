-- From the ashes of defeat, knowledge rises

CREATE TABLE IF NOT EXISTS users (
    id          UUID        PRIMARY KEY DEFAULT uuidv7(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMPTZ,
    platform    VARCHAR(10) NOT NULL CHECK (platform IN ('telegram')),
    platform_id VARCHAR(20) NOT NULL
);

CREATE INDEX IF NOT EXISTS users_platform_id_idx ON users (platform_id);
CREATE INDEX IF NOT EXISTS users_created_at_idx  ON users USING BRIN (created_at);
CREATE INDEX IF NOT EXISTS users_updated_at_idx  ON users USING BRIN (updated_at);
CREATE INDEX IF NOT EXISTS users_deleted_at_idx  ON users USING BRIN (deleted_at);

-- -----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS providers (
    id         UUID        PRIMARY KEY DEFAULT uuidv7(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    code       VARCHAR(10) NOT NULL UNIQUE CHECK (code IN ('nesco', 'dpdc', 'desco')),
    enabled    BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS providers_created_at_idx ON providers USING BRIN (created_at);
CREATE INDEX IF NOT EXISTS providers_updated_at_idx ON providers USING BRIN (updated_at);
CREATE INDEX IF NOT EXISTS providers_deleted_at_idx ON providers USING BRIN (deleted_at);

INSERT INTO providers (code, enabled) VALUES
    ('desco', TRUE),
    ('dpdc',  FALSE),
    ('nesco', FALSE);

-- -----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS meters (
    id                  UUID        PRIMARY KEY DEFAULT uuidv7(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMPTZ,
    user_id             UUID        NOT NULL REFERENCES users(id),
    provider            VARCHAR(10) NOT NULL CHECK (provider IN ('nesco', 'dpdc', 'desco')),
    meter_number        VARCHAR(20) NOT NULL,
    account_number      VARCHAR(20) NOT NULL,
    nickname            VARCHAR(30),
    threshold           FLOAT8      NOT NULL DEFAULT 100,
    notify_mode         VARCHAR(10) NOT NULL CHECK (notify_mode IN ('single', 'daily')),
    balance             FLOAT8      NOT NULL DEFAULT 0,
    last_fetch_at       TIMESTAMPTZ,
    fetch_status        VARCHAR(10) NOT NULL CHECK (fetch_status IN ('pending', 'success', 'failed')),
    notification_status VARCHAR(10) NOT NULL CHECK (notification_status IN ('not_needed', 'pending', 'success', 'failed'))
);

CREATE INDEX IF NOT EXISTS meters_user_id_idx    ON meters (user_id);
CREATE INDEX IF NOT EXISTS meters_created_at_idx ON meters USING BRIN (created_at);
CREATE INDEX IF NOT EXISTS meters_updated_at_idx ON meters USING BRIN (updated_at);
CREATE INDEX IF NOT EXISTS meters_deleted_at_idx ON meters USING BRIN (deleted_at);

-- -----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS notification_logs (
    id          UUID        PRIMARY KEY DEFAULT uuidv7(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  TIMESTAMPTZ,
    user_id     UUID        NOT NULL REFERENCES users(id),
    meter_id    UUID        NOT NULL REFERENCES meters(id),
    platform    VARCHAR(10) NOT NULL CHECK (platform IN ('telegram')),
    platform_id VARCHAR(20) NOT NULL,
    balance     FLOAT8      NOT NULL
);

CREATE INDEX IF NOT EXISTS notification_logs_user_id_idx    ON notification_logs (user_id);
CREATE INDEX IF NOT EXISTS notification_logs_meter_id_idx   ON notification_logs (meter_id);
CREATE INDEX IF NOT EXISTS notification_logs_created_at_idx ON notification_logs USING BRIN (created_at);
CREATE INDEX IF NOT EXISTS notification_logs_updated_at_idx ON notification_logs USING BRIN (updated_at);
CREATE INDEX IF NOT EXISTS notification_logs_deleted_at_idx ON notification_logs USING BRIN (deleted_at);
