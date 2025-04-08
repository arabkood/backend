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
    pre_args text [],

    deleted_at timestamp with time zone,

    PRIMARY KEY (id),
    CONSTRAINT fk_track_id FOREIGN KEY (track_id) REFERENCES class.tracks (id) ON DELETE CASCADE
);


-- +goose Down


DROP TABLE IF EXISTS class.tracks_sections;
DROP TABLE IF EXISTS class.modules;
