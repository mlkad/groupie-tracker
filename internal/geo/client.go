package geo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const (
	baseURL   = "https://nominatim.openstreetmap.org/search"
	userAgent = "groupie-tracker-geolocalization"
)

var ErrNotFound = errors.New("geo: address not found")

type Coordinates struct {
	Lat float64
	Lon float64
}

type nominateResult struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

type Client struct {
	baseURL     string
	httpClient  *http.Client
	minInterval time.Duration

	mu      sync.Mutex
	cache   map[string]Coordinates
	lastReq time.Time
}

func NewClient() *Client {
	return NewClientWithBaseURL(baseURL)
}

func NewClientWithBaseURL(u string) *Client {
	return &Client{
		baseURL:     u,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		minInterval: time.Second,
		cache:       make(map[string]Coordinates),
	}
}

func (c *Client) Geocode(ctx context.Context, address string) (Coordinates, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	coords, ok := c.cache[address]
	if ok {
		return coords, nil
	}

	wait := c.minInterval - time.Since(c.lastReq)
	if wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return Coordinates{}, ctx.Err()
		}
	}

	params := url.Values{}
	params.Set("q", address)
	params.Set("format", "json")
	params.Set("limit", "1")

	endpoint := c.baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Coordinates{}, fmt.Errorf("creating request for %s: %w", endpoint, err)
	}

	req.Header.Set("User-Agent", userAgent)

	c.lastReq = time.Now()
	resp, err := c.httpClient.Do(req) //отправка запроса
	if err != nil {
		return Coordinates{}, fmt.Errorf("fetching %s: %w", endpoint, err)
	}
	defer resp.Body.Close() //закрыть от утечки
	if resp.StatusCode != http.StatusOK {
		return Coordinates{}, fmt.Errorf("unexpected status: %s in %s", resp.Status, endpoint)
	}

	var results []nominateResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return Coordinates{}, fmt.Errorf("decoding %s: %w", endpoint, err)
	}

	if len(results) == 0 {
		return Coordinates{}, fmt.Errorf("geocode %q: %w", address, ErrNotFound)
	}

	lat, err := strconv.ParseFloat(results[0].Lat, 64)
	if err != nil {
		return Coordinates{}, fmt.Errorf("parsing lat %q: %w", results[0].Lat, err)
	}

	lon, err := strconv.ParseFloat(results[0].Lon, 64)
	if err != nil {
		return Coordinates{}, fmt.Errorf("parsing lon %q: %w", results[0].Lon, err)
	}

	coords = Coordinates{Lat: lat, Lon: lon}
	c.cache[address] = coords
	return coords, nil
}
