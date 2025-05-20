-- +goose Up

CREATE TABLE IF NOT EXISTS users.track (
    user_id UUID NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    track_id UUID NOT NULL REFERENCES class.tracks (id) ON DELETE CASCADE,

    completed_items INTEGER NOT NULL DEFAULT 0,

    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (user_id, track_id)
);

-- One current attempt per item per user
CREATE TABLE IF NOT EXISTS users.code_attempt (
    id UUID NOT NULL DEFAULT GEN_RANDOM_UUID() PRIMARY KEY,

    user_id UUID NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES class.items (id) ON DELETE CASCADE,

    status TEXT NOT NULL DEFAULT 'wait', -- wait, fail, error, pass
    attempts INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    user_files JSONB,
    args JSONB,
    results JSONB,

    UNIQUE (item_id, user_id)
);

-- One successful submission per item/user
CREATE TABLE IF NOT EXISTS users.code_submission (
    id UUID NOT NULL DEFAULT GEN_RANDOM_UUID() PRIMARY KEY,

    user_id UUID NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES class.items (id) ON DELETE CASCADE,

    xp_reward INTEGER NOT NULL DEFAULT 0,
    attempts INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    user_files JSONB,
    args JSONB,
    results JSONB,

    UNIQUE (item_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS users.user_tracks;
DROP TABLE IF EXISTS users.user_items_submission;
DROP TABLE IF EXISTS users.user_items_attempt;
