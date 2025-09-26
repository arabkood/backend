-- +goose Up
DROP TABLE IF EXISTS users.code_attempt;
DROP TABLE IF EXISTS users.code_submission;

-- +goose Down
