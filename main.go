package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
	"github.com/pedroaguia8/Letterboxle-backend/internal/handlers"
)
import _ "github.com/lib/pq"

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Couldn't load environment variables from .env file")
	}

	apiConfig := handlers.ApiConfig{}

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err)
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Fatalf("Couldn't close database: %v", err)
		}
	}(db)
	dbQueries := database.New(db)
	apiConfig.Db = dbQueries

	apiConfig.Platform = os.Getenv("PLATFORM")

	apiConfig.Port = os.Getenv("PORT")

	mux := http.NewServeMux()

	mux.Handle("GET /api/movie_of_the_day/{date}", http.HandlerFunc(apiConfig.GetMovieOfTheDay))
	mux.Handle("GET /api/movies/{search_query}", http.HandlerFunc(apiConfig.SearchMovies))

	server := http.Server{
		Addr:    ":" + apiConfig.Port,
		Handler: mux,
	}
	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
