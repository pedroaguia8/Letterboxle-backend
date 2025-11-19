package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
	"github.com/pedroaguia8/Letterboxle-backend/internal/handlers"
	"github.com/pedroaguia8/Letterboxle-backend/internal/workers"
)
import _ "github.com/lib/pq"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	err := godotenv.Load()
	if err != nil {
		log.Printf("No .env file found, using environment variables")
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

	apiConfig.TmdbApiKey = os.Getenv("TMDB_API_KEY")

	posterFetcher := workers.NewPosterFetcher(dbQueries, apiConfig.TmdbApiKey)
	posterFetcher.StartDailyWorker(ctx)

	mux := http.NewServeMux()

	mux.Handle("GET /api/movie_of_the_day/{date}", http.HandlerFunc(apiConfig.GetMovieOfTheDay))
	mux.Handle("GET /api/movies", http.HandlerFunc(apiConfig.SearchMovies))

	server := http.Server{
		Addr:    ":" + apiConfig.Port,
		Handler: mux,
	}

	go func() {
		log.Printf("Server starting on port %s", apiConfig.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutdown signal received. Shutting down...")

	// Graceful shutdown sequence.
	// Give the server 5 seconds to finish active requests.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if err := db.Close(); err != nil {
		log.Printf("Error closing database: %v", err)
	}

	log.Println("Server exited cleanly")
}
