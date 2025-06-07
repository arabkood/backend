-- +goose Up

CREATE TABLE IF NOT EXISTS users.xp_events (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    xp_amount INT NOT NULL,

    source_type VARCHAR(50) NOT NULL,
    source_id UUID NOT NULL,

    created_at TIMESTAMPTZ DEFAULT NOW()
);


-- +goose Down
DROP TABLE IF EXISTS users.xp_events;
