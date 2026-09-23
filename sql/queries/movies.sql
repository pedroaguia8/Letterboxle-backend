-- name: SearchMovies :many
SELECT title, year
FROM movies
WHERE title ILIKE $1
LIMIT 20;

-- name: UpdateMoviePoster :exec
UPDATE movies
SET poster_url = $1
WHERE id = $2;

-- name: GetAllMovieIDs :many
SELECT id FROM movies;

-- name: PickRandomUnusedMovie :one
SELECT id
FROM movies
WHERE id NOT IN (SELECT movie_id FROM movie_of_the_day)
    AND tagline IS NOT NULL
    AND genres IS NOT NULL
    AND director IS NOT NULL
    AND actor1 IS NOT NULL
    AND actor2 IS NOT NULL
    AND year IS NOT NULL
    AND poster_path IS NOT NULL
ORDER BY random()
LIMIT 1;

-- name: InsertMovie :exec
INSERT INTO movies (
    id, title, year, tagline, genres, budget, director, actor1, actor2,
    popularity, poster_url, original_title, original_language, overview,
    status, homepage, imdb_id, release_date, runtime, revenue, vote_average,
    vote_count, adult, video, poster_path, backdrop_path,
    belongs_to_collection, production_companies, production_countries,
    spoken_languages, origin_country, credits_cast, credits_crew
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9,
    $10, $11, $12, $13, $14,
    $15, $16, $17, $18, $19, $20, $21,
    $22, $23, $24, $25, $26,
    $27, $28, $29,
    $30, $31, $32, $33
)
ON CONFLICT (id) DO NOTHING;