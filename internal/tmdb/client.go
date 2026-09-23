package tmdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/sqlc-dev/pqtype"
)

const DefaultBaseURL = "https://api.themoviedb.org/3"
const ImageBaseURL = "https://image.tmdb.org/t/p/w500"

type Client struct {
	readAccessToken string
	baseURL         string
	httpClient      *http.Client
}

func NewClient(readAccessToken string) *Client {
	return &Client{
		readAccessToken: readAccessToken,
		baseURL:         DefaultBaseURL,
		httpClient:      &http.Client{},
	}
}

// Helper for tests to override the URL
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

type SearchResponse struct {
	Results []struct {
		PosterPath string `json:"poster_path"`
	} `json:"results"`
}

func (c *Client) SearchMovie(ctx context.Context, title string, year int) (string, error) {
	searchURL := fmt.Sprintf("%s/search/movie?query=%s&year=%d",
		c.baseURL,
		url.QueryEscape(title),
		year,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.readAccessToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from TMDB: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TMDB API returned status: %d", resp.StatusCode)
	}

	searchRes := SearchResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&searchRes); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(searchRes.Results) == 0 {
		return "", errors.New("no poster found")
	}
	if searchRes.Results[0].PosterPath == "" {
		return "", errors.New("no poster found")
	}

	posterURL := ImageBaseURL + searchRes.Results[0].PosterPath
	return posterURL, nil
}

type DiscoverResponse struct {
	Results []struct {
		ID int `json:"id"`
	} `json:"results"`
	TotalPages   int `json:"total_pages"`
	TotalResults int `json:"total_results"`
}

type DiscoverResult struct {
	MovieIDs     []int
	TotalPages   int
	TotalResults int
}

