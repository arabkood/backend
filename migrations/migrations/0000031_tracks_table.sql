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
