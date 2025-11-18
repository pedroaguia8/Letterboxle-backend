package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
)

type ApiConfig struct {
	Db        *database.Queries
	Platform  string
	JwtSecret string
	PolkaKey  string
	Port      string
}

type Movie struct {
	Title    string `json:"title"`
	Tagline  string `json:"tagline"`
	Genres   string `json:"genres"`
	Director string `json:"director"`
	Actor1   string `json:"actor1"`
	Actor2   string `json:"actor2"`
	Year     string `json:"year"`
}

func dbMovieOfTheDayToMovie(dbMovie database.GetMovieOfTheDayRow) Movie {
	return Movie{
		Title:    dbMovie.Title,
		Tagline:  dbMovie.Tagline,
		Genres:   dbMovie.Genres,
		Director: dbMovie.Director,
		Actor1:   dbMovie.Actor1,
		Actor2:   dbMovie.Actor2,
		Year:     strconv.Itoa(int(dbMovie.Year)),
	}
}

func (cfg *ApiConfig) GetMovieOfTheDay(w http.ResponseWriter, req *http.Request) {
	dateParam := req.PathValue("date")

	date := time.Time{}
	if dateParam == "today" {
		date = time.Now()
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
