-- +goose Up
-- +goose StatementBegin
ALTER TABLE movies
ALTER COLUMN genres TYPE JSONB USING to_jsonb(genres);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE movies
ALTER COLUMN genres TYPE TEXT USING genres::TEXT;
-- +goose StatementEnd