// DiscoverMovies fetches a single page of /discover/movie. TMDB caps discover
// at page 500 and reports that cap via total_pages, so callers should loop
// while page <= TotalPages rather than until an empty page.
func (c *Client) DiscoverMovies(ctx context.Context, voteCountGte int, sortBy string, page int) (DiscoverResult, error) {
	discoverURL := fmt.Sprintf("%s/discover/movie?vote_count.gte=%d&sort_by=%s&page=%d",
		c.baseURL,
		voteCountGte,
		url.QueryEscape(sortBy),
		page,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", discoverURL, nil)
	if err != nil {
		return DiscoverResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.readAccessToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return DiscoverResult{}, fmt.Errorf("failed to fetch from TMDB: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return DiscoverResult{}, fmt.Errorf("TMDB API returned status: %d", resp.StatusCode)
	}

	discoverRes := DiscoverResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&discoverRes); err != nil {
		return DiscoverResult{}, fmt.Errorf("failed to decode response: %w", err)
	}

	movieIDs := make([]int, len(discoverRes.Results))
	for i, result := range discoverRes.Results {
		movieIDs[i] = result.ID
	}

	return DiscoverResult{
		MovieIDs:     movieIDs,
		TotalPages:   discoverRes.TotalPages,
		TotalResults: discoverRes.TotalResults,
	}, nil
}

// MovieDetails maps 1:1 onto the widened `movies` table columns. Nested TMDB
// objects are kept as raw JSON so they can be inserted directly into the
// matching JSONB columns.
type MovieDetails struct {
	ID                  int32
	Title               string
	OriginalTitle       sql.NullString
	OriginalLanguage    sql.NullString
	Overview            sql.NullString
	Status              sql.NullString
	Homepage            sql.NullString
	ImdbID              sql.NullString
	ReleaseDate         sql.NullTime
	Year                sql.NullInt32
	Runtime             sql.NullInt32
	Budget              sql.NullInt64
	Revenue             sql.NullInt64
	Popularity          sql.NullFloat64
	VoteAverage         sql.NullFloat64
	VoteCount           sql.NullInt32
	Adult               sql.NullBool
	Video               sql.NullBool
	PosterPath          sql.NullString
	BackdropPath        sql.NullString
	Tagline             sql.NullString
	Genres              pqtype.NullRawMessage
	BelongsToCollection pqtype.NullRawMessage
	ProductionCompanies pqtype.NullRawMessage
	ProductionCountries pqtype.NullRawMessage
	SpokenLanguages     pqtype.NullRawMessage
	OriginCountry       pqtype.NullRawMessage
	CreditsCast         pqtype.NullRawMessage
	CreditsCrew         pqtype.NullRawMessage
	Director            sql.NullString
	Actor1              sql.NullString
	Actor2              sql.NullString
}

type movieDetailsResponse struct {
	ID                  int32           `json:"id"`
	Title               string          `json:"title"`
	OriginalTitle       string          `json:"original_title"`
	OriginalLanguage    string          `json:"original_language"`
	Overview            string          `json:"overview"`
	Status              string          `json:"status"`
	Homepage            *string         `json:"homepage"`
	ImdbID              *string         `json:"imdb_id"`
	ReleaseDate         string          `json:"release_date"`
	Runtime             int32           `json:"runtime"`
	Budget              int64           `json:"budget"`
	Revenue             int64           `json:"revenue"`
	Popularity          float64         `json:"popularity"`
	VoteAverage         float64         `json:"vote_average"`
	VoteCount           int32           `json:"vote_count"`
	Adult               bool            `json:"adult"`
	Video               bool            `json:"video"`
	PosterPath          *string         `json:"poster_path"`
	BackdropPath        *string         `json:"backdrop_path"`
	Tagline             *string         `json:"tagline"`
	Genres              json.RawMessage `json:"genres"`
	BelongsToCollection json.RawMessage `json:"belongs_to_collection"`
	ProductionCompanies json.RawMessage `json:"production_companies"`
	ProductionCountries json.RawMessage `json:"production_countries"`
	SpokenLanguages     json.RawMessage `json:"spoken_languages"`
	OriginCountry       json.RawMessage `json:"origin_country"`
	Credits             struct {
		Cast json.RawMessage `json:"cast"`
		Crew json.RawMessage `json:"crew"`
	} `json:"credits"`
}

// GetMovieDetails fetches /movie/{id} with credits attached, storing
// everything TMDB returns so future changes to daily-pick filtering don't
// require re-hitting TMDB.
func (c *Client) GetMovieDetails(ctx context.Context, id int) (MovieDetails, error) {
	detailsURL := fmt.Sprintf("%s/movie/%d?append_to_response=credits", c.baseURL, id)

	req, err := http.NewRequestWithContext(ctx, "GET", detailsURL, nil)
	if err != nil {
		return MovieDetails{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.readAccessToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return MovieDetails{}, fmt.Errorf("failed to fetch from TMDB: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return MovieDetails{}, fmt.Errorf("TMDB API returned status: %d", resp.StatusCode)
	}

	var raw movieDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return MovieDetails{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return movieDetailsFromResponse(raw)
}

// nullableString treats both a JSON null and TMDB's empty-string sentinel
// for a missing value as SQL NULL.
func nullableString(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullableRawMessage(raw json.RawMessage) pqtype.NullRawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return pqtype.NullRawMessage{}
	}
	return pqtype.NullRawMessage{RawMessage: raw, Valid: true}
}

func movieDetailsFromResponse(raw movieDetailsResponse) (MovieDetails, error) {
	var releaseDate sql.NullTime
	var year sql.NullInt32
	if raw.ReleaseDate != "" {
		t, err := time.Parse(time.DateOnly, raw.ReleaseDate)
		if err != nil {
			return MovieDetails{}, fmt.Errorf("failed to parse release_date %q: %w", raw.ReleaseDate, err)
		}
		releaseDate = sql.NullTime{Time: t, Valid: true}
		year = sql.NullInt32{Int32: int32(t.Year()), Valid: true} // #nosec G115 -- a release year never overflows int32
	}

	director := sql.NullString{}
	if len(raw.Credits.Crew) > 0 {
		var crew []struct {
			Name string `json:"name"`
			Job  string `json:"job"`
		}
		if err := json.Unmarshal(raw.Credits.Crew, &crew); err != nil {
			return MovieDetails{}, fmt.Errorf("failed to decode credits.crew: %w", err)
		}
		for _, member := range crew {
			if member.Job == "Director" {
				director = sql.NullString{String: member.Name, Valid: true}
				break
			}
		}
	}

	actor1, actor2 := sql.NullString{}, sql.NullString{}
	if len(raw.Credits.Cast) > 0 {
		var cast []struct {
			Name  string `json:"name"`
			Order int    `json:"order"`
		}
		if err := json.Unmarshal(raw.Credits.Cast, &cast); err != nil {
			return MovieDetails{}, fmt.Errorf("failed to decode credits.cast: %w", err)
		}
		sort.Slice(cast, func(i, j int) bool { return cast[i].Order < cast[j].Order })
		if len(cast) > 0 {
			actor1 = sql.NullString{String: cast[0].Name, Valid: true}
		}
		if len(cast) > 1 {
			actor2 = sql.NullString{String: cast[1].Name, Valid: true}
		}
	}

	return MovieDetails{
		ID:                  raw.ID,
		Title:               raw.Title,
		OriginalTitle:       nullableString(&raw.OriginalTitle),
		OriginalLanguage:    nullableString(&raw.OriginalLanguage),
		Overview:            nullableString(&raw.Overview),
		Status:              nullableString(&raw.Status),
		Homepage:            nullableString(raw.Homepage),
		ImdbID:              nullableString(raw.ImdbID),
		ReleaseDate:         releaseDate,
		Year:                year,
		Runtime:             sql.NullInt32{Int32: raw.Runtime, Valid: true},
		Budget:              sql.NullInt64{Int64: raw.Budget, Valid: true},
		Revenue:             sql.NullInt64{Int64: raw.Revenue, Valid: true},
		Popularity:          sql.NullFloat64{Float64: raw.Popularity, Valid: true},
		VoteAverage:         sql.NullFloat64{Float64: raw.VoteAverage, Valid: true},
		VoteCount:           sql.NullInt32{Int32: raw.VoteCount, Valid: true},
		Adult:               sql.NullBool{Bool: raw.Adult, Valid: true},
		Video:               sql.NullBool{Bool: raw.Video, Valid: true},
		PosterPath:          nullableString(raw.PosterPath),
		BackdropPath:        nullableString(raw.BackdropPath),
		Tagline:             nullableString(raw.Tagline),
		Genres:              nullableRawMessage(raw.Genres),
		BelongsToCollection: nullableRawMessage(raw.BelongsToCollection),
		ProductionCompanies: nullableRawMessage(raw.ProductionCompanies),
		ProductionCountries: nullableRawMessage(raw.ProductionCountries),
		SpokenLanguages:     nullableRawMessage(raw.SpokenLanguages),
		OriginCountry:       nullableRawMessage(raw.OriginCountry),
		CreditsCast:         nullableRawMessage(raw.Credits.Cast),
		CreditsCrew:         nullableRawMessage(raw.Credits.Crew),
		Director:            director,
		Actor1:              actor1,
		Actor2:              actor2,
	}, nil
}
