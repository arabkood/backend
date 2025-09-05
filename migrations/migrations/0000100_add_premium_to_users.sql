-- +goose Up
ALTER TABLE auth.users
ADD COLUMN premium_active boolean NOT NULL DEFAULT false,
ADD COLUMN polar_last_synced_at timestamptz NULL,
ADD COLUMN polar_customer_id uuid NULL,
ADD COLUMN polar_subscription_ids uuid[] NULL;

-- +goose Down
ALTER TABLE auth.users
DROP COLUMN IF EXISTS premium_active,
DROP COLUMN IF EXISTS polar_last_synced_at,
DROP COLUMN IF EXISTS polar_customer_id,
DROP COLUMN IF EXISTS polar_subscription_ids;
