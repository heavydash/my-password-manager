-- +goose Up
CREATE TABLE IF NOT EXISTS secrets (
                                       id VARCHAR(255) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    data TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );

CREATE INDEX idx_secrets_user_id ON secrets(user_id);

-- +goose Down
DROP TABLE IF EXISTS secrets;
