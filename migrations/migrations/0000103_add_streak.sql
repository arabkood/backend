-- +goose Up
ALTER TABLE users.stats ADD COLUMN current_streak INT NOT NULL DEFAULT 0;
ALTER TABLE users.stats ADD COLUMN last_active_date DATE NOT NULL DEFAULT CURRENT_DATE;

-- +goose statementbegin
CREATE OR REPLACE FUNCTION public.handle_streak_update()
RETURNS TRIGGER AS $$
DECLARE
    user_stats RECORD;
    today DATE := NEW.date;
BEGIN
    -- Step 1: Lock the user's row in the stats table to prevent race conditions.
    SELECT * INTO user_stats FROM users.stats WHERE user_id = NEW.user_id FOR UPDATE;
    
    -- Step 2: Handle the case for a brand new user who has no stats row yet.
    IF user_stats IS NULL THEN
        INSERT INTO users.stats (user_id, current_streak, longest_streak, last_active_date)
        VALUES (NEW.user_id, 1, 1, today);
        RETURN NEW;
    END IF;
    
    -- Step 3: Core streak logic for an existing user.
    IF user_stats.last_active_date = today - INTERVAL '1 day' THEN
        UPDATE users.stats
        SET
            current_streak = user_stats.current_streak + 1,
            longest_streak = GREATEST(user_stats.longest_streak, user_stats.current_streak + 1),
            last_active_date = today
        WHERE user_id = NEW.user_id;
    ELSIF user_stats.last_active_date < today - INTERVAL '1 day' THEN
        UPDATE users.stats
        SET
            current_streak = 1,
            last_active_date = today
        WHERE user_id = NEW.user_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose statementend

-- Attaches the function to the table as a trigger.
CREATE TRIGGER trigger_update_streak_on_new_day
AFTER INSERT ON users.daily_stats
FOR EACH ROW
EXECUTE FUNCTION public.handle_streak_update();

-- +goose Down


DROP TRIGGER IF EXISTS trigger_update_streak_on_new_day ON users.daily_stats;
DROP FUNCTION IF EXISTS public.handle_streak_update();
ALTER TABLE users.stats DROP COLUMN IF EXISTS last_active_date;
ALTER TABLE users.stats DROP COLUMN IF EXISTS current_streak;


