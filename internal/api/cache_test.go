package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	srv := fakeAPI(t)
	return NewCache(newTestClient(srv.URL + "/api"))
}

func TestCacheLoad(t *testing.T) {
	c := newTestCache(t)

	if err := c.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(c.artists) != 2 {
		t.Errorf("len(artists) = %d, ожидалось 2", len(c.artists))
	}
	if len(c.locations) != 2 {
		t.Errorf("len(locations) = %d, ожидалось 2", len(c.locations))
	}
	if len(c.dates) != 2 {
		t.Errorf("len(dates) = %d, ожидалось 2", len(c.dates))
	}
	if len(c.relations) != 2 {
		t.Errorf("len(relations) = %d, ожидалось 2", len(c.relations))
	}
}

func TestCacheArtists(t *testing.T) {
	c := newTestCache(t)
	if err := c.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	artists := c.Artists()

	if len(artists) != 2 {
		t.Fatalf("len(Artists()) = %d, ожидалось 2", len(artists))
	}
	if artists[0].Name != "Queen" {
		t.Errorf("artists[0].Name = %q, ожидалось \"Queen\"", artists[0].Name)
	}
	if artists[1].Name != "SOJA" {
		t.Errorf("artists[1].Name = %q, ожидалось \"SOJA\"", artists[1].Name)
	}
}

func TestCacheArtistsBeforeLoad(t *testing.T) {
	c := newTestCache(t)

	if got := c.Artists(); len(got) != 0 {
		t.Errorf("len(Artists()) = %d до Load, ожидалось 0", len(got))
	}
}

func TestCacheArtistFound(t *testing.T) {
	c := newTestCache(t)
	if err := c.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	a, err := c.Artist(2)
	if err != nil {
		t.Fatalf("Artist(2): %v", err)
	}
	if a.Name != "SOJA" {
		t.Errorf("Artist(2).Name = %q, ожидалось \"SOJA\"", a.Name)
	}
	if a.ID != 2 {
		t.Errorf("Artist(2).ID = %d, ожидалось 2", a.ID)
	}
}

func TestCacheArtistNotFound(t *testing.T) {
	c := newTestCache(t)
	if err := c.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, id := range []int{0, -1, 999} {
		_, err := c.Artist(id)
		if err == nil {
			t.Errorf("Artist(%d): ожидалась ошибка, получено nil", id)
			continue
		}
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("Artist(%d): errors.Is(err, ErrNotFound) = false, ошибка: %v", id, err)
		}
	}
}

func TestCacheArtistZeroValueOnError(t *testing.T) {
	c := newTestCache(t)
	if err := c.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	a, err := c.Artist(999)
	if err == nil {
		t.Fatal("ожидалась ошибка")
	}
	if a.ID != 0 || a.Name != "" || a.Members != nil {
		t.Errorf("при ошибке ожидалось нулевое значение, получено %+v", a)
	}
}

func TestCacheLoadFailureKeepsOldData(t *testing.T) {
	srv := fakeAPI(t)
	c := NewCache(newTestClient(srv.URL + "/api"))

	if err := c.Load(context.Background()); err != nil {
		t.Fatalf("первый Load: %v", err)
	}
	before := len(c.Artists())

	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer broken.Close()
	c.client = newTestClient(broken.URL)

	if err := c.Load(context.Background()); err == nil {
		t.Fatal("ожидалась ошибка при недоступном API")
	}

	if got := len(c.Artists()); got != before {
		t.Errorf("после неудачного Load len(Artists()) = %d, ожидалось %d — старые данные потеряны", got, before)
	}
}

func TestCacheLoadErrorMentionsStage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewCache(newTestClient(srv.URL))

	err := c.Load(context.Background())
	if err == nil {
		t.Fatal("ожидалась ошибка")
	}
	if !strings.Contains(err.Error(), "discovering") {
		t.Errorf("ошибка не указывает на этап discovery: %v", err)
	}
}

func TestCacheConcurrentReads(t *testing.T) {
	c := newTestCache(t)
	if err := c.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if len(c.Artists()) != 2 {
				t.Error("Artists() вернул неполные данные")
			}
			if _, err := c.Artist(1); err != nil {
				t.Errorf("Artist(1): %v", err)
			}
		}()
	}
	wg.Wait()
}
