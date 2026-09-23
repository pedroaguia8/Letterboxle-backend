package tmdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/sqlc-dev/pqtype"
)

func TestSearchMovie(t *testing.T) {
	tests := map[string]struct {
		queryTitle     string
		queryYear      int
		mockResponse   string
		mockStatusCode int
		wantPosterURL  string
		wantErr        bool
	}{
		"successful_search": {
			queryTitle:     "Inception",
			queryYear:      2010,
			mockStatusCode: 200,
			mockResponse:   `{"results": [{"poster_path": "/inception.jpg"}]}`,
			wantPosterURL:  "https://image.tmdb.org/t/p/w500/inception.jpg",
			wantErr:        false,
		},
		"no_results": {
			queryTitle:     "NonExistentMovie",
			queryYear:      2025,
			mockStatusCode: 200,
			mockResponse:   `{"results": []}`,
			wantPosterURL:  "",
			wantErr:        true,
		},
		"api_error": {
			queryTitle:     "ErrorMovie",
			queryYear:      2010,
			mockStatusCode: 500,
			mockResponse:   `{}`,
			wantPosterURL:  "",
			wantErr:        true,
		},
		"malformed_json": {
			queryTitle:     "BadJson",
			queryYear:      2010,
			mockStatusCode: 200,
			mockResponse:   `{invalid-json`,
			wantPosterURL:  "",
			wantErr:        true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create a local test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/search/movie"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				queryParams := r.URL.Query()

				if gotTitle := queryParams.Get("query"); gotTitle != tc.queryTitle {
					t.Errorf("Expected ?query=%s, got %s", tc.queryTitle, gotTitle)
				}

				expectedYear := fmt.Sprintf("%d", tc.queryYear)
				if gotYear := queryParams.Get("year"); gotYear != expectedYear {
					t.Errorf("Expected ?year=%s, got %s", expectedYear, gotYear)
				}

				w.WriteHeader(tc.mockStatusCode)
				_, _ = w.Write([]byte(tc.mockResponse))
			}))
			defer server.Close()

			// Initialize client and inject the test server URL
			client := NewClient("fake-api-key")
			client.SetBaseURL(server.URL)

			got, err := client.SearchMovie(context.Background(), tc.queryTitle, tc.queryYear)

			// Check error expectation
			if (err != nil) != tc.wantErr {
				t.Fatalf("SearchMovie() error = %v, wantErr %v", err, tc.wantErr)
			}

			// Compare results using cmp.Diff
			if diff := cmp.Diff(tc.wantPosterURL, got); diff != "" {
				t.Errorf("SearchMovie() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDiscoverMovies(t *testing.T) {
	tests := map[string]struct {
		voteCountGte   int
		sortBy         string
		page           int
		mockResponse   string
		mockStatusCode int
		wantResult     DiscoverResult
		wantErr        bool
	}{
		"successful_discover": {
			voteCountGte:   5000,
			sortBy:         "vote_count.desc",
			page:           1,
			mockStatusCode: 200,
			mockResponse:   `{"results": [{"id": 27205}, {"id": 155}], "total_pages": 50, "total_results": 1000}`,
			wantResult: DiscoverResult{
				MovieIDs:     []int{27205, 155},
				TotalPages:   50,
				TotalResults: 1000,
			},
		},
		"empty_page": {
			voteCountGte:   5000,
			sortBy:         "vote_count.desc",
			page:           50,
			mockStatusCode: 200,
			mockResponse:   `{"results": [], "total_pages": 50, "total_results": 1000}`,
			wantResult: DiscoverResult{
				MovieIDs:     []int{},
				TotalPages:   50,
				TotalResults: 1000,
			},
		},
		"api_error": {
			voteCountGte:   5000,
			sortBy:         "vote_count.desc",
			page:           1,
			mockStatusCode: 500,
			mockResponse:   `{}`,
			wantErr:        true,
		},
		"malformed_json": {
			voteCountGte:   5000,
			sortBy:         "vote_count.desc",
			page:           1,
			mockStatusCode: 200,
			mockResponse:   `{invalid-json`,
			wantErr:        true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := "/discover/movie"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				queryParams := r.URL.Query()

				expectedVoteCountGte := fmt.Sprintf("%d", tc.voteCountGte)
				if got := queryParams.Get("vote_count.gte"); got != expectedVoteCountGte {
					t.Errorf("Expected ?vote_count.gte=%s, got %s", expectedVoteCountGte, got)
				}
				if got := queryParams.Get("sort_by"); got != tc.sortBy {
					t.Errorf("Expected ?sort_by=%s, got %s", tc.sortBy, got)
				}
				expectedPage := fmt.Sprintf("%d", tc.page)
				if got := queryParams.Get("page"); got != expectedPage {
					t.Errorf("Expected ?page=%s, got %s", expectedPage, got)
				}

				w.WriteHeader(tc.mockStatusCode)
				_, _ = w.Write([]byte(tc.mockResponse))
			}))
			defer server.Close()

			client := NewClient("fake-api-key")
			client.SetBaseURL(server.URL)

			got, err := client.DiscoverMovies(context.Background(), tc.voteCountGte, tc.sortBy, tc.page)

			if (err != nil) != tc.wantErr {
				t.Fatalf("DiscoverMovies() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if diff := cmp.Diff(tc.wantResult, got); diff != "" {
				t.Errorf("DiscoverMovies() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDoGetRetriesOn429(t *testing.T) {
	tests := map[string]struct {
		retryAfterHeader  string
		requestsBefore200 int
		wantErr           bool
	}{
		"retries_with_retry_after_header": {
			retryAfterHeader:  "0",
			requestsBefore200: 2,
		},
		"retries_with_no_retry_after_header": {
			requestsBefore200: 1,
		},
		"gives_up_after_max_retries": {
			retryAfterHeader:  "0",
			requestsBefore200: maxRetries + 1,
			wantErr:           true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				if requestCount <= tc.requestsBefore200 {
					if tc.retryAfterHeader != "" {
						w.Header().Set("Retry-After", tc.retryAfterHeader)
					}
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"results": [{"poster_path": "/inception.jpg"}]}`))
			}))
			defer server.Close()

			client := NewClient("fake-api-key")
			client.SetBaseURL(server.URL)
			client.SetRetryDelay(0)

			got, err := client.SearchMovie(context.Background(), "Inception", 2010)

			if (err != nil) != tc.wantErr {
				t.Fatalf("SearchMovie() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			wantPosterURL := "https://image.tmdb.org/t/p/w500/inception.jpg"
			if diff := cmp.Diff(wantPosterURL, got); diff != "" {
				t.Errorf("SearchMovie() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetMovieDetails(t *testing.T) {
	timeComparer := cmp.Comparer(func(a, b time.Time) bool { return a.Equal(b) })

	tests := map[string]struct {
		movieID        int
		mockResponse   string
		mockStatusCode int
		wantResult     MovieDetails
		wantErr        bool
	}{
		"complete_details": {
			movieID:        550,
			mockStatusCode: 200,
			mockResponse: `{"adult":false,"backdrop_path":"/backdrop.jpg","belongs_to_collection":null,"budget":63000000,` +
				`"genres":[{"id":18,"name":"Drama"}],"homepage":"","id":550,"imdb_id":"tt0137523",` +
				`"origin_country":["US"],"original_language":"en","original_title":"Fight Club",` +
				`"overview":"A ticking-time-bomb insomniac.","popularity":61.416,"poster_path":"/poster.jpg",` +
				`"production_companies":[{"id":508,"name":"Regency Enterprises"}],` +
				`"production_countries":[{"iso_3166_1":"US","name":"United States of America"}],` +
				`"release_date":"1999-10-15","revenue":100853753,"runtime":139,` +
				`"spoken_languages":[{"iso_639_1":"en","name":"English"}],"status":"Released",` +
				`"tagline":"Mischief. Mayhem. Soap.","title":"Fight Club","video":false,` +
				`"vote_average":8.433,"vote_count":26280,` +
				`"credits":{"cast":[{"name":"Edward Norton","order":0},{"name":"Brad Pitt","order":1},` +
				`{"name":"Helena Bonham Carter","order":2}],` +
				`"crew":[{"name":"Jim Uhls","job":"Screenplay"},{"name":"David Fincher","job":"Director"}]}}`,
			wantResult: MovieDetails{
				ID:                  550,
				Title:               "Fight Club",
				OriginalTitle:       sql.NullString{String: "Fight Club", Valid: true},
				OriginalLanguage:    sql.NullString{String: "en", Valid: true},
				Overview:            sql.NullString{String: "A ticking-time-bomb insomniac.", Valid: true},
				Status:              sql.NullString{String: "Released", Valid: true},
				Homepage:            sql.NullString{},
				ImdbID:              sql.NullString{String: "tt0137523", Valid: true},
				ReleaseDate:         sql.NullTime{Time: time.Date(1999, 10, 15, 0, 0, 0, 0, time.UTC), Valid: true},
				Year:                sql.NullInt32{Int32: 1999, Valid: true},
				Runtime:             sql.NullInt32{Int32: 139, Valid: true},
				Budget:              sql.NullInt64{Int64: 63000000, Valid: true},
				Revenue:             sql.NullInt64{Int64: 100853753, Valid: true},
				Popularity:          sql.NullFloat64{Float64: 61.416, Valid: true},
				VoteAverage:         sql.NullFloat64{Float64: 8.433, Valid: true},
				VoteCount:           sql.NullInt32{Int32: 26280, Valid: true},
				Adult:               sql.NullBool{Bool: false, Valid: true},
				Video:               sql.NullBool{Bool: false, Valid: true},
				PosterPath:          sql.NullString{String: "/poster.jpg", Valid: true},
				BackdropPath:        sql.NullString{String: "/backdrop.jpg", Valid: true},
				Tagline:             sql.NullString{String: "Mischief. Mayhem. Soap.", Valid: true},
				Genres:              pqtype.NullRawMessage{RawMessage: []byte(`[{"id":18,"name":"Drama"}]`), Valid: true},
				BelongsToCollection: pqtype.NullRawMessage{},
				ProductionCompanies: pqtype.NullRawMessage{
					RawMessage: []byte(`[{"id":508,"name":"Regency Enterprises"}]`), Valid: true,
				},
				ProductionCountries: pqtype.NullRawMessage{
					RawMessage: []byte(`[{"iso_3166_1":"US","name":"United States of America"}]`), Valid: true,
				},
				SpokenLanguages: pqtype.NullRawMessage{
					RawMessage: []byte(`[{"iso_639_1":"en","name":"English"}]`), Valid: true,
				},
				OriginCountry: pqtype.NullRawMessage{RawMessage: []byte(`["US"]`), Valid: true},
				CreditsCast: pqtype.NullRawMessage{
					RawMessage: []byte(`[{"name":"Edward Norton","order":0},{"name":"Brad Pitt","order":1},` +
						`{"name":"Helena Bonham Carter","order":2}]`),
					Valid: true,
				},
				CreditsCrew: pqtype.NullRawMessage{
					RawMessage: []byte(`[{"name":"Jim Uhls","job":"Screenplay"},{"name":"David Fincher","job":"Director"}]`),
					Valid:      true,
				},
				Director: sql.NullString{String: "David Fincher", Valid: true},
				Actor1:   sql.NullString{String: "Edward Norton", Valid: true},
				Actor2:   sql.NullString{String: "Brad Pitt", Valid: true},
			},
		},
		"missing_hint_fields_normalise_to_null": {
			movieID:        1,
			mockStatusCode: 200,
			mockResponse: `{"adult":false,"backdrop_path":null,"belongs_to_collection":null,"budget":0,` +
				`"genres":[],"homepage":null,"id":1,"imdb_id":"","origin_country":[],"original_language":"",` +
				`"original_title":"","overview":"","popularity":0,"poster_path":null,` +
				`"production_companies":[],"production_countries":[],"release_date":"","revenue":0,"runtime":0,` +
				`"spoken_languages":[],"status":"","tagline":"","title":"Untitled","video":false,` +
				`"vote_average":0,"vote_count":0,"credits":{"cast":[],"crew":[]}}`,
			wantResult: MovieDetails{
				ID:                  1,
				Title:               "Untitled",
				Runtime:             sql.NullInt32{Int32: 0, Valid: true},
				Budget:              sql.NullInt64{Int64: 0, Valid: true},
				Revenue:             sql.NullInt64{Int64: 0, Valid: true},
				Popularity:          sql.NullFloat64{Float64: 0, Valid: true},
				VoteAverage:         sql.NullFloat64{Float64: 0, Valid: true},
				VoteCount:           sql.NullInt32{Int32: 0, Valid: true},
				Adult:               sql.NullBool{Bool: false, Valid: true},
				Video:               sql.NullBool{Bool: false, Valid: true},
				Genres:              pqtype.NullRawMessage{RawMessage: []byte(`[]`), Valid: true},
				ProductionCompanies: pqtype.NullRawMessage{RawMessage: []byte(`[]`), Valid: true},
				ProductionCountries: pqtype.NullRawMessage{RawMessage: []byte(`[]`), Valid: true},
				SpokenLanguages:     pqtype.NullRawMessage{RawMessage: []byte(`[]`), Valid: true},
				OriginCountry:       pqtype.NullRawMessage{RawMessage: []byte(`[]`), Valid: true},
				CreditsCast:         pqtype.NullRawMessage{RawMessage: []byte(`[]`), Valid: true},
				CreditsCrew:         pqtype.NullRawMessage{RawMessage: []byte(`[]`), Valid: true},
			},
		},
		"invalid_release_date": {
			movieID:        2,
			mockStatusCode: 200,
			mockResponse:   `{"id":2,"title":"Bad Date","release_date":"not-a-date","credits":{"cast":[],"crew":[]}}`,
			wantErr:        true,
		},
		"api_error": {
			movieID:        3,
			mockStatusCode: 500,
			mockResponse:   `{}`,
			wantErr:        true,
		},
		"malformed_json": {
			movieID:        4,
			mockStatusCode: 200,
			mockResponse:   `{invalid-json`,
			wantErr:        true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/movie/%d", tc.movieID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}
				if got := r.URL.Query().Get("append_to_response"); got != "credits" {
					t.Errorf("Expected ?append_to_response=credits, got %s", got)
				}

				w.WriteHeader(tc.mockStatusCode)
				_, _ = w.Write([]byte(tc.mockResponse))
			}))
			defer server.Close()

			client := NewClient("fake-api-key")
			client.SetBaseURL(server.URL)

			got, err := client.GetMovieDetails(context.Background(), tc.movieID)

			if (err != nil) != tc.wantErr {
				t.Fatalf("GetMovieDetails() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			if diff := cmp.Diff(tc.wantResult, got, timeComparer); diff != "" {
				t.Errorf("GetMovieDetails() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
