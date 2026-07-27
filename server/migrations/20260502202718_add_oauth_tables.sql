-- +goose Up
CREATE TABLE IF NOT EXISTS oauth_states (
    state          VARCHAR(64) PRIMARY KEY,
    provider       VARCHAR(20)  NOT NULL CHECK (provider IN ('google', 'yandex')),
    one_time_code  VARCHAR(64),
    user_id        VARCHAR(255),
    expires_at     TIMESTAMP    NOT NULL,
    created_at     TIMESTAMP    DEFAULT NOW()
    );

CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states (expires_at);

-- +goose Down
DROP TABLE IF EXISTS oauth_states;
