package geo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(u string) *Client {
	c := NewClientWithBaseURL(u)
	c.minInterval = 0
	c.httpClient = &http.Client{Timeout: 2 * time.Second}
	return c
}

func fakeGeo(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGeocodeOK(t *testing.T) {
	srv := fakeGeo(t, http.StatusOK, `[{"lat":"49.9929","lon":"8.2473"}]`)
	c := newTestClient(srv.URL)

	got, err := c.Geocode(context.Background(), "Mainz, Germany")
	if err != nil {
		t.Fatalf("Geocode: %v", err)
	}

	if got.Lat != 49.9929 {
		t.Errorf("Lat = %v, ожидалось 49.9929", got.Lat)
	}
	if got.Lon != 8.2473 {
		t.Errorf("Lon = %v, ожидалось 8.2473", got.Lon)
	}
}

func TestGeocodeNotFound(t *testing.T) {
	srv := fakeGeo(t, http.StatusOK, `[]`)
	c := newTestClient(srv.URL)

	_, err := c.Geocode(context.Background(), "asdfgh")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("errors.Is(err, ErrNotFound) = false, ошибка: %v", err)
	}
}

func TestGeocodeBadStatus(t *testing.T) {
	srv := fakeGeo(t, http.StatusInternalServerError, `oops`)
	c := newTestClient(srv.URL)

	if _, err := c.Geocode(context.Background(), "Mainz, Germany"); err == nil {
		t.Error("Geocode вернул nil, ожидалась ошибка из-за статуса 500")
	}
}

func TestGeocodeBadJSON(t *testing.T) {
	srv := fakeGeo(t, http.StatusOK, `not json`)
	c := newTestClient(srv.URL)

	if _, err := c.Geocode(context.Background(), "Mainz, Germany"); err == nil {
		t.Error("Geocode вернул nil, ожидалась ошибка декодирования")
	}
}

func TestGeocodeBadCoordinates(t *testing.T) {
	tests := map[string]string{
		"битая широта":  `[{"lat":"abc","lon":"8.2473"}]`,
		"битая долгота": `[{"lat":"49.9929","lon":"xyz"}]`,
	}

	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			srv := fakeGeo(t, http.StatusOK, body)
			c := newTestClient(srv.URL)

			if _, err := c.Geocode(context.Background(), "Mainz, Germany"); err == nil {
				t.Error("Geocode вернул nil, ожидалась ошибка ParseFloat")
			}
		})
	}
}

func TestGeocodeCache(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`[{"lat":"49.9929","lon":"8.2473"}]`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(srv.URL)
	ctx := context.Background()

	first, err := c.Geocode(ctx, "Mainz, Germany")
	if err != nil {
		t.Fatalf("первый Geocode: %v", err)
	}

	second, err := c.Geocode(ctx, "Mainz, Germany")
	if err != nil {
		t.Fatalf("второй Geocode: %v", err)
	}

	if calls != 1 {
		t.Errorf("запросов к серверу = %d, ожидался 1 (второй должен взяться из кэша)", calls)
	}
	if first != second {
		t.Errorf("из кэша пришло %+v, ожидалось %+v", second, first)
	}
}

func TestGeocodeSendsQuery(t *testing.T) {
	var gotQuery, gotUA, gotFormat, gotLimit string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		gotFormat = r.URL.Query().Get("format")
		gotLimit = r.URL.Query().Get("limit")
		gotUA = r.Header.Get("User-Agent")
		w.Write([]byte(`[{"lat":"49.9929","lon":"8.2473"}]`))
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(srv.URL)
	if _, err := c.Geocode(context.Background(), "Mainz, Germany"); err != nil {
		t.Fatalf("Geocode: %v", err)
	}

	if gotQuery != "Mainz, Germany" {
		t.Errorf("q = %q, ожидалось \"Mainz, Germany\"", gotQuery)
	}
	if gotFormat != "json" {
		t.Errorf("format = %q, ожидалось \"json\"", gotFormat)
	}
	if gotLimit != "1" {
		t.Errorf("limit = %q, ожидалось \"1\"", gotLimit)
	}
	if gotUA != userAgent {
		t.Errorf("User-Agent = %q, ожидалось %q", gotUA, userAgent)
	}
}

func TestGeocodeCanceledContext(t *testing.T) {
	srv := fakeGeo(t, http.StatusOK, `[{"lat":"49.9929","lon":"8.2473"}]`)

	c := NewClientWithBaseURL(srv.URL)
	c.lastReq = time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := c.Geocode(ctx, "Mainz, Germany"); !errors.Is(err, context.Canceled) {
		t.Errorf("errors.Is(err, context.Canceled) = false, ошибка: %v", err)
	}
}

func TestGeocodeConcurrent(t *testing.T) {
	srv := fakeGeo(t, http.StatusOK, `[{"lat":"49.9929","lon":"8.2473"}]`)
	c := newTestClient(srv.URL)

	addresses := []string{"Mainz, Germany", "London, UK", "Mainz, Germany", "Berlin, Germany"}
	done := make(chan struct{})

	for _, addr := range addresses {
		go func(a string) {
			defer func() { done <- struct{}{} }()
			if _, err := c.Geocode(context.Background(), a); err != nil {
				t.Errorf("Geocode(%q): %v", a, err)
			}
		}(addr)
	}

	for range addresses {
		<-done
	}
}
