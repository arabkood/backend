-- +goose Up

---------
-- table: stats
---------
CREATE TABLE users.stats (
    user_id uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    total_xp bigint NOT NULL DEFAULT 0,
    completed_items int NOT NULL DEFAULT 0,
    longest_streak int NOT NULL DEFAULT 0,
    last_active_at timestamptz NULL,
    PRIMARY KEY (user_id)
);

---------
-- table: daily_stats
---------
CREATE TABLE users.daily_stats (
    user_id uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    date DATE NOT NULL DEFAULT CURRENT_DATE,
    xp_earned bigint NOT NULL DEFAULT 0,
    items_completed int NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, date)
);

-- +goose Down
DROP TABLE IF EXISTS users.stats;
DROP TABLE IF EXISTS users.daily_stats;
