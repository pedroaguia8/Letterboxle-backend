-- +goose Up
-- +goose StatementBegin
CREATE INDEX idx_movies_title ON movies (title);
-- +goose StatementEnd



-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_movies_title;
-- +goose StatementEnd