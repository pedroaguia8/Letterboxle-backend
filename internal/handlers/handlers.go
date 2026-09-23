package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
	"github.com/pedroaguia8/Letterboxle-backend/internal/workers"
)

type ApiConfig struct {
	Db            *database.Queries
	Platform      string
	Port          string
	PosterFetcher *workers.PosterFetcher
}

type Movie struct {
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
		Title:     dbMovie.Title,
		Tagline:   nullString(dbMovie.Tagline),
		Genres:    genreNames(dbMovie.Genres),
		Director:  nullString(dbMovie.Director),
		Actor1:    nullString(dbMovie.Actor1),
		Actor2:    nullString(dbMovie.Actor2),
		Year:      strconv.Itoa(int(nullInt32(dbMovie.Year))),
		PosterUrl: nullString(dbMovie.PosterUrl),
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

	if !dbMovie.PosterUrl.Valid {
		log.Printf("Poster missing for %s, fetching on demand", dbMovie.Title)
		posterURL, err := cfg.PosterFetcher.EnsurePosterURL(req.Context(), dbMovie.ID, dbMovie.Title, nullInt32(dbMovie.Year), dbMovie.PosterUrl)
		if err != nil {
			log.Printf("Failed to fetch on-demand poster: %v", err)
		} else {
			dbMovie.PosterUrl = sql.NullString{String: posterURL, Valid: true}
		}
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

func (cfg *ApiConfig) SearchMovies(w http.ResponseWriter, req *http.Request) {
	searchQuery := req.URL.Query().Get("search_query")

	if searchQuery == "" {
		err := RespondWithJSON(w, http.StatusOK, []interface{}{})
		if err != nil {
			log.Printf("Failed to send error response to client: %v", err)
			return
		}
		return
	}

	searchWords := strings.Fields(searchQuery)
	searchPattern := "%" + strings.Join(searchWords, "%") + "%"

	dbMovies, err := cfg.Db.SearchMovies(req.Context(), searchPattern)
	if err != nil {
		log.Printf("ERROR: Failed to search movies from database: %v", err)
		err := RespondWithError(w, http.StatusBadRequest, "Failed to get movies")
		if err != nil {
			log.Printf("Failed to send error response to client: %v", err)
			return
		}
		return
	}

	type movieDto struct {
		Title string `json:"title"`
		Year  string `json:"year"`
	}
	res := []movieDto{}
	for _, dbMovie := range dbMovies {
		movie := movieDto{
			Title: dbMovie.Title,
			Year:  strconv.Itoa(int(nullInt32(dbMovie.Year))),
		}
		res = append(res, movie)
	}

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
