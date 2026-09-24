package workers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
)

// SelectionWindowDays is how many days ahead (including today) the
// selection worker keeps movie_of_the_day filled. A window bigger than 1
// lets an admin hand-override upcoming days and means a catalog sync outage
// doesn't take the game down immediately.
const SelectionWindowDays = 7

// LowEligibleWarningThreshold is the eligible-unused movie count below which
// Fill logs a warning, since the catalog is close to running out of movies
// that haven't been used yet.
const LowEligibleWarningThreshold = 30

const (
	// selectionInterval is how often Fill re-runs after a successful run.
	selectionInterval = 4 * time.Hour
	// selectionRetryInterval is how soon Fill retries after a failed run, so
	// a worker that starts before migrations/the catalog sync have finished
	// doesn't wait hours to try again. Fill is cheap and idempotent, so this
	// can be short.
	selectionRetryInterval = 1 * time.Minute
)

type Selector struct {
	db *database.Queries
}

func NewSelector(db *database.Queries) *Selector {
	return &Selector{db: db}
}

// Fill ensures every date from today through today+SelectionWindowDays-1
// (UTC) has a movie_of_the_day row, picking a random unused eligible movie
// for each date that's missing one. It's idempotent: dates that already
// have a row are left untouched, and the insert is ON CONFLICT (date) DO
// NOTHING so overlapping runs can't clobber each other.
//
// The returned bool reports whether every date in the window ended up
// filled. It's false when the catalog didn't have enough eligible movies
// for some date (e.g. the catalog sync hasn't populated movies yet on a
// fresh DB) — that's not an error, but StartWorker uses it to retry sooner
// instead of waiting a full selectionInterval.
func (s *Selector) Fill(ctx context.Context) (bool, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	lastDate := today.AddDate(0, 0, SelectionWindowDays-1)

	filled, err := s.db.GetMovieOfTheDayDatesInRange(ctx, database.GetMovieOfTheDayDatesInRangeParams{
		Date:   today,
		Date_2: lastDate,
	})
	if err != nil {
		return false, fmt.Errorf("failed to load filled dates: %w", err)
	}
	// Keyed by formatted date rather than time.Time itself: lib/pq scans
	// DATE columns into a fixed-offset Location distinct from the time.UTC
	// singleton, so == (which a map key relies on) is false even though the
	// instants are .Equal() and represent the same calendar day.
	filledDates := make(map[string]bool, len(filled))
	for _, d := range filled {
		filledDates[d.Format(time.DateOnly)] = true
	}

	picked := 0
	allFilled := true
	for date := today; !date.After(lastDate); date = date.AddDate(0, 0, 1) {
		if filledDates[date.Format(time.DateOnly)] {
			continue
		}

		movieID, err := s.db.PickRandomUnusedMovie(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("Selection: no eligible unused movie available for %s", date.Format(time.DateOnly))
				allFilled = false
				continue
			}
			return false, fmt.Errorf("failed to pick a movie for %s: %w", date.Format(time.DateOnly), err)
		}

		if err := s.db.InsertMovieOfTheDay(ctx, database.InsertMovieOfTheDayParams{
			Date:    date,
			MovieID: movieID,
		}); err != nil {
			return false, fmt.Errorf("failed to insert movie of the day for %s: %w", date.Format(time.DateOnly), err)
		}
		picked++
	}

	if picked > 0 {
		log.Printf("Selection: filled %d date(s) in the movie_of_the_day window", picked)
	}

	eligible, err := s.db.CountEligibleUnusedMovies(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to count eligible unused movies: %w", err)
	}
	if eligible < LowEligibleWarningThreshold {
		log.Printf("Selection: WARNING only %d eligible unused movie(s) left", eligible)
	}

	return allFilled, nil
}

// StartWorker runs Fill in the background: immediately at startup, then
// every selectionInterval once the window is fully filled. A failed run, or
// one that leaves a date unfilled (e.g. the catalog sync hasn't caught up
// yet), retries after selectionRetryInterval instead of waiting for the
// next scheduled run.
func (s *Selector) StartWorker(ctx context.Context) {
	go func() {
		log.Println("Starting selection worker (runs every 4 hours)...")

		for {
			wait := selectionInterval
			allFilled, err := s.Fill(ctx)
			if err != nil {
				log.Printf("Selection failed: %v", err)
				wait = selectionRetryInterval
			} else if !allFilled {
				wait = selectionRetryInterval
			}

			select {
			case <-ctx.Done():
				log.Println("Stopping selection worker...")
				return
			case <-time.After(wait):
			}
		}
	}()
}
