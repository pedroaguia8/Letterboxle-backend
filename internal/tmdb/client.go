package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
