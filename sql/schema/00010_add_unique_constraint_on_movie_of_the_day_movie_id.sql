-- +goose Up
-- +goose StatementBegin
ALTER TABLE movie_of_the_day
ADD CONSTRAINT movie_of_the_day_movie_id_key UNIQUE (movie_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE movie_of_the_day
DROP CONSTRAINT movie_of_the_day_movie_id_key;
-- +goose StatementEnd
