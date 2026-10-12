package api

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mlkad/groupie-tracker/internal/models"
)

//чтоб не ходить в хероку на каждый запрос

var ErrNotFound = errors.New("api: artist not found")

type Cache struct {
	mu        sync.RWMutex
	client    *Client
	artists   []models.Artist
	locations []models.Location
	dates     []models.Date
	relations []models.Relation
}

type ArtistFilter struct {
	CreationFrom int
	CreationTo   int

	AlbumFrom int
	AlbumTo   int

	MembersFrom int
	MembersTo   int

	Locations []string

	MembersExact int
}

type SearchResult struct {
	Artist models.Artist
	Types  []string
}

const (
	searchTypeBand         = "band"
	searchTypeMember       = "member"
	searchTypeLocation     = "location"
	searchTypeCreationDate = "creation_date"
	searchTypeFirstAlbum   = "first_album"
)

func NewCache(client *Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) Load(ctx context.Context) error {
	if err := c.client.Discover(ctx); err != nil {
		return fmt.Errorf("discovering api: %w", err)
	}

	artists, err := c.client.FetchArtists(ctx)
	if err != nil {
		return fmt.Errorf("loading artists: %w", err)
	}

	locations, err := c.client.FetchLocations(ctx)
	if err != nil {
		return fmt.Errorf("loading locations: %w", err)
	}

	dates, err := c.client.FetchDates(ctx)
	if err != nil {
		return fmt.Errorf("loading dates: %w", err)
	}

	relations, err := c.client.FetchRelations(ctx)
	if err != nil {
		return fmt.Errorf("loading relations: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.artists = artists
	c.locations = locations
	c.dates = dates
	c.relations = relations

	return nil
}

// возвращает всех артистов из кэша
func (c *Cache) Artists() []models.Artist {
	c.mu.RLock()         //я сейчас читаю данные, другие читатели пусть читают тоже, а вот писателю надо подождать
	defer c.mu.RUnlock() //открыть замок
	return c.artists
}

func (c *Cache) Relation(id int) (models.Relation, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, r := range c.relations {
		if r.ID == id {
			return r, nil
		}
	}
	return models.Relation{}, ErrNotFound
}

func (c *Cache) Artist(id int) (models.Artist, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, a := range c.artists {
		if a.ID == id {
			return a, nil
		}
	}
	return models.Artist{}, ErrNotFound
}

func (c *Cache) Search(q string) []SearchResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return nil
	}

	found := make([]SearchResult, 0)

	//фильтр по именам артистов
	for _, a := range c.artists {
		types := make([]string, 0, 5)

		if containsIgnoreCase(a.Name, q) {
			types = append(types, searchTypeBand)
		}

		if containsAnyIgnoreCase(a.Members, q) {
			types = append(types, searchTypeMember)
		}

		if c.locationMatches(a.ID, q) {
			types = append(types, searchTypeLocation)
		}

		if strings.Contains(strconv.Itoa(a.CreationDate), q) {
			types = append(types, searchTypeCreationDate)
		}

		if containsIgnoreCase(a.FirstAlbum, q) {
			types = append(types, searchTypeFirstAlbum)
		}

		if len(types) > 0 {
			found = append(found, SearchResult{
				Artist: a,
				Types:  types,
			})
		}
	}
	return found
}

func containsIgnoreCase(value, query string) bool {
	return strings.Contains(strings.ToLower(value), query)
}

func containsAnyIgnoreCase(values []string, query string) bool {
	for _, value := range values {
		if containsIgnoreCase(value, query) {
			return true
		}
	}
	return false
}

func (c *Cache) locationMatches(artistID int, query string) bool {
	for _, location := range c.locations {
		if location.ID == artistID {
			return containsAnyIgnoreCase(location.Locations, query)
		}
	}
	return false
}

func firstAlbumInRange(date string, from, to int) bool {
	parts := strings.Split(date, "-")
	if len(parts) != 3 {
		return false
	}

	year, err := strconv.Atoi(parts[2])
	if err != nil {
		return false
	}

	return year >= from && year <= to
}

func (c *Cache) Filter(f ArtistFilter) []models.Artist {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make([]models.Artist, 0)

	for _, artist := range c.artists {
		if artist.CreationDate < f.CreationFrom || artist.CreationDate > f.CreationTo {
			continue
		}

		if f.MembersExact > 0 {
			if len(artist.Members) != f.MembersExact {
				continue
			}
		} else if len(artist.Members) < f.MembersFrom || len(artist.Members) > f.MembersTo {
			continue
		}

		if !firstAlbumInRange(artist.FirstAlbum, f.AlbumFrom, f.AlbumTo) {
			continue
		}

		if len(f.Locations) != 0 {
			matched := false
			for _, loc := range f.Locations {
				if c.hasLocation(artist.ID, loc) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		res = append(res, artist)
	}

	return res
}

func (c *Cache) hasLocation(id int, query string) bool {
	query = strings.ToLower(query)

	for _, location := range c.locations {
		if location.ID != id {
			continue
		}

		for _, loc := range location.Locations {
			if strings.Contains(strings.ToLower(loc), query) {
				return true
			}
		}
		return false
	}
	return false
}

func (c *Cache) AllLocations() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	seen := make(map[string]bool)
	for _, l := range c.locations {
		for _, loc := range l.Locations {
			seen[loc] = true
		}
	}

	out := make([]string, 0, len(seen))
	for loc := range seen {
		out = append(out, loc)
	}
	sort.Strings(out)

	return out
}
