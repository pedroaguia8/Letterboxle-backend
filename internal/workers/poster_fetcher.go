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

func NewPosterFetcher(db *database.Queries, tmdbApiReadAccessToken string) *PosterFetcher {
	return &PosterFetcher{
		db:         db,
		tmdbClient: tmdb.NewClient(tmdbApiReadAccessToken),
	}
}

// EnsurePosterURL returns the cached poster URL for a movie, or fetches it
// from TMDB and persists it (including an empty result, to avoid refetching
// on every call) if it isn't cached yet.
func (pf *PosterFetcher) EnsurePosterURL(ctx context.Context, movieID int32, title string, year int32, currentPosterURL sql.NullString) (string, error) {
	if currentPosterURL.Valid && currentPosterURL.String != "" {
		return currentPosterURL.String, nil
	}

	log.Printf("Fetching poster from TMDB for: %s (%d)", title, year)

	posterURL, err := pf.tmdbClient.SearchMovie(ctx, title, int(year))
	if err != nil {
		log.Printf("Error fetching from TMDB: %v", err)
		// prevent refetching
		posterURL = ""
	}

	err = pf.db.UpdateMoviePoster(ctx, database.UpdateMoviePosterParams{
		PosterUrl: sql.NullString{String: posterURL, Valid: true},
		ID:        movieID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to update poster URL: %w", err)
	}

	if posterURL != "" {
		log.Printf("Successfully fetched and saved poster URL for: %s", title)
	} else {
		log.Printf("No poster found for: %s. Saved empty placeholder.", title)
	}

	return posterURL, nil
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

	_, err = pf.EnsurePosterURL(ctx, movie.ID, movie.Title, movie.Year.Int32, movie.PosterUrl)
	return err
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
