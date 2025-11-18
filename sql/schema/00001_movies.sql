-- +goose Up
-- +goose StatementBegin
CREATE TABLE movies (
    id INT PRIMARY KEY,
    title TEXT NOT NULL,
    year INT,
    tagline TEXT,
    genres TEXT,
    budget TEXT,
    director TEXT,
    actor1 TEXT,
    actor2 TEXT,
    popularity DOUBLE PRECISION,
    poster_url TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE movies;
-- +goose StatementEnd
