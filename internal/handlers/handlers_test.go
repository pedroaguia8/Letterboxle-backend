package handlers

import (
	"database/sql"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pedroaguia8/Letterboxle-backend/internal/database"
)

func TestDbMovieOfTheDayToMovie(t *testing.T) {
	// Table driven test
	tests := map[string]struct {
		input database.GetMovieOfTheDayRow
		want  Movie
	}{
		"complete_movie": {
			input: database.GetMovieOfTheDayRow{
				Title:    "The Matrix",
				Tagline:  "Welcome to the Real World",
				Genres:   "Sci-Fi",
				Director: "Wachowskis",
				Actor1:   "Keanu Reeves",
				Actor2:   "Laurence Fishburne",
				Year:     1999,
				PosterUrl: sql.NullString{
					String: "http://poster.url",
					Valid:  true,
				},
			},
			want: Movie{
				Title:     "The Matrix",
				Tagline:   "Welcome to the Real World",
				Genres:    "Sci-Fi",
				Director:  "Wachowskis",
				Actor1:    "Keanu Reeves",
				Actor2:    "Laurence Fishburne",
				Year:      "1999",
				PosterUrl: "http://poster.url",
				Date:      "", // The function doesn't set Date, so we expect empty
			},
		},
		"null_poster": {
			input: database.GetMovieOfTheDayRow{
				Title: "Unknown",
				Year:  2020,
				PosterUrl: sql.NullString{
					String: "",
					Valid:  false,
				},
			},
			want: Movie{
				Title:     "Unknown",
				Year:      "2020",
				PosterUrl: "",
				Date:      "",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := dbMovieOfTheDayToMovie(tc.input)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("dbMovieOfTheDayToMovie() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
