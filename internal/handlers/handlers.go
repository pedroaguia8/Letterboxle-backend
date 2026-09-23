package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
)

type ApiConfig struct {
	Db       *database.Queries
	Platform string
	Port     string
}

type Movie struct {
	ID        int32    `json:"id"`
	Title     string   `json:"title"`
	Tagline   string   `json:"tagline"`
	Genres    []string `json:"genres"`
	Director  string   `json:"director"`
	Actor1    string   `json:"actor1"`
	Actor2    string   `json:"actor2"`
	Year      string   `json:"year"`
	PosterUrl string   `json:"poster_url"`
	Date      string   `json:"date"`
}

func dbMovieOfTheDayToMovie(dbMovie database.GetMovieOfTheDayRow) Movie {
	return Movie{
		ID:        dbMovie.ID,
		Title:     dbMovie.Title,
		Tagline:   nullString(dbMovie.Tagline),
		Genres:    genreNames(dbMovie.Genres),
		Director:  nullString(dbMovie.Director),
		Actor1:    nullString(dbMovie.Actor1),
		Actor2:    nullString(dbMovie.Actor2),
		Year:      strconv.Itoa(int(nullInt32(dbMovie.Year))),
		PosterUrl: posterURL(dbMovie.PosterPath),
	}
}

func (cfg *ApiConfig) GetMovieOfTheDay(w http.ResponseWriter, req *http.Request) {
	dateParam := req.PathValue("date")

	// Later we change this when we implement playing past day's games
	if dateParam != "today" {
		log.Printf("ERROR: Request for movie of date other than 'today'")
		err := RespondWithError(w, http.StatusBadRequest, "Failed to get movie")
		if err != nil {
			log.Printf("Failed to send error response to client: %v", err)
			return
		}
		return
	}
	date := time.Now().UTC()

	dbMovie, err := cfg.Db.GetMovieOfTheDay(req.Context(), date)
	if err != nil {
		log.Printf("ERROR: Failed to get movie of the day from database")
		err := RespondWithError(w, http.StatusBadRequest, "Failed to get movie")
		if err != nil {
			log.Printf("Failed to send error response to client: %v", err)
			return
		}
		return
	}

	movie := dbMovieOfTheDayToMovie(dbMovie)
	movie.Date = date.Format(time.DateOnly)

	err = RespondWithJSON(w, http.StatusOK, movie)
	if err != nil {
		log.Printf("ERROR: Failed to respond with json")
		err := RespondWithError(w, http.StatusInternalServerError, "Failed to get movie")
		if err != nil {
			log.Printf("Failed to send error response to client: %v", err)
			return
		}
		return
	}
}

// ListMovies serves the full lightweight movie list (id, title, year) for the
// frontend's one-time autocomplete fetch. Cached for a few hours since the
// catalog only changes on the weekly sync.
func (cfg *ApiConfig) ListMovies(w http.ResponseWriter, req *http.Request) {
	dbMovies, err := cfg.Db.ListAllMovies(req.Context())
	if err != nil {
		log.Printf("ERROR: Failed to list movies from database: %v", err)
		err := RespondWithError(w, http.StatusInternalServerError, "Failed to get movies")
		if err != nil {
			log.Printf("Failed to send error response to client: %v", err)
			return
		}
		return
	}

	type movieDto struct {
		ID    int32  `json:"id"`
		Title string `json:"title"`
		Year  string `json:"year"`
	}
	res := make([]movieDto, 0, len(dbMovies))
	for _, dbMovie := range dbMovies {
		res = append(res, movieDto{
			ID:    dbMovie.ID,
			Title: dbMovie.Title,
			Year:  strconv.Itoa(int(nullInt32(dbMovie.Year))),
		})
	}

	w.Header().Set("Cache-Control", "public, max-age=14400")

	err = RespondWithJSON(w, http.StatusOK, res)
	if err != nil {
		log.Printf("ERROR: Failed to respond with json of movieDto array")
		err := RespondWithError(w, http.StatusInternalServerError, "Failed to get movies")
		if err != nil {
			log.Printf("Failed to send error response to client: %v", err)
			return
		}
		return
	}
}
