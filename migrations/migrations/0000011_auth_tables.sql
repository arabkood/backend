-- +goose Up

---------
-- table: users
---------
CREATE TABLE auth.users (
    -- Core Identity
    id uuid PRIMARY KEY NOT NULL,
    email varchar(254) UNIQUE NOT NULL,
    username varchar(30) UNIQUE NOT NULL,
    role varchar(255) NOT NULL DEFAULT 'user',
    -- Authentication
    encrypted_password text NOT NULL,
    email_verified boolean NOT NULL DEFAULT false,
    email_verified_at timestamptz NULL,
    -- Timestamps
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX users_active_email_lower_idx
ON auth.users (lower(email));

CREATE INDEX users_active_username_lower_idx
ON auth.users (lower(username));

CREATE TRIGGER users_auto_updated_at
BEFORE UPDATE ON auth.users
FOR EACH ROW
EXECUTE FUNCTION auto_updated_at();

---------
-- table: session_tokens
---------
CREATE TABLE auth.session_tokens (
    token text PRIMARY KEY CHECK (char_length(token) > 0),
    user_id uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    last_used_at timestamptz NULL
);

---------
-- table: one_time_tokens
---------
CREATE TYPE auth.one_time_token_type AS ENUM (
    'email_confirmation',
    'email_change',
    'password_change',
    'password_recovery'
);

CREATE TABLE auth.one_time_tokens (
    user_id uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,
    type auth.one_time_token_type NOT NULL,
    token text NOT NULL CHECK (char_length(token) > 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    metadata jsonb NULL, -- optional metadata
    PRIMARY KEY (user_id, type)
);

CREATE TRIGGER one_time_tokens_auto_updated_at
BEFORE UPDATE ON auth.one_time_tokens
FOR EACH ROW
EXECUTE FUNCTION auto_updated_at();

---------
-- table: audit_logs
---------
CREATE TYPE auth.audit_log_type AS ENUM (
    'signup',
    'signin',
    'signout',
    'password_change',
    'email_change',
    'session_token'
);
CREATE TABLE auth.audit_logs (
    id bigserial PRIMARY KEY,
    user_id uuid NULL REFERENCES auth.users (id) ON DELETE SET NULL,
    type auth.audit_log_type NOT NULL,
    ip_address inet NULL,
    user_agent text NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    metadata jsonb NULL -- optional additional info
);
CREATE INDEX idx_audit_logs_user_time
ON auth.audit_logs (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS auth.audit_logs;
DROP TYPE IF EXISTS auth.audit_log_type;

DROP TABLE IF EXISTS auth.one_time_tokens;
DROP TYPE IF EXISTS auth.one_time_token_type;

DROP TABLE IF EXISTS auth.session_tokens;

DROP TABLE IF EXISTS auth.users;
