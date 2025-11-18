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

const BaseURL = "https://api.themoviedb.org/3"
const ImageBaseURL = "https://image.tmdb.org/t/p/w500"

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

type SearchResponse struct {
	Results []struct {
		PosterPath string `json:"poster_path"`
	} `json:"results"`
}

func (c *Client) SearchMovie(ctx context.Context, title string, year int) (string, error) {
	searchURL := fmt.Sprintf("%s/search/movie?query=%s&year=%d",
		BaseURL,
		url.QueryEscape(title),
		year,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

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
