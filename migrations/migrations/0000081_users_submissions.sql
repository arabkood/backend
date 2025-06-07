-- +goose Up

-- One successful submission per item/user
CREATE TABLE IF NOT EXISTS users.submission (
    id UUID NOT NULL DEFAULT GEN_RANDOM_UUID() PRIMARY KEY,

    user_id UUID NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES class.items (id) ON DELETE CASCADE,

    status TEXT NOT NULL DEFAULT 'wait', -- 'wait', 'pass', 'fail', 'error'
    xp_reward INTEGER NOT NULL DEFAULT 0,
    attempts INTEGER NOT NULL DEFAULT 1,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    metadata JSONB, -- for custom metadata, e.g. arguments for running code
    data JSONB, -- for user submitted data
    results JSONB, -- results from user submission, e.g. unit tests

    UNIQUE (item_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS users.submission;
