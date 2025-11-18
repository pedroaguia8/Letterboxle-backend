package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
)

type ApiConfig struct {
	Db         *database.Queries
	Platform   string
	JwtSecret  string
	PolkaKey   string
	Port       string
	TmdbApiKey string
}

type Movie struct {
	Title     string `json:"title"`
	Tagline   string `json:"tagline"`
	Genres    string `json:"genres"`
	Director  string `json:"director"`
	Actor1    string `json:"actor1"`
	Actor2    string `json:"actor2"`
	Year      string `json:"year"`
	PosterUrl string `json:"poster_url"`
}

func dbMovieOfTheDayToMovie(dbMovie database.GetMovieOfTheDayRow) Movie {
	posterUrl := ""
	if dbMovie.PosterUrl.Valid {
		posterUrl = dbMovie.PosterUrl.String
	}

	return Movie{
		Title:     dbMovie.Title,
		Tagline:   dbMovie.Tagline,
		Genres:    dbMovie.Genres,
		Director:  dbMovie.Director,
		Actor1:    dbMovie.Actor1,
		Actor2:    dbMovie.Actor2,
		Year:      strconv.Itoa(int(dbMovie.Year)),
		PosterUrl: posterUrl,
	}
}

func (cfg *ApiConfig) GetMovieOfTheDay(w http.ResponseWriter, req *http.Request) {
	dateParam := req.PathValue("date")

	date := time.Time{}
	if dateParam == "today" {
		date = time.Now().UTC()
	} else {
		log.Printf("ERROR: Request for movie of date other than 'today'")
		err := RespondWithError(w, http.StatusBadRequest, "Failed to get movie")
		if err != nil {
			log.Printf("Failed to send error response to client: %v", err)
			return
		}
		return
	}

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
	searchQuery := req.PathValue("search_query")

	searchWords := strings.Fields(searchQuery)
	searchPattern := "%" + strings.Join(searchWords, "%") + "%"

	dbMovies, err := cfg.Db.SearchMovies(req.Context(), searchPattern)
	if err != nil {
		log.Printf("ERROR: Failed to search movies from database with query: %v", searchQuery)
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
			Year:  strconv.Itoa(int(dbMovie.Year)),
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
