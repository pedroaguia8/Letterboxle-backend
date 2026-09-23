-- name: GetMovieOfTheDay :one
SELECT
    movies.id,
    movies.title,
    movies.tagline,
    movies.genres,
    movies.director,
    movies.actor1,
    movies.actor2,
    movies.year,
    movies.poster_path
FROM movie_of_the_day
INNER JOIN movies
ON movie_of_the_day.movie_id = movies.id
WHERE movie_of_the_day.date = $1;

-- name: GetMovieOfTheDayDatesInRange :many
SELECT date
FROM movie_of_the_day
WHERE date >= $1 AND date <= $2;

-- name: InsertMovieOfTheDay :exec
INSERT INTO movie_of_the_day (date, movie_id)
VALUES ($1, $2)
ON CONFLICT (date) DO NOTHING;