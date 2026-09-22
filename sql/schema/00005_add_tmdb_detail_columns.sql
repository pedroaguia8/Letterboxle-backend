-- +goose Up
-- +goose StatementBegin
ALTER TABLE movies
ADD COLUMN original_title TEXT,
ADD COLUMN original_language TEXT,
ADD COLUMN overview TEXT,
ADD COLUMN status TEXT,
ADD COLUMN homepage TEXT,
ADD COLUMN imdb_id TEXT,
ADD COLUMN release_date DATE,
ADD COLUMN runtime INT,
ADD COLUMN revenue BIGINT,
ADD COLUMN vote_average DOUBLE PRECISION,
ADD COLUMN vote_count INT,
ADD COLUMN adult BOOLEAN,
ADD COLUMN video BOOLEAN,
ADD COLUMN poster_path TEXT,
ADD COLUMN backdrop_path TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE movies
DROP COLUMN original_title,
DROP COLUMN original_language,
DROP COLUMN overview,
DROP COLUMN status,
DROP COLUMN homepage,
DROP COLUMN imdb_id,
DROP COLUMN release_date,
DROP COLUMN runtime,
DROP COLUMN revenue,
DROP COLUMN vote_average,
DROP COLUMN vote_count,
DROP COLUMN adult,
DROP COLUMN video,
DROP COLUMN poster_path,
DROP COLUMN backdrop_path;
-- +goose StatementEnd
