package workers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
	"github.com/pedroaguia8/Letterboxle-backend/internal/tmdb"
)

type PosterFetcher struct {
	db         *database.Queries
	tmdbClient *tmdb.Client
}

func NewPosterFetcher(db *database.Queries, tmdbApiKey string) *PosterFetcher {
	return &PosterFetcher{
		db:         db,
		tmdbClient: tmdb.NewClient(tmdbApiKey),
	}
}

func (pf *PosterFetcher) FetchPosterForDate(ctx context.Context, date time.Time) error {
	log.Printf("Fetching poster for movie on date: %s", date.Format(time.DateOnly))

	movie, err := pf.db.GetMovieOfTheDay(ctx, date)
	if err != nil {
		return fmt.Errorf("failed to get movie for date %s: %w", date.Format(time.DateOnly), err)
	}

	if movie.PosterUrl.Valid && movie.PosterUrl.String != "" {
		log.Printf("Poster already cached for movie: %s", movie.Title)
		return nil
	}

	log.Printf("Fetching poster from TMDB for: %s (%d)", movie.Title, movie.Year)

	posterURL, err := pf.tmdbClient.SearchMovie(ctx, movie.Title, int(movie.Year))
	if err != nil {
		log.Printf("Error fetching from TMDB: %v", err)
		// prevent refetching
		posterURL = ""
	}

	posterURLSql := sql.NullString{
		String: posterURL,
		Valid:  true,
	}

	err = pf.db.UpdateMoviePoster(ctx, database.UpdateMoviePosterParams{
		PosterUrl: posterURLSql,
		ID:        movie.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to update poster URL: %w", err)
	}

	if posterURL != "" {
		log.Printf("Successfully fetched and saved poster URL for: %s", movie.Title)
	} else {
		log.Printf("No poster found for: %s. Saved empty placeholder.", movie.Title)
	}

	return nil
}

func (pf *PosterFetcher) StartDailyWorker(ctx context.Context) {
	go func() {
		log.Println("Starting poster fetcher worker (runs every 4 hours)...")

		runFetch := func() {
			if err := pf.FetchPosterForDate(ctx, time.Now().UTC()); err != nil {
				log.Printf("Error fetching poster for tomorrow: %v", err)
			}
			tomorrow := time.Now().UTC().AddDate(0, 0, 1)
			if err := pf.FetchPosterForDate(ctx, tomorrow); err != nil {
				log.Printf("Error fetching poster for tomorrow: %v", err)
			}
		}

		runFetch()

		ticker := time.NewTicker(4 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping poster fetcher worker...")
				return
			case <-ticker.C:
				runFetch()
			}
		}
	}()
}
