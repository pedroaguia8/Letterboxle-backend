-- +goose Up
-- +goose StatementBegin
ALTER TABLE movies
ALTER COLUMN budget TYPE BIGINT USING budget::BIGINT,
ALTER COLUMN budget DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE movies
ALTER COLUMN budget TYPE TEXT USING budget::TEXT,
ALTER COLUMN budget SET NOT NULL;
-- +goose StatementEnd
