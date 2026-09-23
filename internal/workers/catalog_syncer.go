package workers

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
	"github.com/pedroaguia8/Letterboxle-backend/internal/tmdb"
)

// DefaultVoteCountThreshold is the vote_count.gte floor used to walk TMDB's
// discover endpoint. A higher threshold means fewer, more well-known movies
// and more years of runway before eligible movies run out (see task 8's
// tuning pass).
const DefaultVoteCountThreshold = 5000

type CatalogSyncer struct {
	db                 *database.Queries
	tmdbClient         *tmdb.Client
	voteCountThreshold int
}

func NewCatalogSyncer(db *database.Queries, tmdbApiReadAccessToken string) *CatalogSyncer {
	return &CatalogSyncer{
		db:                 db,
		tmdbClient:         tmdb.NewClient(tmdbApiReadAccessToken),
		voteCountThreshold: DefaultVoteCountThreshold,
	}
}

// Sync walks the full TMDB discover list (vote_count.gte=voteCountThreshold,
// sorted by vote_count desc), diffs the returned IDs against what's already
// in the movies table, fetches full details for anything new, and inserts
// it. The first run against an empty DB is the seed; the same method re-run
// weekly is the updater, since re-walking the full list is cheap (~50 calls)
// and complete, unlike a popularity-based heuristic that could miss a movie
// crossing the vote threshold outside the popularity top page.
func (cs *CatalogSyncer) Sync(ctx context.Context) error {
	existingIDs, err := cs.db.GetAllMovieIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to load existing movie IDs: %w", err)
	}
	existing := make(map[int32]bool, len(existingIDs))
	for _, id := range existingIDs {
		existing[id] = true
	}

	var newIDs []int32
	totalResults := 0
	for page := 1; ; page++ {
		result, err := cs.tmdbClient.DiscoverMovies(ctx, cs.voteCountThreshold, "vote_count.desc", page)
		if err != nil {
			return fmt.Errorf("failed to discover movies (page %d): %w", page, err)
		}
		totalResults = result.TotalResults

		for _, id := range result.MovieIDs {
			id32 := int32(id) // #nosec G115 -- TMDB movie IDs fit comfortably in int32
			if !existing[id32] {
				newIDs = append(newIDs, id32)
			}
		}

		if page >= result.TotalPages {
			break
		}
	}

	log.Printf("Catalog sync: discover returned %d total results, %d new movie(s) to fetch", totalResults, len(newIDs))

	inserted := 0
	for _, id := range newIDs {
		if err := ctx.Err(); err != nil {
			return err
		}

		details, err := cs.tmdbClient.GetMovieDetails(ctx, int(id))
		if err != nil {
			log.Printf("Catalog sync: failed to fetch details for movie %d: %v", id, err)
			continue
		}

		if err := cs.db.InsertMovie(ctx, insertMovieParams(details)); err != nil {
			log.Printf("Catalog sync: failed to insert movie %d: %v", id, err)
			continue
		}
		inserted++
	}

	log.Printf("Catalog sync: inserted %d/%d new movie(s)", inserted, len(newIDs))
	return nil
}

func insertMovieParams(d tmdb.MovieDetails) database.InsertMovieParams {
	posterURL := sql.NullString{}
	if d.PosterPath.Valid {
		posterURL = sql.NullString{String: tmdb.ImageBaseURL + d.PosterPath.String, Valid: true}
	}

	return database.InsertMovieParams{
		ID:                  d.ID,
		Title:               d.Title,
		Year:                d.Year,
		Tagline:             d.Tagline,
		Genres:              d.Genres,
		Budget:              d.Budget,
		Director:            d.Director,
		Actor1:              d.Actor1,
		Actor2:              d.Actor2,
		Popularity:          d.Popularity,
		PosterUrl:           posterURL,
		OriginalTitle:       d.OriginalTitle,
		OriginalLanguage:    d.OriginalLanguage,
		Overview:            d.Overview,
		Status:              d.Status,
		Homepage:            d.Homepage,
		ImdbID:              d.ImdbID,
		ReleaseDate:         d.ReleaseDate,
		Runtime:             d.Runtime,
		Revenue:             d.Revenue,
		VoteAverage:         d.VoteAverage,
		VoteCount:           d.VoteCount,
		Adult:               d.Adult,
		Video:               d.Video,
		PosterPath:          d.PosterPath,
		BackdropPath:        d.BackdropPath,
		BelongsToCollection: d.BelongsToCollection,
		ProductionCompanies: d.ProductionCompanies,
		ProductionCountries: d.ProductionCountries,
		SpokenLanguages:     d.SpokenLanguages,
		OriginCountry:       d.OriginCountry,
		CreditsCast:         d.CreditsCast,
		CreditsCrew:         d.CreditsCrew,
	}
}
