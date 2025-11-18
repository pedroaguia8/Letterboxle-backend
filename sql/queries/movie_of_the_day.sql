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
    movies.poster_url
FROM movie_of_the_day
INNER JOIN movies
ON movie_of_the_day.movie_id = movies.id
WHERE movie_of_the_day.date = $1;