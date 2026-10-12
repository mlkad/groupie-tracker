package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}
}

func fakeAPI(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	var srv *httptest.Server

	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"artists":   "` + srv.URL + `/api/artists",
			"locations": "` + srv.URL + `/api/locations",
			"dates":     "` + srv.URL + `/api/dates",
			"relation":  "` + srv.URL + `/api/relation"
		}`))
	})

	mux.HandleFunc("/api/artists", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[
			{"id":1,"image":"queen.jpeg","name":"Queen",
			 "members":["Freddie Mercury","Brian May","John Daecon","Roger Meddows-Taylor","Mike Grose","Barry Mitchell","Doug Fogie"],
			 "creationDate":1970,"firstAlbum":"14-12-1973",
			 "locations":"` + srv.URL + `/api/locations/1",
			 "concertDates":"` + srv.URL + `/api/dates/1",
			 "relations":"` + srv.URL + `/api/relation/1"},
			{"id":2,"image":"soja.jpeg","name":"SOJA",
			 "members":["Jacob Hemphill","Bob Jefferson"],
			 "creationDate":1997,"firstAlbum":"28-07-2002",
			 "locations":"` + srv.URL + `/api/locations/2",
			 "concertDates":"` + srv.URL + `/api/dates/2",
			 "relations":"` + srv.URL + `/api/relation/2"}
		]`))
	})

	mux.HandleFunc("/api/locations", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"index":[
			{"id":1,"locations":["north_carolina-usa","georgia-usa"],"dates":"` + srv.URL + `/api/dates/1"},
			{"id":2,"locations":["playa_del_carmen-mexico"],"dates":"` + srv.URL + `/api/dates/2"}
		]}`))
	})

	mux.HandleFunc("/api/dates", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"index":[
			{"id":1,"dates":["*23-08-2019","*22-08-2019"]},
			{"id":2,"dates":["*05-12-2019"]}
		]}`))
	})

	mux.HandleFunc("/api/relation", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"index":[
			{"id":1,"datesLocations":{"georgia-usa":["22-08-2019"],"north_carolina-usa":["23-08-2019"]}},
			{"id":2,"datesLocations":{"playa_del_carmen-mexico":["05-12-2019","06-12-2019"]}}
		]}`))
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestDiscover(t *testing.T) {
	srv := fakeAPI(t)
	c := newTestClient(srv.URL + "/api")

	if err := c.Discover(context.Background()); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if c.endpoints.Artists != srv.URL+"/api/artists" {
		t.Errorf("endpoints.Artists = %q", c.endpoints.Artists)
	}
	if c.endpoints.Locations == "" || c.endpoints.Dates == "" || c.endpoints.Relation == "" {
		t.Errorf("endpoints заполнены не полностью: %+v", c.endpoints)
	}
}

func TestFetchArtists(t *testing.T) {
	srv := fakeAPI(t)
	c := newTestClient(srv.URL + "/api")
	ctx := context.Background()

	if err := c.Discover(ctx); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	artists, err := c.FetchArtists(ctx)
	if err != nil {
		t.Fatalf("FetchArtists: %v", err)
	}

	if len(artists) != 2 {
		t.Fatalf("len(artists) = %d, ожидалось 2", len(artists))
	}
	if artists[0].Name != "Queen" {
		t.Errorf("artists[0].Name = %q, ожидалось \"Queen\"", artists[0].Name)
	}
	if artists[0].CreationDate != 1970 {
		t.Errorf("artists[0].CreationDate = %d, ожидалось 1970", artists[0].CreationDate)
	}
	if len(artists[0].Members) != 7 {
		t.Errorf("len(artists[0].Members) = %d, ожидалось 7", len(artists[0].Members))
	}
}

func TestFetchLocations(t *testing.T) {
	srv := fakeAPI(t)
	c := newTestClient(srv.URL + "/api")
	ctx := context.Background()

	if err := c.Discover(ctx); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	locations, err := c.FetchLocations(ctx)
	if err != nil {
		t.Fatalf("FetchLocations: %v", err)
	}

	if len(locations) != 2 {
		t.Fatalf("len(locations) = %d, ожидалось 2 (обёртка index развёрнута?)", len(locations))
	}
	if len(locations[0].Locations) != 2 {
		t.Errorf("len(locations[0].Locations) = %d, ожидалось 2", len(locations[0].Locations))
	}
	if locations[0].Locations[0] != "north_carolina-usa" {
		t.Errorf("locations[0].Locations[0] = %q", locations[0].Locations[0])
	}
}

func TestFetchDates(t *testing.T) {
	srv := fakeAPI(t)
	c := newTestClient(srv.URL + "/api")
	ctx := context.Background()

	if err := c.Discover(ctx); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	dates, err := c.FetchDates(ctx)
	if err != nil {
		t.Fatalf("FetchDates: %v", err)
	}

	if len(dates) != 2 {
		t.Fatalf("len(dates) = %d, ожидалось 2", len(dates))
	}
	if len(dates[0].Dates) != 2 {
		t.Errorf("len(dates[0].Dates) = %d, ожидалось 2", len(dates[0].Dates))
	}
}

func TestFetchRelations(t *testing.T) {
	srv := fakeAPI(t)
	c := newTestClient(srv.URL + "/api")
	ctx := context.Background()

	if err := c.Discover(ctx); err != nil {
		t.Fatalf("Discover: %v", err)
	}

	relations, err := c.FetchRelations(ctx)
	if err != nil {
		t.Fatalf("FetchRelations: %v", err)
	}

	if len(relations) != 2 {
		t.Fatalf("len(relations) = %d, ожидалось 2", len(relations))
	}
	if len(relations[0].DatesLocations) != 2 {
		t.Fatalf("len(relations[0].DatesLocations) = %d, ожидалось 2", len(relations[0].DatesLocations))
	}
	got := relations[1].DatesLocations["playa_del_carmen-mexico"]
	if len(got) != 2 {
		t.Errorf("DatesLocations[\"playa_del_carmen-mexico\"] = %v, ожидалось 2 даты", got)
	}
}

func TestGetServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)

	err := c.Discover(context.Background())
	if err == nil {
		t.Fatal("ожидалась ошибка при статусе 500, получено nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("ошибка не упоминает код статуса: %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)

	if err := c.Discover(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка при статусе 404, получено nil")
	}
}

func TestGetMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"artists": `))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)

	err := c.Discover(context.Background())
	if err == nil {
		t.Fatal("ожидалась ошибка при битом JSON, получено nil")
	}
	if !strings.Contains(err.Error(), "decoding") {
		t.Errorf("ошибка не помечена как ошибка разбора: %v", err)
	}
}

func TestGetHTMLInsteadOfJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><body>maintenance</body></html>`))
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)

	if err := c.Discover(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка при HTML вместо JSON, получено nil")
	}
}

func TestGetUnreachableHost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	c := newTestClient(url)

	if err := c.Discover(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка при недоступном хосте, получено nil")
	}
}

func TestGetInvalidURL(t *testing.T) {
	c := newTestClient("://not-a-url")

	err := c.Discover(context.Background())
	if err == nil {
		t.Fatal("ожидалась ошибка при некорректном URL, получено nil")
	}
	if !strings.Contains(err.Error(), "creating request") {
		t.Errorf("ошибка не помечена как ошибка создания запроса: %v", err)
	}
}

func TestGetContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := c.Discover(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("ожидалась ошибка при отменённом контексте, получено nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("errors.Is(err, context.DeadlineExceeded) = false, ошибка: %v", err)
	}
	if elapsed > 300*time.Millisecond {
		t.Errorf("запрос не был отменён вовремя: занял %v", elapsed)
	}
}

func TestFetchWithoutDiscover(t *testing.T) {
	c := newTestClient("http://example.invalid/api")

	if _, err := c.FetchArtists(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка при вызове FetchArtists без Discover, получено nil")
	}
}
