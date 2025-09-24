-- +goose Up
ALTER TABLE class.tracks
ADD COLUMN difficulty TEXT,
ADD COLUMN tags TEXT[];

-- +goose Down
ALTER TABLE class.tracks
DROP COLUMN tags,
DROP COLUMN difficulty;
