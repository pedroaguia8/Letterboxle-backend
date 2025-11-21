package tmdb

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
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
