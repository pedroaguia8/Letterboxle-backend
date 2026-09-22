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
				Tagline:  sql.NullString{String: "Welcome to the Real World", Valid: true},
				Genres:   sql.NullString{String: "Sci-Fi", Valid: true},
				Director: sql.NullString{String: "Wachowskis", Valid: true},
				Actor1:   sql.NullString{String: "Keanu Reeves", Valid: true},
				Actor2:   sql.NullString{String: "Laurence Fishburne", Valid: true},
				Year:     sql.NullInt32{Int32: 1999, Valid: true},
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
				Year:  sql.NullInt32{Int32: 2020, Valid: true},
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
		"null_hint_fields": {
			input: database.GetMovieOfTheDayRow{
				Title: "Untagged",
			},
			want: Movie{
				Title: "Untagged",
				Year:  "0",
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
