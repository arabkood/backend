-- +goose Up

CREATE TABLE IF NOT EXISTS users.track (
    user_id UUID NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    track_id UUID NOT NULL REFERENCES class.tracks (id) ON DELETE CASCADE,

    completed_modules INTEGER NOT NULL DEFAULT 0,

    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_activity_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (user_id, track_id)
);

-- One current attempt per module per user
CREATE TABLE IF NOT EXISTS users.modules_attempt (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,

    user_id UUID NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES class.modules (id) ON DELETE CASCADE,

    status TEXT NOT NULL DEFAULT 'wait', -- wait, fail, error, pass
    attempts INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    user_files JSONB,
    args JSONB,
    results JSONB,

    UNIQUE (module_id, user_id)
);

-- One successful submission per module/user
CREATE TABLE IF NOT EXISTS users.modules_submission (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,

    user_id UUID NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    module_id UUID NOT NULL REFERENCES class.modules (id) ON DELETE CASCADE,

    xp_reward INTEGER NOT NULL DEFAULT 0,
    attempts INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    user_files JSONB,
    args JSONB,
    results JSONB,

    UNIQUE (module_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS users.user_tracks;
DROP TABLE IF EXISTS users.user_modules_submission;
DROP TABLE IF EXISTS users.user_modules_attempt;
