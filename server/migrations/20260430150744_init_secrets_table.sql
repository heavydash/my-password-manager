-- +goose Up
CREATE TABLE IF NOT EXISTS secrets (
    id            TEXT PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type          TEXT NOT NULL CHECK (type IN ('password', 'note', 'card', 'ssh_key', 'custom')),
    title         TEXT NOT NULL,
    data          TEXT NOT NULL,
    metadata      TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );


CREATE INDEX IF NOT EXISTS idx_secrets_user_id ON secrets(user_id);
CREATE INDEX IF NOT EXISTS idx_secrets_created_at ON secrets(created_at DESC);

COMMENT ON TABLE secrets IS 'Encrypted user secrets (passwords, notes, cards, SSH keys, etc.)';

-- +goose Down
DROP TABLE IF EXISTS secrets;
