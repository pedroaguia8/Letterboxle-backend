package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
	"github.com/pedroaguia8/Letterboxle-backend/internal/handlers"
	"github.com/pedroaguia8/Letterboxle-backend/internal/migrations"
	"github.com/pedroaguia8/Letterboxle-backend/internal/workers"
)
import _ "github.com/lib/pq"

// embeddedSchema holds the goose migrations, so the binary can apply them
// without the sql/ folder or the goose CLI.
//
//go:embed sql/schema/*.sql
var embeddedSchema embed.FS

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

	schema, err := fs.Sub(embeddedSchema, "sql/schema")
	if err != nil {
		log.Fatalf("Couldn't read embedded migrations: %v", err)
	}

	// "app migrate" applies pending migrations and exits. The deploy runs it
	// in a separate container before switching to the new image.
	if len(os.Args) > 1 {
		if os.Args[1] != "migrate" {
			log.Fatal("Unknown command (the only command is \"migrate\")")
		}
		if err := migrations.Up(ctx, db, schema); err != nil {
			log.Fatalf("Migrations failed: %v", err)
		}
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
		return
	}

	// Refuse to serve against a schema older than this binary expects.
	if err := migrations.CheckNoPending(ctx, db, schema); err != nil {
		log.Fatalf("Refusing to start: %v", err)
	}

	dbQueries := database.New(db)
	apiConfig.Db = dbQueries

	apiConfig.Platform = os.Getenv("PLATFORM")

	apiConfig.Port = os.Getenv("PORT")

	catalogSyncer := workers.NewCatalogSyncer(dbQueries, os.Getenv("TMDB_API_READ_ACCESS_TOKEN"))
	catalogSyncer.StartWorker(ctx)

	selector := workers.NewSelector(dbQueries)
	selector.StartWorker(ctx)

	mux := http.NewServeMux()

	mux.Handle("GET /api/movie_of_the_day/{date}", http.HandlerFunc(apiConfig.GetMovieOfTheDay))
	mux.Handle("GET /api/movies", http.HandlerFunc(apiConfig.ListMovies))

	server := http.Server{
		Addr:              ":" + apiConfig.Port,
		Handler:           mux,
		ReadHeaderTimeout: time.Duration(5 * time.Second),
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
}
