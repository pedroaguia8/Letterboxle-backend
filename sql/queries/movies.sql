-- name: SearchMovies :many
SELECT title, year
FROM movies
WHERE title ILIKE $1
LIMIT 20;

-- name: UpdateMoviePoster :exec
UPDATE movies
SET poster_url = $1
WHERE id = $2;