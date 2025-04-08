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
