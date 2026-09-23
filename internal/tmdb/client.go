package tmdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/sqlc-dev/pqtype"
	"golang.org/x/time/rate"
)

const DefaultBaseURL = "https://api.themoviedb.org/3"
const ImageBaseURL = "https://image.tmdb.org/t/p/w500"

// TMDB doesn't publish a hard rate limit; their docs describe "somewhere in
// the 40 requests per second range" for bulk use, so stay comfortably under
// it rather than trying to hug the exact number.
const requestsPerSecond = 30
const maxRetries = 5
const defaultRetryDelay = 1 * time.Second

type Client struct {
	readAccessToken string
	baseURL         string
	httpClient      *http.Client
	limiter         *rate.Limiter
	// retryDelay is the fallback backoff used when a 429 response carries no
	// Retry-After header. Overridable for tests.
	retryDelay time.Duration
}

func NewClient(readAccessToken string) *Client {
	return &Client{
		readAccessToken: readAccessToken,
		baseURL:         DefaultBaseURL,
		httpClient:      &http.Client{},
		limiter:         rate.NewLimiter(rate.Limit(requestsPerSecond), 1),
		retryDelay:      defaultRetryDelay,
	}
}

// Helper for tests to override the URL
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// Helper for tests to avoid waiting out the real fallback retry delay.
func (c *Client) SetRetryDelay(d time.Duration) {
	c.retryDelay = d
}

// doGet issues a GET request against the given URL with standard TMDB
// headers, throttled to requestsPerSecond and retried on 429 (honouring
// Retry-After when present), and decodes a 200 response into out.
func (c *Client) doGet(ctx context.Context, requestURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.readAccessToken))

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("TMDB API returned status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

// do sends req, waiting on the rate limiter first, and retries on 429 up to
// maxRetries times.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		if err := c.limiter.Wait(req.Context()); err != nil {
			return nil, fmt.Errorf("rate limiter wait failed: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch from TMDB: %w", err)
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		wait := c.retryAfter(resp)
		_ = resp.Body.Close()

		if attempt >= maxRetries {
			return nil, fmt.Errorf("TMDB API returned status 429 after %d retries", attempt)
		}

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(wait):
		}
	}
}

func (c *Client) retryAfter(resp *http.Response) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return c.retryDelay
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

	discoverRes := DiscoverResponse{}
	if err := c.doGet(ctx, discoverURL, &discoverRes); err != nil {
		return DiscoverResult{}, err
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

	var raw movieDetailsResponse
	if err := c.doGet(ctx, detailsURL, &raw); err != nil {
		return MovieDetails{}, err
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
