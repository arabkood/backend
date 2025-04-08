-- +goose Up



CREATE SCHEMA IF NOT EXISTS auth;

CREATE SCHEMA IF NOT EXISTS users;

CREATE SCHEMA IF NOT EXISTS class;



-- +goose Down



DROP SCHEMA IF EXISTS auth CASCADE;

DROP SCHEMA IF EXISTS users CASCADE;

DROP SCHEMA IF EXISTS class CASCADE;

-- +goose Up



-- +goose statementbegin

CREATE OR REPLACE FUNCTION auto_updated_at()

RETURNS TRIGGER AS $$

BEGIN

    NEW.updated_at = now();

    RETURN NEW;

END;

$$ LANGUAGE plpgsql;

-- +goose statementend



-- +goose Down

DROP FUNCTION IF EXISTS auto_updated_at();

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

    date timestamptz NOT NULL,

    xp_earned bigint NOT NULL DEFAULT 0,

    items_completed int NOT NULL DEFAULT 0,

    PRIMARY KEY (user_id, date)

);



-- +goose Down

DROP TABLE IF EXISTS users.stats;

DROP TABLE IF EXISTS users.daily_stats;

-- +goose Up



CREATE TABLE IF NOT EXISTS class.tracks

(

    id uuid NOT NULL DEFAULT gen_random_uuid(),

    title character varying(255) NOT NULL,

    slug character varying(100) NOT NULL,

    description text,

    logo text,

    programming_languages text[],

    difficulty text DEFAULT 'beginner', -- novice, beginner, intermediate, advanced

    premium_only boolean NOT NULL DEFAULT false,

    skills text[],

    tags text[],

    total_xp integer NOT NULL DEFAULT 0,

    total_modules integer NOT NULL DEFAULT 0,

    estimated_hours integer,

    students integer NOT NULL DEFAULT 0,

    created_at timestamp with time zone NOT NULL DEFAULT NOW(),

    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),

    deleted_at timestamp with time zone,

    outcomes text[],

    requirements text[],

    PRIMARY KEY (id),

    CONSTRAINT slug UNIQUE (slug)

);



-- +goose Down



DROP TABLE IF EXISTS class.tracks;

-- +goose Up

--

CREATE TABLE IF NOT EXISTS class.tracks_sections

(

    id uuid NOT NULL DEFAULT gen_random_uuid(),

    track_id uuid NOT NULL REFERENCES class.tracks (id) ON DELETE CASCADE,

    title character varying(255) NOT NULL,

    description text NOT NULL,

    order_number integer NOT NULL,

    created_at timestamp with time zone NOT NULL DEFAULT NOW(),

    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),

    deleted_at timestamp with time zone,



    PRIMARY KEY (id)

);



CREATE TABLE IF NOT EXISTS class.modules

(

    id uuid NOT NULL DEFAULT gen_random_uuid(),

    slug character varying(100) NOT NULL UNIQUE,

    track_id uuid NOT NULL,

    title character varying(255) NOT NULL,

    description text,

    order_number integer,

    created_at timestamp with time zone NOT NULL DEFAULT NOW(),

    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),

    xp_reward integer NOT NULL DEFAULT 0,

    estimated_minutes integer,

    difficulty text,

    dependencies uuid [],

    premium_only boolean NOT NULL DEFAULT false,

    section_id uuid REFERENCES class.tracks_sections (id) ON DELETE CASCADE,



    type text NOT NULL, 

    source text NOT NULL,



    deleted_at timestamp with time zone,



    PRIMARY KEY (id),

    CONSTRAINT fk_track_id FOREIGN KEY (track_id) REFERENCES class.tracks (id) ON DELETE CASCADE

);





-- +goose Down





DROP TABLE IF EXISTS class.tracks_sections;

DROP TABLE IF EXISTS class.modules;

-- +goose Up



CREATE TABLE IF NOT EXISTS class.submissions (

    id uuid NOT NULL DEFAULT gen_random_uuid(),

    user_id uuid NOT NULL,

    module_id uuid NOT NULL,

    user_files JSONB,

    args JSONB,     

    status TEXT NOT NULL,

    attempts INTEGER NOT NULL DEFAULT 1,

    results JSONB,     

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    

    UNIQUE (module_id, user_id),

    PRIMARY KEY (id),

    FOREIGN KEY (module_id) REFERENCES class.modules(id),

    FOREIGN KEY (user_id) REFERENCES auth.users(id)

);



-- +goose Down



DROP TABLE IF EXISTS class.submissions;

-- +goose Up



CREATE TABLE IF NOT EXISTS class.user_tracks

(

    user_id uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,

    track_id uuid NOT NULL REFERENCES class.tracks (id) ON DELETE CASCADE,

    completed_modules integer NOT NULL DEFAULT 0,

    last_activity_at timestamp with time zone NOT NULL DEFAULT NOW(),

    started_at timestamp with time zone NOT NULL DEFAULT NOW(),

    completed_at timestamp with time zone,

    PRIMARY KEY (user_id, track_id)

);





CREATE TABLE IF NOT EXISTS class.user_modules_attempts

(

    user_id uuid NOT NULL REFERENCES auth.users (id) ON DELETE CASCADE,

    module_id uuid NOT NULL REFERENCES class.modules (id) ON DELETE CASCADE,

    status text NOT NULL DEFAULT 'inprogress',

    attempts integer NOT NULL,

    last_attempt_at timestamp with time zone NOT NULL DEFAULT NOW(),

    completed_at timestamp with time zone,

    user_input jsonb,

    server_output jsonb,

    PRIMARY KEY (user_id, module_id)

);



-- +goose Down

DROP TABLE IF EXISTS class.user_tracks;

-- +goose Up







CREATE SCHEMA IF NOT EXISTS auth;



CREATE SCHEMA IF NOT EXISTS users;



CREATE SCHEMA IF NOT EXISTS class;







-- +goose Down







DROP SCHEMA IF EXISTS auth CASCADE;



DROP SCHEMA IF EXISTS users CASCADE;



DROP SCHEMA IF EXISTS class CASCADE;



-- +goose Up







-- +goose statementbegin



CREATE OR REPLACE FUNCTION auto_updated_at()



RETURNS TRIGGER AS $$



BEGIN



    NEW.updated_at = now();



    RETURN NEW;



END;



$$ LANGUAGE plpgsql;



-- +goose statementend







-- +goose Down



DROP FUNCTION IF EXISTS auto_updated_at();



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



    date timestamptz NOT NULL,



    xp_earned bigint NOT NULL DEFAULT 0,



    items_completed int NOT NULL DEFAULT 0,



    PRIMARY KEY (user_id, date)



);







-- +goose Down



DROP TABLE IF EXISTS users.stats;



DROP TABLE IF EXISTS users.daily_stat
