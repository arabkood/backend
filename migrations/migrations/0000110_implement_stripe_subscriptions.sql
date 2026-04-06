-- +goose Up

-- Remove old Premium implementation
ALTER TABLE auth.users
DROP COLUMN IF EXISTS premium_active,
DROP COLUMN IF EXISTS polar_last_synced_at,
DROP COLUMN IF EXISTS polar_customer_id,
DROP COLUMN IF EXISTS polar_subscription_ids;

-- Plan types
CREATE TYPE auth.plan_type AS ENUM ('free', 'pro', 'past_due');
CREATE TYPE auth.plan_interval AS ENUM ('monthly', 'yearly');

---------
-- table: user_subscriptions
---------
CREATE TABLE auth.user_subscriptions (
    user_id                uuid PRIMARY KEY REFERENCES auth.users (id) ON DELETE CASCADE,
    stripe_customer_id     text UNIQUE NOT NULL,
    stripe_subscription_id text UNIQUE NULL,
    plan                   auth.plan_type NOT NULL DEFAULT 'free',
    plan_interval          auth.plan_interval NULL,
    pro_until              timestamptz NULL,
    cancel_at_period_end   boolean NOT NULL DEFAULT false,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_subscriptions_stripe_customer
ON auth.user_subscriptions (stripe_customer_id);

CREATE TRIGGER user_subscriptions_auto_updated_at
BEFORE UPDATE ON auth.user_subscriptions
FOR EACH ROW
EXECUTE FUNCTION auto_updated_at();

-- +goose Down
DROP TABLE IF EXISTS auth.user_subscriptions;
DROP TYPE IF EXISTS auth.plan_interval;
DROP TYPE IF EXISTS auth.plan_type;

ALTER TABLE auth.users
ADD COLUMN premium_active boolean NOT NULL DEFAULT false,
ADD COLUMN polar_last_synced_at timestamptz NULL,
ADD COLUMN polar_customer_id uuid NULL,
ADD COLUMN polar_subscription_ids uuid[] NULL;
