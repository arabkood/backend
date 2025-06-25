-- +goose Up

CREATE TABLE IF NOT EXISTS class.topics
(
    id uuid NOT NULL DEFAULT GEN_RANDOM_UUID(),
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),

    title text NOT NULL,
    blurb text,
    logo text,
    hash text,


    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS class.tracks
(
    topic_id uuid NOT NULL REFERENCES class.topics (id) ON DELETE CASCADE,
    id uuid NOT NULL DEFAULT GEN_RANDOM_UUID(),
    slug text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),

    title text NOT NULL,
    blurb text,
    logo text,
    hash text,

    premium_only boolean DEFAULT false,

    PRIMARY KEY (id),
    CONSTRAINT slug UNIQUE (slug)
);

CREATE TABLE IF NOT EXISTS class.modules
(
    track_id uuid NOT NULL REFERENCES class.tracks (id) ON DELETE CASCADE,
    id uuid NOT NULL DEFAULT GEN_RANDOM_UUID(),
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    hash text,

    title text NOT NULL,
    position integer,
    premium_only boolean DEFAULT false,

    PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS class.items
(
    module_id uuid NOT NULL REFERENCES class.modules (id) ON DELETE CASCADE,
    id uuid NOT NULL DEFAULT GEN_RANDOM_UUID(),
    slug text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp with time zone NOT NULL DEFAULT NOW(),
    hash text,

    type text,
    position integer,
    title text NOT NULL,
    blurb text,
    difficulty text,
    premium_only boolean DEFAULT false,
    base_xp integer DEFAULT 1,
    s3_path text,

    PRIMARY KEY (id)
);


-- +goose Down

DROP TABLE IF EXISTS class.items CASCADE;
DROP TABLE IF EXISTS class.modules CASCADE;
DROP TABLE IF EXISTS class.tracks CASCADE;
DROP TABLE IF EXISTS class.topics CASCADE;
