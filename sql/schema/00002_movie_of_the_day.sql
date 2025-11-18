-- +goose Up
-- +goose StatementBegin
CREATE TABLE movie_of_the_day (
    date DATE PRIMARY KEY,
    movie_id INT NOT NULL REFERENCES movies(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE movie_of_the_day;
-- +goose StatementEnd
