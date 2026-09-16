package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mlkad/groupie-tracker/internal/models"
)

const baseURL = "https://groupietrackers.herokuapp.com/api"

type endpoints struct {
	Artists   string `json:"artists"`
	Locations string `json:"locations"`
	Dates     string `json:"dates"`
	Relation  string `json:"relation"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	endpoints  endpoints
}

func NewClient() *Client {
	return NewClientWithBaseURL(baseURL)
}

func NewClientWithBaseURL(url string) *Client {
	return &Client{
		baseURL:    url,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) get(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request for %s: %w", url, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s for %s", resp.Status, url)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decoding %s: %w", url, err)
	}
	return nil
}

func (c *Client) Discover(ctx context.Context) error {
	return c.get(ctx, c.baseURL, &c.endpoints)
}

func (c *Client) FetchArtists(ctx context.Context) ([]models.Artist, error) {
	var artists []models.Artist
	if err := c.get(ctx, c.endpoints.Artists, &artists); err != nil {
		return nil, err
	}
	return artists, nil
}

func (c *Client) FetchLocations(ctx context.Context) ([]models.Location, error) {
	var locations models.LocationsResponse
	if err := c.get(ctx, c.endpoints.Locations, &locations); err != nil {
		return nil, err
	}
	return locations.Index, nil
}

func (c *Client) FetchDates(ctx context.Context) ([]models.Date, error) {
	var dates models.DatesResponse
	if err := c.get(ctx, c.endpoints.Dates, &dates); err != nil {
		return nil, err
	}
	return dates.Index, nil
}

func (c *Client) FetchRelations(ctx context.Context) ([]models.Relation, error) {
	var rel models.RelationResponse
	if err := c.get(ctx, c.endpoints.Relation, &rel); err != nil {
		return nil, err
	}
	return rel.Index, nil
}
