-- +goose Up
-- +goose StatementBegin
ALTER TABLE movies
DROP COLUMN poster_url;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE movies
ADD COLUMN poster_url TEXT;
-- +goose StatementEnd
