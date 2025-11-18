-- name: SearchMovies :many
SELECT title, year
FROM movies
WHERE title ILIKE $1
LIMIT 20;